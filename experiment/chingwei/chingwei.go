package chingwei

import (
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
	"strings"
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

	if tidbErr != nil {
		panic(tidbErr)
	} else if mysqlErr != nil {
		panic(mysqlErr)
	}

	// Feedback to issue.
	// diff expected v.s. mysqlOutput
	// diff actual v.s. tidbOutput into folded section
	_ = DiffSubmittedAndExecuted(expected, mysqlOutput)
	_ = DiffSubmittedAndExecuted(actual, tidbOutput)

	// compare result
	_ = CompareResult(mysqlOutput, tidbOutput)

	return nil
}

type DBConnInfo struct {
	Host     string
	Port     string
	User     string
	Database string
}

func Reproduce(dbconninfo DBConnInfo, query string) (string, error) {

	return "", nil
}

// mock diff
func DiffSubmittedAndExecuted(submitted string, executed string) string {
	return "no differences"
}

// mock simple comparison
func CompareResult(expected string, actual string) bool {
	return strings.Compare(expected, actual) == 0
}
