package registry

import (
	"os"
	"sort"
	"sync"
	"time"

	"github.com/albertrdixon/gitlab-cli/util"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

func ListImageTags(projectName, registryName string) error {
	gc, er := util.NewGitlabClient()
	if er != nil {
		return er
	}

	project, er := util.GetProject(gc, projectName)
	if er != nil {
		return er
	}

	registry, er := util.GetRegistry(gc, project.ID, registryName)
	if er != nil {
		return er
	}

	tags := getAllRegistryTags(gc, project.ID, registry.ID)
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Tag", "Location", "Created At"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	for _, t := range tags {
		created := "n/a"
		if t.CreatedAt != nil {
			created = t.CreatedAt.Format(time.Stamp)
		}
		table.Append([]string{
			t.Name,
			t.Location,
			created,
		})
	}
	table.Render()
	return nil
}

func getAllRegistryTags(gc *gitlab.Client, projectID, registryID int) []gitlab.RegistryRepositoryTag {
	var (
		tags = make([]gitlab.RegistryRepositoryTag, 0)
		ch   = make(chan gitlab.RegistryRepositoryTag)
		lo   = gitlab.ListRegistryRepositoryTagsOptions(gitlab.ListOptions{Page: 1, PerPage: 20})
	)

	go func(client *gitlab.Client, opts gitlab.ListRegistryRepositoryTagsOptions, pid, rid int, c chan gitlab.RegistryRepositoryTag) {
		defer close(c)
		var wg sync.WaitGroup
		for {
			images, resp, er := gc.ContainerRegistry.ListRegistryRepositoryTags(pid, rid, &opts)
			if er != nil {
				log.Error().Err(er).Msg("Error communicating with Gitlab")
				return
			}

			for _, img := range images {
				wg.Add(1)
				go func(p, r int, name string, c chan gitlab.RegistryRepositoryTag) {
					defer wg.Done()
					log.Debug().Str("image-name", name).Msg("Getting image detail")
					detail, _, er := gc.ContainerRegistry.GetRegistryRepositoryTagDetail(p, r, name)
					if er != nil {
						log.Error().Err(er).Msg("Error communicating with Gitlab")
						return
					}
					c <- *detail
				}(pid, rid, img.Name, c)
			}
			if resp.CurrentPage >= resp.TotalPages {
				break
			}
			opts.Page = resp.NextPage
		}
		wg.Wait()
	}(gc, lo, projectID, registryID, ch)

	for tag := range ch {
		log.Debug().Interface("image-tag", tag).Msg("Got image tag")
		tags = append(tags, tag)
	}

	sort.SliceStable(tags, func(i, j int) bool {
		b, a := tags[i].CreatedAt, tags[j].CreatedAt
		if b == nil || a == nil {
			return false
		}
		return b.After(*a)
	})
	return tags
}
