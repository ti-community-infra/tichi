package tars

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
)

const (
	// PluginName is the name of this plugin.
	PluginName = "ti-community-tars"
	// branchRefsPrefix specifies the prefix of branch refs.
	// See also: https://docs.github.com/en/rest/reference/git#references.
	branchRefsPrefix = "refs/heads/"
)

const configInfoAutoUpdatedMessagePrefix = "Auto updated message: "

var sleep = time.Sleep

type githubClient interface {
	CreateComment(org, repo string, number int, comment string) error
	BotUserChecker() (func(candidate string) bool, error)
	DeleteStaleComments(org, repo string, number int,
		comments []github.IssueComment, isStale func(github.IssueComment) bool) error
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetSingleCommit(org, repo, SHA string) (github.RepositoryCommit, error)
	ListPRCommits(org, repo string, number int) ([]github.RepositoryCommit, error)
	UpdatePullRequestBranch(org, repo string, number int, expectedHeadSha *string) error
	Query(context.Context, interface{}, map[string]interface{}) error
}

// See: https://developer.github.com/v4/object/pullrequest/.
type pullRequest struct {
	Number     githubql.Int
	Repository struct {
		Name  githubql.String
		Owner struct {
			Login githubql.String
		}
	}
	Author struct {
		Login githubql.String
	}
	BaseRef struct {
		Name githubql.String
	}
	Commits struct {
		Nodes []struct {
			Commit struct {
				OID     githubql.GitObjectID `graphql:"oid"`
				Parents struct {
					Nodes []struct {
						OID githubql.GitObjectID `graphql:"oid"`
					}
				} `graphql:"parents(first:100)"`
			}
		}
	} `graphql:"commits(last:1)"`
	Labels struct {
		Nodes []struct {
			Name githubql.String
		}
	} `graphql:"labels(first:100)"`
	Mergeable githubql.MergeableState
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
func HelpProvider(epa *externalplugins.ConfigAgent) func(
	enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()
		for _, repo := range enabledRepos {
			opts := cfg.TarsFor(repo.Org, repo.Repo)
			var isConfigured bool
			var configInfoStrings []string

			configInfoStrings = append(configInfoStrings, "The plugin has these configurations:<ul>")

			if len(opts.Message) != 0 {
				isConfigured = true
			}

			configInfoStrings = append(configInfoStrings, "<li>"+configInfoAutoUpdatedMessagePrefix+opts.Message+"</li>")

			configInfoStrings = append(configInfoStrings, "</ul>")
			if isConfigured {
				configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
			}
		}
		pluginHelp := &pluginhelp.PluginHelp{
			Description: `The tars plugin help you update your out-of-date PR.`,
			Config:      configInfo,
		}

		return pluginHelp, nil
	}
}

// HandlePullRequestEvent handles a GitHub pull request event and update the PR
// if the issue is a PR based on whether the PR out-of-date.
func HandlePullRequestEvent(log *logrus.Entry, ghc githubClient, pre *github.PullRequestEvent,
	cfg *externalplugins.Configuration) error {
	if pre.Action != github.PullRequestActionOpened &&
		pre.Action != github.PullRequestActionSynchronize && pre.Action != github.PullRequestActionReopened {
		return nil
	}

	return handlePullRequest(log, ghc, &pre.PullRequest, cfg)
}

// HandleIssueCommentEvent handles a GitHub issue comment event and update the PR
// if the issue is a PR based on whether the PR out-of-date.
func HandleIssueCommentEvent(log *logrus.Entry, ghc githubClient, ice *github.IssueCommentEvent,
	cfg *externalplugins.Configuration) error {
	if !ice.Issue.IsPullRequest() {
		return nil
	}
	pr, err := ghc.GetPullRequest(ice.Repo.Owner.Login, ice.Repo.Name, ice.Issue.Number)
	if err != nil {
		return err
	}

	return handlePullRequest(log, ghc, pr, cfg)
}

func handlePullRequest(log *logrus.Entry, ghc githubClient,
	pr *github.PullRequest, cfg *externalplugins.Configuration) error {
	if pr.Merged {
		return nil
	}

	org := pr.Base.Repo.Owner.Login
	repo := pr.Base.Repo.Name
	number := pr.Number
	mergeable := false
	tars := cfg.TarsFor(org, repo)

	// If the OnlyWhenLabel configuration is set, the pr will only be updated if it has this label.
	if len(tars.OnlyWhenLabel) != 0 {
		hasTriggerLabel := false
		for _, label := range pr.Labels {
			if label.Name == tars.OnlyWhenLabel {
				hasTriggerLabel = true
			}
		}
		if !hasTriggerLabel {
			log.Infof("Ignore PR %s/%s#%d without trigger label %s.", org, repo, number, tars.OnlyWhenLabel)
			return nil
		}
	}

	prCommits, err := ghc.ListPRCommits(org, repo, pr.Number)
	if err != nil {
		return err
	}
	if len(prCommits) == 0 {
		return nil
	}

	// Check if we update the base into PR.
	currentBaseCommit, err := ghc.GetSingleCommit(org, repo, pr.Base.Ref)
	if err != nil {
		return err
	}
	for _, prCommit := range prCommits {
		for _, parentCommit := range prCommit.Parents {
			if parentCommit.SHA == currentBaseCommit.SHA {
				mergeable = true
			}
		}
	}

	if mergeable {
		return nil
	}

	lastCommitIndex := len(prCommits) - 1
	return takeAction(log, ghc, org, repo, number, &prCommits[lastCommitIndex].SHA, pr.User.Login, tars.Message)
}

// HandlePushEvent handles a GitHub push event and update the PR.
func HandlePushEvent(log *logrus.Entry, ghc githubClient, pe *github.PushEvent,
	cfg *externalplugins.Configuration) error {
	if !strings.HasPrefix(pe.Ref, branchRefsPrefix) {
		log.Infof("Ignoring ref %s push event.", pe.Ref)
		return nil
	}

	org := pe.Repo.Owner.Login
	repo := pe.Repo.Name
	branch := getRefBranch(pe.Ref)
	log.Infof("Checking %s/%s#%s PRs.", org, repo, branch)

	var buf bytes.Buffer
	fmt.Fprint(&buf, "archived:false is:pr is:open sort:created-asc")
	fmt.Fprintf(&buf, " repo:\"%s/%s\"", org, repo)
	fmt.Fprintf(&buf, " base:\"%s\"", branch)

	prs, err := search(context.Background(), log, ghc, buf.String())
	if err != nil {
		return err
	}
	log.Infof("Considering %d PRs.", len(prs))
	for i := range prs {
		pr := prs[i]
		org := string(pr.Repository.Owner.Login)
		repo := string(pr.Repository.Name)
		num := int(pr.Number)
		l := log.WithFields(logrus.Fields{
			"org":  org,
			"repo": repo,
			"pr":   num,
		})

		// Only one PR is processed at a time, because even if other PRs are updated,
		// they still need to be queued for another update and merge.
		// To save testing resources we only process one PR at a time.
		err = handle(l, ghc, &pr, cfg)
		if err != nil {
			l.WithError(err).Error("Error handling PR.")
		} else {
			break
		}
	}
	return nil
}

func getRefBranch(ref string) string {
	return strings.TrimPrefix(ref, branchRefsPrefix)
}

// HandleAll checks all orgs and repos that enabled this plugin for open PRs to
// determine if the issue is a PR based on whether the PR out-of-date.
func HandleAll(log *logrus.Entry, ghc githubClient, config *plugins.Configuration,
	externalConfig *externalplugins.Configuration) error {
	log.Info("Checking all PRs.")
	orgs, repos := config.EnabledReposForExternalPlugin(PluginName)
	if len(orgs) == 0 && len(repos) == 0 {
		log.Warnf("No repos have been configured for the %s plugin", PluginName)
		return nil
	}
	var buf bytes.Buffer
	fmt.Fprint(&buf, "archived:false is:pr is:open sort:created-asc")
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
	for i := range prs {
		pr := prs[i]
		org := string(pr.Repository.Owner.Login)
		repo := string(pr.Repository.Name)
		num := int(pr.Number)
		l := log.WithFields(logrus.Fields{
			"org":  org,
			"repo": repo,
			"pr":   num,
		})

		err = handle(l, ghc, &pr, externalConfig)
		if err != nil {
			l.WithError(err).Error("Error handling PR.")
		}
	}
	return nil
}

func handle(log *logrus.Entry, ghc githubClient, pr *pullRequest, cfg *externalplugins.Configuration) error {
	// Skips PRs that cannot be conflicting.
	if pr.Mergeable != githubql.MergeableStateMergeable {
		return nil
	}

	org := string(pr.Repository.Owner.Login)
	repo := string(pr.Repository.Name)
	number := int(pr.Number)
	mergeable := false
	tars := cfg.TarsFor(org, repo)

	// If the OnlyWhenLabel configuration is set, the pr will only be updated if it has this label.
	if len(tars.OnlyWhenLabel) != 0 {
		hasTriggerLabel := false
		for _, labelName := range pr.Labels.Nodes {
			if string(labelName.Name) == tars.OnlyWhenLabel {
				hasTriggerLabel = true
			}
		}
		if !hasTriggerLabel {
			log.Infof("Ignore PR %s/%s#%d without trigger label %s.", org, repo, number, tars.OnlyWhenLabel)
			return nil
		}
	}

	// Must have last commit.
	if len(pr.Commits.Nodes) == 0 || len(pr.Commits.Nodes) != 1 {
		return nil
	}

	// Check if we update the base into PR.
	currentBaseCommit, err := ghc.GetSingleCommit(org, repo, string(pr.BaseRef.Name))
	if err != nil {
		return err
	}
	for _, prCommitParent := range pr.Commits.Nodes[0].Commit.Parents.Nodes {
		if string(prCommitParent.OID) == currentBaseCommit.SHA {
			mergeable = true
		}
	}

	if mergeable {
		return nil
	}

	lastCommitIndex := 0
	lastCommitSHA := string(pr.Commits.Nodes[lastCommitIndex].Commit.OID)
	return takeAction(log, ghc, org, repo, number, &lastCommitSHA, string(pr.Author.Login), tars.Message)
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
	author string, message string) error {
	botUserChecker, err := ghc.BotUserChecker()
	if err != nil {
		return err
	}
	needsReply := len(message) != 0

	if needsReply {
		err = ghc.DeleteStaleComments(org, repo, num, nil, shouldPrune(botUserChecker, message))
		if err != nil {
			return err
		}
	}

	log.Infof("Update PR %s/%s#%d.", org, repo, num)
	err = ghc.UpdatePullRequestBranch(org, repo, num, expectedHeadSha)
	if err != nil {
		return err
	}
	if needsReply {
		// Delay the reply because we may trigger the test in the reply.
		// See: https://github.com/ti-community-infra/tichi/issues/181.
		sleep(time.Second * 5)
		msg := externalplugins.FormatSimpleResponse(author, message)
		return ghc.CreateComment(org, repo, num, msg)
	}
	return nil
}

func shouldPrune(isBot func(string) bool, message string) func(github.IssueComment) bool {
	return func(ic github.IssueComment) bool {
		return isBot(ic.User.Login) &&
			strings.Contains(ic.Body, message)
	}
}
