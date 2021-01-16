package chingwei

import (
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
)

type githubClient interface {
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
}

func Reproducing(log *logrus.Entry, ghc githubClient) error {
	log.Info("Staring search pull request.")
	query := `repo:"tidb" label:"status/needs-reproduction"`

	issues, err := ghc.FindIssues(query, "", false)
	if err != nil {
		return err
	}

	for _, issue := range issues {
		log.WithField("issue", issue).Info("Get issue success.")
	}

	return nil
}
