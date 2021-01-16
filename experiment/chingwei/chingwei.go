package chingwei

import (
	"github.com/sirupsen/logrus"
	k8s.io/test-infra/prow/github"
	docker "github.com/fsouza/go-dockerclient"
)

type githubClient interface {
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
}

func Reproducing(log *logrus.Entry, ghc githubClient) error {
	log.Info("Staring search pull request.")
	filter := `repo:"tidb" label:"status/needs-reproduction"`

	issues, err := ghc.FindIssues(filter, "", false)
	if err != nil {
		return err
	}

	// only reproduce first issue
	issue := issues[0]
	// parse minimal reproduce step and version from issue
	query, mysqlVersion, tidbVersion, expected, actual := parseIssue(issue)

	// try send a SQL query to tidb with a specific version
	tidbInfo, err := PrepareTiDB(tidbVersion)
	mysqlInfo, err := PrepareMySQL(mysqlVersion)

	// reproduce by connecting to tidb and mysql
	tidbOutput, tidbErr := Reproduce(tidbInfo, query)
	mysqlOutput, mysqlErr := Reproduce(mysqlInfo, query)
   
	// Feedback to issue.
	// diff expected v.s. mysqlOutput
	// diff actual v.s. tidbOutput into folded section
	return nil
}

type DBConnInfo struct {
	Host string
	Port string
	User string
	Database string
}

func Reproduce(dbconninfo DBConnInfo, query string) (string, error) {
	
	return 
}