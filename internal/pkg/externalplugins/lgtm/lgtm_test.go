// nolint:lll
package lgtm

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/ownersclient"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
)

var (
	lgtmOne = fmt.Sprintf("%s%d", externalplugins.LgtmLabelPrefix, 1)
	lgtmTwo = fmt.Sprintf("%s%d", externalplugins.LgtmLabelPrefix, 2)
)

const botName = "ti-chi-bot"

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

type fakeGithubClient struct {
	IssueCommentID int
	IssueComments  map[int][]github.IssueComment

	// All Labels That Exist In The Repo
	RepoLabelsExisting []string
	// org/repo#number:label
	IssueLabelsAdded    []string
	IssueLabelsExisting []string
	IssueLabelsRemoved  []string

	// org/repo#number:body
	IssueCommentsAdded []string
	// org/repo#issuecommentid
	IssueCommentsDeleted []string

	PullRequests  map[int]*github.PullRequest
	Collaborators []string

	// lock to be thread safe
	lock sync.RWMutex
}

// AddLabel adds a label
func (f *fakeGithubClient) AddLabel(owner, repo string, number int, label string) error {
	return f.AddLabels(owner, repo, number, label)
}

// AddLabels adds a list of labels
func (f *fakeGithubClient) AddLabels(owner, repo string, number int, labels ...string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	for _, label := range labels {
		labelString := fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, label)
		if sets.NewString(f.IssueLabelsAdded...).Has(labelString) {
			return fmt.Errorf("cannot add %v to %s/%s/#%d", label, owner, repo, number)
		}
		if f.RepoLabelsExisting == nil {
			f.IssueLabelsAdded = append(f.IssueLabelsAdded, labelString)
			continue
		}

		var repoLabelExists bool
		for _, l := range f.RepoLabelsExisting {
			if label == l {
				f.IssueLabelsAdded = append(f.IssueLabelsAdded, labelString)
				repoLabelExists = true
				break
			}
		}
		if !repoLabelExists {
			return fmt.Errorf("cannot add %v to %s/%s/#%d", label, owner, repo, number)
		}
	}
	return nil
}

// RemoveLabel removes a label
func (f *fakeGithubClient) RemoveLabel(owner, repo string, number int, label string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	labelString := fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, label)
	if !sets.NewString(f.IssueLabelsRemoved...).Has(labelString) {
		f.IssueLabelsRemoved = append(f.IssueLabelsRemoved, labelString)
		return nil
	}
	return fmt.Errorf("cannot remove %v from %s/%s/#%d", label, owner, repo, number)
}

// GetIssueLabels gets labels on an issue
func (f *fakeGithubClient) GetIssueLabels(owner, repo string, number int) ([]github.Label, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	re := regexp.MustCompile(fmt.Sprintf(`^%s/%s#%d:(.*)$`, owner, repo, number))
	la := []github.Label{}
	allLabels := sets.NewString(f.IssueLabelsExisting...)
	allLabels.Insert(f.IssueLabelsAdded...)
	allLabels.Delete(f.IssueLabelsRemoved...)
	for _, l := range allLabels.List() {
		groups := re.FindStringSubmatch(l)
		if groups != nil {
			la = append(la, github.Label{Name: groups[1]})
		}
	}
	return la, nil
}

// CreateComment adds a comment to a PR
func (f *fakeGithubClient) CreateComment(owner, repo string, number int, comment string) error {
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

// EditComment edits a comment. Its a stub that does nothing.
func (f *fakeGithubClient) EditComment(org, repo string, id int, comment string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	for num, ics := range f.IssueComments {
		for i, ic := range ics {
			if ic.ID == id {
				f.IssueComments[num][i].Body = comment
				return nil
			}
		}
	}
	return fmt.Errorf("could not find issue comment %d", id)
}

// DeleteComment deletes a comment.
func (f *fakeGithubClient) DeleteComment(owner, repo string, id int) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.IssueCommentsDeleted = append(f.IssueCommentsDeleted, fmt.Sprintf("%s/%s#%d", owner, repo, id))
	for num, ics := range f.IssueComments {
		for i, ic := range ics {
			if ic.ID == id {
				f.IssueComments[num] = append(ics[:i], ics[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("could not find issue comment %d", id)
}

// ListIssueComments returns comments.
func (f *fakeGithubClient) ListIssueComments(owner, repo string, number int) ([]github.IssueComment, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return append([]github.IssueComment{}, f.IssueComments[number]...), nil
}

func (f *fakeGithubClient) BotUserChecker() (func(candidate string) bool, error) {
	return func(candidate string) bool {
		candidate = strings.TrimSuffix(candidate, "[bot]")
		return candidate == botName
	}, nil
}

func getNotificationMessage(reviewers []string) string {
	ownersLink := fmt.Sprintf(ownersclient.OwnersURLFmt, "https://prow-dev.tidb.io/tichi", "org", "repo", 5)
	message, err := getMessage(reviewers,
		"https://prow-dev.tidb.io/command-help",
		"https://book.prow.tidb.io/#/en/workflows/pr",
		ownersLink, "org", "repo")

	if err != nil {
		return ""
	}

	return *message
}

// compareComments used to determine whether two comment lists are equal.
func compareComments(actualComments []string, expectComments []string) bool {
	if len(actualComments) != len(expectComments) {
		return false
	}

	if len(actualComments) == 0 {
		return true
	}

	for i := 0; i < len(actualComments); i++ {
		if expectComments[i] != actualComments[i] {
			return false
		}
	}

	return true
}

func TestLGTMFromApproveReview(t *testing.T) {
	var testcases = []struct {
		name           string
		comments       []github.IssueComment
		state          github.ReviewState
		action         github.ReviewEventAction
		body           string
		reviewer       string
		currentLabel   string
		isCancel       bool
		shouldToggle   bool
		expectComments []github.IssueComment
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
			name: "Request changes review by reviewer, lgtm on pr",
			comments: []github.IssueComment{
				{
					ID: 1000,
					User: github.User{
						Login: botName,
					},
					Body: getNotificationMessage([]string{"collab1"}),
				},
			},
			state:        github.ReviewStateChangesRequested,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			currentLabel: lgtmOne,
			isCancel:     true,
			shouldToggle: true,
			expectComments: []github.IssueComment{
				{
					ID: 1000,
					User: github.User{
						Login: botName,
					},
					Body: getNotificationMessage(nil),
				},
			},
		},
		{
			name: "Approve review by reviewer, no lgtm on pr",
			comments: []github.IssueComment{
				{
					ID: 1000,
					User: github.User{
						Login: botName,
					},
					Body: getNotificationMessage([]string{"collab2"}),
				},
			},
			state:        github.ReviewStateApproved,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			currentLabel: lgtmOne,
			shouldToggle: true,
			expectComments: []github.IssueComment{
				{
					ID: 1000,
					User: github.User{
						Login: botName,
					},
					Body: getNotificationMessage([]string{"collab1", "collab2"}),
				},
			},
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
			name:         "Comment Review by issue author, no lgtm on pr",
			state:        github.ReviewStateCommented,
			action:       github.ReviewActionSubmitted,
			reviewer:     "author",
			body:         "/lgtm",
			shouldToggle: false,
		},
		{
			name:         "(Deprecated) Comment Review with /lgtm comment",
			state:        github.ReviewStateCommented,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			body:         "/lgtm",
			currentLabel: lgtmOne,
			shouldToggle: false,
		},
		{
			name:         "(Deprecated) Comment Review with /lgtm cancel comment",
			state:        github.ReviewStateCommented,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			body:         "/lgtm cancel",
			currentLabel: lgtmOne,
			shouldToggle: false,
		},
		{
			name:         "Comment Review with random comment",
			state:        github.ReviewStateCommented,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			body:         "/random content",
			currentLabel: lgtmOne,
			shouldToggle: false,
		},
		{
			name: "(Deprecated) Comment body has /lgtm cancel on Approve Review",
			comments: []github.IssueComment{
				{
					ID:   100,
					User: github.User{Login: botName},
					Body: getNotificationMessage([]string{"collab2"}),
				},
			},
			state:        github.ReviewStateApproved,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			body:         "/lgtm cancel",
			currentLabel: lgtmOne,
			shouldToggle: true,
			expectComments: []github.IssueComment{
				{
					ID:   100,
					User: github.User{Login: botName},
					Body: getNotificationMessage([]string{"collab1", "collab2"}),
				},
			},
		},
		{
			name: "The comment list contains redundant comments",
			comments: []github.IssueComment{
				{
					ID: 1001,
					User: github.User{
						Login: botName,
					},
					Body: "redundant comment",
				},
				{
					ID: 1002,
					User: github.User{
						Login: botName,
					},
					Body: getNotificationMessage([]string{"collab1"}),
				},
			},
			state:        github.ReviewStateChangesRequested,
			action:       github.ReviewActionSubmitted,
			reviewer:     "collab1",
			currentLabel: lgtmOne,
			isCancel:     true,
			shouldToggle: true,
			expectComments: []github.IssueComment{
				{
					ID: 1001,
					User: github.User{
						Login: botName,
					},
					Body: "redundant comment",
				},
				{
					ID: 1002,
					User: github.User{
						Login: botName,
					},
					Body: getNotificationMessage(nil),
				},
			},
		},
	}
	for _, tc := range testcases {
		fc := &fakeGithubClient{
			IssueComments: map[int][]github.IssueComment{
				5: tc.comments,
			},
			IssueLabelsExisting: []string{},
			IssueLabelsAdded:    []string{},
			IssueLabelsRemoved:  []string{},
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
		cfg.CommandHelpLink = "https://prow-dev.tidb.io/command-help"
		cfg.PRProcessLink = "https://book.prow.tidb.io/#/en/workflows/pr"
		cfg.TichiWebURL = "https://prow-dev.tidb.io/tichi"
		cfg.TiCommunityLgtm = []externalplugins.TiCommunityLgtm{
			{
				Repos:              []string{"org/repo"},
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

			var actualComments []string
			for _, actualComment := range fc.IssueComments[5] {
				actualComments = append(actualComments, actualComment.Body)
			}

			var expectComments []string
			for _, expectComment := range tc.expectComments {
				expectComments = append(expectComments, expectComment.Body)
			}

			if compareComments(actualComments, expectComments) == false {
				t.Errorf("expect comments: %s, but got comments: %s", expectComments, actualComments)
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
			expectComment: "org/repo#101:[REVIEW NOTIFICATION]\n\nThis pull request has not been approved.\n\n\nTo complete the [pull request process](https://prProcessLink), please ask the reviewers in the [list](https://tichiWebLink/repos/org/repo/pulls/101/owners) to review by filling `/cc @reviewer` in the comment.\nAfter your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list](https://tichiWebLink/repos/org/repo/pulls/101/owners) by filling  `/assign @committer` in the comment to help you merge this pull request.\n\nThe full list of commands accepted by this bot can be found [here](https://commandHelpLink?repo=org%2Frepo).\n\n<details>\n\nReviewer can indicate their review by submitting an approval review.\nReviewer can cancel approval by submitting a request changes review.\n</details>\n\n<!--Review Notification Identifier-->",
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
			name:         "Empty config",
			config:       &externalplugins.Configuration{},
			enabledRepos: enabledRepos,
		},
		{
			name: "All configs enabled",
			config: &externalplugins.Configuration{
				TiCommunityLgtm: []externalplugins.TiCommunityLgtm{
					{
						Repos:              []string{"org2/repo"},
						PullOwnersEndpoint: "https://fake",
					},
				},
			},
			enabledRepos: enabledRepos,
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
