package main

import (
	"os"

	"github.com/albertrdixon/gitlab-cli/file"
	"github.com/albertrdixon/gitlab-cli/registry"
	"github.com/albertrdixon/gitlab-cli/util"

	"github.com/rs/zerolog"

	"github.com/rs/zerolog/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

const defaultURL = "https://gitlab.com/"

var (
	app         = kingpin.New("gitlab", "A CLI for the Gitlab API").DefaultEnvars()
	gitlabToken = app.Flag("token", "Gitlab API token").String()
	debug       = app.Flag("debug", "Debug log output").Bool()
	quiet       = app.Flag("quiet", "Only error logs").Bool()
	baseURL     = app.Flag("base-url", "Gitlab base url").Default(defaultURL).URL()
	projectName = app.Flag("project-name", "Project name").Short('p').Required().String()

	regCmd        = app.Command("registry", "registry commands")
	regCmdRegName = regCmd.Flag("registry-name", "Registry name").Short('r').Default("default").String()

	regCmdListTagsCmd = regCmd.Command("list-tags", "list image tags")

	fileCmd         = app.Command("get-file", "download a repo file")
	fileCmdRef      = fileCmd.Flag("file-ref", "the name of the branch, tag or commit desired").Short('r').Default("master").String()
	fileCmdOutPath  = fileCmd.Flag("file-output", "output file for downloaded content").Short('O').String()
	fileCmdFilePath = fileCmd.Arg("file-path", "file path in repo").Required().String()
)

func configure() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if *quiet {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}

	util.Configure((*baseURL).String(), *gitlabToken)
}

func main() {
	app.Version(shortVersion())
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	configure()

	switch cmd {
	case regCmdListTagsCmd.FullCommand():
		log.Logger = log.With().Str("project-name", *projectName).Str("registry-name", *regCmdRegName).Logger()
		if er := registry.ListImageTags(*projectName, *regCmdRegName); er != nil {
			log.Error().Err(er).Msg("Failed to list tags")
			os.Exit(1)
		}
	case fileCmd.FullCommand():
		log.Logger = log.With().Str("project-name", *projectName).Str("file-path", *fileCmdFilePath).Logger()
		content, er := file.GetRepoFileContent(*projectName, *fileCmdFilePath, *fileCmdRef)
		if er != nil {
			log.Error().Err(er).Msg("Failed to download file")
			os.Exit(1)
		}

		if er := util.WriteFile(*fileCmdFilePath, *fileCmdOutPath, content); er != nil {
			log.Error().Err(er).Msg("Failed to write file content")
			os.Exit(1)
		}
	}
}
