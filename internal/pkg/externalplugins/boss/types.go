package boss

import "k8s.io/test-infra/prow/github"

// PluginName defines this plugin's registered name.
const PluginName = "ti-community-boss"

type githubClient interface {
	RequestReview(org, repo string, number int, logins []string) error
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
	CreateComment(owner, repo string, number int, comment string) error
	AddLabel(owner, repo string, number int, label string) error
	RemoveLabel(owner, repo string, number int, label string) error
	GetRepoLabels(owner, repo string) ([]github.Label, error)
}
