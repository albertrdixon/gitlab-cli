package main

import (
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

func findProject() chan *gitlab.Project {
	var (
		c   = make(chan *gitlab.Project, 1)
		gc  = newGitlabClient()
		lpo = gitlab.ListProjectsOptions{
			ListOptions: gitlab.ListOptions{Page: 1, PerPage: 50},
			Membership:  newBool(true),
		}
	)

	go func(client *gitlab.Client, opts gitlab.ListProjectsOptions, ch chan *gitlab.Project) {
		defer close(ch)

		for {
			projects, resp, er := client.Projects.ListProjects(&opts)
			if er != nil {
				log.Error().Err(er).Msg("Error communicating with Gitlab")
				return
			}

			for _, p := range projects {
				log.Debug().Str("name", p.Name).Str("path", p.Path).Msg("Looking at project")
				if p.Name == *projectName || strings.Contains(p.Path, *projectName) {
					ch <- p
					return
				}
			}
			if resp.CurrentPage >= resp.TotalPages {
				return
			}
			opts.Page = resp.NextPage
		}

	}(gc, lpo, c)
	return c
}

func findRegistry(pid int) chan *gitlab.RegistryRepository {
	var (
		c  = make(chan *gitlab.RegistryRepository, 1)
		gc = newGitlabClient()
		lo = gitlab.ListRegistryRepositoriesOptions(gitlab.ListOptions{Page: 1, PerPage: 50})
	)
	go func(projectID int, client *gitlab.Client, opts gitlab.ListRegistryRepositoriesOptions, ch chan *gitlab.RegistryRepository) {
		defer close(ch)

		for {
			repos, resp, er := client.ContainerRegistry.ListRegistryRepositories(projectID, &opts)
			if er != nil {
				log.Error().Err(er).Msg("Error communicating with Gitlab")
				return
			}

			for _, r := range repos {
				log.Debug().Str("name", r.Name).Msg("Looking at registry")
				switch r.Name {
				case *regCmdRegName:
					c <- r
					return
				case "":
					if *regCmdRegName == "default" {
						c <- r
						return
					}
				}
			}

			if resp.CurrentPage >= resp.TotalPages {
				return
			}
			opts.Page = resp.NextPage
		}
	}(pid, gc, lo, c)
	return c
}

func getAllRegistryTags(projectID, registryID int) []gitlab.RegistryRepositoryTag {
	var (
		tags = make([]gitlab.RegistryRepositoryTag, 0)
		ch   = make(chan gitlab.RegistryRepositoryTag)
		gc   = newGitlabClient()
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

func listImageTags(projectName, registryName string) error {
	log.Info().Msg("Searching for project")
	project, ok := <-findProject()
	if !ok && project == nil {
		return errors.New("did not find project")
	}
	log.Info().Msg("Found project")

	log.Info().Msg("Searching for registry")
	registry, ok := <-findRegistry(project.ID)
	if !ok && registry == nil {
		return errors.New("did not find registry")
	}
	log.Info().Msg("found registry")

	tags := getAllRegistryTags(project.ID, registry.ID)
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
