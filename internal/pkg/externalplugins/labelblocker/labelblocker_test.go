package labelblocker

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/lib"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
)

const botName = "ti-chi-bot"

// fghc is a fake GitHub client.
type fghc struct {
	Issues  map[int]*github.Issue
	IssueID int

	IssueComments  map[int][]github.IssueComment
	IssueCommentID int
	// org/repo#number:body
	IssueCommentsAdded []string
	// org/repo#issuecommentid
	IssueCommentsDeleted []string

	// org/repo#number:label
	IssueLabelsAdded    []string
	IssueLabelsExisting []string
	IssueLabelsRemoved  []string

	PullRequests     map[int]*github.PullRequest
	CombinedStatuses map[string]*github.CombinedStatus
	CreatedStatuses  map[string][]github.Status
	// Error will be returned if set. Currently only implemented for CreateStatus
	Error error

	// QueryResult will be returned for Query interface.
	QueryResult interface{}

	// lock to be thread safe
	lock sync.RWMutex
}

func (f *fghc) AddLabel(org, repo string, number int, label string) error {
	return f.AddLabels(org, repo, number, label)
}

func (f *fghc) AddLabels(org, repo string, number int, labels ...string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	for _, label := range labels {
		labelString := fmt.Sprintf("%s/%s#%d:%s", org, repo, number, label)
		if sets.NewString(f.IssueLabelsAdded...).Has(labelString) {
			return fmt.Errorf("cannot add %v to %s/%s/#%d", label, org, repo, number)
		}
		f.IssueLabelsAdded = append(f.IssueLabelsAdded, labelString)
	}
	return nil
}

// RemoveLabel removes a label
func (f *fghc) RemoveLabel(owner, repo string, number int, label string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	labelString := fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, label)
	if !sets.NewString(f.IssueLabelsRemoved...).Has(labelString) {
		f.IssueLabelsRemoved = append(f.IssueLabelsRemoved, labelString)
		return nil
	}
	return fmt.Errorf("cannot remove %v from %s/%s/#%d", label, owner, repo, number)
}

// ListTeams return a list of fake teams that correspond to the fake team members returned by ListTeamAllMembers
func (f *fghc) ListTeams(org string) ([]github.Team, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return []github.Team{
		{
			ID:   0,
			Name: "Admins",
			Slug: "Admins",
		},
		{
			ID:   42,
			Name: "Leads",
			Slug: "Admins",
		},
	}, nil
}

func (f *fghc) GetTeamBySlug(slug string, org string) (*github.Team, error) {
	teams, _ := f.ListTeams(org)
	for _, team := range teams {
		if team.Name == slug {
			return &team, nil
		}
	}
	return &github.Team{}, nil
}

// CreateComment adds a comment to a PR
func (f *fghc) CreateComment(owner, repo string, number int, comment string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.IssueCommentID++
	f.IssueCommentsAdded = append(f.IssueCommentsAdded, fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, comment))
	f.IssueComments[number] = append(f.IssueComments[number], github.IssueComment{
		ID:   f.IssueCommentID,
		Body: comment,
		User: github.User{Login: botName},
	})
	return nil
}

func (f *fghc) Query(ctx context.Context, q interface{}, vars map[string]interface{}) error {
	sq, ok := q.(*lib.TeamMembersQuery)
	if !ok {
		return errors.New("unexpected query type")
	}

	var res lib.TeamMembersQuery
	members := make([]lib.MemberEdge, 0)
	members = append(members, lib.MemberEdge{
		Node: lib.MemberNode{
			Login: "default-sig-lead",
		},
	})

	res.Organization.Team.Members.Edges = members
	sq.Organization = res.Organization
	sq.RateLimit = res.RateLimit

	return nil
}

func TestLabelBlockerPullRequest(t *testing.T) {
	var testcases = []struct {
		name        string
		label       string
		sender      string
		action      github.PullRequestEventAction
		blockLabels []externalplugins.BlockLabel

		expectLabelsRemoved []string
		expectLabelsAdded   []string
		expectCommentsAdded []string
	}{
		{
			name:   "no match block label",
			label:  "sig/community-infra",
			sender: "default-sig-lead",
			action: github.PullRequestActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"labeled", "unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{},
		},
		{
			name:   "no match action",
			label:  "status/can-merge",
			sender: "default-sig-lead",
			action: github.PullRequestActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{},
		},
		{
			name:   "sender is the member of trusted team",
			label:  "status/can-merge",
			sender: "default-sig-lead",
			action: github.PullRequestActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"labeled", "unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{},
		},
		{
			name:   "sender is trusted user",
			label:  "status/can-merge",
			sender: "ti-chi-bot",
			action: github.PullRequestActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"labeled", "unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{},
		},
		{
			name:   "add label blocked and the sender is not trusted",
			label:  "status/can-merge",
			sender: "no-trust-user",
			action: github.PullRequestActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"labeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
					Message:      "You can't add the status/can-merge label.",
				},
			},
			expectLabelsRemoved: []string{"org/repo#5:status/can-merge"},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{
				"org/repo#5:@no-trust-user: You can't add the status/can-merge label.\n\n" +
					"<details>\n\n" +
					"In response to adding label named status/can-merge.\n\n" +
					"Instructions for interacting with me using PR comments are available " +
					"[here](https://prow.tidb.io/command-help).  " +
					"If you have questions or suggestions related to my behavior, please file an issue against the " +
					"[ti-community-infra/tichi](https://github.com/ti-community-infra/tichi/issues/new?title=Prow%20issue:) " +
					"repository.\n" +
					"</details>",
			},
		},
		{
			name:   "remove label blocked and the sender is not trusted",
			label:  "status/can-merge",
			sender: "no-trust-user",
			action: github.PullRequestActionUnlabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
					Message:      "You can't remove the status/can-merge label.",
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{"org/repo#5:status/can-merge"},
			expectCommentsAdded: []string{
				"org/repo#5:@no-trust-user: You can't remove the status/can-merge label.\n\n" +
					"<details>\n\n" +
					"In response to removing label named status/can-merge.\n\n" +
					"Instructions for interacting with me using PR comments are available " +
					"[here](https://prow.tidb.io/command-help).  " +
					"If you have questions or suggestions related to my behavior, please file an issue against the " +
					"[ti-community-infra/tichi](https://github.com/ti-community-infra/tichi/issues/new?title=Prow%20issue:) " +
					"repository.\n" +
					"</details>",
			},
		},
		{
			name:   "illegal action",
			label:  "status/can-merge",
			sender: "no-trust-user",
			action: "nop",
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			var fc = &fghc{
				PullRequests: map[int]*github.PullRequest{
					5: {
						Number: 5,
						Labels: []github.Label{
							{Name: tc.label},
						},
					},
				},
				IssueComments:      map[int][]github.IssueComment{},
				IssueCommentsAdded: []string{},
				IssueLabelsAdded:   []string{},
				IssueLabelsRemoved: []string{},
			}
			e := &github.PullRequestEvent{
				Action:      tc.action,
				Repo:        github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
				PullRequest: *fc.PullRequests[5],
				Label: github.Label{
					Name: tc.label,
				},
				Sender: github.User{
					Login: tc.sender,
				},
			}

			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityLabelBlocker = []externalplugins.TiCommunityLabelBlocker{
				{
					Repos:       []string{"org/repo"},
					BlockLabels: tc.blockLabels,
				},
			}

			if err := HandlePullRequestEvent(fc, e, cfg, logrus.WithField("plugin", PluginName)); err != nil {
				t.Errorf("didn't expect error from %s: %v", PluginName, err)
			}

			if !reflect.DeepEqual(fc.IssueLabelsAdded, tc.expectLabelsAdded) {
				t.Errorf("labels added for pull request mismatch: got %v, want %v", fc.IssueLabelsAdded, tc.expectLabelsAdded)
			}

			if !reflect.DeepEqual(fc.IssueLabelsRemoved, tc.expectLabelsRemoved) {
				t.Errorf("labels removed for pull request mismatch: got %v, want %v", fc.IssueLabelsRemoved, tc.expectLabelsRemoved)
			}

			if !reflect.DeepEqual(fc.IssueCommentsAdded, tc.expectCommentsAdded) {
				t.Errorf("message is mismatch: got %v, want %v", fc.IssueCommentsAdded, tc.expectCommentsAdded)
			}
		})
	}
}

func TestLabelBlockerIssue(t *testing.T) {
	var testcases = []struct {
		name        string
		label       string
		sender      string
		action      github.IssueEventAction
		blockLabels []externalplugins.BlockLabel

		expectLabelsRemoved []string
		expectLabelsAdded   []string
		expectCommentsAdded []string
	}{
		{
			name:   "no match action",
			label:  "status/can-merge",
			sender: "default-sig-lead",
			action: github.IssueActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{},
		},
		{
			name:   "no match block label",
			label:  "sig/community-infra",
			sender: "default-sig-lead",
			action: github.IssueActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"labeled", "unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{},
		},
		{
			name:   "sender is the member of trusted team",
			label:  "status/can-merge",
			sender: "default-sig-lead",
			action: github.IssueActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"labeled", "unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{},
		},
		{
			name:   "sender is trusted user",
			label:  "status/can-merge",
			sender: "ti-chi-bot",
			action: github.IssueActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"labeled", "unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{},
		},
		{
			name:   "add label blocked and the sender is not trusted",
			label:  "status/can-merge",
			sender: "no-trust-user",
			action: github.IssueActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"labeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
					Message:      "You can't add the status/can-merge label.",
				},
			},
			expectLabelsRemoved: []string{"org/repo#5:status/can-merge"},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{
				"org/repo#5:@no-trust-user: You can't add the status/can-merge label.\n\n" +
					"<details>\n\n" +
					"In response to adding label named status/can-merge.\n\n" +
					"Instructions for interacting with me using PR comments are available " +
					"[here](https://prow.tidb.io/command-help).  " +
					"If you have questions or suggestions related to my behavior, please file an issue against the " +
					"[ti-community-infra/tichi](https://github.com/ti-community-infra/tichi/issues/new?title=Prow%20issue:) " +
					"repository.\n" +
					"</details>",
			},
		},
		{
			name:   "remove label blocked and the sender is not trusted",
			label:  "status/can-merge",
			sender: "no-trust-user",
			action: github.IssueActionUnlabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
					Message:      "You can't remove the status/can-merge label.",
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{"org/repo#5:status/can-merge"},
			expectCommentsAdded: []string{
				"org/repo#5:@no-trust-user: You can't remove the status/can-merge label.\n\n" +
					"<details>\n\n" +
					"In response to removing label named status/can-merge.\n\n" +
					"Instructions for interacting with me using PR comments are available " +
					"[here](https://prow.tidb.io/command-help).  " +
					"If you have questions or suggestions related to my behavior, please file an issue against the " +
					"[ti-community-infra/tichi](https://github.com/ti-community-infra/tichi/issues/new?title=Prow%20issue:) " +
					"repository.\n" +
					"</details>",
			},
		},
		{
			name:   "illegal action",
			label:  "status/can-merge",
			sender: "no-trust-user",
			action: "nop",
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `^status/can-merge$`,
					Actions:      []string{"unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
			expectCommentsAdded: []string{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			var fc = &fghc{
				Issues: map[int]*github.Issue{
					5: {
						Number: 5,
						Labels: []github.Label{
							{Name: tc.label},
						},
					},
				},
				IssueComments:      map[int][]github.IssueComment{},
				IssueCommentsAdded: []string{},
				IssueLabelsAdded:   []string{},
				IssueLabelsRemoved: []string{},
			}

			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityLabelBlocker = []externalplugins.TiCommunityLabelBlocker{
				{
					Repos:       []string{"org/repo"},
					BlockLabels: tc.blockLabels,
				},
			}

			e := &github.IssueEvent{
				Action: tc.action,
				Repo:   github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
				Issue:  *fc.Issues[5],
				Label: github.Label{
					Name: tc.label,
				},
				Sender: github.User{
					Login: tc.sender,
				},
			}

			if err := HandleIssueEvent(fc, e, cfg, logrus.WithField("plugin", PluginName)); err != nil {
				t.Errorf("didn't expect error from %s: %v", PluginName, err)
			}

			if !reflect.DeepEqual(fc.IssueLabelsAdded, tc.expectLabelsAdded) {
				t.Errorf("labels added for issue mismatch: got %v, want %v", fc.IssueLabelsAdded, tc.expectLabelsAdded)
			}

			if !reflect.DeepEqual(fc.IssueLabelsRemoved, tc.expectLabelsRemoved) {
				t.Errorf("labels removed for issue mismatch: got %v, want %v", fc.IssueLabelsRemoved, tc.expectLabelsRemoved)
			}

			if !reflect.DeepEqual(fc.IssueCommentsAdded, tc.expectCommentsAdded) {
				t.Errorf("message is mismatch: got %v, want %v", fc.IssueCommentsAdded, tc.expectCommentsAdded)
			}
		})
	}
}

func TestHelpProvider(t *testing.T) {
	enabledRepos := []config.OrgRepo{
		{Org: "org1", Repo: "repo"},
		{Org: "org2", Repo: "repo"},
	}
	testcases := []struct {
		name         string
		config       *externalplugins.Configuration
		enabledRepos []config.OrgRepo
		err          bool

		configInfoIncludes []string
		configInfoExcludes []string
	}{
		{
			name:               "Empty config",
			config:             &externalplugins.Configuration{},
			enabledRepos:       enabledRepos,
			configInfoExcludes: []string{"trusted team", "trusted user"},
		},
		{
			name: "All configs enabled",
			config: &externalplugins.Configuration{
				TiCommunityLabelBlocker: []externalplugins.TiCommunityLabelBlocker{
					{
						Repos: []string{"org2/repo"},
						BlockLabels: []externalplugins.BlockLabel{
							{
								Regex:        `^status/can-merge$`,
								Actions:      []string{"labeled", "unlabeled"},
								TrustedTeams: []string{"Admins"},
								TrustedUsers: []string{"ti-chi-bot", "mini256"},
							},
						},
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{"trusted team", "trusted user"},
		},
	}
	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			epa := &externalplugins.ConfigAgent{}
			epa.Set(tc.config)

			helpProvider := HelpProvider(epa)
			pluginHelp, err := helpProvider(tc.enabledRepos)
			if err != nil && !tc.err {
				t.Fatalf("helpProvider error: %v", err)
			}
			for _, msg := range tc.configInfoExcludes {
				if strings.Contains(pluginHelp.Config["org2/repo"], msg) {
					t.Fatalf("helpProvider.Config error mismatch: got %v, but didn't want it", msg)
				}
			}
			for _, msg := range tc.configInfoIncludes {
				if !strings.Contains(pluginHelp.Config["org2/repo"], msg) {
					t.Fatalf("helpProvider.Config error mismatch: didn't get %v, but wanted it", msg)
				}
			}
		})
	}
}
