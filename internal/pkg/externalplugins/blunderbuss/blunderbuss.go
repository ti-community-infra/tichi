package blunderbuss

import (
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/externalplugins"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/ownersclient"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pkg/layeredsets"
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

func HandlePullRequest(gc githubClient, pe *github.PullRequestEvent,
	cfg *externalplugins.Configuration, ol ownersclient.OwnersLoader, log *logrus.Entry) error {
	pr := &pe.PullRequest
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

func HandleIssueCommentEvent(gc githubClient, ce github.IssueCommentEvent, cfg *externalplugins.Configuration,
	ol ownersclient.OwnersLoader, log *logrus.Entry) error {
	if ce.Action != github.IssueCommentActionCreated || !ce.Issue.IsPullRequest() || ce.Issue.State == "closed" {
		return nil
	}

	if !match.MatchString(ce.Issue.Body) {
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

	reviewers := getReviewers(pr.User.Login, owners.Reviewers, opts.ReviewerCount, log)
	maxReviewerCount := opts.MaxReviewerCount

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

func getReviewers(author string, reviewers []string, minReviewers int, log *logrus.Entry) []string {
	authorSet := sets.NewString(github.NormLogin(author))
	reviewersSet := sets.NewString()
	reviewersSet.Insert(reviewers...)

	var result []string
	// Exclude the author.
	availableReviewers := layeredsets.NewString(reviewersSet.Difference(authorSet).List()...)

	if availableReviewers.Len() < minReviewers {
		log.Debugf("Not enough reviewers found in sig. %d/%d reviewers found.", len(reviewers), minReviewers)
	}

	for i := 0; i < minReviewers; i++ {
		reviewer := availableReviewers.PopRandom()
		result = append(result, reviewer)
		log.Infof("Added %s as reviewers. %d/%d reviewers found.", reviewer, len(result), minReviewers)
	}

	return result
}
