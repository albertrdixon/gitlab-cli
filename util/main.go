package util

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

const defaultUrl = "https://gitlab.com/"

var (
	fs = afero.NewOsFs()

	url   string
	token string
)

func newBool(b bool) *bool { return &b }

func Configure(u, t string) {
	if u == "" {
		u = defaultUrl
	}
	url, token = u, t
}

func NewGitlabClient() (*gitlab.Client, error) {
	log.Debug().Str("url", url).Msg("creating gitlab client")
	gc := gitlab.NewClient(nil, token)
	return gc, gc.SetBaseURL(url)
}

func GetProject(gc *gitlab.Client, name string) (*gitlab.Project, error) {
	log.Debug().Msg("searching for project")
	project, ok := <-findProject(gc, name)
	if !ok && project == nil {
		return nil, errors.New("did not find project")
	}
	log.Debug().Msg("Found project")
	return project, nil
}

func GetRegistry(gc *gitlab.Client, pid int, name string) (*gitlab.RegistryRepository, error) {
	log.Debug().Msg("searching for registry")
	registry, ok := <-findRegistry(gc, pid, name)
	if !ok && registry == nil {
		return nil, errors.New("did not find registry")
	}
	log.Debug().Msg("found registry")
	return registry, nil
}

func findProject(gc *gitlab.Client, name string) chan *gitlab.Project {
	var (
		c   = make(chan *gitlab.Project, 1)
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
				if p.Name == name || strings.Contains(p.Path, name) {
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

func findRegistry(gc *gitlab.Client, pid int, name string) chan *gitlab.RegistryRepository {
	var (
		c  = make(chan *gitlab.RegistryRepository, 1)
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
				case name:
					c <- r
					return
				case "":
					if name == "default" {
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

func WriteFile(src, dest string, content []byte) error {
	switch dest {
	case "-":
		fmt.Println(string(content))
	case "":
		dest = filepath.Base(src)
		fallthrough
	default:
		if er := afero.WriteFile(fs, dest, content, 0644); er != nil {
			return er
		}
	}
	return nil
}
