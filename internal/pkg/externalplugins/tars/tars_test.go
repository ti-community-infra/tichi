//nolint:scopelint
package tars

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/prow-github/pkg/github"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/plugins"
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

func getPullRequest() *github.PullRequest {
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

func TestHandleIssueCommentEvent(t *testing.T) {
	currentBaseSHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	outOfDateSHA := "0bd3ed50c88cd53a0931609dsa9d-0a9d0-as9d0"

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
	updatedPrCommits := func() []github.RepositoryCommit {
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
		message    string

		expectComment  bool
		expectDeletion bool
		expectUpdate   bool
	}{
		{
			name: "No pull request, ignoring",
		},
		{
			name:       "updated no-op",
			pr:         getPullRequest(),
			baseCommit: baseCommit,
			prCommits:  updatedPrCommits(),
			outOfDate:  false,
		},
		{
			name:           "out of date with message",
			pr:             getPullRequest(),
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
		{
			name:           "out of date with empty message",
			pr:             getPullRequest(),
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "",
			expectDeletion: false,
			expectComment:  false,
			expectUpdate:   true,
		},
		{
			name:   "merged pr is ignored",
			pr:     getPullRequest(),
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
			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityTars = []externalplugins.TiCommunityTars{
				{
					Repos:   []string{"org/repo"},
					Message: tc.message,
				},
			}
			if err := HandleIssueCommentEvent(logrus.WithField("plugin", PluginName), fc, ice, cfg); err != nil {
				t.Fatalf("error handling issue comment event: %v", err)
			}
			fc.compareExpected(t, "org", "repo", 5, tc.expectComment, tc.expectDeletion, tc.outOfDate)
		})
	}
}

func TestHandlePullRequestEvent(t *testing.T) {
	currentBaseSHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	outOfDateSHA := "0bd3ed50c88cd53a0931609dsa9d-0a9d0-as9d0"
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
		merged     bool
		baseCommit github.RepositoryCommit
		prCommits  []github.RepositoryCommit
		outOfDate  bool
		message    string

		expectComment  bool
		expectDeletion bool
		expectUpdate   bool
	}{
		{
			name:       "updated no-op",
			baseCommit: baseCommit,
			prCommits:  updatePrCommits(),
			outOfDate:  false,
		},
		{
			name:           "out of date with message",
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
		{
			name:           "out of date with empty message",
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "",
			expectDeletion: false,
			expectComment:  false,
			expectUpdate:   true,
		},
		{
			name:   "merged pr is ignored",
			merged: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fc := newFakeGithubClient(nil, nil, tc.baseCommit, tc.prCommits, tc.outOfDate)
			pre := &github.PullRequestEvent{
				Action: github.PullRequestActionSynchronize,
				PullRequest: github.PullRequest{
					Base: github.PullRequestBranch{
						Repo: github.Repo{
							Name:  "repo",
							Owner: github.User{Login: "org"},
						},
					},
					Merged: tc.merged,
					Number: 5,
				},
			}
			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityTars = []externalplugins.TiCommunityTars{
				{
					Repos:   []string{"org/repo"},
					Message: tc.message,
				},
			}
			if err := HandlePullRequestEvent(logrus.WithField("plugin", PluginName), fc, pre, cfg); err != nil {
				t.Fatalf("Unexpected error handling event: %v.", err)
			}
			fc.compareExpected(t, "org", "repo", 5, tc.expectComment, tc.expectDeletion, tc.expectUpdate)
		})
	}
}

func TestHandleAll(t *testing.T) {
	currentBaseSHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	outOfDateSHA := "0bd3ed50c88cd53a0931609dsa9d-0a9d0-as9d0"

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

	updatedPrCommits := func() []github.RepositoryCommit {
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

	testCases := []struct {
		name       string
		pr         *github.PullRequest
		baseCommit github.RepositoryCommit
		prCommits  []github.RepositoryCommit
		outOfDate  bool
		message    string

		expectComment  bool
		expectDeletion bool
		expectUpdate   bool
	}{
		{
			name: "No pull request, ignoring",
		},
		{
			name:       "updated no-op",
			pr:         getPullRequest(),
			baseCommit: baseCommit,
			prCommits:  updatedPrCommits(),
			outOfDate:  false,
		},
		{
			name:           "out of date with message",
			pr:             getPullRequest(),
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
		{
			name:           "out of date with empty message",
			pr:             getPullRequest(),
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "",
			expectDeletion: false,
			expectComment:  false,
			expectUpdate:   true,
		},
		{
			name: "merged pr is ignored",
			pr:   getPullRequest(),
		},
	}

	oldSleep := sleep
	sleep = func(time.Duration) {}
	defer func() { sleep = oldSleep }()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var prs []pullRequest
			if tc.pr != nil {
				pr := pullRequest{}
				pr.Number = githubql.Int(tc.pr.Number)
				pr.Repository.Name = "repo"
				pr.Repository.Owner.Login = "org"
				prs = append(prs, pr)
			}
			fc := newFakeGithubClient(prs, tc.pr, tc.baseCommit, tc.prCommits, tc.outOfDate)
			config := &plugins.Configuration{
				ExternalPlugins: map[string][]plugins.ExternalPlugin{"/": {{Name: PluginName}}},
			}
			externalConfig := &externalplugins.Configuration{}
			externalConfig.TiCommunityTars = []externalplugins.TiCommunityTars{
				{
					Repos:   []string{"org/repo"},
					Message: tc.message,
				},
			}
			if err := HandleAll(logrus.WithField("plugin", PluginName), fc, config, externalConfig); err != nil {
				t.Fatalf("Unexpected error handling all prs: %v.", err)
			}
			fc.compareExpected(t, "org", "repo", 5, tc.expectComment, tc.expectDeletion, tc.outOfDate)
		})
	}
}

func TestShouldPrune(t *testing.T) {
	botName := "ti-community-bot"
	message := "updated"

	f := shouldPrune(botName, message)

	testCases := []struct {
		name    string
		comment github.IssueComment

		shouldPrune bool
	}{
		{
			name: "not bot comment",
			comment: github.IssueComment{
				Body: "updated",
				User: github.User{
					Login: "user",
				},
			},
			shouldPrune: false,
		},
		{
			name: "random body",
			comment: github.IssueComment{
				Body: "random",
				User: github.User{
					Login: "user",
				},
			},
			shouldPrune: false,
		},
		{
			name: "bot updated comment",
			comment: github.IssueComment{
				Body: "updated",
				User: github.User{
					Login: "ti-community-bot",
				},
			},
			shouldPrune: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shouldPrune := f(tc.comment)
			if shouldPrune != tc.shouldPrune {
				t.Errorf("Mismatch should prune except %v, but got %v.", tc.shouldPrune, shouldPrune)
			}
		})
	}
}
