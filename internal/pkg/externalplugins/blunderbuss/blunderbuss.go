package blunderbuss

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/ti-community-prow/internal/pkg/externalplugins"
	"github.com/ti-community-infra/ti-community-prow/internal/pkg/ownersclient"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pkg/layeredsets"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
	"k8s.io/test-infra/prow/plugins/assign"
)

const (
	// PluginName defines this plugin's registered name.
	PluginName = "ti-community-blunderbuss"
)

var (
	match = regexp.MustCompile(`(?mi)^/auto-cc\s*$`)
)

type githubClient interface {
	RequestReview(org, repo string, number int, logins []string) error
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *externalplugins.ConfigAgent) func(
	enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.BlunderbussFor(repo.Org, repo.Repo)
			var isConfigured bool
			var configInfoStrings []string
			configInfoStrings = append(configInfoStrings, "The plugin has these configurations:<ul>")
			if opts.MaxReviewerCount > 0 {
				configInfoStrings = append(configInfoStrings, "<li>"+configString(opts.MaxReviewerCount)+"</li>")
				isConfigured = true
			}
			configInfoStrings = append(configInfoStrings, "</ul>")
			if isConfigured {
				configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
			}
		}
		yamlSnippet, err := plugins.CommentMap.GenYaml(&externalplugins.Configuration{
			TiCommunityBlunderbuss: []externalplugins.TiCommunityBlunderbuss{
				{
					Repos:              []string{"ti-community-infra/test-dev"},
					MaxReviewerCount:   2,
					ExcludeReviewers:   []string{},
					PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
		}
		pluginHelp := &pluginhelp.PluginHelp{
			Description: "The blunderbuss plugin automatically requests reviews from reviewers when a new PR is created.",
			Config:      configInfo,
			Snippet:     yamlSnippet,
		}
		pluginHelp.AddCommand(pluginhelp.Command{
			Usage:       "/auto-cc",
			Featured:    false,
			Description: "Manually request reviews from reviewers for a PR.",
			Examples:    []string{"/auto-cc"},
			WhoCanUse:   "Everyone",
		})
		return pluginHelp, nil
	}
}

func configString(maxReviewerCount int) string {
	var pluralSuffix string
	if maxReviewerCount > 1 {
		pluralSuffix = "s"
	}
	return fmt.Sprintf("Blunderbuss is currently configured to request reviews from %d reviewer%s.",
		maxReviewerCount, pluralSuffix)
}

// HandleIssueCommentEvent handles a GitHub pull request event and requests review.
func HandlePullRequestEvent(gc githubClient, pe *github.PullRequestEvent,
	cfg *externalplugins.Configuration, ol ownersclient.OwnersLoader, log *logrus.Entry) error {
	pr := &pe.PullRequest
	// Only for open PR and non /cc PR.
	if pe.Action != github.PullRequestActionOpened || assign.CCRegexp.MatchString(pr.Body) {
		return nil
	}
	repo := &pe.Repo
	opts := cfg.BlunderbussFor(repo.Owner.Login, repo.Name)

	return handle(
		gc,
		opts,
		repo,
		pr,
		log,
		ol,
	)
}

// HandleIssueCommentEvent handles a GitHub issue comment event and requests review.
func HandleIssueCommentEvent(gc githubClient, ce *github.IssueCommentEvent, cfg *externalplugins.Configuration,
	ol ownersclient.OwnersLoader, log *logrus.Entry) error {
	// Only consider open PRs and new comments.
	if ce.Action != github.IssueCommentActionCreated || !ce.Issue.IsPullRequest() || ce.Issue.State == "closed" {
		return nil
	}

	if !match.MatchString(ce.Comment.Body) {
		return nil
	}

	repo := &ce.Repo

	pr, err := gc.GetPullRequest(repo.Owner.Login, repo.Name, ce.Issue.Number)
	if err != nil {
		return fmt.Errorf("error loading PullRequest: %v", err)
	}

	opts := cfg.BlunderbussFor(repo.Owner.Login, repo.Name)

	return handle(
		gc,
		opts,
		repo,
		pr,
		log,
		ol,
	)
}

func handle(ghc githubClient, opts *externalplugins.TiCommunityBlunderbuss, repo *github.Repo, pr *github.PullRequest,
	log *logrus.Entry, ol ownersclient.OwnersLoader) error {
	owners, err := ol.LoadOwners(opts.PullOwnersEndpoint, repo.Owner.Login, repo.Name, pr.Number)
	if err != nil {
		return fmt.Errorf("error loading RepoOwners: %v", err)
	}

	reviewers := getReviewers(pr.User.Login, owners.Reviewers, opts.ExcludeReviewers, log)
	maxReviewerCount := opts.MaxReviewerCount

	// If the maximum count of reviewers greater than 0, it needs to be split.
	if maxReviewerCount > 0 && len(reviewers) > maxReviewerCount {
		log.Infof("Limiting request of %d reviewers to %d maxReviewers.", len(reviewers), maxReviewerCount)
		reviewers = reviewers[:maxReviewerCount]
	}

	if len(reviewers) > 0 {
		log.Infof("Requesting reviews from users %s.", reviewers)
		return ghc.RequestReview(repo.Owner.Login, repo.Name, pr.Number, reviewers)
	}
	return nil
}

func getReviewers(author string, reviewers []string, excludeReviewers []string, log *logrus.Entry) []string {
	authorSet := sets.NewString(github.NormLogin(author))
	excludeReviewersSet := sets.NewString(excludeReviewers...)
	reviewersSet := sets.NewString()
	reviewersSet.Insert(reviewers...)

	var result []string
	// Exclude the author.
	availableReviewers := layeredsets.NewString(
		reviewersSet.Difference(authorSet).Difference(excludeReviewersSet).List()...)

	for availableReviewers.Len() > 0 {
		reviewer := availableReviewers.PopRandom()
		result = append(result, reviewer)
		log.Infof("Added %s as reviewers. %d reviewers found.", reviewer, len(result))
	}

	return result
}
