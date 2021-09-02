package tars

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
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

func (f *fakeGithub) BotUserChecker() (func(candidate string) bool, error) {
	return func(candidate string) bool {
		return candidate == "tichi"
	}, nil
}

func (f *fakeGithub) CreateComment(org, repo string, number int, _ string) error {
	f.commentCreated[testKey(org, repo, number)] = true
	return nil
}

func (f *fakeGithub) DeleteStaleComments(org, repo string, number int,
	_ []github.IssueComment, _ func(github.IssueComment) bool) error {
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

func (f *fakeGithub) GetSingleCommit(string, string, string) (github.RepositoryCommit, error) {
	return f.baseCommit, nil
}

func (f *fakeGithub) ListPRCommits(string, string, int) ([]github.RepositoryCommit, error) {
	return f.prCommits, nil
}

func (f *fakeGithub) UpdatePullRequestBranch(string, string, int, *string) error {
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

func getPullRequest(org string, repo string, num int) *github.PullRequest {
	pr := github.PullRequest{
		Base: github.PullRequestBranch{
			Repo: github.Repo{
				Name:  repo,
				Owner: github.User{Login: org},
			},
		},
		Number: num,
	}
	return &pr
}

func TestHandleIssueCommentEvent(t *testing.T) {
	currentBaseSHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	outOfDateSHA := "0bd3ed50c88cd53a0931609dsa9d-0a9d0-as9d0"
	triggerLabel := "trigger-update"
	excludeLabel := "exclude"

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

	testcases := []struct {
		name       string
		pr         *github.PullRequest
		labels     []github.Label
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
			name: "updated no-op",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit: baseCommit,
			prCommits:  updatedPrCommits(),
			outOfDate:  false,
		},
		{
			name: "out of date with message",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
		{
			name: "out of date with empty message",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "",
			expectDeletion: false,
			expectComment:  false,
			expectUpdate:   true,
		},
		{
			name: "out of date with message and without trigger label",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: "random",
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: false,
			expectComment:  false,
			expectUpdate:   false,
		},
		{
			name: "out of date with message and with exclude label",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
				{
					Name: excludeLabel,
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: false,
			expectComment:  false,
			expectUpdate:   false,
		},
		{
			name: "out of date with message trigger label",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			fc := newFakeGithubClient(nil, tc.pr, tc.baseCommit, tc.prCommits, tc.outOfDate)
			ice := &github.IssueCommentEvent{}
			if tc.pr != nil {
				ice.Issue.PullRequest = &struct{}{}
			}
			if len(tc.labels) != 0 {
				tc.pr.Labels = tc.labels
			}
			cfg := &externalplugins.Configuration{}
			cfg.TiCommunityTars = []externalplugins.TiCommunityTars{
				{
					Repos:         []string{"org/repo"},
					Message:       tc.message,
					OnlyWhenLabel: triggerLabel,
					ExcludeLabels: []string{excludeLabel},
				},
			}
			if err := HandleIssueCommentEvent(logrus.WithField("plugin", PluginName), fc, ice, cfg); err != nil {
				t.Fatalf("error handling issue comment event: %v", err)
			}
			fc.compareExpected(t, "org", "repo", 5, tc.expectComment, tc.expectDeletion, tc.expectUpdate)
		})
	}
}

func TestHandlePushEvent(t *testing.T) {
	currentBaseSHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	outOfDateSHA := "0bd3ed50c88cd53a0931609dsa9d-0a9d0-as9d0"
	triggerLabel := "trigger-update"

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

	testcases := []struct {
		name       string
		pe         *github.PushEvent
		pr         *github.PullRequest
		labels     []github.Label
		baseCommit github.RepositoryCommit
		prCommits  []github.RepositoryCommit
		outOfDate  bool
		message    string

		expectComment  bool
		expectDeletion bool
		expectUpdate   bool
	}{
		{
			name: "tags ref, ignoring",
			pe: &github.PushEvent{
				Ref: "refs/tags/simple-tag",
			},
		},
		{
			name: "updated no-op",
			pe: &github.PushEvent{
				Ref: "refs/heads/main",
			},
			pr: getPullRequest("org1", "repo1", 6),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit: baseCommit,
			prCommits:  updatedPrCommits(),
			outOfDate:  false,
		},
		{
			name: "out of date with message",
			pr:   getPullRequest("org1", "repo1", 6),
			pe: &github.PushEvent{
				Ref: "refs/heads/main",
			},
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
		{
			name: "out of date with empty message",
			pe: &github.PushEvent{
				Ref: "refs/heads/main",
			},
			pr: getPullRequest("org1", "repo1", 6),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "",
			expectDeletion: false,
			expectComment:  false,
			expectUpdate:   true,
		},
		{
			name: "out of date with message and trigger label",
			pe: &github.PushEvent{
				Ref: "refs/heads/main",
			},
			pr: getPullRequest("org1", "repo1", 6),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
	}

	oldSleep := sleep
	sleep = func(time.Duration) {}
	defer func() { sleep = oldSleep }()

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			// For now we only add one pr.
			var prs []pullRequest
			if tc.pr != nil {
				prs = generatePullRequests("org1", "repo1", tc.pr, tc.prCommits, tc.labels)
			}
			fc := newFakeGithubClient(prs, tc.pr, tc.baseCommit, tc.prCommits, tc.outOfDate)
			externalConfig := &externalplugins.Configuration{}
			externalConfig.TiCommunityTars = []externalplugins.TiCommunityTars{
				{
					Repos:         []string{"org1/repo1"},
					Message:       tc.message,
					OnlyWhenLabel: triggerLabel,
				},
			}
			if err := HandlePushEvent(logrus.WithField("plugin", PluginName), fc, tc.pe, externalConfig); err != nil {
				t.Fatalf("error handling issue comment event: %v", err)
			}
			fc.compareExpected(t, "org1", "repo1", 6, tc.expectComment, tc.expectDeletion, tc.expectUpdate)
		})
	}
}

func TestHandleAll(t *testing.T) {
	currentBaseSHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	outOfDateSHA := "0bd3ed50c88cd53a0931609dsa9d-0a9d0-as9d0"
	triggerLabel := "trigger-update"

	baseCommit := github.RepositoryCommit{
		SHA: currentBaseSHA,
	}

	outOfDatePrCommits := func() []github.RepositoryCommit {
		prCommits := []github.RepositoryCommit{
			{
				Parents: []github.GitCommit{
					{
						SHA: "314506403e9c39bf599b53be524e16b25c8124cf",
					},
				},
			},
			{
				Parents: []github.GitCommit{
					{
						SHA: "fc57072d7d4fc4220d4856ceb6bd6c861e9ef3a0",
					},
				},
			},
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

	testcases := []struct {
		name       string
		pr         *github.PullRequest
		labels     []github.Label
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
			name: "the first commit is based on current base commit, ignoring",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit: baseCommit,
			prCommits: []github.RepositoryCommit{
				{
					SHA: "314506403e9c39bf599b53be524e16b25c8124cf",
					Parents: []github.GitCommit{
						{
							SHA: currentBaseSHA,
						},
					},
				},
				{
					SHA: "546c9f753db86f80e50f26f4da588d0d24b78c51",
					Parents: []github.GitCommit{
						{
							SHA: "314506403e9c39bf599b53be524e16b25c8124cf",
						},
					},
				},
				{
					SHA: "e8de18edf50ed760701f3b8b91245142b9d0d974",
					Parents: []github.GitCommit{
						{
							SHA: "546c9f753db86f80e50f26f4da588d0d24b78c51",
						},
					},
				},
			},
			outOfDate: false,
		},
		{
			name: "PR have already merged the latest base, ignoring",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit: baseCommit,
			prCommits: []github.RepositoryCommit{
				{
					SHA: "314506403e9c39bf599b53be524e16b25c8124cf",
					Parents: []github.GitCommit{
						{
							SHA: "f0a713b8f7b52e3aebfd16e68911b1b69129bf11",
						},
					},
				},
				{
					SHA: "546c9f753db86f80e50f26f4da588d0d24b78c51",
					Parents: []github.GitCommit{
						{
							SHA: "314506403e9c39bf599b53be524e16b25c8124cf",
						},
						{
							SHA: currentBaseSHA,
						},
					},
				},
				{
					SHA: "e8de18edf50ed760701f3b8b91245142b9d0d974",
					Parents: []github.GitCommit{
						{
							SHA: "546c9f753db86f80e50f26f4da588d0d24b78c51",
						},
					},
				},
			},
			outOfDate: false,
		},
		{
			name: "out of date with message",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
		{
			name: "out of date with empty message",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "",
			expectDeletion: false,
			expectComment:  false,
			expectUpdate:   true,
		},
		{
			name: "out of date with message and trigger label",
			pr:   getPullRequest("org", "repo", 5),
			labels: []github.Label{
				{
					Name: triggerLabel,
				},
			},
			baseCommit:     baseCommit,
			prCommits:      outOfDatePrCommits(),
			outOfDate:      true,
			message:        "updated",
			expectDeletion: true,
			expectComment:  true,
			expectUpdate:   true,
		},
	}

	oldSleep := sleep
	sleep = func(time.Duration) {}
	defer func() { sleep = oldSleep }()

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			// For now we only add one pr.
			var prs []pullRequest
			if tc.pr != nil {
				prs = generatePullRequests("org", "repo", tc.pr, tc.prCommits, tc.labels)
			}
			fc := newFakeGithubClient(prs, tc.pr, tc.baseCommit, tc.prCommits, tc.outOfDate)
			cfg := &plugins.Configuration{
				ExternalPlugins: map[string][]plugins.ExternalPlugin{"/": {{Name: PluginName}}},
			}
			externalConfig := &externalplugins.Configuration{}
			externalConfig.TiCommunityTars = []externalplugins.TiCommunityTars{
				{
					Repos:         []string{"org/repo"},
					Message:       tc.message,
					OnlyWhenLabel: triggerLabel,
				},
			}
			if err := HandleAll(logrus.WithField("plugin", PluginName), fc, cfg, externalConfig); err != nil {
				t.Fatalf("Unexpected error handling all prs: %v.", err)
			}
			fc.compareExpected(t, "org", "repo", 5, tc.expectComment, tc.expectDeletion, tc.expectUpdate)
		})
	}
}

func generatePullRequests(org string, repo string, pr *github.PullRequest,
	prCommits []github.RepositoryCommit, labels []github.Label) []pullRequest {
	var prs []pullRequest

	graphPr := pullRequest{}
	// Set the basic info.
	graphPr.Number = githubql.Int(pr.Number)
	graphPr.Repository.Name = githubql.String(repo)
	graphPr.Repository.Owner.Login = githubql.String(org)
	graphPr.Author.Login = githubql.String(pr.User.Login)
	graphPr.BaseRef.Name = githubql.String(pr.Base.Ref)

	// Convert the commit.
	lastCommit := prCommits[len(prCommits)-1]
	graphCommit := struct {
		Commit struct {
			OID     githubql.GitObjectID `graphql:"oid"`
			Parents struct {
				Nodes []struct {
					OID githubql.GitObjectID `graphql:"oid"`
				}
			} `graphql:"parents(first:10)"`
		}
	}{}
	for _, parent := range lastCommit.Parents {
		s := struct {
			OID githubql.GitObjectID `graphql:"oid"`
		}{
			OID: githubql.GitObjectID(parent.SHA),
		}
		graphCommit.Commit.Parents.Nodes = append(graphCommit.Commit.Parents.Nodes, s)
	}

	// Set the labels.
	if len(labels) != 0 {
		pr.Labels = labels
		for _, label := range pr.Labels {
			s := struct {
				Name githubql.String
			}{
				Name: githubql.String(label.Name),
			}
			graphPr.Labels.Nodes = append(graphPr.Labels.Nodes, s)
		}
	}

	graphPr.Commits.Nodes = append(graphPr.Commits.Nodes, graphCommit)
	prs = append(prs, graphPr)

	return prs
}

func TestShouldPrune(t *testing.T) {
	message := "updated"
	isBot := func(candidate string) bool {
		return candidate == "ti-community-bot"
	}
	f := shouldPrune(isBot, message)

	testcases := []struct {
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

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			shouldPrune := f(tc.comment)
			if shouldPrune != tc.shouldPrune {
				t.Errorf("Mismatch should prune expect %v, but got %v.", tc.shouldPrune, shouldPrune)
			}
		})
	}
}

func TestHelpProvider(t *testing.T) {
	enabledRepos := []config.OrgRepo{
		{Org: "org1", Repo: "repo"},
		{Org: "org2", Repo: "repo"},
	}
	testcases := []struct {
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
			configInfoExcludes: []string{configInfoAutoUpdatedMessagePrefix},
		},
		{
			name: "All configs enabled",
			config: &externalplugins.Configuration{
				TiCommunityTars: []externalplugins.TiCommunityTars{
					{
						Repos:   []string{"org2/repo"},
						Message: "updated",
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{configInfoAutoUpdatedMessagePrefix},
		},
	}
	for _, testcase := range testcases {
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
