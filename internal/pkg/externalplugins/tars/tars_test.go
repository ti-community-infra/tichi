//nolint:scopelint
package tars

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/prow-github/pkg/github"
)

func testKey(org, repo string, num int) string {
	return fmt.Sprintf("%s/%s#%d", org, repo, num)
}

type fakeGithub struct {
	allPRs []struct {
		PullRequest pullRequest `graphql:"... on PullRequest"`
	}
	pr *github.PullRequest

	baseCommit github.RepositoryCommit

	prCommits []github.RepositoryCommit

	outOfDate bool

	// The following are maps are keyed using 'testKey'
	commentCreated, commentDeleted map[string]bool
}

func newFakeGithubClient(prs []pullRequest, pr *github.PullRequest,
	baseCommit github.RepositoryCommit, prCommits []github.RepositoryCommit, outOfDate bool) *fakeGithub {
	f := &fakeGithub{
		commentCreated: make(map[string]bool),
		commentDeleted: make(map[string]bool),
		pr:             pr,
		baseCommit:     baseCommit,
		prCommits:      prCommits,
		outOfDate:      outOfDate,
	}

	for _, pr := range prs {
		s := struct {
			PullRequest pullRequest `graphql:"... on PullRequest"`
		}{pr}
		f.allPRs = append(f.allPRs, s)
	}

	return f
}

func (f *fakeGithub) BotName() (string, error) {
	return "ti-community-prow", nil
}

func (f *fakeGithub) CreateComment(org, repo string, number int, comment string) error {
	f.commentCreated[testKey(org, repo, number)] = true
	return nil
}

func (f *fakeGithub) DeleteStaleComments(org, repo string, number int,
	comments []github.IssueComment, isStale func(github.IssueComment) bool) error {
	f.commentDeleted[testKey(org, repo, number)] = true
	return nil
}

func (f *fakeGithub) Query(_ context.Context, q interface{}, _ map[string]interface{}) error {
	query, ok := q.(*searchQuery)
	if !ok {
		return errors.New("invalid query format")
	}
	query.Search.Nodes = f.allPRs
	return nil
}

func (f *fakeGithub) GetPullRequest(org, repo string, number int) (*github.PullRequest, error) {
	if f.pr != nil {
		return f.pr, nil
	}
	return nil, fmt.Errorf("didn't find pull request %s/%s#%d", org, repo, number)
}

func (f *fakeGithub) GetSingleCommit(org, repo, ref string) (github.RepositoryCommit, error) {
	return f.baseCommit, nil
}

func (f *fakeGithub) ListPRCommits(org, repo string, number int) ([]github.RepositoryCommit, error) {
	return f.prCommits, nil
}

func (f *fakeGithub) UpdatePullRequestBranch(org, repo string, number int, expectedHeadSha *string) error {
	if f.outOfDate {
		f.outOfDate = false
	}
	return nil
}

func (f *fakeGithub) compareExpected(t *testing.T, org, repo string,
	num int, expectComment bool, expectDeletion bool, expectUpdate bool) {
	key := testKey(org, repo, num)

	if expectComment && !f.commentCreated[key] {
		t.Errorf("Expected a comment to be created on %s, but none was.", key)
	} else if !expectComment && f.commentCreated[key] {
		t.Errorf("Unexpected comment on %s.", key)
	}
	if expectDeletion && !f.commentDeleted[key] {
		t.Errorf("Expected a comment to be deleted from %s, but none was.", key)
	} else if !expectDeletion && f.commentDeleted[key] {
		t.Errorf("Unexpected comment deletion on %s.", key)
	}

	if expectUpdate {
		if f.outOfDate {
			t.Errorf("Expected update pull request %s, but still out of date.", key)
		}
	}
}

func TestHandleIssueCommentEvent(t *testing.T) {
	currentBaseSHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	outOfDateSHA := "0bd3ed50c88cd53a0931609dsa9d-0a9d0-as9d0"

	pr := func() *github.PullRequest {
		pr := github.PullRequest{
			Base: github.PullRequestBranch{
				Repo: github.Repo{
					Name:  "repo",
					Owner: github.User{Login: "org"},
				},
			},
			Number: 5,
		}
		return &pr
	}
	baseCommit := github.RepositoryCommit{
		SHA: currentBaseSHA,
	}
	outOfDatePrCommits := func() []github.RepositoryCommit {
		prCommits := []github.RepositoryCommit{
			{
				Parents: []github.GitCommit{
					{
						SHA: outOfDateSHA,
					},
				},
			},
		}
		return prCommits
	}
	updatePrCommits := func() []github.RepositoryCommit {
		prCommits := []github.RepositoryCommit{
			{
				Parents: []github.GitCommit{
					{
						SHA: currentBaseSHA,
					},
				},
			},
		}
		return prCommits
	}

	oldSleep := sleep
	sleep = func(time.Duration) {}
	defer func() { sleep = oldSleep }()

	testCases := []struct {
		name       string
		pr         *github.PullRequest
		merged     bool
		baseCommit github.RepositoryCommit
		prCommits  []github.RepositoryCommit
		outOfDate  bool

		expectComment  bool
		expectDeletion bool
		expectUpdate   bool
	}{
		{
			name: "No pull request, ignoring",
		},
		{
			name:       "updated no-op",
			pr:         pr(),
			baseCommit: baseCommit,
			prCommits:  updatePrCommits(),
			outOfDate:  false,
		},
		{
			name:           "first time out of date",
			pr:             pr(),
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
		{
			name:           "second time out of date",
			pr:             pr(),
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
		{
			name:   "merged pr is ignored",
			pr:     pr(),
			merged: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fc := newFakeGithubClient(nil, tc.pr, tc.baseCommit, tc.prCommits, tc.outOfDate)
			ice := &github.IssueCommentEvent{}
			if tc.pr != nil {
				ice.Issue.PullRequest = &struct{}{}
			}
			if err := HandleIssueCommentEvent(logrus.WithField("plugin", PluginName), fc, ice); err != nil {
				t.Fatalf("error handling issue comment event: %v", err)
			}
			fc.compareExpected(t, "org", "repo", 5, tc.expectComment, tc.expectDeletion, tc.outOfDate)
		})
	}
}
