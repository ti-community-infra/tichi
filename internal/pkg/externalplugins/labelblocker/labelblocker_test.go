package labelblocker

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
)

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
				"org/repo#5:@no-trust-user: You can't add the status/can-merge label.\n" +
					"<details>\n" +
					"Only trusted users or members of the trusted team can do this, the operation of " +
					"others will be revoked by the robot.\n" +
					"trusted teams: Admins\n" +
					"trusted users: ti-chi-bot, mini256\n" +
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
				"org/repo#5:@no-trust-user: You can't remove the status/can-merge label.\n" +
					"<details>\n" +
					"Only trusted users or members of the trusted team can do this, the operation of " +
					"others will be revoked by the robot.\n" +
					"trusted teams: Admins\n" +
					"trusted users: ti-chi-bot, mini256\n" +
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
			var fc = &fakegithub.FakeClient{
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
				"org/repo#5:@no-trust-user: You can't add the status/can-merge label.\n" +
					"<details>\n" +
					"Only trusted users or members of the trusted team can do this, the operation " +
					"of others will be revoked by the robot.\n" +
					"trusted teams: Admins\n" +
					"trusted users: ti-chi-bot, mini256\n" +
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
				"org/repo#5:@no-trust-user: You can't remove the status/can-merge label.\n" +
					"<details>\n" +
					"Only trusted users or members of the trusted team can do this, the operation of " +
					"others will be revoked by the robot.\n" +
					"trusted teams: Admins\n" +
					"trusted users: ti-chi-bot, mini256\n" +
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
			var fc = &fakegithub.FakeClient{
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
