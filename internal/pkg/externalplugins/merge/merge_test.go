//nolint:scopelint
package merge

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
	approvers []string
	needsLgtm int
}

func (f *fakeOwnersClient) LoadOwners(_ string,
	_, _ string, _ int) (*ownersclient.Owners, error) {
	return &ownersclient.Owners{
		Approvers: f.approvers,
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

func TestMergeIssueAndReviewComment(t *testing.T) {
	var testcases = []struct {
		name             string
		body             string
		commenter        string
		currentLGTMLabel string
		canMergeLabel    string
		isCancel         bool
		shouldToggle     bool
		shouldComment    bool
		storeTreeHash    bool
	}{
		{
			name:         "non-merge comment",
			body:         "uh oh",
			commenter:    "collab1",
			shouldToggle: false,
		},
		{
			name:          "merge comment by approver collab1, no lgtm and no can merge on pr",
			body:          "/merge",
			commenter:     "collab1",
			shouldToggle:  false,
			shouldComment: true,
		},
		{
			name:          "MERGE comment by approver collab1, no lgtm and no can merge on pr",
			body:          "/MERGE",
			commenter:     "collab1",
			shouldToggle:  false,
			shouldComment: true,
		},
		{
			name:             "merge comment by approver collab1, lgtm satisfy and no can merge on pr",
			body:             "/merge",
			commenter:        "collab1",
			currentLGTMLabel: lgtmTwo,
			shouldToggle:     true,
			shouldComment:    true,
		},
		{
			name:          "merge comment by author",
			body:          "/merge",
			commenter:     "author",
			shouldToggle:  false,
			shouldComment: true,
		},
		{
			name:             "merge cancel by author",
			body:             "/merge cancel",
			commenter:        "author",
			currentLGTMLabel: lgtmTwo,
			isCancel:         true,
			shouldToggle:     true,
			shouldComment:    false,
		},
		{
			name:             "merge comment by approver collab2",
			body:             "/merge",
			commenter:        "collab2",
			currentLGTMLabel: lgtmTwo,
			shouldToggle:     true,
			shouldComment:    true,
		},
		{
			name:             "merge comment by approver collab2, with trailing space",
			body:             "/merge ",
			commenter:        "collab2",
			currentLGTMLabel: lgtmTwo,
			shouldToggle:     true,
			shouldComment:    true,
		},
		{
			name:          "merge comment by random",
			body:          "/merge",
			commenter:     "not-in-the-org",
			shouldToggle:  false,
			shouldComment: true,
		},
		{
			name:             "merge cancel by approver collab2",
			body:             "/merge cancel",
			commenter:        "collab2",
			currentLGTMLabel: lgtmTwo,
			isCancel:         true,
			shouldToggle:     true,
			shouldComment:    false,
		},
		{
			name:             "merge cancel by random",
			body:             "/merge cancel",
			commenter:        "not-in-the-org",
			currentLGTMLabel: lgtmTwo,
			isCancel:         true,
			shouldToggle:     false,
			shouldComment:    true,
		},
		{
			name:             "merge cancel comment by approver collab1",
			body:             "/merge cancel",
			commenter:        "collab1",
			currentLGTMLabel: lgtmTwo,
			isCancel:         true,
			shouldToggle:     true,
			shouldComment:    false,
		},
		{
			name:             "merge cancel comment by approver collab1, with trailing space",
			body:             "/merge cancel \r",
			commenter:        "collab1",
			currentLGTMLabel: lgtmTwo,
			isCancel:         true,
			shouldToggle:     true,
			shouldComment:    false,
		},
		{
			name:          "merge cancel comment by reviewer collab1, no merge",
			body:          "/merge cancel",
			commenter:     "collab1",
			isCancel:      true,
			shouldToggle:  false,
			shouldComment: false,
		},
		{
			name:             "merge comment by approver collab2, can merge is exist",
			body:             "/merge ",
			commenter:        "collab2",
			currentLGTMLabel: lgtmTwo,
			canMergeLabel:    canMergeLabel,
			shouldToggle:     false,
			shouldComment:    false,
		},
		{
			name:             "merge comment by random, can merge is exist",
			body:             "/merge ",
			commenter:        "not-in-the-org",
			currentLGTMLabel: lgtmTwo,
			canMergeLabel:    canMergeLabel,
			shouldToggle:     false,
			shouldComment:    true,
		},
		{
			name:             "merge comment by approver collab1, lgtm not satisfy and no can merge on pr",
			body:             "/merge",
			commenter:        "collab1",
			currentLGTMLabel: lgtmOne,
			shouldToggle:     false,
			shouldComment:    true,
		},
	}
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	prName := "org/repo#5"
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
				CommitMap: map[string][]github.RepositoryCommit{
					prName: {
						{
							SHA: SHA,
						},
					},
				},
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
			if tc.currentLGTMLabel != "" {
				fc.IssueLabelsAdded = append(fc.IssueLabelsAdded, prName+":"+tc.currentLGTMLabel)
			}
			if tc.canMergeLabel != "" {
				fc.IssueLabelsAdded = append(fc.IssueLabelsAdded, prName+":"+tc.canMergeLabel)
			}

			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityMerge = []externalplugins.TiCommunityMerge{
				{
					Repos:              []string{"org/repo"},
					StoreTreeHash:      true,
					PullOwnersEndpoint: "https://fake/ti-community-bot",
				},
			}

			foc := &fakeOwnersClient{
				approvers: []string{"collab1", "collab2"},
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
				if tc.canMergeLabel != "" {
					if tc.isCancel {
						if len(fc.IssueLabelsRemoved) != 1 {
							t.Error("should have removed " + canMergeLabel + ".")
						}
					}
				} else {
					if !tc.isCancel {
						if len(fc.IssueLabelsAdded) == 0 {
							t.Error("should have added " + canMergeLabel + ".")
						}
					}
				}
			} else if len(fc.IssueLabelsRemoved) > 0 {
				t.Error("should not have removed " + canMergeLabel + ".")
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
				CommitMap: map[string][]github.RepositoryCommit{
					prName: {
						{
							SHA: SHA,
						},
					},
				},
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
			if tc.currentLGTMLabel != "" {
				fc.IssueLabelsAdded = append(fc.IssueLabelsAdded, "org/repo#5:"+tc.currentLGTMLabel)
			}
			if tc.canMergeLabel != "" {
				fc.IssueLabelsAdded = append(fc.IssueLabelsAdded, "org/repo#5:"+tc.canMergeLabel)
			}

			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityMerge = []externalplugins.TiCommunityMerge{
				{
					Repos:              []string{"org/repo"},
					StoreTreeHash:      true,
					PullOwnersEndpoint: "https://fake/ti-community-bot",
				},
			}

			foc := &fakeOwnersClient{
				approvers: []string{"collab1", "collab2"},
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
				if tc.canMergeLabel != "" {
					if tc.isCancel {
						if len(fc.IssueLabelsRemoved) != 1 {
							t.Error("should have removed " + canMergeLabel + ".")
						}
					}
				} else {
					if !tc.isCancel {
						if len(fc.IssueLabelsAdded) == 0 {
							t.Error("should have added " + canMergeLabel + ".")
						}
					}
				}
			} else if len(fc.IssueLabelsRemoved) > 0 {
				t.Error("should not have removed " + canMergeLabel + ".")
			}

			if tc.shouldComment && len(fc.IssueComments[5]) != 1 {
				t.Error("should have commented.")
			} else if !tc.shouldComment && len(fc.IssueComments[5]) != 0 {
				t.Error("should not have commented.")
			}
		}
	}
}

func TestMergeReviewCommentWithMergeNoti(t *testing.T) {
	var testcases = []struct {
		name         string
		body         string
		commenter    string
		shouldDelete bool
	}{
		{
			name:         "non-merge comment",
			body:         "uh oh",
			commenter:    "collab2",
			shouldDelete: false,
		},
		{
			name:         "merge comment by approver collab1, no can merge on pr",
			body:         "/merge",
			commenter:    "collab1",
			shouldDelete: true,
		},
		{
			name:         "MERGE comment by approver collab1, no can merge on pr",
			body:         "/MERGE",
			commenter:    "collab1",
			shouldDelete: true,
		},
		{
			name:         "merge comment by author",
			body:         "/merge",
			commenter:    "author",
			shouldDelete: false,
		},
		{
			name:         "merge comment by approver collab2",
			body:         "/merge",
			commenter:    "collab2",
			shouldDelete: true,
		},
		{
			name:         "merge comment by approver collab2, with trailing space",
			body:         "/merge ",
			commenter:    "collab2",
			shouldDelete: true,
		},
		{
			name:         "merge comment by random",
			body:         "/merge",
			commenter:    "not-in-the-org",
			shouldDelete: false,
		},
		{
			name:         "merge cancel comment by reviewer collab1, no can merge",
			body:         "/merge cancel",
			commenter:    "collab1",
			shouldDelete: false,
		},
	}
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	prName := "org/repo#5"
	for _, tc := range testcases {
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
			Collaborators: []string{"collab1", "collab2"},
			CommitMap: map[string][]github.RepositoryCommit{
				prName: {
					{
						SHA: SHA,
					},
				},
			},
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
		botName, err := fc.BotName()
		if err != nil {
			t.Fatalf("For case %s, could not get Bot nam", tc.name)
		}
		ic := github.IssueComment{
			User: github.User{
				Login: botName,
			},
			Body: removeCanMergeLabelNoti,
		}
		fc.IssueComments[5] = append(fc.IssueComments[5], ic)
		fc.IssueLabelsAdded = append(fc.IssueLabelsAdded, prName+":"+lgtmTwo)

		cfg := &externalplugins.Configuration{}
		cfg.TiCommunityMerge = []externalplugins.TiCommunityMerge{
			{
				Repos:              []string{"org/repo"},
				StoreTreeHash:      true,
				PullOwnersEndpoint: "https://fake/ti-community-bot",
			},
		}

		foc := &fakeOwnersClient{
			approvers: []string{"collab1", "collab2"},
			needsLgtm: 2,
		}

		cp := &fakePruner{
			GitHubClient:  fc,
			IssueComments: fc.IssueComments[5],
		}

		if err := HandlePullReviewCommentEvent(fc, e, cfg, foc, cp, logrus.WithField("plugin", PluginName)); err != nil {
			t.Errorf("For case %s, didn't expect error from lgtmComment: %v", tc.name, err)
			continue
		}

		deleted := false
		for _, body := range fc.IssueCommentsDeleted {
			if body == removeCanMergeLabelNoti {
				deleted = true
				break
			}
		}
		if tc.shouldDelete {
			if !deleted {
				t.Errorf("For case %s, status/can-merge removed notification should have been deleted", tc.name)
			}
		} else {
			if deleted {
				t.Errorf("For case %s, status/can-merge removed notification should not have been deleted", tc.name)
			}
		}
	}
}

func TestHandlePullRequest(t *testing.T) {
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"
	prName := "kubernetes/kubernetes#101"
	cases := []struct {
		name             string
		event            github.PullRequestEvent
		prCommits        map[string][]github.RepositoryCommit
		removeLabelErr   error
		createCommentErr error

		err                error
		IssueLabelsAdded   []string
		IssueLabelsRemoved []string
		issueComments      map[int][]github.IssueComment

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
			prCommits: map[string][]github.RepositoryCommit{
				prName: {
					{
						SHA: SHA,
					},
				},
			},
			IssueLabelsRemoved: []string{canMergeLabel},
			issueComments: map[int][]github.IssueComment{
				101: {
					{
						Body: removeCanMergeLabelNoti,
						User: github.User{Login: fakegithub.Bot},
					},
				},
			},
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
			prCommits: map[string][]github.RepositoryCommit{
				prName: {
					{
						SHA: SHA,
					},
				},
			},
			issueComments: map[int][]github.IssueComment{
				101: {
					{
						Body: fmt.Sprintf(addCanMergeLabelNotification, treeSHA),
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
			prCommits: map[string][]github.RepositoryCommit{
				prName: {
					{
						SHA: SHA,
					},
				},
			},
			IssueLabelsRemoved: []string{canMergeLabel},
			issueComments: map[int][]github.IssueComment{
				101: {
					{
						Body:      fmt.Sprintf(addCanMergeLabelNotification, treeSHA),
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
			prCommits: map[string][]github.RepositoryCommit{
				prName: {
					{
						SHA: SHA,
					},
				},
			},
			issueComments: map[int][]github.IssueComment{
				101: {
					{
						Body: fmt.Sprintf(addCanMergeLabelNotification, "older_treeSHA"),
						User: github.User{Login: fakegithub.Bot},
					},
					{
						Body: fmt.Sprintf(addCanMergeLabelNotification, treeSHA),
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
				CommitMap:        c.prCommits,
			}
			fakeGitHub.IssueLabelsAdded = append(fakeGitHub.IssueLabelsAdded, prName+":"+canMergeLabel)
			commit := github.SingleCommit{}
			commit.Commit.Tree.SHA = treeSHA
			fakeGitHub.Commits[SHA] = commit

			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityMerge = []externalplugins.TiCommunityMerge{
				{
					Repos:              []string{"kubernetes/kubernetes"},
					StoreTreeHash:      true,
					PullOwnersEndpoint: "https://fake/ti-community-bot",
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
	c := struct {
		name          string
		author        string
		trustedTeam   string
		expectTreeSha bool
	}{
		name:          "Tree SHA added",
		author:        "Bob",
		expectTreeSha: true,
	}

	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"
	prName := "kubernetes/kubernetes#101"
	cfg := &externalplugins.Configuration{}
	cfg.TiCommunityMerge = []externalplugins.TiCommunityMerge{
		{
			Repos:              []string{"kubernetes/kubernetes"},
			StoreTreeHash:      true,
			PullOwnersEndpoint: "https://fake/ti-community-bot",
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
		body:   "/merge",
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
		CommitMap: map[string][]github.RepositoryCommit{
			prName: {
				{
					SHA: SHA,
				},
			},
		},
	}
	fc.IssueLabelsAdded = []string{prName + ":" + lgtmTwo}

	commit := github.SingleCommit{}
	commit.Commit.Tree.SHA = treeSHA
	fc.Commits[SHA] = commit

	foc := &fakeOwnersClient{
		approvers: []string{"collab1"},
		needsLgtm: 2,
	}

	_ = handle(true, cfg, rc, fc, foc, &fakePruner{}, logrus.WithField("plugin", PluginName))
	found := false
	for _, body := range fc.IssueCommentsAdded {
		if addCanMergeLabelNotificationRe.MatchString(body) {
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
}

func TestRemoveTreeHashComment(t *testing.T) {
	treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"
	cfg := &externalplugins.Configuration{}
	cfg.TiCommunityMerge = []externalplugins.TiCommunityMerge{
		{
			Repos:              []string{"kubernetes/kubernetes"},
			StoreTreeHash:      true,
			PullOwnersEndpoint: "https://fake/ti-community-bot",
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
		body:   "/merge cancel",
	}
	fc := &fakegithub.FakeClient{
		IssueComments: map[int][]github.IssueComment{
			101: {
				{
					Body: fmt.Sprintf(addCanMergeLabelNotification, treeSHA),
					User: github.User{Login: fakegithub.Bot},
				},
			},
		},
		Collaborators: []string{"collab1", "collab2"},
	}
	fc.IssueLabelsAdded = []string{"kubernetes/kubernetes#101:" + canMergeLabel}
	fp := &fakePruner{
		GitHubClient:  fc,
		IssueComments: fc.IssueComments[101],
	}

	foc := &fakeOwnersClient{
		approvers: []string{"collab1"},
		needsLgtm: 2,
	}

	_ = handle(false, cfg, rc, fc, foc, fp, logrus.WithField("plugin", PluginName))
	found := false
	for _, body := range fc.IssueCommentsDeleted {
		if addCanMergeLabelNotificationRe.MatchString(body) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected deleted tree_hash comment but got none")
	}
}

func TestGetCurrentLabelNumber(t *testing.T) {
	var testcases = []struct {
		name      string
		labels    []github.Label
		needsLgtm int
		isSatisfy bool
	}{
		{
			name:      "Current no LGTM",
			labels:    []github.Label{},
			needsLgtm: 1,
			isSatisfy: false,
		},
		{
			name:      "Current no LGTM",
			labels:    []github.Label{},
			needsLgtm: 2,
			isSatisfy: false,
		},
		{
			name: "Current LGT1, needs 1 LGTM",
			labels: []github.Label{
				{
					Name: lgtmOne,
				},
			},
			needsLgtm: 1,
			isSatisfy: true,
		},
		{
			name: "Current LGT1, needs 2 LGTM",
			labels: []github.Label{
				{
					Name: lgtmOne,
				},
			},
			needsLgtm: 2,
			isSatisfy: false,
		},
		{
			name: "Current LGT2, needs 2 LGTM",
			labels: []github.Label{
				{
					Name: lgtmTwo,
				},
			},
			needsLgtm: 2,
			isSatisfy: true,
		},
	}

	// scopelint:ignore
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			isSatisfy := isLGTMSatisfy(LabelPrefix, tc.labels, tc.needsLgtm)

			if isSatisfy != tc.isSatisfy {
				t.Fatalf("satisify mismatch: got %v, want %v", isSatisfy, tc.isSatisfy)
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
			configInfoExcludes: []string{configInfoStoreTreeHash},
		},
		{
			name: "StoreTreeHash enabled",
			config: &externalplugins.Configuration{
				TiCommunityMerge: []externalplugins.TiCommunityMerge{
					{
						Repos:         []string{"org2/repo"},
						StoreTreeHash: true,
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{configInfoStoreTreeHash},
		},
		{
			name: "All configs enabled",
			config: &externalplugins.Configuration{
				TiCommunityMerge: []externalplugins.TiCommunityMerge{
					{
						Repos:              []string{"org2/repo"},
						StoreTreeHash:      true,
						PullOwnersEndpoint: "https://fake",
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{configInfoStoreTreeHash},
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

func TestAllGuaranteed(t *testing.T) {
	treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"

	var testcases = []struct {
		name             string
		lastCanMergeSha  string
		commits          []github.RepositoryCommit
		exceptGuaranteed bool
	}{
		{
			name:            "Only one commit",
			lastCanMergeSha: treeSHA,
			commits: []github.RepositoryCommit{
				{
					SHA: treeSHA,
				},
			},
			exceptGuaranteed: true,
		},
		{
			name:            "All authored commits",
			lastCanMergeSha: treeSHA,
			commits: []github.RepositoryCommit{
				{
					SHA: "some-sha",
				},
				{
					SHA: treeSHA,
				},
			},
			exceptGuaranteed: true,
		},
		{
			name:            "Guaranteed by github",
			lastCanMergeSha: treeSHA,
			commits: []github.RepositoryCommit{
				{
					SHA: "some-sha",
				},
				{
					SHA: treeSHA,
				},
				{
					SHA: "some-sha",
					Committer: github.User{
						Login: githubUpdateCommitter,
					},
				},
			},
			exceptGuaranteed: true,
		},
		{
			name:            "New commit not guaranteed",
			lastCanMergeSha: treeSHA,
			commits: []github.RepositoryCommit{
				{
					SHA: "some-sha",
				},
				{
					SHA: treeSHA,
				},
				{
					SHA: "some-sha",
				},
			},
			exceptGuaranteed: false,
		},
		{
			name:            "New commit and github update",
			lastCanMergeSha: treeSHA,
			commits: []github.RepositoryCommit{
				{
					SHA: "some-sha",
				},
				{
					SHA: treeSHA,
				},
				{
					SHA: "some-sha",
					Committer: github.User{
						Login: githubUpdateCommitter,
					},
				},
				{
					SHA: "some-sha",
				},
			},
			exceptGuaranteed: false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			isGuaranteed := isAllGuaranteed(testcase.commits, testcase.lastCanMergeSha, logrus.WithField("plugin", PluginName))

			if isGuaranteed != testcase.exceptGuaranteed {
				t.Fatalf("=guarantee mismatch: got %v, want %v", isGuaranteed, testcase.exceptGuaranteed)
			}
		})
	}
}
