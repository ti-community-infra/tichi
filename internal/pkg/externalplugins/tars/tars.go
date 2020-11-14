package tars

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/prow-github/pkg/github"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/externalplugins"
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
	Query(context.Context, interface{}, map[string]interface{}) error
}

type pullRequest struct {
	Number githubql.Int
	Author struct {
		Login githubql.String
	}
	Repository struct {
		Name  githubql.String
		Owner struct {
			Login githubql.String
		}
	}
	Commits struct {
		Nodes []struct {
			Commit struct {
				oid githubql.GitObjectID
			}
		}
	} `graphql:"commits(last:1)"`
}

type searchQuery struct {
	RateLimit struct {
		Cost      githubql.Int
		Remaining githubql.Int
	}
	Search struct {
		PageInfo struct {
			HasNextPage githubql.Boolean
			EndCursor   githubql.String
		}
		Nodes []struct {
			PullRequest pullRequest `graphql:"... on PullRequest"`
		}
	} `graphql:"search(type: ISSUE, first: 100, after: $searchCursor, query: $query)"`
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

	lastCommitIndex := len(prCommits) - 1
	return takeAction(log, ghc, org, repo, number, &prCommits[lastCommitIndex].SHA, pr.User.Login)
}

// HandleAll checks all orgs and repos that enabled this plugin for open PRs to
// determine if the issue is a PR based on whether the PR out-of-date.
func HandleAll(log *logrus.Entry, ghc githubClient, config *plugins.Configuration) error {
	log.Info("Checking all PRs.")
	orgs, repos := config.EnabledReposForExternalPlugin(PluginName)
	if len(orgs) == 0 && len(repos) == 0 {
		log.Warnf("No repos have been configured for the %s plugin", PluginName)
		return nil
	}
	var buf bytes.Buffer
	fmt.Fprint(&buf, "archived:false is:pr is:open")
	for _, org := range orgs {
		fmt.Fprintf(&buf, " org:\"%s\"", org)
	}
	for _, repo := range repos {
		fmt.Fprintf(&buf, " repo:\"%s\"", repo)
	}
	prs, err := search(context.Background(), log, ghc, buf.String())
	if err != nil {
		return err
	}
	log.Infof("Considering %d PRs.", len(prs))
	for _, pr := range prs {
		org := string(pr.Repository.Owner.Login)
		repo := string(pr.Repository.Name)
		num := int(pr.Number)
		lastCommitIndex := len(pr.Commits.Nodes) - 1
		sha := string(pr.Commits.Nodes[lastCommitIndex].Commit.oid)
		author := string(pr.Author.Login)
		l := log.WithFields(logrus.Fields{
			"org":  org,
			"repo": repo,
			"pr":   num,
		})
		err := takeAction(log, ghc, org, repo, num, &sha, author)
		if err != nil {
			l.WithError(err).Error("Error handling PR.")
		}
	}
	return nil
}

func search(ctx context.Context, log *logrus.Entry, ghc githubClient, q string) ([]pullRequest, error) {
	var ret []pullRequest
	vars := map[string]interface{}{
		"query":        githubql.String(q),
		"searchCursor": (*githubql.String)(nil),
	}
	var totalCost int
	var remaining int
	for {
		sq := searchQuery{}
		if err := ghc.Query(ctx, &sq, vars); err != nil {
			return nil, err
		}
		totalCost += int(sq.RateLimit.Cost)
		remaining = int(sq.RateLimit.Remaining)
		for _, n := range sq.Search.Nodes {
			ret = append(ret, n.PullRequest)
		}
		if !sq.Search.PageInfo.HasNextPage {
			break
		}
		vars["searchCursor"] = githubql.NewString(sq.Search.PageInfo.EndCursor)
	}
	log.Infof("Search for query \"%s\" cost %d point(s). %d remaining.", q, totalCost, remaining)
	return ret, nil
}

// takeAction updates the PR and comment ont it.
func takeAction(log *logrus.Entry, ghc githubClient, org, repo string, num int, expectedHeadSha *string,
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
	err = ghc.UpdatePullRequestBranch(org, repo, num, expectedHeadSha)
	if err != nil {
		return err
	}

	msg := externalplugins.FormatSimpleResponse(author, autoUpdateMessage)
	return ghc.CreateComment(org, repo, num, msg)
}

func shouldPrune(botName string) func(github.IssueComment) bool {
	return func(ic github.IssueComment) bool {
		return github.NormLogin(botName) == github.NormLogin(ic.User.Login) &&
			strings.Contains(ic.Body, autoUpdateMessage)
	}
}
