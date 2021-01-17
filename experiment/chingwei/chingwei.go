package chingwei

import (
	"fmt"
	"os/exec"

	"github.com/google/martian/log"
	"github.com/kylelemons/godebug/diff"
	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/github"
)

const needsReproLabel = "status/needs-reproduction"
const reproducedByChingweiLabel = "status/reproduced-by-chingwei"

type githubClient interface {
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
	CreateComment(owner, repo string, number int, comment string) error
	RemoveLabel(owner, repo string, number int, label string) error
	AddLabel(owner, repo string, number int, label string) error
}

func Reproducing(log *logrus.Entry, ghc githubClient) error {
	owner := "ti-community-infra"
	repo := "test-dev"

	log.Info("Staring search pull request.")
	filter := "repo:" + owner + "/" + repo + " label:" + needsReproLabel

	issues, err := ghc.FindIssues(filter, "", false)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		return nil
	}

	// For now, only reproduce first issue.
	issue := issues[0]
	log.Infof("Got issue: %v with body: %q", issue, issue.Body)

	// Parse minimal reproduce step and version from GitHub issue.
	issueBasicInfo := parseIssue(issue.Body)

	// Prepare TiDB and MySQL.
	tidbConInfo, tidbCleanup, err := PrepareTiDB(issueBasicInfo.tidbVersion)
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
	tidbCleanup()

	mysqlOutput, err := reproduce(mysqlConInfo, issueBasicInfo.query)
	if err != nil {
		return err
	}

	log.Infof("mysql output: %s", mysqlOutput)

	diffOutput := diff.Diff(mysqlOutput, tidbOutput)
	var resp string
	if diffOutput == "" {
		resp = fmt.Sprintf("This issue is **NOT** reproduced on MySQL (%s) and TiDB (%s). "+
			"The output of TiDB is the same as MySQL.", issueBasicInfo.mysqlVersion, issueBasicInfo.tidbVersion)
	} else {
		resp = fmt.Sprintf("This issue is reproduced on MySQL (%s) and TiDB (%s). "+
			"Here is the diff between MySQL output and TiDB output.\n```diff\n+++ tidb.log\n--- mysql.log\n\n%s\n```\n",
			issueBasicInfo.mysqlVersion, issueBasicInfo.tidbVersion, diffOutput)
	}

	resp += fmt.Sprintf("TiDB output:\n```\n%s\n```\nMySQL output:\n```\n%s\n```\n", tidbOutput, mysqlOutput)

	// Remove needs reproduction label and chingwei label.
	err = ghc.RemoveLabel(owner, repo, issue.Number, needsReproLabel)
	if err != nil {
		return err
	}

	err = ghc.AddLabel(owner, repo, issue.Number, reproducedByChingweiLabel)
	if err != nil {
		return err
	}

	// Feedback to github issue.
	return ghc.CreateComment(owner, repo, issue.Number,
		externalplugins.FormatResponseRaw(issue.Body, issue.HTMLURL, issue.User.Login, resp))
}

func reproduce(info *DBConnInfo, query string) (string, error) {
	//nolint:gosec
	cmd := exec.Command("mysql", "--host", info.Host,
		"--port", info.Port, "-u", info.User, info.Database, "-e", query, "-t")
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
