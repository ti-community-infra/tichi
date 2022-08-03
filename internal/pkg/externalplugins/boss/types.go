package boss

import "k8s.io/test-infra/prow/github"

// PluginName defines this plugin's registered name.
const PluginName = "ti-community-boss"

type githubClient interface {
	RequestReview(org, repo string, number int, logins []string) error
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
	ListFileCommits(org, repo, path string) ([]github.RepositoryCommit, error)
}
