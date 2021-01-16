package chingwei

import (
	"fmt"
	"os/exec"

	"github.com/google/go-cmp/cmp"
	"github.com/google/martian/log"
	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/github"
)

type githubClient interface {
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
	CreateComment(owner, repo string, number int, comment string) error
	RemoveLabel(owner, repo string, number int, label string) error
}

func Reproducing(log *logrus.Entry, ghc githubClient) error {
	owner := "ti-community-infra"
	repo := "test-dev"

	log.Info("Staring search pull request.")
	filter := "repo:" + owner + "/" + repo + " label:status/needs-reproduction"

	issues, err := ghc.FindIssues(filter, "", false)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		return nil
	}

	// For now, only reproduce first issue.
	issue := issues[0]
	log.Infof("Got issue: %v with body: %s", issue, issue.Body)

	// Parse minimal reproduce step and version from GitHub issue.
	issueBasicInfo := parseIssue(issue.Body)

	// Prepare TiDB and MySQL.
	tidbConInfo, err := PrepareTiDB(issueBasicInfo.tidbVersion)
	if err != nil {
		return err
	}
	mysqlConInfo, err := PrepareMySQL(issueBasicInfo.mysqlVersion)
	if err != nil {
		return err
	}

	// reproduce by connecting to tidb and mysql
	tidbOutput, err := reproduce(tidbConInfo, issueBasicInfo.query)
	if err != nil {
		return err
	}
	log.Infof("tidb cluster output: %s", tidbOutput)

	mysqlOutput, err := reproduce(mysqlConInfo, issueBasicInfo.query)
	if err != nil {
		return err
	}

	log.Infof("mysql output: %s", mysqlOutput)

	// MySQL output v.s. expected.
	expectedDiff := diff(issueBasicInfo.expected, mysqlOutput)

	// TiDB output v.s. actual.
	actualDiff := diff(issueBasicInfo.actual, tidbOutput)

	resp := expectedDiff + actualDiff

	// Feedback to github issue.
	return ghc.CreateComment(owner, repo, issue.Number,
		externalplugins.FormatResponseRaw(issue.Body, issue.HTMLURL, issue.User.Login, resp))
}

func reproduce(info *DBConnInfo, query string) (string, error) {
	//nolint:gosec
	cmd := exec.Command("mysql", "--host", info.Host,
		"--port", info.Port, "-u", info.User, info.Database, "-e", query)
	if info.Password != "" {
		cmd.Args = append(cmd.Args, "-p"+info.Password)
	}
	log.Debugf("reproduce command: %v", cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("connect to mysql failed: output: %s, error: %w", string(output), err)
	}
	return string(output), nil
}

func diff(want, got string) string {
	var result string
	diff := cmp.Diff(want, got)
	if diff == "" {
		result = fmt.Sprintf("want: %s\n, got: %s\n", want, got)
	} else {
		result = "```diff\n" + diff + "```\n"
	}

	return result
}
