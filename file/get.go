package file

import (
	"github.com/albertrdixon/gitlab-cli/util"
	"github.com/xanzy/go-gitlab"
)

func GetRepoFileContent(projectName, filePath, ref string) ([]byte, error) {
	gc, er := util.NewGitlabClient()
	if er != nil {
		return []byte(nil), er
	}

	project, er := util.GetProject(gc, projectName)
	if er != nil {
		return []byte(nil), er
	}

	content, _, er := gc.RepositoryFiles.GetRawFile(project.ID, filePath, &gitlab.GetRawFileOptions{Ref: &ref})
	return content, er
}
