package contribution

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
)

func TestHandlePullRequest(t *testing.T) {
	testcases := []struct {
		name              string
		action            github.PullRequestEventAction
		author            string
		authorAssociation string
		message           string

		expectAddedLabels []string
		shouldComment     bool
	}{
		{
			name:              "Not a member opens a pull request without message configuration",
			action:            github.PullRequestActionOpened,
			author:            "author",
			authorAssociation: "CONTRIBUTOR",
			expectAddedLabels: []string{"org/repo#101:contribution"},
			shouldComment:     false,
		},
		{
			name:              "Not a member opens a pull request with a message configuration",
			action:            github.PullRequestActionOpened,
			author:            "author",
			message:           "Message",
			authorAssociation: "CONTRIBUTOR",
			expectAddedLabels: []string{"org/repo#101:contribution"},
			shouldComment:     true,
		},
		{
			name:              "Not a member reopens a pull request without message configuration",
			action:            github.PullRequestActionReopened,
			author:            "author",
			authorAssociation: "CONTRIBUTOR",
			expectAddedLabels: []string{},
			shouldComment:     false,
		},
		{
			name:              "Member opens a pull request without message configuration",
			action:            github.PullRequestActionOpened,
			author:            "member1",
			authorAssociation: "MEMBER",
			expectAddedLabels: []string{},
			shouldComment:     false,
		},
		{
			name:              "Member opens a pull request with a message configuration",
			action:            github.PullRequestActionOpened,
			author:            "member1",
			message:           "Message",
			authorAssociation: "MEMBER",
			expectAddedLabels: []string{},
			shouldComment:     false,
		},
		{
			name:              "Not a member first time opens a pull request to repo without message configuration",
			action:            github.PullRequestActionOpened,
			author:            "author",
			authorAssociation: firstTimeContributor,
			expectAddedLabels: []string{"org/repo#101:contribution", "org/repo#101:first-time-contributor"},
			shouldComment:     false,
		},
		{
			name:              "Not a member first time opens a pull request to repo with a message configuration",
			action:            github.PullRequestActionOpened,
			author:            "author",
			message:           "Message",
			authorAssociation: firstTimeContributor,
			expectAddedLabels: []string{"org/repo#101:contribution", "org/repo#101:first-time-contributor"},
			shouldComment:     true,
		},
		{
			name:              "Not a member first time opens a pull request to GitHub without message configuration",
			action:            github.PullRequestActionOpened,
			author:            "author",
			authorAssociation: firstTimer,
			expectAddedLabels: []string{"org/repo#101:contribution", "org/repo#101:first-time-contributor"},
			shouldComment:     false,
		},
		{
			name:              "Not a member first time opens a pull request to GitHub with a message configuration",
			action:            github.PullRequestActionOpened,
			author:            "author",
			message:           "Message",
			authorAssociation: firstTimer,
			expectAddedLabels: []string{"org/repo#101:contribution", "org/repo#101:first-time-contributor"},
			shouldComment:     true,
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		fc := &fakegithub.FakeClient{
			IssueComments:    make(map[int][]github.IssueComment),
			IssueLabelsAdded: []string{},
			OrgMembers: map[string][]string{
				"org": {
					"member1",
					"member2",
					"member4",
				},
			},
		}

		cfg := &externalplugins.Configuration{}
		cfg.TiCommunityContribution = []externalplugins.TiCommunityContribution{
			{
				Repos:   []string{"org/repo"},
				Message: tc.message,
			},
		}

		pe := &github.PullRequestEvent{
			Action: tc.action,
			Number: 101,
			PullRequest: github.PullRequest{
				User: github.User{
					Login: tc.author,
				},
				AuthorAssociation: tc.authorAssociation,
			},
			Repo: github.Repo{
				Owner: github.User{
					Login: "org",
				},
				Name: "repo",
			},
		}
		err := HandlePullRequestEvent(fc, pe, cfg, logrus.WithField("plugin", PluginName))
		if err != nil {
			t.Errorf("For case %s, didn't expect error: %v", tc.name, err)
		}

		sort.Strings(tc.expectAddedLabels)
		sort.Strings(fc.IssueLabelsAdded)
		if !reflect.DeepEqual(tc.expectAddedLabels, fc.IssueLabelsAdded) {
			t.Errorf("expected the labels %q to be added, but %q were added",
				tc.expectAddedLabels, fc.IssueLabelsAdded)
		}

		if !tc.shouldComment && len(fc.IssueCommentsAdded) != 0 {
			t.Errorf("unexpected comment %v", fc.IssueCommentsAdded)
		}

		if tc.shouldComment && len(fc.IssueCommentsAdded) == 0 {
			t.Fatalf("expected comments but got none")
		}
	}
}

func TestHelpProvider(t *testing.T) {
	enabledRepos := []config.OrgRepo{
		{Org: "org1", Repo: "repo"},
		{Org: "org2", Repo: "repo"},
	}
	cases := []struct {
		name               string
		config             *externalplugins.Configuration
		enabledRepos       []config.OrgRepo
		err                bool
		configInfoIncludes []string
		configInfoExcludes []string
	}{
		{
			name:               "Empty config",
			config:             &externalplugins.Configuration{},
			enabledRepos:       enabledRepos,
			configInfoExcludes: []string{"message: "},
		},
		{
			name: "All configs enabled",
			config: &externalplugins.Configuration{
				TiCommunityContribution: []externalplugins.TiCommunityContribution{
					{
						Repos:   []string{"org2/repo"},
						Message: "Message",
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{"message: "},
		},
	}
	for _, testcase := range cases {
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
