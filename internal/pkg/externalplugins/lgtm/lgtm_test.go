//nolint:scopelint
package lgtm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/externalplugins"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/ownersclient"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
)

var (
	lgtmOne = fmt.Sprintf("%s%d", LabelPrefix, 1)
	lgtmTwo = fmt.Sprintf("%s%d", LabelPrefix, 2)
)

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

type fakePruner struct {
	GitHubClient  *fakegithub.FakeClient
	IssueComments []github.IssueComment
}

func (fp *fakePruner) PruneComments(shouldPrune func(github.IssueComment) bool) {
	for _, comment := range fp.IssueComments {
		if shouldPrune(comment) {
			fp.GitHubClient.IssueCommentsDeleted = append(fp.GitHubClient.IssueCommentsDeleted, comment.Body)
		}
	}
}

func TestLGTMIssueAndReviewComment(t *testing.T) {
	var testcases = []struct {
		name          string
		body          string
		commenter     string
		currentLabel  string
		isCancel      bool
		shouldToggle  bool
		storeTreeHash bool
	}{
		{
			name:         "non-lgtm comment",
			body:         "uh oh",
			commenter:    "collab1",
			shouldToggle: false,
		},
		{
			name:         "lgtm comment by reviewer collab1, no lgtm on pr",
			body:         "/lgtm",
			commenter:    "collab1",
			shouldToggle: true,
		},
		{
			name:         "LGTM comment by reviewer collab1, no lgtm on pr",
			body:         "/LGTM",
			commenter:    "collab1",
			shouldToggle: true,
		},
		{
			name:         "lgtm comment by reviewer collab1, lgtm on pr",
			body:         "/lgtm",
			commenter:    "collab1",
			currentLabel: lgtmOne,
			shouldToggle: true,
		},
		{
			name:         "lgtm comment by author",
			body:         "/lgtm",
			commenter:    "author",
			shouldToggle: false,
		},
		{
			name:         "lgtm cancel by author",
			body:         "/lgtm cancel",
			commenter:    "author",
			currentLabel: lgtmOne,
			isCancel:     true,
			shouldToggle: true,
		},
		{
			name:         "lgtm comment by reviewer collab2",
			body:         "/lgtm",
			commenter:    "collab2",
			shouldToggle: true,
		},
		{
			name:         "lgtm comment by reviewer collab2, with trailing space",
			body:         "/lgtm ",
			commenter:    "collab2",
			shouldToggle: true,
		},
		{
			name:         "lgtm comment by reviewer collab2, with no-issue",
			body:         "/lgtm no-issue",
			commenter:    "collab2",
			shouldToggle: true,
		},
		{
			name:         "lgtm comment by reviewer collab2, with no-issue and trailing space",
			body:         "/lgtm no-issue \r",
			commenter:    "collab2",
			shouldToggle: true,
		},
		{
			name:         "lgtm comment by random",
			body:         "/lgtm",
			commenter:    "not-in-the-org",
			shouldToggle: false,
		},
		{
			name:         "lgtm cancel by reviewer collab2",
			body:         "/lgtm cancel",
			commenter:    "collab2",
			currentLabel: lgtmOne,
			isCancel:     true,
			shouldToggle: true,
		},
		{
			name:         "lgtm cancel by random",
			body:         "/lgtm cancel",
			commenter:    "not-in-the-org",
			currentLabel: lgtmOne,
			isCancel:     true,
			shouldToggle: false,
		},
		{
			name:         "lgtm cancel comment by reviewer collab1",
			body:         "/lgtm cancel",
			commenter:    "collab1",
			currentLabel: lgtmOne,
			isCancel:     true,
			shouldToggle: true,
		},
		{
			name:         "lgtm cancel comment by reviewer collab1, with trailing space",
			body:         "/lgtm cancel \r",
			commenter:    "collab1",
			currentLabel: lgtmOne,
			isCancel:     true,
			shouldToggle: true,
		},
		{
			name:         "lgtm cancel comment by reviewer collab1, no lgtm",
			body:         "/lgtm cancel",
			commenter:    "collab1",
			isCancel:     true,
			shouldToggle: false,
		},
		{
			name:         "lgtm comment by reviewer collab2, LGTM is enough",
			body:         "/lgtm ",
			commenter:    "collab2",
			currentLabel: lgtmTwo,
			shouldToggle: false,
		},
		{
			name:         "lgtm comment by random, LGTM is enough",
			body:         "/lgtm ",
			commenter:    "not-in-the-org",
			currentLabel: lgtmTwo,
			shouldToggle: false,
		},
	}
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	for _, tc := range testcases {
		t.Logf("Running scenario %q", tc.name)
		// Test issue comments.
		{
			fc := &fakegithub.FakeClient{
				IssueComments: make(map[int][]github.IssueComment),
				PullRequests: map[int]*github.PullRequest{
					5: {
						Base: github.PullRequestBranch{
							Ref: "master",
						},
						Head: github.PullRequestBranch{
							SHA: SHA,
						},
						User:   github.User{Login: "author"},
						Number: 5,
						State:  "open",
					},
				},
				PullRequestChanges: map[int][]github.PullRequestChange{
					5: {
						{Filename: "doc/README.md"},
					},
				},
				Collaborators: []string{"collab1", "collab2"},
			}
			e := &github.IssueCommentEvent{
				Action: github.IssueCommentActionCreated,
				Issue: github.Issue{
					User:   github.User{Login: "author"},
					Number: 5,
					State:  "open",
					PullRequest: &struct {
					}{},
				},
				Comment: github.IssueComment{
					Body:    tc.body,
					User:    github.User{Login: tc.commenter},
					HTMLURL: "<url>",
				},
				Repo: github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
			}
			if tc.currentLabel != "" {
				fc.IssueLabelsAdded = []string{"org/repo#5:" + tc.currentLabel}
			}

			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityLgtm = []externalplugins.TiCommunityLgtm{
				{
					Repos:              []string{"org/repo"},
					ReviewActsAsLgtm:   true,
					PullOwnersEndpoint: "https://fake/ti-community-bot",
				},
			}

			foc := &fakeOwnersClient{
				reviewers: []string{"collab1", "collab2"},
				needsLgtm: 2,
			}

			if err := HandleIssueCommentEvent(fc, e, cfg, foc, logrus.WithField("plugin", PluginName)); err != nil {
				t.Errorf("didn't expect error from lgtmComment: %v", err)
				continue
			}

			if tc.shouldToggle {
				if tc.currentLabel != "" {
					if len(fc.IssueLabelsRemoved) != 1 {
						t.Error("should have removed " + lgtmOne + ".")
					} else if len(fc.IssueLabelsAdded) != 2 && !tc.isCancel {
						t.Error("should have added " + lgtmTwo + ".")
					}
				} else {
					if len(fc.IssueLabelsAdded) == 0 {
						t.Error("should have added " + lgtmOne + ".")
					} else if len(fc.IssueLabelsRemoved) > 0 {
						t.Error("should not have removed " + lgtmOne)
					}
				}
			} else if len(fc.IssueLabelsRemoved) > 0 {
				t.Error("should not have removed " + lgtmOne + ".")
			}
		}

		// Test review comments.
		{
			fc := &fakegithub.FakeClient{
				IssueComments: make(map[int][]github.IssueComment),
				PullRequests: map[int]*github.PullRequest{
					5: {
						Base: github.PullRequestBranch{
							Ref: "master",
						},
						Head: github.PullRequestBranch{
							SHA: SHA,
						},
						User:   github.User{Login: "author"},
						Number: 5,
						State:  "open",
					},
				},
				PullRequestChanges: map[int][]github.PullRequestChange{
					5: {
						{Filename: "doc/README.md"},
					},
				},
				Collaborators: []string{"collab1", "collab2"},
			}
			e := &github.ReviewCommentEvent{
				Action: github.ReviewCommentActionCreated,
				Comment: github.ReviewComment{
					Body:    tc.body,
					User:    github.User{Login: tc.commenter},
					HTMLURL: "<url>",
				},
				Repo:        github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
				PullRequest: *fc.PullRequests[5],
			}
			if tc.currentLabel != "" {
				fc.IssueLabelsAdded = []string{"org/repo#5:" + tc.currentLabel}
			}

			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityLgtm = []externalplugins.TiCommunityLgtm{
				{
					Repos:              []string{"org/repo"},
					ReviewActsAsLgtm:   true,
					PullOwnersEndpoint: "https://fake/ti-community-bot",
				},
			}

			foc := &fakeOwnersClient{
				reviewers: []string{"collab1", "collab2"},
				needsLgtm: 2,
			}

			if err := HandlePullReviewCommentEvent(fc, e, cfg, foc, logrus.WithField("plugin", PluginName)); err != nil {
				t.Errorf("didn't expect error from lgtmComment: %v", err)
				continue
			}

			if tc.shouldToggle {
				if tc.currentLabel != "" {
					if len(fc.IssueLabelsRemoved) != 1 {
						t.Error("should have removed " + lgtmOne + ".")
					} else if len(fc.IssueLabelsAdded) != 2 && !tc.isCancel {
						t.Error("should have added " + lgtmTwo + ".")
					}
				} else {
					if len(fc.IssueLabelsAdded) == 0 {
						t.Error("should have added " + lgtmOne + ".")
					} else if len(fc.IssueLabelsRemoved) > 0 {
						t.Error("should not have removed " + lgtmOne)
					}
				}
			} else if len(fc.IssueLabelsRemoved) > 0 {
				t.Error("should not have removed " + lgtmOne + ".")
			}
		}
	}
}

func TestLGTMFromApproveReview(t *testing.T) {
	var testcases = []struct {
		name          string
		state         github.ReviewState
		action        github.ReviewEventAction
		body          string
		reviewer      string
		currentLabel  string
		isCancel      bool
		shouldToggle  bool
		storeTreeHash bool
	}{
		{
			name:          "Edit approve review by reviewer, no lgtm on pr",
			state:         github.ReviewStateApproved,
			action:        github.ReviewActionEdited,
			reviewer:      "collab1",
			shouldToggle:  false,
			storeTreeHash: true,
		},
		{
			name:          "Dismiss approve review by reviewer, no lgtm on pr",
			state:         github.ReviewStateApproved,
			action:        github.ReviewActionDismissed,
			reviewer:      "collab1",
			shouldToggle:  false,
			storeTreeHash: true,
		},
		{
			name:         "Request changes review by reviewer, no lgtm on pr",
			state:        github.ReviewStateChangesRequested,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			shouldToggle: false,
		},
		{
			name:         "Request changes review by reviewer, lgtm on pr",
			state:        github.ReviewStateChangesRequested,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			currentLabel: lgtmOne,
			isCancel:     true,
			shouldToggle: true,
		},
		{
			name:          "Approve review by reviewer, no lgtm on pr",
			state:         github.ReviewStateApproved,
			action:        github.ReviewActionSubmitted,
			reviewer:      "collab1",
			currentLabel:  lgtmOne,
			shouldToggle:  true,
			storeTreeHash: true,
		},
		{
			name:         "Approve review by reviewer, no lgtm on pr, do not store tree_hash",
			state:        github.ReviewStateApproved,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			currentLabel: lgtmOne,
			shouldToggle: true,
		},
		{
			name:         "Approve review by reviewer, LGTM is enough",
			state:        github.ReviewStateApproved,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			currentLabel: lgtmTwo,
			shouldToggle: false,
		},
		{
			name:          "Approve review by non-reviewer, no lgtm on pr",
			state:         github.ReviewStateApproved,
			action:        github.ReviewActionSubmitted,
			reviewer:      "collab2",
			shouldToggle:  false,
			storeTreeHash: true,
		},
		{
			name:         "Request changes review by non-reviewer, no lgtm on pr",
			state:        github.ReviewStateChangesRequested,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab2",
			shouldToggle: false,
		},
		{
			name:         "Approve review by random",
			state:        github.ReviewStateApproved,
			action:       github.ReviewActionSubmitted,
			reviewer:     "not-in-the-org",
			shouldToggle: false,
		},
		{
			name:         "Comment review by issue author, no lgtm on pr",
			state:        github.ReviewStateCommented,
			action:       github.ReviewActionSubmitted,
			reviewer:     "author",
			shouldToggle: false,
		},
		{
			name:         "Comment body has /lgtm on Comment Review ",
			state:        github.ReviewStateCommented,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			body:         "/lgtm",
			currentLabel: lgtmOne,
			shouldToggle: false,
		},
		{
			name:         "Comment body has /lgtm cancel on Approve Review",
			state:        github.ReviewStateApproved,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			body:         "/lgtm cancel",
			currentLabel: lgtmOne,
			isCancel:     true,
			shouldToggle: false,
		},
	}
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	for _, tc := range testcases {
		fc := &fakegithub.FakeClient{
			IssueComments:    make(map[int][]github.IssueComment),
			IssueLabelsAdded: []string{},
			PullRequests: map[int]*github.PullRequest{
				5: {
					Head: github.PullRequestBranch{
						SHA: SHA,
					},
				},
			},
			Collaborators: []string{"collab1", "collab2"},
		}
		e := &github.ReviewEvent{
			Action: tc.action,
			Review: github.Review{Body: tc.body, State: tc.state, HTMLURL: "<url>", User: github.User{Login: tc.reviewer}},
			PullRequest: github.PullRequest{
				User: github.User{Login: "author"},
				Assignees: []github.User{
					{Login: "collab1"},
					{Login: "assignee1"}},
				Number: 5},
			Repo: github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
		}
		if tc.currentLabel != "" {
			fc.IssueLabelsAdded = append(fc.IssueLabelsAdded, "org/repo#5:"+tc.currentLabel)
		}

		cfg := &externalplugins.Configuration{}
		cfg.TiCommunityLgtm = []externalplugins.TiCommunityLgtm{
			{
				Repos:              []string{"org/repo"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://fake/ti-community-bot",
			},
		}

		foc := &fakeOwnersClient{
			reviewers: []string{"collab1"},
			needsLgtm: 2,
		}

		if err := HandlePullReviewEvent(fc, e, cfg, foc, logrus.WithField("plugin", PluginName)); err != nil {
			t.Errorf("For case %s, didn't expect error from pull request review: %v", tc.name, err)
			continue
		}

		if tc.shouldToggle {
			if tc.currentLabel != "" {
				if len(fc.IssueLabelsRemoved) != 1 {
					t.Error("should have removed " + lgtmOne + ".")
				} else if len(fc.IssueLabelsAdded) != 2 && !tc.isCancel {
					t.Error("should have added " + lgtmTwo + ".")
				}
			} else {
				if len(fc.IssueLabelsAdded) == 0 {
					t.Error("should have added " + lgtmOne + ".")
				} else if len(fc.IssueLabelsRemoved) > 0 {
					t.Error("should not have removed " + lgtmOne)
				}
			}
		} else if len(fc.IssueLabelsRemoved) > 0 {
			t.Error("should not have removed " + lgtmOne + ".")
		}
	}
}

func TestGetCurrentAndNextLabel(t *testing.T) {
	var testcases = []struct {
		name               string
		labels             []github.Label
		needsLgtm          int
		exceptCurrentLabel string
		exceptNextLabel    string
	}{
		{
			name:               "Current no LGTM, needs 1 LGTM",
			labels:             []github.Label{},
			needsLgtm:          1,
			exceptCurrentLabel: "",
			exceptNextLabel:    lgtmOne,
		},
		{
			name:               "Current no LGTM, needs 2 LGTM",
			labels:             []github.Label{},
			needsLgtm:          2,
			exceptCurrentLabel: "",
			exceptNextLabel:    lgtmOne,
		},
		{
			name: "Current LGT1, needs 1 LGTM",
			labels: []github.Label{
				{
					Name: lgtmOne,
				},
			},
			needsLgtm:          1,
			exceptCurrentLabel: lgtmOne,
			exceptNextLabel:    "",
		},
		{
			name: "Current LGT1, needs 2 LGTM",
			labels: []github.Label{
				{
					Name: lgtmOne,
				},
			},
			needsLgtm:          2,
			exceptCurrentLabel: lgtmOne,
			exceptNextLabel:    lgtmTwo,
		},
		{
			name: "Current LGT2, needs 2 LGTM",
			labels: []github.Label{
				{
					Name: lgtmTwo,
				},
			},
			needsLgtm:          2,
			exceptCurrentLabel: lgtmTwo,
			exceptNextLabel:    "",
		},
	}

	// scopelint:ignore
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			currentLabel, nextLabel := getCurrentAndNextLabel(LabelPrefix, tc.labels, tc.needsLgtm)

			if currentLabel != tc.exceptCurrentLabel {
				t.Fatalf("currentLabel mismatch: got %v, want %v", currentLabel, tc.exceptCurrentLabel)
			}

			if nextLabel != tc.exceptNextLabel {
				t.Fatalf("nextLabel mismatch: got %v, want %v", nextLabel, tc.exceptNextLabel)
			}
		})
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
			configInfoExcludes: []string{configInfoReviewActsAsLgtm},
		},
		{
			name: "All configs enabled",
			config: &externalplugins.Configuration{
				TiCommunityLgtm: []externalplugins.TiCommunityLgtm{
					{
						Repos:              []string{"org2/repo"},
						ReviewActsAsLgtm:   true,
						PullOwnersEndpoint: "https://fake",
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{configInfoReviewActsAsLgtm},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			helpProvider := HelpProvider(c.config)
			pluginHelp, err := helpProvider(c.enabledRepos)
			if err != nil && !c.err {
				t.Fatalf("helpProvider error: %v", err)
			}
			for _, msg := range c.configInfoExcludes {
				if strings.Contains(pluginHelp.Config["org2/repo"], msg) {
					t.Fatalf("helpProvider.Config error mismatch: got %v, but didn't want it", msg)
				}
			}
			for _, msg := range c.configInfoIncludes {
				if !strings.Contains(pluginHelp.Config["org2/repo"], msg) {
					t.Fatalf("helpProvider.Config error mismatch: didn't get %v, but wanted it", msg)
				}
			}
		})
	}
}
