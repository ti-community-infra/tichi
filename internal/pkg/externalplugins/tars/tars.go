package tars

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/prow-github/pkg/github"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
)

const (
	// PluginName is the name of this plugin
	PluginName        = "ti-community-tars"
	autoUpdateMessage = "PR auto updated."
)

var sleep = time.Sleep

type githubClient interface {
	CreateComment(org, repo string, number int, comment string) error
	BotName() (string, error)
	DeleteStaleComments(org, repo string, number int,
		comments []github.IssueComment, isStale func(github.IssueComment) bool) error
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetSingleCommit(org, repo, SHA string) (github.RepositoryCommit, error)
	ListPRCommits(org, repo string, number int) ([]github.RepositoryCommit, error)
	UpdatePullRequestBranch(org, repo string, number int, expectedHeadSha *string) error
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return &pluginhelp.PluginHelp{
			Description: `The tars plugin help you update your out-of-date PR.`,
		},
		nil
}

// HandlePullRequestEvent handles a GitHub pull request event and update the PR
// if the issue is a PR based on whether the PR out-of-date.
func HandlePullRequestEvent(log *logrus.Entry, ghc githubClient, pre *github.PullRequestEvent) error {
	if pre.Action != github.PullRequestActionOpened &&
		pre.Action != github.PullRequestActionSynchronize && pre.Action != github.PullRequestActionReopened {
		return nil
	}

	return handle(log, ghc, &pre.PullRequest)
}

// HandleIssueCommentEvent handles a GitHub issue comment event and update the PR
// if the issue is a PR based on whether the PR out-of-date.
func HandleIssueCommentEvent(log *logrus.Entry, ghc githubClient, ice *github.IssueCommentEvent) error {
	if !ice.Issue.IsPullRequest() {
		return nil
	}
	pr, err := ghc.GetPullRequest(ice.Repo.Owner.Login, ice.Repo.Name, ice.Issue.Number)
	if err != nil {
		return err
	}

	return handle(log, ghc, pr)
}

func handle(log *logrus.Entry, ghc githubClient, pr *github.PullRequest) error {
	if pr.Merged {
		return nil
	}
	// Before checking mergeability wait a few seconds to give github a chance to calculate it.
	// This initial delay prevents us from always wasting the first API token.
	sleep(time.Second * 5)

	org := pr.Base.Repo.Owner.Login
	repo := pr.Base.Repo.Name
	number := pr.Number
	mergeable := false

	prCommits, err := ghc.ListPRCommits(org, repo, pr.Number)
	if err != nil {
		return err
	}
	if len(prCommits) == 0 {
		return nil
	}

	// Check if we update the base into PR.
	baseCommit, err := ghc.GetSingleCommit(org, repo, pr.Base.Ref)
	if err != nil {
		return err
	}
	for _, prCommit := range prCommits {
		for _, parentCommit := range prCommit.Parents {
			if parentCommit.SHA == baseCommit.SHA {
				mergeable = true
			}
		}
	}

	if mergeable {
		return nil
	}

	return takeAction(log, ghc, org, repo, number, pr.User.Login)
}

// takeAction updates the PR and comment ont it.
func takeAction(log *logrus.Entry, ghc githubClient, org, repo string, num int,
	author string) error {
	botName, err := ghc.BotName()
	if err != nil {
		return err
	}
	err = ghc.DeleteStaleComments(org, repo, num, nil, shouldPrune(botName))
	if err != nil {
		return err
	}

	log.Infof("Update PR %s/%s#%d.", org, repo, num)
	err = ghc.UpdatePullRequestBranch(org, repo, num, nil)
	if err != nil {
		return err
	}

	msg := plugins.FormatSimpleResponse(author, autoUpdateMessage)
	return ghc.CreateComment(org, repo, num, msg)
}

func shouldPrune(botName string) func(github.IssueComment) bool {
	return func(ic github.IssueComment) bool {
		return github.NormLogin(botName) == github.NormLogin(ic.User.Login) &&
			strings.Contains(ic.Body, autoUpdateMessage)
	}
}
