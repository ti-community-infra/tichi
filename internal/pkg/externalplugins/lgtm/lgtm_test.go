//nolint:scopelint
package lgtm

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/externalplugins"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/ownersclient"
	"k8s.io/apimachinery/pkg/api/equality"
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
		shouldComment bool
		storeTreeHash bool
	}{
		{
			name:         "non-lgtm comment",
			body:         "uh oh",
			commenter:    "collab1",
			shouldToggle: false,
		},
		{
			name:          "lgtm comment by reviewer collab1, no lgtm on pr",
			body:          "/lgtm",
			commenter:     "collab1",
			shouldToggle:  true,
			shouldComment: true,
		},
		{
			name:          "LGTM comment by reviewer collab1, no lgtm on pr",
			body:          "/LGTM",
			commenter:     "collab1",
			shouldToggle:  true,
			shouldComment: true,
		},
		{
			name:          "lgtm comment by reviewer collab1, lgtm on pr",
			body:          "/lgtm",
			commenter:     "collab1",
			currentLabel:  lgtmOne,
			shouldToggle:  true,
			shouldComment: true,
		},
		{
			name:          "lgtm comment by author",
			body:          "/lgtm",
			commenter:     "author",
			shouldToggle:  false,
			shouldComment: true,
		},
		{
			name:          "lgtm cancel by author",
			body:          "/lgtm cancel",
			commenter:     "author",
			currentLabel:  lgtmOne,
			isCancel:      true,
			shouldToggle:  true,
			shouldComment: false,
		},
		{
			name:          "lgtm comment by reviewer collab2",
			body:          "/lgtm",
			commenter:     "collab2",
			shouldToggle:  true,
			shouldComment: true,
		},
		{
			name:          "lgtm comment by reviewer collab2, with trailing space",
			body:          "/lgtm ",
			commenter:     "collab2",
			shouldToggle:  true,
			shouldComment: true,
		},
		{
			name:          "lgtm comment by reviewer collab2, with no-issue",
			body:          "/lgtm no-issue",
			commenter:     "collab2",
			shouldToggle:  true,
			shouldComment: true,
		},
		{
			name:          "lgtm comment by reviewer collab2, with no-issue and trailing space",
			body:          "/lgtm no-issue \r",
			commenter:     "collab2",
			shouldToggle:  true,
			shouldComment: true,
		},
		{
			name:          "lgtm comment by random",
			body:          "/lgtm",
			commenter:     "not-in-the-org",
			shouldToggle:  false,
			shouldComment: true,
		},
		{
			name:          "lgtm cancel by reviewer collab2",
			body:          "/lgtm cancel",
			commenter:     "collab2",
			currentLabel:  lgtmOne,
			isCancel:      true,
			shouldToggle:  true,
			shouldComment: false,
		},
		{
			name:          "lgtm cancel by random",
			body:          "/lgtm cancel",
			commenter:     "not-in-the-org",
			currentLabel:  lgtmOne,
			isCancel:      true,
			shouldToggle:  false,
			shouldComment: true,
		},
		{
			name:          "lgtm cancel comment by reviewer collab1",
			body:          "/lgtm cancel",
			commenter:     "collab1",
			currentLabel:  lgtmOne,
			isCancel:      true,
			shouldToggle:  true,
			shouldComment: false,
		},
		{
			name:          "lgtm cancel comment by reviewer collab1, with trailing space",
			body:          "/lgtm cancel \r",
			commenter:     "collab1",
			currentLabel:  lgtmOne,
			isCancel:      true,
			shouldToggle:  true,
			shouldComment: false,
		},
		{
			name:          "lgtm cancel comment by reviewer collab1, no lgtm",
			body:          "/lgtm cancel",
			commenter:     "collab1",
			isCancel:      true,
			shouldToggle:  false,
			shouldComment: false,
		},
		{
			name:          "lgtm comment by reviewer collab2, LGTM is enough",
			body:          "/lgtm ",
			commenter:     "collab2",
			currentLabel:  lgtmTwo,
			shouldToggle:  false,
			shouldComment: false,
		},
		{
			name:          "lgtm comment by random, LGTM is enough",
			body:          "/lgtm ",
			commenter:     "not-in-the-org",
			currentLabel:  lgtmTwo,
			shouldToggle:  false,
			shouldComment: true,
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
					Repos:            []string{"org/repo"},
					ReviewActsAsLgtm: true,
					StoreTreeHash:    true,
					PullOwnersURL:    "https://fake/ti-community-bot",
				},
			}

			foc := &fakeOwnersClient{
				reviewers: []string{"collab1", "collab2"},
				needsLgtm: 2,
			}

			cp := &fakePruner{
				GitHubClient:  fc,
				IssueComments: fc.IssueComments[5],
			}

			if err := HandleIssueCommentEvent(fc, e, cfg, foc, cp, logrus.WithField("plugin", PluginName)); err != nil {
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

			if tc.shouldComment && len(fc.IssueComments[5]) != 1 {
				t.Error("should have commented.")
			} else if !tc.shouldComment && len(fc.IssueComments[5]) != 0 {
				t.Error("should not have commented.")
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
					Repos:            []string{"org/repo"},
					ReviewActsAsLgtm: true,
					StoreTreeHash:    true,
					PullOwnersURL:    "https://fake/ti-community-bot",
				},
			}

			foc := &fakeOwnersClient{
				reviewers: []string{"collab1", "collab2"},
				needsLgtm: 2,
			}

			cp := &fakePruner{
				GitHubClient:  fc,
				IssueComments: fc.IssueComments[5],
			}

			if err := HandlePullReviewCommentEvent(fc, e, cfg, foc, cp, logrus.WithField("plugin", PluginName)); err != nil {
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

			if tc.shouldComment && len(fc.IssueComments[5]) != 1 {
				t.Error("should have commented.")
			} else if !tc.shouldComment && len(fc.IssueComments[5]) != 0 {
				t.Error("should not have commented.")
			}
		}
	}
}

func TestLGTMIssueCommentWithLGTMNoti(t *testing.T) {
	var testcases = []struct {
		name         string
		body         string
		commenter    string
		shouldDelete bool
	}{
		{
			name:         "non-lgtm comment",
			body:         "uh oh",
			commenter:    "collab2",
			shouldDelete: false,
		},
		{
			name:         "lgtm comment by reviewer collab1, no lgtm on pr",
			body:         "/lgtm",
			commenter:    "collab1",
			shouldDelete: true,
		},
		{
			name:         "LGTM comment by reviewer collab1, no lgtm on pr",
			body:         "/LGTM",
			commenter:    "collab1",
			shouldDelete: true,
		},
		{
			name:         "lgtm comment by author",
			body:         "/lgtm",
			commenter:    "author",
			shouldDelete: false,
		},
		{
			name:         "lgtm comment by reviewer collab2",
			body:         "/lgtm",
			commenter:    "collab2",
			shouldDelete: true,
		},
		{
			name:         "lgtm comment by reviewer collab2, with trailing space",
			body:         "/lgtm ",
			commenter:    "collab2",
			shouldDelete: true,
		},
		{
			name:         "lgtm comment by reviewer collab2, with no-issue",
			body:         "/lgtm no-issue",
			commenter:    "collab2",
			shouldDelete: true,
		},
		{
			name:         "lgtm comment by reviewer collab2, with no-issue and trailing space",
			body:         "/lgtm no-issue \r",
			commenter:    "collab2",
			shouldDelete: true,
		},
		{
			name:         "lgtm comment by random",
			body:         "/lgtm",
			commenter:    "not-in-the-org",
			shouldDelete: false,
		},
		{
			name:         "lgtm cancel comment by reviewer collab1, no lgtm",
			body:         "/lgtm cancel",
			commenter:    "collab1",
			shouldDelete: false,
		},
	}
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	for _, tc := range testcases {
		fc := &fakegithub.FakeClient{
			IssueComments: make(map[int][]github.IssueComment),
			PullRequests: map[int]*github.PullRequest{
				5: {
					Head: github.PullRequestBranch{
						SHA: SHA,
					},
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
		botName, err := fc.BotName()
		if err != nil {
			t.Fatalf("For case %s, could not get Bot nam", tc.name)
		}
		ic := github.IssueComment{
			User: github.User{
				Login: botName,
			},
			Body: removeLGTMLabelNoti,
		}
		fc.IssueComments[5] = append(fc.IssueComments[5], ic)

		cfg := &externalplugins.Configuration{}
		cfg.TiCommunityLgtm = []externalplugins.TiCommunityLgtm{
			{
				Repos:            []string{"org/repo"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				PullOwnersURL:    "https://fake/ti-community-bot",
			},
		}

		foc := &fakeOwnersClient{
			reviewers: []string{"collab1", "collab2"},
			needsLgtm: 2,
		}

		cp := &fakePruner{
			GitHubClient:  fc,
			IssueComments: fc.IssueComments[5],
		}

		if err := HandleIssueCommentEvent(fc, e, cfg, foc, cp, logrus.WithField("plugin", PluginName)); err != nil {
			t.Errorf("For case %s, didn't expect error from lgtmComment: %v", tc.name, err)
			continue
		}

		deleted := false
		for _, body := range fc.IssueCommentsDeleted {
			if body == removeLGTMLabelNoti {
				deleted = true
				break
			}
		}
		if tc.shouldDelete {
			if !deleted {
				t.Errorf("For case %s, LGTM removed notification should have been deleted", tc.name)
			}
		} else {
			if deleted {
				t.Errorf("For case %s, LGTM removed notification should not have been deleted", tc.name)
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
		shouldComment bool
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
			name:          "Request changes review by reviewer, no lgtm on pr",
			state:         github.ReviewStateChangesRequested,
			action:        github.ReviewActionSubmitted,
			reviewer:      "collab1",
			shouldToggle:  false,
			shouldComment: false,
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
			shouldComment: true,
			storeTreeHash: true,
		},
		{
			name:          "Approve review by reviewer, no lgtm on pr, do not store tree_hash",
			state:         github.ReviewStateApproved,
			action:        github.ReviewActionSubmitted,
			reviewer:      "collab1",
			currentLabel:  lgtmOne,
			shouldToggle:  true,
			shouldComment: false,
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
			shouldComment: true,
			storeTreeHash: true,
		},
		{
			name:          "Request changes review by non-reviewer, no lgtm on pr",
			state:         github.ReviewStateChangesRequested,
			action:        github.ReviewActionSubmitted,
			reviewer:      "collab2",
			shouldToggle:  false,
			shouldComment: true,
		},
		{
			name:          "Approve review by random",
			state:         github.ReviewStateApproved,
			action:        github.ReviewActionSubmitted,
			reviewer:      "not-in-the-org",
			shouldToggle:  false,
			shouldComment: true,
		},
		{
			name:          "Comment review by issue author, no lgtm on pr",
			state:         github.ReviewStateCommented,
			action:        github.ReviewActionSubmitted,
			reviewer:      "author",
			shouldToggle:  false,
			shouldComment: false,
		},
		{
			name:          "Comment body has /lgtm on Comment Review ",
			state:         github.ReviewStateCommented,
			action:        github.ReviewActionSubmitted,
			reviewer:      "collab1",
			body:          "/lgtm",
			currentLabel:  lgtmOne,
			shouldToggle:  false,
			shouldComment: false,
		},
		{
			name:          "Comment body has /lgtm cancel on Approve Review",
			state:         github.ReviewStateApproved,
			action:        github.ReviewActionSubmitted,
			reviewer:      "collab1",
			body:          "/lgtm cancel",
			currentLabel:  lgtmOne,
			isCancel:      true,
			shouldToggle:  false,
			shouldComment: false,
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
				Repos:            []string{"org/repo"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    tc.storeTreeHash,
				PullOwnersURL:    "https://fake/ti-community-bot",
			},
		}

		foc := &fakeOwnersClient{
			reviewers: []string{"collab1"},
			needsLgtm: 2,
		}

		fp := &fakePruner{
			GitHubClient:  fc,
			IssueComments: fc.IssueComments[5],
		}
		if err := HandlePullReviewEvent(fc, e, cfg, foc, fp, logrus.WithField("plugin", PluginName)); err != nil {
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

		if tc.shouldComment && len(fc.IssueComments[5]) != 1 {
			t.Error("should have commented.")
		} else if !tc.shouldComment && len(fc.IssueComments[5]) != 0 {
			t.Error("should not have commented.")
		}
	}
}

func TestHandlePullRequest(t *testing.T) {
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"
	cases := []struct {
		name             string
		event            github.PullRequestEvent
		removeLabelErr   error
		createCommentErr error

		err                error
		IssueLabelsAdded   []string
		IssueLabelsRemoved []string
		issueComments      map[int][]github.IssueComment
		trustedTeam        string

		expectNoComments bool
	}{
		{
			name: "pr_synchronize, no RemoveLabel error",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionSynchronize,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
					},
					Head: github.PullRequestBranch{
						SHA: SHA,
					},
				},
			},
			IssueLabelsRemoved: []string{lgtmOne},
			issueComments: map[int][]github.IssueComment{
				101: {
					{
						Body: removeLGTMLabelNoti,
						User: github.User{Login: fakegithub.Bot},
					},
				},
			},
			expectNoComments: false,
		},
		{
			name: "Sticky LGTM for trusted team members",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionSynchronize,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
					},
					User: github.User{
						Login: "sig-lead",
					},
					MergeSHA: &SHA,
				},
			},
			trustedTeam:      "Leads",
			expectNoComments: true,
		},
		{
			name: "LGTM not sticky for trusted user if disabled",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionSynchronize,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
					},
					User: github.User{
						Login: "sig-lead",
					},
					MergeSHA: &SHA,
				},
			},
			IssueLabelsRemoved: []string{lgtmOne},
			issueComments: map[int][]github.IssueComment{
				101: {
					{
						Body: removeLGTMLabelNoti,
						User: github.User{Login: fakegithub.Bot},
					},
				},
			},
			expectNoComments: false,
		},
		{
			name: "LGTM not sticky for non trusted user",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionSynchronize,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
					},
					User: github.User{
						Login: "sig-lead",
					},
					MergeSHA: &SHA,
				},
			},
			IssueLabelsRemoved: []string{lgtmOne},
			issueComments: map[int][]github.IssueComment{
				101: {
					{
						Body: removeLGTMLabelNoti,
						User: github.User{Login: fakegithub.Bot},
					},
				},
			},
			trustedTeam:      "Committers",
			expectNoComments: false,
		},
		{
			name: "pr_assigned",
			event: github.PullRequestEvent{
				Action: "assigned",
			},
			expectNoComments: true,
		},
		{
			name: "pr_synchronize, same tree-hash, keep label",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionSynchronize,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
					},
					Head: github.PullRequestBranch{
						SHA: SHA,
					},
				},
			},
			issueComments: map[int][]github.IssueComment{
				101: {
					{
						Body: fmt.Sprintf(addLGTMLabelNotification, treeSHA),
						User: github.User{Login: fakegithub.Bot},
					},
				},
			},
			expectNoComments: true,
		},
		{
			name: "pr_synchronize, same tree-hash, keep label, edited comment",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionSynchronize,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
					},
					Head: github.PullRequestBranch{
						SHA: SHA,
					},
				},
			},
			IssueLabelsRemoved: []string{lgtmOne},
			issueComments: map[int][]github.IssueComment{
				101: {
					{
						Body:      fmt.Sprintf(addLGTMLabelNotification, treeSHA),
						User:      github.User{Login: fakegithub.Bot},
						CreatedAt: time.Date(1981, 2, 21, 12, 30, 0, 0, time.UTC),
						UpdatedAt: time.Date(1981, 2, 21, 12, 31, 0, 0, time.UTC),
					},
				},
			},
			expectNoComments: false,
		},
		{
			name: "pr_synchronize, 2 tree-hash comments, keep label",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionSynchronize,
				PullRequest: github.PullRequest{
					Number: 101,
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Owner: github.User{
								Login: "kubernetes",
							},
							Name: "kubernetes",
						},
					},
					Head: github.PullRequestBranch{
						SHA: SHA,
					},
				},
			},
			issueComments: map[int][]github.IssueComment{
				101: {
					{
						Body: fmt.Sprintf(addLGTMLabelNotification, "older_treeSHA"),
						User: github.User{Login: fakegithub.Bot},
					},
					{
						Body: fmt.Sprintf(addLGTMLabelNotification, treeSHA),
						User: github.User{Login: fakegithub.Bot},
					},
				},
			},
			expectNoComments: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fakeGitHub := &fakegithub.FakeClient{
				IssueComments: c.issueComments,
				PullRequests: map[int]*github.PullRequest{
					101: {
						Base: github.PullRequestBranch{
							Ref: "master",
						},
						Head: github.PullRequestBranch{
							SHA: SHA,
						},
					},
				},
				Commits:          make(map[string]github.SingleCommit),
				Collaborators:    []string{"collab"},
				IssueLabelsAdded: c.IssueLabelsAdded,
			}
			fakeGitHub.IssueLabelsAdded = append(fakeGitHub.IssueLabelsAdded, "kubernetes/kubernetes#101:"+lgtmOne)
			commit := github.SingleCommit{}
			commit.Commit.Tree.SHA = treeSHA
			fakeGitHub.Commits[SHA] = commit

			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityLgtm = []externalplugins.TiCommunityLgtm{
				{
					Repos:            []string{"kubernetes/kubernetes"},
					ReviewActsAsLgtm: true,
					StoreTreeHash:    true,
					StickyLgtmTeam:   c.trustedTeam,
					PullOwnersURL:    "https://fake/ti-community-bot",
				},
			}

			err := HandlePullRequestEvent(
				fakeGitHub,
				&c.event,
				cfg,
				logrus.WithField("plugin", "approve"),
			)

			if err != nil && c.err == nil {
				t.Fatalf("handlePullRequest error: %v", err)
			}

			if err == nil && c.err != nil {
				t.Fatalf("handlePullRequest wanted error: %v, got nil", c.err)
			}

			if got, want := err, c.err; !equality.Semantic.DeepEqual(got, want) {
				t.Fatalf("handlePullRequest error mismatch: got %v, want %v", got, want)
			}

			if got, want := len(fakeGitHub.IssueLabelsRemoved), len(c.IssueLabelsRemoved); got != want {
				t.Logf("IssueLabelsRemoved: got %v, want: %v", fakeGitHub.IssueLabelsRemoved, c.IssueLabelsRemoved)
				t.Fatalf("IssueLabelsRemoved length mismatch: got %d, want %d", got, want)
			}

			if got, want := fakeGitHub.IssueComments, c.issueComments; !equality.Semantic.DeepEqual(got, want) {
				t.Fatalf("LGTM revmoved notifications mismatch: got %v, want %v", got, want)
			}
			if c.expectNoComments && len(fakeGitHub.IssueCommentsAdded) > 0 {
				t.Fatalf("expected no comments but got %v", fakeGitHub.IssueCommentsAdded)
			}
			if !c.expectNoComments && len(fakeGitHub.IssueCommentsAdded) == 0 {
				t.Fatalf("expected comments but got none")
			}
		})
	}
}

func TestAddTreeHashComment(t *testing.T) {
	cases := []struct {
		name          string
		author        string
		trustedTeam   string
		expectTreeSha bool
	}{
		{
			name:          "Tree SHA added",
			author:        "Bob",
			expectTreeSha: true,
		},
		{
			name:          "Tree SHA if sticky lgtm off",
			author:        "sig-lead",
			expectTreeSha: true,
		},
		{
			name:          "No Tree SHA if sticky lgtm",
			author:        "sig-lead",
			trustedTeam:   "Leads",
			expectTreeSha: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
			treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"
			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityLgtm = []externalplugins.TiCommunityLgtm{
				{
					Repos:            []string{"kubernetes/kubernetes"},
					ReviewActsAsLgtm: true,
					StoreTreeHash:    true,
					StickyLgtmTeam:   c.trustedTeam,
					PullOwnersURL:    "https://fake/ti-community-bot",
				},
			}
			rc := reviewCtx{
				author:      "collab1",
				issueAuthor: c.author,
				repo: github.Repo{
					Owner: github.User{
						Login: "kubernetes",
					},
					Name: "kubernetes",
				},
				number: 101,
				body:   "/lgtm",
			}
			fc := &fakegithub.FakeClient{
				Commits:       make(map[string]github.SingleCommit),
				IssueComments: map[int][]github.IssueComment{},
				PullRequests: map[int]*github.PullRequest{
					101: {
						Base: github.PullRequestBranch{
							Ref: "master",
						},
						Head: github.PullRequestBranch{
							SHA: SHA,
						},
					},
				},
				Collaborators: []string{"collab1", "collab2"},
			}
			commit := github.SingleCommit{}
			commit.Commit.Tree.SHA = treeSHA
			fc.Commits[SHA] = commit

			foc := &fakeOwnersClient{
				reviewers: []string{"collab1"},
				needsLgtm: 2,
			}

			_ = handle(true, cfg, rc, fc, foc, &fakePruner{}, logrus.WithField("plugin", PluginName))
			found := false
			for _, body := range fc.IssueCommentsAdded {
				if addLGTMLabelNotificationRe.MatchString(body) {
					found = true
					break
				}
			}
			if c.expectTreeSha {
				if !found {
					t.Fatalf("expected tree_hash comment but got none")
				}
			} else {
				if found {
					t.Fatalf("expected no tree_hash comment but got one")
				}
			}
		})
	}
}

func TestRemoveTreeHashComment(t *testing.T) {
	treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"
	config := &externalplugins.Configuration{}
	config.TiCommunityLgtm = []externalplugins.TiCommunityLgtm{
		{
			Repos:            []string{"kubernetes/kubernetes"},
			ReviewActsAsLgtm: true,
			StoreTreeHash:    true,
			PullOwnersURL:    "https://fake/ti-community-bot",
		},
	}
	rc := reviewCtx{
		author:      "collab1",
		issueAuthor: "bob",
		repo: github.Repo{
			Owner: github.User{
				Login: "kubernetes",
			},
			Name: "kubernetes",
		},
		number: 101,
		body:   "/lgtm cancel",
	}
	fc := &fakegithub.FakeClient{
		IssueComments: map[int][]github.IssueComment{
			101: {
				{
					Body: fmt.Sprintf(addLGTMLabelNotification, treeSHA),
					User: github.User{Login: fakegithub.Bot},
				},
			},
		},
		Collaborators: []string{"collab1", "collab2"},
	}
	fc.IssueLabelsAdded = []string{"kubernetes/kubernetes#101:" + lgtmOne}
	fp := &fakePruner{
		GitHubClient:  fc,
		IssueComments: fc.IssueComments[101],
	}

	foc := &fakeOwnersClient{
		reviewers: []string{"collab1"},
		needsLgtm: 2,
	}

	_ = handle(false, config, rc, fc, foc, fp, logrus.WithField("plugin", PluginName))
	found := false
	for _, body := range fc.IssueCommentsDeleted {
		if addLGTMLabelNotificationRe.MatchString(body) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected deleted tree_hash comment but got none")
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
			configInfoExcludes: []string{configInfoReviewActsAsLgtm, configInfoStoreTreeHash, configInfoStickyLgtmTeam("team1")},
		},
		{
			name: "StoreTreeHash enabled",
			config: &externalplugins.Configuration{
				TiCommunityLgtm: []externalplugins.TiCommunityLgtm{
					{
						Repos:         []string{"org2/repo"},
						StoreTreeHash: true,
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoExcludes: []string{configInfoReviewActsAsLgtm, configInfoStickyLgtmTeam("team1")},
			configInfoIncludes: []string{configInfoStoreTreeHash},
		},
		{
			name: "All configs enabled",
			config: &externalplugins.Configuration{
				TiCommunityLgtm: []externalplugins.TiCommunityLgtm{
					{
						Repos:            []string{"org2/repo"},
						ReviewActsAsLgtm: true,
						StoreTreeHash:    true,
						StickyLgtmTeam:   "team1",
						PullOwnersURL:    "https://fake",
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{configInfoReviewActsAsLgtm, configInfoStoreTreeHash, configInfoStickyLgtmTeam("team1")},
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
