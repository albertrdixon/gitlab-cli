package main

import (
	"os"

	"github.com/rs/zerolog"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app          = kingpin.New("gitlab", "A CLI for the Gitlab API").DefaultEnvars()
	debug        = app.Flag("debug", "Debug log output").Bool()
	baseURL      = app.Flag("base-url", "Gitlab base url").Default("https://gitlab.com/").URL()
	projectOwner = app.Flag("project-owner", "Project Owner").Short('O').Default("lumoslabs").String()
	projectName  = app.Flag("project-name", "Project name").Short('p').Required().String()

	regCmd        = app.Command("registry", "registry commands")
	regCmdRegName = regCmd.Flag("registry-name", "Registry name").Short('r').Default("default").String()

	regCmdListTagsCmd      = regCmd.Command("list-tags", "list image tags")
	regCmdGetTagCmd        = regCmd.Command("get-tag", "get image tag details")
	regCmdGetTagCmdTagName = regCmdGetTagCmd.Arg("tag", "registry tag to get").Required().String()
)

func newBool(b bool) *bool { return &b }

func newGitlabClient() *gitlab.Client {
	gc := gitlab.NewClient(nil, os.Getenv("GITLAB_TOKEN"))
	gc.SetBaseURL((*baseURL).String())
	return gc
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	app.Version(shortVersion())
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	switch cmd {
	case regCmdListTagsCmd.FullCommand():
		log.Logger = log.With().Str("project-name", *projectName).Str("registry-name", *regCmdRegName).Logger()
		if er := listImageTags(*projectName, *regCmdRegName); er != nil {
			log.Error().Err(er).Msg("Failed to list tags")
			os.Exit(1)
		}
	case regCmdGetTagCmd.FullCommand():
	}
}
