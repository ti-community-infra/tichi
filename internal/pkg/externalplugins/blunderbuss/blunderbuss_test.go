//nolint:scopelint
package blunderbuss

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/ti-community-prow/internal/pkg/externalplugins"
	"github.com/ti-community-infra/ti-community-prow/internal/pkg/ownersclient"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
)

type fakeGitHubClient struct {
	pr        *github.PullRequest
	requested []string
}

func newFakeGitHubClient(pr *github.PullRequest) *fakeGitHubClient {
	return &fakeGitHubClient{pr: pr}
}

func (c *fakeGitHubClient) RequestReview(org, repo string, number int, logins []string) error {
	if org != "org" {
		return errors.New("org should be 'org'")
	}
	if repo != "repo" {
		return errors.New("repo should be 'repo'")
	}
	if number != 5 {
		return errors.New("number should be 5")
	}
	c.requested = append(c.requested, logins...)
	return nil
}

func (c *fakeGitHubClient) GetPullRequest(_, _ string, _ int) (*github.PullRequest, error) {
	return c.pr, nil
}

type fakeOwnersClient struct {
	reviewers []string
	needsLgtm int
}

func (f *fakeOwnersClient) LoadOwners(_ string,
	_, _ string, _ int) (*ownersclient.Owners, error) {
	return &ownersclient.Owners{
		Reviewers: f.reviewers,
		NeedsLgtm: f.needsLgtm,
	}, nil
}

func TestHandleIssueCommentEvent(t *testing.T) {
	var testcases = []struct {
		name              string
		action            github.IssueCommentEventAction
		issueState        string
		isPR              bool
		body              string
		maxReviewersCount int
		excludeReviewers  []string

		expectReviewerCount int
	}{
		{
			name:                "no-auto-cc comment",
			action:              github.IssueCommentActionCreated,
			issueState:          "open",
			isPR:                true,
			body:                "uh oh",
			maxReviewersCount:   1,
			expectReviewerCount: 0,
		},
		{
			name:                "comment with a valid command in an open PR triggers auto-assignment",
			action:              github.IssueCommentActionCreated,
			issueState:          "open",
			isPR:                true,
			body:                "/auto-cc",
			maxReviewersCount:   1,
			expectReviewerCount: 1,
		},
		{
			name:                "comment with an invalid command in an open PR will not trigger auto-assignment",
			action:              github.IssueCommentActionCreated,
			issueState:          "open",
			isPR:                true,
			body:                "/automatic-review",
			maxReviewersCount:   1,
			expectReviewerCount: 0,
		},
		{
			name:                "comment with a valid command in a closed PR will not trigger auto-assignment",
			action:              github.IssueCommentActionCreated,
			issueState:          "closed",
			isPR:                true,
			body:                "/auto-cc",
			maxReviewersCount:   2,
			expectReviewerCount: 0,
		},
		{
			name:                "comment deleted from an open PR will not trigger auto-assignment",
			action:              github.IssueCommentActionDeleted,
			issueState:          "open",
			isPR:                true,
			body:                "/auto-cc",
			maxReviewersCount:   2,
			expectReviewerCount: 0,
		},
		{
			name:                "comment with valid command in an open issue will not trigger auto-assignment",
			action:              github.IssueCommentActionCreated,
			issueState:          "open",
			isPR:                false,
			body:                "/auto-cc",
			maxReviewersCount:   2,
			expectReviewerCount: 0,
		},
		{
			name:       "comment with a valid command in an open PR triggers auto-assignment and exclude some reviewers",
			action:     github.IssueCommentActionCreated,
			issueState: "open",
			isPR:       true,
			body:       "/auto-cc",
			excludeReviewers: []string{
				"collab2",
			},
			maxReviewersCount:   2,
			expectReviewerCount: 1,
		},
	}

	for _, tc := range testcases {
		t.Logf("Running scenario %q", tc.name)
		pr := github.PullRequest{Number: 5, User: github.User{Login: "author"}}
		fc := newFakeGitHubClient(&pr)
		e := &github.IssueCommentEvent{
			Action: tc.action,
			Issue: github.Issue{
				User:   github.User{Login: "author"},
				Number: 5,
				State:  tc.issueState,
			},
			Comment: github.IssueComment{
				Body:    tc.body,
				User:    github.User{Login: "commenter"},
				HTMLURL: "<url>",
			},
			Repo: github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
		}
		if tc.isPR {
			e.Issue.PullRequest = &struct {
			}{}
		}
		cfg := &externalplugins.Configuration{}
		cfg.TiCommunityBlunderbuss = []externalplugins.TiCommunityBlunderbuss{
			{
				Repos:              []string{"org/repo"},
				MaxReviewerCount:   tc.maxReviewersCount,
				ExcludeReviewers:   tc.excludeReviewers,
				PullOwnersEndpoint: "https://fake/ti-community-bot",
			},
		}

		foc := &fakeOwnersClient{
			reviewers: []string{"collab1", "collab2"},
			needsLgtm: 2,
		}

		if err := HandleIssueCommentEvent(fc, e, cfg, foc, logrus.WithField("plugin", PluginName)); err != nil {
			t.Errorf("didn't expect error from autoccComment: %v", err)
			continue
		}

		if len(fc.requested) != tc.expectReviewerCount {
			t.Fatalf("reviewers count mismatch: got %v, want %v", len(fc.requested), tc.expectReviewerCount)
		}
	}
}

func TestHandlePullRequest(t *testing.T) {
	var testcases = []struct {
		name              string
		action            github.PullRequestEventAction
		body              string
		maxReviewersCount int
		excludeReviewers  []string

		expectReviewerCount int
	}{
		{
			name:                "PR opened",
			action:              github.PullRequestActionOpened,
			body:                "/auto-cc",
			maxReviewersCount:   2,
			expectReviewerCount: 2,
		},
		{
			name:                "PR opened with /cc command",
			action:              github.PullRequestActionOpened,
			body:                "/cc",
			maxReviewersCount:   2,
			expectReviewerCount: 0,
		},
		{
			name:                "PR closed",
			action:              github.PullRequestActionClosed,
			body:                "/auto-cc",
			maxReviewersCount:   2,
			expectReviewerCount: 0,
		},
		{
			name:              "PR opened with exclude reviewers",
			action:            github.PullRequestActionOpened,
			body:              "/auto-cc",
			maxReviewersCount: 2,
			excludeReviewers: []string{
				"collab2",
			},

			expectReviewerCount: 1,
		},
	}

	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	for _, tc := range testcases {
		t.Logf("Running scenario %q", tc.name)
		pr := github.PullRequest{Number: 5, User: github.User{Login: "author"}, Body: tc.body}
		fc := newFakeGitHubClient(&pr)
		e := &github.PullRequestEvent{
			Action: tc.action,
			PullRequest: github.PullRequest{
				Number: 5,
				Body:   tc.body,
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						Owner: github.User{
							Login: "org",
						},
						Name: "repo",
					},
				},
				Head: github.PullRequestBranch{
					SHA: SHA,
				},
			},
			Repo: github.Repo{
				Owner: github.User{Login: "org"},
				Name:  "repo",
			},
		}

		cfg := &externalplugins.Configuration{}
		cfg.TiCommunityBlunderbuss = []externalplugins.TiCommunityBlunderbuss{
			{
				Repos:              []string{"org/repo"},
				MaxReviewerCount:   tc.maxReviewersCount,
				ExcludeReviewers:   tc.excludeReviewers,
				PullOwnersEndpoint: "https://fake/ti-community-bot",
			},
		}

		foc := &fakeOwnersClient{
			reviewers: []string{"collab1", "collab2"},
			needsLgtm: 2,
		}

		if err := HandlePullRequestEvent(fc, e, cfg, foc, logrus.WithField("plugin", PluginName)); err != nil {
			t.Errorf("didn't expect error from autoccComment: %v", err)
			continue
		}

		if len(fc.requested) != tc.expectReviewerCount {
			t.Fatalf("reviewers count mismatch: got %v, want %v", len(fc.requested), tc.expectReviewerCount)
		}
	}
}

func TestGetReviewers(t *testing.T) {
	var testcases = []struct {
		name             string
		author           string
		reviewers        []string
		excludeReviewers []string

		expectReviewers []string
	}{
		{
			name:   "non exclude reviewers",
			author: "author",
			reviewers: []string{
				"author", "reviewers1", "reviewers2", "reviewers3",
			},
			expectReviewers: []string{
				"reviewers1", "reviewers2", "reviewers3",
			},
		},
		{
			name:   "exclude reviewers",
			author: "author",
			reviewers: []string{
				"author", "reviewers1", "reviewers2", "reviewers3",
			},
			excludeReviewers: []string{
				"reviewers2",
			},
			expectReviewers: []string{
				"reviewers1", "reviewers3",
			},
		},
	}

	for _, tc := range testcases {
		reviewers := getReviewers(tc.author, tc.reviewers, tc.excludeReviewers, logrus.WithField("plugin", PluginName))
		sort.Strings(reviewers)
		sort.Strings(tc.expectReviewers)
		if !reflect.DeepEqual(reviewers, tc.expectReviewers) {
			t.Errorf("[%s] expected the requested reviewers to be %q, but got %q.", tc.name, tc.excludeReviewers, reviewers)
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
	}{
		{
			name:               "Empty config",
			config:             &externalplugins.Configuration{},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{},
		},
		{
			name: "ReviewerCount specified",
			config: &externalplugins.Configuration{
				TiCommunityBlunderbuss: []externalplugins.TiCommunityBlunderbuss{
					{
						Repos:              []string{"org2/repo"},
						MaxReviewerCount:   2,
						ExcludeReviewers:   []string{},
						PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{configString(2)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			epa := &externalplugins.ConfigAgent{}
			epa.Set(c.config)

			helpProvider := HelpProvider(epa)
			pluginHelp, err := helpProvider(c.enabledRepos)
			if err != nil && !c.err {
				t.Fatalf("helpProvider error: %v", err)
			}
			for _, msg := range c.configInfoIncludes {
				if !strings.Contains(pluginHelp.Config["org2/repo"], msg) {
					t.Fatalf("helpProvider.Config error mismatch: didn't get %v, but wanted it", msg)
				}
			}
		})
	}
}
