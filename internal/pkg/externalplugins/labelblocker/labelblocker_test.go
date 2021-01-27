package labelblocker

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
)

func TestLabelBlockerPullRequest(t *testing.T) {
	var testcases = []struct {
		name                string
		label               string
		sender              string
		action              github.PullRequestEventAction
		blockLabels         []externalplugins.BlockLabel
		expectLabelsRemoved []string
		expectLabelsAdded   []string
	}{
		{
			name:   "no match block label",
			label:  "sig/community-infra",
			sender: "default-sig-lead",
			action: github.PullRequestActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `(?mi)^status/can-merge$`,
					Actions: 	  []string{"labeled", "unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
		},
		{
			name:   "no match action",
			label:  "status/can-merge",
			sender: "default-sig-lead",
			action: github.PullRequestActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `(?mi)^status/can-merge$`,
					Actions:      []string{"unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
		},
		{
			name:   "sender is the member of trusted team",
			label:  "status/can-merge",
			sender: "default-sig-lead",
			action: github.PullRequestActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `(?mi)^status/can-merge$`,
					Actions: 	  []string{"labeled", "unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
		},
		{
			name:   "sender is trusted user",
			label:  "status/can-merge",
			sender: "ti-chi-bot",
			action: github.PullRequestActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `(?mi)^status/can-merge$`,
					Actions: 	  []string{"labeled", "unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{},
		},
		{
			name:   "add label blocked and the sender is not trusted",
			label:  "status/can-merge",
			sender: "no-trust-user",
			action: github.PullRequestActionLabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `(?mi)^status/can-merge$`,
					Actions:      []string{"labeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{"org/repo#5:status/can-merge"},
			expectLabelsAdded:   []string{},
		},
		{
			name:   "remove label blocked and the sender is not trusted",
			label:  "status/can-merge",
			sender: "no-trust-user",
			action: github.PullRequestActionUnlabeled,
			blockLabels: []externalplugins.BlockLabel{
				{
					Regex:        `(?mi)^status/can-merge$`,
					Actions:      []string{"unlabeled"},
					TrustedTeams: []string{"Admins"},
					TrustedUsers: []string{"ti-chi-bot", "mini256"},
				},
			},
			expectLabelsRemoved: []string{},
			expectLabelsAdded:   []string{"org/repo#5:status/can-merge"},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			fc := &fakegithub.FakeClient{
				PullRequests: map[int]*github.PullRequest{
					5: {
						Number: 5,
						Labels: []github.Label{
							{Name: tc.label},
						},
					},
				},
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
		})
	}
}
