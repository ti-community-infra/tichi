package chingwei

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
)

type githubClient interface {
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
}

func Reproducing(log *logrus.Entry, ghc githubClient) error {
	log.Info("Staring search pull request.")
	// filter := `repo:"tidb" label:"status/needs-reproduction"`

	// issues, err := ghc.FindIssues(filter, "", false)
	// if err != nil {
	// 	return err
	// }
	// var issue

	// only reproduce first issue
	var issue github.Issue
	// parse minimal reproduce step and version from issue
	query, mysqlVersion, tidbVersion, expected, actual := parseIssue(issue)

	// try send a SQL query to tidb with a specific version
	tidbInfo, err := PrepareTiDB(tidbVersion)
	if err != nil {
		panic(err)
	}
	_ = mysqlVersion
	// mysqlInfo, err := PrepareMySQL(mysqlVersion)
	// if err != nil {
	// 	panic(err)
	// }

	// reproduce by connecting to tidb and mysql
	tidbOutput, tidbErr := Reproduce(tidbInfo, query)
	// mysqlOutput, mysqlErr := Reproduce(mysqlInfo, query)

	if tidbErr != nil {
		panic(tidbErr)
	}

	fmt.Println("tidb output:", tidbOutput)

	// Feedback to issue.
	// diff expected v.s. mysqlOutput
	// diff actual v.s. tidbOutput into folded section
	_ = expected
	// _ = DiffSubmittedAndExecuted(expected, mysqlOutput)
	_ = DiffSubmittedAndExecuted(actual, tidbOutput)

	// compare result
	// _ = CompareResult(mysqlOutput, tidbOutput)

	return nil
}

type DBConnInfo struct {
	Host     string
	Port     string
	User     string
	Database string
}

func Reproduce(info *DBConnInfo, query string) (string, error) {
	cmd := exec.Command("mysql", "--host", info.Host, "--port", info.Port, "-u", info.User, info.Database, "-e", query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mysql client failed: %w\n", err)
	}
	return string(output), nil
}

// mock diff
func DiffSubmittedAndExecuted(submitted string, executed string) string {
	return "no differences"
}

// mock simple comparison
func CompareResult(expected string, actual string) bool {
	return strings.Compare(expected, actual) == 0
}
