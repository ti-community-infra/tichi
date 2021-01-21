// nolint:lll
package lgtm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/ownersclient"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
)

var (
	lgtmOne = fmt.Sprintf("%s%d", externalplugins.LgtmLabelPrefix, 1)
	lgtmTwo = fmt.Sprintf("%s%d", externalplugins.LgtmLabelPrefix, 2)
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

func TestLGTMIssueAndReviewComment(t *testing.T) {
	type commentCase struct {
		name         string
		body         string
		commenter    string
		currentLabel string
		lgtmComment  string
		isCancel     bool

		shouldToggle        bool
		shouldComment       bool
		expectComment       string
		shouldDeleteComment bool
	}

	var testcases = []commentCase{
		{
			name:          "non-lgtm comment",
			body:          "uh oh",
			commenter:     "collab1",
			shouldToggle:  false,
			shouldComment: false,
		},
		{
			name:          "lgtm comment by reviewer collab1, no lgtm on pr",
			body:          "/lgtm",
			commenter:     "collab1",
			shouldToggle:  true,
			shouldComment: true,
			expectComment: "org/repo#5:[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab1\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
		},
		{
			name:          "LGTM comment by reviewer collab1, no lgtm on pr",
			body:          "/LGTM",
			commenter:     "collab1",
			shouldToggle:  true,
			shouldComment: true,
			expectComment: "org/repo#5:[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab1\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
		},
		{
			name:                "lgtm comment by reviewer collab1, lgtm on pr",
			body:                "/lgtm",
			commenter:           "collab1",
			currentLabel:        lgtmOne,
			lgtmComment:         "[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab2\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
			shouldToggle:        true,
			shouldComment:       true,
			shouldDeleteComment: true,
			expectComment:       "org/repo#5:[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab1\n- collab2\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
		},
		{
			name:          "lgtm comment by author",
			body:          "/lgtm",
			commenter:     "author",
			shouldToggle:  false,
			shouldComment: true,
			expectComment: "org/repo#5:@author: you cannot `/lgtm` your own PR.\n\n<details>\n\nIn response to [this](<url>):\n\n>/lgtm\n\n\nInstructions for interacting with me using PR comments are available [here](https://prow.tidb.io/command-help).  If you have questions or suggestions related to my behavior, please file an issue against the [ti-community-infra/tichi](https://github.com/ti-community-infra/tichi/issues/new?title=Prow%20issue:) repository.\n</details>",
		},
		{
			name:          "lgtm cancel by reviewer author",
			body:          "/lgtm cancel",
			commenter:     "author",
			currentLabel:  lgtmOne,
			isCancel:      true,
			shouldToggle:  true,
			shouldComment: true,
			expectComment: "org/repo#5:[REVIEW NOTIFICATION]\n\nThis pull request has not been approved.\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
		},
		{
			name:          "lgtm comment by reviewer collab2",
			body:          "/lgtm",
			commenter:     "collab2",
			shouldToggle:  true,
			shouldComment: true,
			expectComment: "org/repo#5:[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab2\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
		},
		{
			name:          "lgtm comment by reviewer collab2, with trailing space",
			body:          "/lgtm ",
			commenter:     "collab2",
			shouldToggle:  true,
			shouldComment: true,
			expectComment: "org/repo#5:[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab2\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
		},
		{
			name:          "lgtm comment by random",
			body:          "/lgtm",
			commenter:     "not-in-the-org",
			shouldToggle:  false,
			shouldComment: true,
			expectComment: "org/repo#5:@not-in-the-org: `/lgtm` is only allowed for the reviewers in [list](https://tichiWebLink/repos/org/repo/pulls/5/owners).\n\n<details>\n\nIn response to [this](<url>):\n\n>/lgtm\n\n\nInstructions for interacting with me using PR comments are available [here](https://prow.tidb.io/command-help).  If you have questions or suggestions related to my behavior, please file an issue against the [ti-community-infra/tichi](https://github.com/ti-community-infra/tichi/issues/new?title=Prow%20issue:) repository.\n</details>",
		},
		{
			name:                "lgtm cancel by reviewer collab2",
			body:                "/lgtm cancel",
			commenter:           "collab2",
			currentLabel:        lgtmOne,
			lgtmComment:         "[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab1\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
			isCancel:            true,
			shouldToggle:        true,
			shouldComment:       true,
			shouldDeleteComment: true,
			expectComment:       "org/repo#5:[REVIEW NOTIFICATION]\n\nThis pull request has not been approved.\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
		},
		{
			name:          "lgtm cancel by random",
			body:          "/lgtm cancel",
			commenter:     "not-in-the-org",
			currentLabel:  lgtmOne,
			lgtmComment:   "[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab1\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
			isCancel:      true,
			shouldToggle:  false,
			shouldComment: true,
			expectComment: "org/repo#5:@not-in-the-org: `/lgtm cancel` is only allowed for the PR author or the reviewers in [list](https://tichiWebLink/repos/org/repo/pulls/5/owners).\n\n<details>\n\nIn response to [this](<url>):\n\n>/lgtm cancel\n\n\nInstructions for interacting with me using PR comments are available [here](https://prow.tidb.io/command-help).  If you have questions or suggestions related to my behavior, please file an issue against the [ti-community-infra/tichi](https://github.com/ti-community-infra/tichi/issues/new?title=Prow%20issue:) repository.\n</details>",
		},
		{
			name:                "lgtm cancel comment by reviewer collab1",
			body:                "/lgtm cancel",
			commenter:           "collab1",
			currentLabel:        lgtmOne,
			lgtmComment:         "[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab1\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
			isCancel:            true,
			shouldToggle:        true,
			shouldComment:       true,
			shouldDeleteComment: true,
			expectComment:       "org/repo#5:[REVIEW NOTIFICATION]\n\nThis pull request has not been approved.\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
		},
		{
			name:                "lgtm cancel comment by reviewer collab1, with trailing space",
			body:                "/lgtm cancel \r",
			commenter:           "collab1",
			lgtmComment:         "[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab1\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
			currentLabel:        lgtmOne,
			isCancel:            true,
			shouldToggle:        true,
			shouldComment:       true,
			shouldDeleteComment: true,
			expectComment:       "org/repo#5:[REVIEW NOTIFICATION]\n\nThis pull request has not been approved.\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
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
			expectComment: "org/repo#5:@not-in-the-org: `/lgtm` is only allowed for the reviewers in [list](https://tichiWebLink/repos/org/repo/pulls/5/owners).\n\n<details>\n\nIn response to [this](<url>):\n\n>/lgtm \n\n\nInstructions for interacting with me using PR comments are available [here](https://prow.tidb.io/command-help).  If you have questions or suggestions related to my behavior, please file an issue against the [ti-community-infra/tichi](https://github.com/ti-community-infra/tichi/issues/new?title=Prow%20issue:) repository.\n</details>",
		},
		{
			name:          "lgtm comment by reviewer collab1, lgtm twice",
			body:          "/lgtm",
			commenter:     "collab1",
			currentLabel:  lgtmOne,
			lgtmComment:   "[REVIEW NOTIFICATION]\n\nThis pull request has been approved by:\n\n- collab1\n\n\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/5/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
			shouldToggle:  false,
			shouldComment: false,
		},
	}
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	for _, testcase := range testcases {
		tc := testcase
		t.Logf("Running scenario %q", tc.name)
		cfg := &externalplugins.Configuration{
			TichiWebURL:     "https://tichiWebLink",
			CommandHelpLink: "https://commandHelpLink",
			PRProcessLink:   "https://prProcessLink",
		}
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

		checkResult := func(tc *commentCase, fc *fakegithub.FakeClient) {
			if !tc.shouldComment && len(fc.IssueCommentsAdded) != 0 {
				t.Errorf("unexpected comment %v", fc.IssueCommentsAdded)
			}

			if tc.shouldComment && tc.expectComment != fc.IssueCommentsAdded[0] {
				t.Fatalf("review notifications mismatch: got %q, want %q", fc.IssueCommentsAdded[0], tc.expectComment)
			}

			if tc.shouldDeleteComment && len(fc.IssueCommentsDeleted) == 0 {
				t.Errorf("expected to delete comments but didn't.")
			}

			if !tc.shouldDeleteComment && len(fc.IssueCommentsDeleted) != 0 {
				t.Errorf("expected not to delete comments but deleted.")
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

		// Test issue comments.
		{
			fc := &fakegithub.FakeClient{
				IssueComments: map[int][]github.IssueComment{
					5: {{
						Body: tc.lgtmComment,
						User: github.User{
							Login: "k8s-ci-robot",
						},
					}},
				},
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

			if err := HandleIssueCommentEvent(fc, e, cfg, foc, logrus.WithField("plugin", PluginName)); err != nil {
				t.Errorf("didn't expect error from lgtmComment: %v", err)
				continue
			}

			checkResult(&tc, fc)
		}

		// Test review comments.
		{
			fc := &fakegithub.FakeClient{
				IssueComments: map[int][]github.IssueComment{
					5: {{
						Body: tc.lgtmComment,
						User: github.User{
							Login: "k8s-ci-robot",
						},
					}},
				},
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

			if err := HandlePullReviewCommentEvent(fc, e, cfg, foc, logrus.WithField("plugin", PluginName)); err != nil {
				t.Errorf("didn't expect error from lgtmComment: %v", err)
				continue
			}

			checkResult(&tc, fc)
		}
	}
}

func TestLGTMFromApproveReview(t *testing.T) {
	var testcases = []struct {
		name         string
		state        github.ReviewState
		action       github.ReviewEventAction
		body         string
		reviewer     string
		currentLabel string
		isCancel     bool
		shouldToggle bool
	}{
		{
			name:         "Edit approve review by reviewer, no lgtm on pr",
			state:        github.ReviewStateApproved,
			action:       github.ReviewActionEdited,
			reviewer:     "collab1",
			shouldToggle: false,
		},
		{
			name:         "Dismiss approve review by reviewer, no lgtm on pr",
			state:        github.ReviewStateApproved,
			action:       github.ReviewActionDismissed,
			reviewer:     "collab1",
			shouldToggle: false,
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
			name:         "Approve review by reviewer, no lgtm on pr",
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
			name:         "Approve review by non-reviewer, no lgtm on pr",
			state:        github.ReviewStateApproved,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab2",
			shouldToggle: false,
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
			shouldToggle: true,
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

func TestHandlePullRequest(t *testing.T) {
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	testcases := []struct {
		name  string
		event github.PullRequestEvent

		shouldComment bool
		expectComment string
	}{
		{
			name: "Open a pull request",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionOpened,
				PullRequest: github.PullRequest{
					Number: 101,
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
			},
			shouldComment: true,
			expectComment: "org/repo#101:[REVIEW NOTIFICATION]\n\nThis pull request has not been approved.\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/101/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/101/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by writing `/lgtm` in a comment.\nReviewer can cancel approval by writing `/lgtm cancel` in a comment.\n</details>\n\n<!--Review Notification Identifier-->",
		},
		{
			name: "Reopen a pull request",
			event: github.PullRequestEvent{
				Action: github.PullRequestActionReopened,
				PullRequest: github.PullRequest{
					Number: 101,
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
			},
			shouldComment: false,
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		fc := &fakegithub.FakeClient{
			IssueComments:    make(map[int][]github.IssueComment),
			IssueLabelsAdded: []string{},
			PullRequests: map[int]*github.PullRequest{
				101: &tc.event.PullRequest,
			},
		}
		cfg := &externalplugins.Configuration{
			TichiWebURL:     "https://tichiWebLink",
			CommandHelpLink: "https://commandHelpLink",
			PRProcessLink:   "https://prProcessLink",
		}

		err := HandlePullRequestEvent(fc, &tc.event, cfg, logrus.WithField("plugin", PluginName))
		if err != nil {
			t.Errorf("For case %s, didn't expect error: %v", tc.name, err)
		}

		if !tc.shouldComment && len(fc.IssueCommentsAdded) != 0 {
			t.Errorf("unexpected comment %v", fc.IssueCommentsAdded)
		}

		if tc.shouldComment && tc.expectComment != fc.IssueCommentsAdded[0] {
			t.Fatalf("review notifications mismatch: got %q, want %q", fc.IssueCommentsAdded[0], tc.expectComment)
		}
	}
}

func TestGetCurrentAndNextLabel(t *testing.T) {
	var testcases = []struct {
		name               string
		labels             []github.Label
		needsLgtm          int
		expectCurrentLabel string
		expectNextLabel    string
	}{
		{
			name:               "Current no LGTM, needs 1 LGTM",
			labels:             []github.Label{},
			needsLgtm:          1,
			expectCurrentLabel: "",
			expectNextLabel:    lgtmOne,
		},
		{
			name:               "Current no LGTM, needs 2 LGTM",
			labels:             []github.Label{},
			needsLgtm:          2,
			expectCurrentLabel: "",
			expectNextLabel:    lgtmOne,
		},
		{
			name: "Current LGT1, needs 1 LGTM",
			labels: []github.Label{
				{
					Name: lgtmOne,
				},
			},
			needsLgtm:          1,
			expectCurrentLabel: lgtmOne,
			expectNextLabel:    "",
		},
		{
			name: "Current LGT1, needs 2 LGTM",
			labels: []github.Label{
				{
					Name: lgtmOne,
				},
			},
			needsLgtm:          2,
			expectCurrentLabel: lgtmOne,
			expectNextLabel:    lgtmTwo,
		},
		{
			name: "Current LGT2, needs 2 LGTM",
			labels: []github.Label{
				{
					Name: lgtmTwo,
				},
			},
			needsLgtm:          2,
			expectCurrentLabel: lgtmTwo,
			expectNextLabel:    "",
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			currentLabel, nextLabel := getCurrentAndNextLabel(externalplugins.LgtmLabelPrefix, tc.labels, tc.needsLgtm)

			if currentLabel != tc.expectCurrentLabel {
				t.Fatalf("currentLabel mismatch: got %v, want %v", currentLabel, tc.expectCurrentLabel)
			}

			if nextLabel != tc.expectNextLabel {
				t.Fatalf("nextLabel mismatch: got %v, want %v", nextLabel, tc.expectNextLabel)
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
