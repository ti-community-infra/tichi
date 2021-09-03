package tars

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
)

const (
	// PluginName is the name of this plugin.
	PluginName = "ti-community-tars"
	// branchRefsPrefix specifies the prefix of branch refs.
	// See also: https://docs.github.com/en/rest/reference/git#references.
	branchRefsPrefix = "refs/heads/"
)

const configInfoAutoUpdatedMessagePrefix = "Auto updated message: "
const searchQueryPrefix = "archived:false is:pr is:open sort:created-asc"

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
				} `graphql:"parents(first:10)"`
			}
		}
	} `graphql:"commits(last:100)"`
	Labels struct {
		Nodes []struct {
			Name githubql.String
		}
	} `graphql:"labels(first:100)"`
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
func HelpProvider(epa *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
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
		yamlSnippet, err := plugins.CommentMap.GenYaml(&tiexternalplugins.Configuration{
			TiCommunityTars: []tiexternalplugins.TiCommunityTars{
				{
					Repos:         []string{"ti-community-infra/test-dev"},
					Message:       "Your PR was out of date, I have automatically updated it for you.",
					OnlyWhenLabel: "status/can-merge",
					ExcludeLabels: []string{"do-not-merge/hold"},
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
		}
		pluginHelp := &pluginhelp.PluginHelp{
			Description: `The tars plugin help you update your out-of-date PR.`,
			Config:      configInfo,
			Snippet:     yamlSnippet,
			Events:      []string{tiexternalplugins.IssueCommentEvent, tiexternalplugins.PushEvent},
		}

		return pluginHelp, nil
	}
}

// HandleIssueCommentEvent handles a GitHub issue comment event and update the PR
// if the issue is a PR based on whether the PR out-of-date.
func HandleIssueCommentEvent(log *logrus.Entry, ghc githubClient, ice *github.IssueCommentEvent,
	cfg *tiexternalplugins.Configuration) error {
	if !ice.Issue.IsPullRequest() {
		return nil
	}

	// Ignore comments from bots.
	isBot, err := ghc.BotUserChecker()
	if err != nil {
		return err
	}
	if isBot(ice.Comment.User.Login) {
		return nil
	}

	// Delay for a few seconds to give GitHub time to add or remove the label,
	// as the comment may be a command related to a PR merge(such as /hold or /merge).
	// See: https://github.com/ti-community-infra/tichi/issues/524.
	sleep(time.Second * 5)

	pr, err := ghc.GetPullRequest(ice.Repo.Owner.Login, ice.Repo.Name, ice.Issue.Number)
	if err != nil {
		return err
	}

	return handlePullRequest(log, ghc, pr, cfg)
}

func handlePullRequest(log *logrus.Entry, ghc githubClient,
	pr *github.PullRequest, cfg *tiexternalplugins.Configuration) error {
	org := pr.Base.Repo.Owner.Login
	repo := pr.Base.Repo.Name
	number := pr.Number
	updated := false
	tars := cfg.TarsFor(org, repo)

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

	for _, label := range pr.Labels {
		for _, excludeLabel := range tars.ExcludeLabels {
			if label.Name == excludeLabel {
				log.Infof("Ignore PR %s/%s#%d with exclude label %s.", org, repo, number, label.Name)
				return nil
			}
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
				updated = true
			}
		}
	}

	if updated {
		return nil
	}

	return takeAction(log, ghc, org, repo, number, pr.User.Login, tars.Message)
}

// HandlePushEvent handles a GitHub push event and update the PR.
func HandlePushEvent(log *logrus.Entry, ghc githubClient, pe *github.PushEvent,
	cfg *tiexternalplugins.Configuration) error {
	if !strings.HasPrefix(pe.Ref, branchRefsPrefix) {
		log.Infof("Ignoring ref %s push event.", pe.Ref)
		return nil
	}

	org := pe.Repo.Owner.Login
	repo := pe.Repo.Name
	branch := getRefBranch(pe.Ref)
	tars := cfg.TarsFor(org, repo)
	log.Infof("Checking %s/%s/%s PRs.", org, repo, branch)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, " repo:\"%s/%s\"", org, repo)
	fmt.Fprintf(&buf, " base:\"%s\"", branch)
	fmt.Fprintf(&buf, searchQueryPrefix+" label:\"%s\"", tars.OnlyWhenLabel)
	for _, label := range tars.ExcludeLabels {
		fmt.Fprintf(&buf, " -label:\"%s\"", label)
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

		takenAction, err := handle(l, ghc, &pr, cfg)
		if err != nil {
			l.WithError(err).Error("Error handling PR.")
			continue
		}
		// Only one PR is processed at a time, because even if other PRs are updated,
		// they still need to be queued for another update and merge.
		// To save testing resources we only process one PR at a time.
		if takenAction {
			l.Info("Successfully updated and completed this push event response process.")
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
	externalConfig *tiexternalplugins.Configuration) error {
	log.Info("Checking all PRs.")
	_, repos := config.EnabledReposForExternalPlugin(PluginName)
	if len(repos) == 0 {
		log.Warnf("No repos have been configured for the %s plugin", PluginName)
		return nil
	}

	if len(repos) == 0 {
		return nil
	}

	// Do _not_ parallelize this. It will trigger GitHub's abuse detection and we don't really care anyways except
	// when developing.
	for _, repo := range repos {
		// Construct the query.
		var reposQuery bytes.Buffer
		fmt.Fprint(&reposQuery, searchQueryPrefix)
		slashSplit := strings.Split(repo, "/")
		if n := len(slashSplit); n != 2 {
			log.WithField("repo", repo).Warn("Found repo that was not in org/repo format, ignoring...")
			continue
		}
		org := slashSplit[0]
		repoName := slashSplit[1]
		tars := externalConfig.TarsFor(org, repoName)
		fmt.Fprintf(&reposQuery, " label:\"%s\" repo:\"%s\"", tars.OnlyWhenLabel, repo)
		for _, label := range tars.ExcludeLabels {
			fmt.Fprintf(&reposQuery, " -label:\"%s\"", label)
		}
		query := reposQuery.String()

		prs, err := search(context.Background(), log, ghc, query)
		if err != nil {
			log.WithError(err).Error("Error was encountered when querying GitHub, " +
				"but the remaining repositories will be processed anyway.")
			continue
		}

		log.Infof("Considering %d PRs of %s.", len(prs), repo)
		branches := make(map[string]bool)
		for i := range prs {
			pr := prs[i]
			org := string(pr.Repository.Owner.Login)
			repo := string(pr.Repository.Name)
			num := int(pr.Number)
			base := string(pr.BaseRef.Name)
			l := log.WithFields(logrus.Fields{
				"org":  org,
				"repo": repo,
				"pr":   num,
				"base": base,
			})
			// Process only one PR for per branch at a time, because even if other PRs are updated,
			// they cannot be merged and will generate DOS attacks on the CI system.
			updated, ok := branches[base]
			if ok {
				if updated {
					continue
				}
			} else {
				branches[base] = false
			}

			// Try to update.
			takenAction, err := handle(l, ghc, &pr, externalConfig)
			if err != nil {
				l.WithError(err).Error("The PR update failed, but the remaining PRs will be processed anyway.")
				continue
			}
			if takenAction {
				// Mark this base branch as already having an updated PR.
				branches[base] = takenAction
				l.Info("Successfully updated.")
			}
		}
	}

	return nil
}

func handle(log *logrus.Entry, ghc githubClient, pr *pullRequest, cfg *tiexternalplugins.Configuration) (bool, error) {
	org := string(pr.Repository.Owner.Login)
	repo := string(pr.Repository.Name)
	number := int(pr.Number)
	updated := false
	tars := cfg.TarsFor(org, repo)

	// Must be at least one commit.
	if len(pr.Commits.Nodes) == 0 {
		return false, nil
	}

	// Check if we update the base into PR.
	currentBaseCommit, err := ghc.GetSingleCommit(org, repo, string(pr.BaseRef.Name))
	if err != nil {
		return false, err
	}

check:
	for _, prCommit := range pr.Commits.Nodes {
		for _, prCommitParent := range prCommit.Commit.Parents.Nodes {
			if string(prCommitParent.OID) == currentBaseCommit.SHA {
				updated = true
				break check
			}
		}
	}

	if updated {
		return false, nil
	}

	return true, takeAction(log, ghc, org, repo, number, string(pr.Author.Login), tars.Message)
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
func takeAction(log *logrus.Entry, ghc githubClient, org, repo string, num int,
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
	err = ghc.UpdatePullRequestBranch(org, repo, num, nil)
	if err != nil {
		return err
	}
	if needsReply {
		// Delay the reply because we may trigger the test in the reply.
		// See: https://github.com/ti-community-infra/tichi/issues/181.
		sleep(time.Second * 5)
		msg := tiexternalplugins.FormatSimpleResponse(author, message)
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
