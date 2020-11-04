package needsrebase

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/labels"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
)

const (
	// PluginName is the name of this plugin
	PluginName         = "ti-community-needs-rebase"
	needsRebaseMessage = "PR needs rebase."
)

var sleep = time.Sleep

type githubClient interface {
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	CreateComment(org, repo string, number int, comment string) error
	BotName() (string, error)
	AddLabel(org, repo string, number int, label string) error
	RemoveLabel(org, repo string, number int, label string) error
	IsMergeable(org, repo string, number int, sha string) (bool, error)
	DeleteStaleComments(org, repo string, number int,
		comments []github.IssueComment, isStale func(github.IssueComment) bool) error
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetSingleCommit(org, repo, SHA string) (github.SingleCommit, error)
	ListPRCommits(org, repo string, number int) ([]github.RepositoryCommit, error)
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return &pluginhelp.PluginHelp{
			Description: `The needs-rebase plugin manages the '` + labels.NeedsRebase +
				`' label by removing it from Pull Requests that are mergeable and adding it to those which are not.`,
		},
		nil
}

// HandlePullRequestEvent handles a GitHub pull request event and adds or removes a
// "needs-rebase" label based on whether the GitHub api considers the PR mergeable
func HandlePullRequestEvent(log *logrus.Entry, ghc githubClient, pre *github.PullRequestEvent) error {
	if pre.Action != github.PullRequestActionOpened &&
		pre.Action != github.PullRequestActionSynchronize && pre.Action != github.PullRequestActionReopened {
		return nil
	}

	return handle(log, ghc, &pre.PullRequest)
}

// HandleIssueCommentEvent handles a GitHub issue comment event and adds or removes a
// "needs-rebase" label if the issue is a PR based on whether the GitHub api considers
// the PR mergeable
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

// handle handles a GitHub PR to determine if the "needs-rebase"
// label needs to be added or removed. It depends on GitHub mergeability check
// to decide the need for a rebase.
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

	for _, parent := range prCommits[0].Parents {
		if parent.ID == pr.Base.SHA {
			mergeable = true
		}
	}

	issueLabels, err := ghc.GetIssueLabels(org, repo, number)
	if err != nil {
		return err
	}
	hasLabel := github.HasLabel(labels.NeedsRebase, issueLabels)

	return takeAction(log, ghc, org, repo, number, pr.User.Login, hasLabel, mergeable)
}

// takeAction adds or removes the "needs-rebase" label based on the current
// state of the PR (hasLabel and mergeable). It also handles adding and
// removing GitHub comments notifying the PR author that a rebase is needed.
func takeAction(log *logrus.Entry, ghc githubClient, org, repo string, num int,
	author string, hasLabel, mergeable bool) error {
	if !mergeable && !hasLabel {
		if err := ghc.AddLabel(org, repo, num, labels.NeedsRebase); err != nil {
			log.WithError(err).Errorf("Failed to add %q label.", labels.NeedsRebase)
		}
		msg := plugins.FormatSimpleResponse(author, needsRebaseMessage)
		return ghc.CreateComment(org, repo, num, msg)
	} else if mergeable && hasLabel {
		// remove label and prune comment
		if err := ghc.RemoveLabel(org, repo, num, labels.NeedsRebase); err != nil {
			log.WithError(err).Errorf("Failed to remove %q label.", labels.NeedsRebase)
		}
		botName, err := ghc.BotName()
		if err != nil {
			return err
		}
		return ghc.DeleteStaleComments(org, repo, num, nil, shouldPrune(botName))
	}
	return nil
}

func shouldPrune(botName string) func(github.IssueComment) bool {
	return func(ic github.IssueComment) bool {
		return github.NormLogin(botName) == github.NormLogin(ic.User.Login) &&
			strings.Contains(ic.Body, needsRebaseMessage)
	}
}
