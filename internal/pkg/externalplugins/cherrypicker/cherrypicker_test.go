/*
Copyright 2017 The Kubernetes Authors.
Copyright 2021 The TiChi Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

The original file of the code is at:
https://github.com/kubernetes/test-infra/blob/master/prow/external-plugins/cherrypicker/server_test.go,
which we modified to add support for copying the labels and reviewers.
*/

package cherrypicker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/localgit"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/plugins"
)

var commentFormat = "%s/%s#%d %s"

type fghc struct {
	sync.Mutex
	pr       *github.PullRequest
	isMember bool

	patch      []byte
	comments   []string
	prs        []github.PullRequest
	prComments []github.IssueComment
	prLabels   []github.Label
	orgMembers []github.TeamMember
	issues     []github.Issue
}

func (f *fghc) AddLabels(org, repo string, number int, labels ...string) error {
	f.Lock()
	defer f.Unlock()
	for i := range f.prs {
		if number == f.prs[i].Number {
			for _, label := range labels {
				f.prs[i].Labels = append(f.prs[i].Labels, github.Label{Name: label})
			}
		}
	}
	return nil
}

func (f *fghc) AssignIssue(org, repo string, number int, logins []string) error {
	f.Lock()
	defer f.Unlock()
	var users []github.User
	for _, login := range logins {
		users = append(users, github.User{Login: login})
	}
	for i := range f.prs {
		if number == f.prs[i].Number {
			f.prs[i].Assignees = users
		}
	}
	return nil
}

func (f *fghc) RequestReview(org, repo string, number int, logins []string) error {
	f.Lock()
	defer f.Unlock()
	var users []github.User
	for _, login := range logins {
		users = append(users, github.User{Login: login})
	}
	for i := range f.prs {
		if number == f.prs[i].Number {
			f.prs[i].RequestedReviewers = users
		}
	}
	return nil
}

func (f *fghc) GetPullRequest(org, repo string, number int) (*github.PullRequest, error) {
	f.Lock()
	defer f.Unlock()
	return f.pr, nil
}

func (f *fghc) GetPullRequestPatch(org, repo string, number int) ([]byte, error) {
	f.Lock()
	defer f.Unlock()
	return f.patch, nil
}

func (f *fghc) GetPullRequests(org, repo string) ([]github.PullRequest, error) {
	f.Lock()
	defer f.Unlock()
	return f.prs, nil
}

func (f *fghc) CreateComment(org, repo string, number int, comment string) error {
	f.Lock()
	defer f.Unlock()
	f.comments = append(f.comments, fmt.Sprintf(commentFormat, org, repo, number, comment))
	return nil
}

func (f *fghc) IsMember(org, user string) (bool, error) {
	f.Lock()
	defer f.Unlock()
	return f.isMember, nil
}

func (f *fghc) GetRepo(owner, name string) (github.FullRepo, error) {
	f.Lock()
	defer f.Unlock()
	return github.FullRepo{}, nil
}

func (f *fghc) EnsureFork(forkingUser, org, repo string) (string, error) {
	if repo == "changeme" {
		return "changed", nil
	}
	if repo == "error" {
		return repo, errors.New("errors")
	}
	return repo, nil
}

var prFmt = `title=%q body=%q head=%s base=%s labels=%v reviewers=%v assignees=%v`

func prToString(pr github.PullRequest) string {
	var labels []string
	for _, label := range pr.Labels {
		labels = append(labels, label.Name)
	}

	var reviewers []string
	for _, reviewer := range pr.RequestedReviewers {
		reviewers = append(reviewers, reviewer.Login)
	}

	var assignees []string
	for _, assignee := range pr.Assignees {
		assignees = append(assignees, assignee.Login)
	}
	return fmt.Sprintf(prFmt, pr.Title, pr.Body, pr.Head.Ref, pr.Base.Ref, labels, reviewers, assignees)
}

func (f *fghc) CreateIssue(org, repo, title, body string, milestone int, labels, assignees []string) (int, error) {
	f.Lock()
	defer f.Unlock()

	var ghLabels []github.Label
	var ghAssignees []github.User

	num := len(f.issues) + 1

	for _, label := range labels {
		ghLabels = append(ghLabels, github.Label{Name: label})
	}

	for _, assignee := range assignees {
		ghAssignees = append(ghAssignees, github.User{Login: assignee})
	}

	f.issues = append(f.issues, github.Issue{
		Title:     title,
		Body:      body,
		Number:    num,
		Labels:    ghLabels,
		Assignees: ghAssignees,
	})

	return num, nil
}

func (f *fghc) CreatePullRequest(org, repo, title, body, head, base string, canModify bool) (int, error) {
	f.Lock()
	defer f.Unlock()
	num := len(f.prs) + 1
	f.prs = append(f.prs, github.PullRequest{
		Title:  title,
		Body:   body,
		Number: num,
		Head:   github.PullRequestBranch{Ref: head},
		Base:   github.PullRequestBranch{Ref: base},
	})
	return num, nil
}

func (f *fghc) ListIssueComments(org, repo string, number int) ([]github.IssueComment, error) {
	f.Lock()
	defer f.Unlock()
	return f.prComments, nil
}

func (f *fghc) GetIssueLabels(org, repo string, number int) ([]github.Label, error) {
	f.Lock()
	defer f.Unlock()
	return f.prLabels, nil
}

func (f *fghc) ListOrgMembers(org, role string) ([]github.TeamMember, error) {
	f.Lock()
	defer f.Unlock()
	if role != "all" {
		return nil, fmt.Errorf("all is only supported role, not: %s", role)
	}
	return f.orgMembers, nil
}

func (f *fghc) CreateFork(org, repo string) (string, error) {
	return repo, nil
}

var initialFiles = map[string][]byte{
	"bar.go": []byte(`// Package bar does an interesting thing.
package bar

// Foo does a thing.
func Foo(wow int) int {
	return 42 + wow
}
`),
}

var patch = []byte(`From af468c9e69dfdf39db591f1e3e8de5b64b0e62a2 Mon Sep 17 00:00:00 2001
From: Wise Guy <wise@guy.com>
Date: Thu, 19 Oct 2017 15:14:36 +0200
Subject: [PATCH] Update magic number

---
 bar.go | 3 ++-
 1 file changed, 2 insertions(+), 1 deletion(-)

diff --git a/bar.go b/bar.go
index 1ea52dc..5bd70a9 100644
--- a/bar.go
+++ b/bar.go
@@ -3,5 +3,6 @@ package bar

 // Foo does a thing.
 func Foo(wow int) int {
-	return 42 + wow
+	// Needs to be 49 because of a reason.
+	return 49 + wow
 }
`)

var body = "This PR updates the magic number.\n\n"

func TestCherryPickIC(t *testing.T) {
	t.Parallel()
	testCherryPickIC(localgit.New, t)
}

func TestCherryPickICV2(t *testing.T) {
	t.Parallel()
	testCherryPickIC(localgit.NewV2, t)
}

func testCherryPickIC(clients localgit.Clients, t *testing.T) {
	lg, c, err := clients()
	if err != nil {
		t.Fatalf("Making localgit: %v", err)
	}
	defer func() {
		if err := lg.Clean(); err != nil {
			t.Errorf("Cleaning up localgit: %v", err)
		}
		if err := c.Clean(); err != nil {
			t.Errorf("Cleaning up client: %v", err)
		}
	}()
	if err := lg.MakeFakeRepo("foo", "bar"); err != nil {
		t.Fatalf("Making fake repo: %v", err)
	}
	if err := lg.AddCommit("foo", "bar", initialFiles); err != nil {
		t.Fatalf("Adding initial commit: %v", err)
	}

	expectedBranches := []string{"stage", "release-1.5"}
	for _, branch := range expectedBranches {
		if err := lg.CheckoutNewBranch("foo", "bar", branch); err != nil {
			t.Fatalf("Checking out pull branch: %v", err)
		}
	}

	ghc := &fghc{
		pr: &github.PullRequest{
			Base: github.PullRequestBranch{
				Ref: "master",
			},
			Number: 2,
			Merged: true,
			Title:  "This is a fix for X",
			Body:   body,
			RequestedReviewers: []github.User{
				{
					Login: "user1",
				},
			},
			Assignees: []github.User{
				{
					Login: "user2",
				},
			},
		},
		isMember: true,
		patch:    patch,
	}

	ic := github.IssueCommentEvent{
		Action: github.IssueCommentActionCreated,
		Repo: github.Repo{
			Owner: github.User{
				Login: "foo",
			},
			Name:     "bar",
			FullName: "foo/bar",
		},
		Issue: github.Issue{
			Number:      2,
			State:       "closed",
			PullRequest: &struct{}{},
		},
		Comment: github.IssueComment{
			User: github.User{
				Login: "wiseguy",
			},
			Body: "/cherrypick stage\r\n/cherrypick release-1.5\r\n/cherrypick master",
		},
	}

	botUser := &github.UserData{Login: "ci-robot", Email: "ci-robot@users.noreply.github.com"}
	var expectedFn = func(branch string) string {
		expectedTitle := "This is a fix for X (#2)"
		expectedBody := "This is an automated cherry-pick of #2\n\nThis PR updates the magic number.\n\n"
		expectedHead := fmt.Sprintf(botUser.Login+":"+cherryPickBranchFmt, 2, branch)
		expectedAssignees := []string{"wiseguy"}

		var expectedLabels []string
		for _, label := range ghc.pr.Labels {
			expectedLabels = append(expectedLabels, label.Name)
		}
		expectedLabels = append(expectedLabels, "type/cherrypick-for-"+branch)
		var expectedReviewers []string
		for _, reviewer := range ghc.pr.RequestedReviewers {
			expectedReviewers = append(expectedReviewers, reviewer.Login)
		}
		return fmt.Sprintf(prFmt, expectedTitle, expectedBody, expectedHead,
			branch, expectedLabels, expectedReviewers, expectedAssignees)
	}

	getSecret := func() []byte {
		return []byte("sha=abcdefg")
	}

	cfg := &externalplugins.Configuration{}
	cfg.TiCommunityCherrypicker = []externalplugins.TiCommunityCherrypicker{
		{
			Repos:             []string{"foo/bar"},
			LabelPrefix:       "cherrypick/",
			PickedLabelPrefix: "type/cherrypick-for-",
		},
	}
	ca := &externalplugins.ConfigAgent{}
	ca.Set(cfg)

	s := &Server{
		BotUser:        botUser,
		GitClient:      c,
		ConfigAgent:    ca,
		Push:           func(forkName, newBranch string, force bool) error { return nil },
		GitHubClient:   ghc,
		TokenGenerator: getSecret,
		Log:            logrus.StandardLogger().WithField("client", "cherrypicker"),
		Repos:          []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},
	}

	if err := s.handleIssueComment(logrus.NewEntry(logrus.StandardLogger()), ic); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ghc.prs) != len(expectedBranches) {
		t.Fatalf("Expected %d PRs, got %d", len(expectedBranches), len(ghc.prs))
	}

	expectedPrs := make(map[string]string)
	for _, branch := range expectedBranches {
		expectedPrs[expectedFn(branch)] = branch
	}

	seenBranches := make(map[string]bool)
	for _, p := range ghc.prs {
		pr := prToString(p)
		branch, present := expectedPrs[pr]
		if !present {
			t.Errorf("Unexpected PR:\n%s\nExpected to target one of the following branches: %v\n",
				pr, expectedBranches)
		} else {
			seenBranches[branch] = present
		}
	}
	if len(seenBranches) != len(expectedBranches) {
		t.Fatalf("Expected to see PRs for %d branches, got %d (%v)", len(expectedBranches), len(seenBranches), seenBranches)
	}
}

func TestCherryPickPR(t *testing.T) {
	t.Parallel()
	testCherryPickPR(localgit.New, t)
}

func TestCherryPickPRV2(t *testing.T) {
	t.Parallel()
	testCherryPickPR(localgit.NewV2, t)
}

func testCherryPickPR(clients localgit.Clients, t *testing.T) {
	lg, c, err := clients()
	if err != nil {
		t.Fatalf("Making localgit: %v", err)
	}
	defer func() {
		if err := lg.Clean(); err != nil {
			t.Errorf("Cleaning up localgit: %v", err)
		}
		if err := c.Clean(); err != nil {
			t.Errorf("Cleaning up client: %v", err)
		}
	}()
	if err := lg.MakeFakeRepo("foo", "bar"); err != nil {
		t.Fatalf("Making fake repo: %v", err)
	}
	if err := lg.AddCommit("foo", "bar", initialFiles); err != nil {
		t.Fatalf("Adding initial commit: %v", err)
	}
	expectedBranches := []string{"release-1.5", "release-1.6", "release-1.8"}
	for _, branch := range expectedBranches {
		if err := lg.CheckoutNewBranch("foo", "bar", branch); err != nil {
			t.Fatalf("Checking out pull branch: %v", err)
		}
	}
	if err := lg.CheckoutNewBranch("foo", "bar", "cherry-pick-2-to-release-1.5"); err != nil {
		t.Fatalf("Checking out existing PR branch: %v", err)
	}

	ghc := &fghc{
		orgMembers: []github.TeamMember{
			{
				Login: "approver",
			},
			{
				Login: "merge-bot",
			},
		},
		prComments: []github.IssueComment{
			{
				User: github.User{
					Login: "developer",
				},
				Body: "a review comment",
			},
			{
				User: github.User{
					Login: "approver",
				},
				Body: "/cherrypick release-1.5\r\n/cherrypick release-1.8",
			},
			{
				User: github.User{
					Login: "approver",
				},
				Body: "/cherrypick release-1.6",
			},
			{
				User: github.User{
					Login: "fan",
				},
				Body: "/cherrypick release-1.7",
			},
			{
				User: github.User{
					Login: "approver",
				},
				Body: "/approve",
			},
			{
				User: github.User{
					Login: "merge-bot",
				},
				Body: "Automatic merge from submit-queue.",
			},
		},
		prs: []github.PullRequest{
			{
				Title: "This is a fix for Y (#2)",
				Body:  "This is an automated cherry-pick of #2",
				Base: github.PullRequestBranch{
					Ref: "release-1.5",
				},
				Head: github.PullRequestBranch{
					Ref: "ci-robot:cherry-pick-2-to-release-1.5",
				},
				Labels: []github.Label{
					{
						Name: "test",
					},
				},
				RequestedReviewers: []github.User{
					{
						Login: "user1",
					},
				},
				Assignees: []github.User{
					{
						Login: "approver",
					},
				},
			},
		},
		isMember: true,
		patch:    patch,
	}

	pr := github.PullRequestEvent{
		Action: github.PullRequestActionClosed,
		PullRequest: github.PullRequest{
			Base: github.PullRequestBranch{
				Ref: "master",
				Repo: github.Repo{
					Owner: github.User{
						Login: "foo",
					},
					Name: "bar",
				},
			},
			Number:   2,
			Merged:   true,
			MergeSHA: new(string),
			Title:    "This is a fix for Y",
			Labels: []github.Label{
				{
					Name: "test",
				},
			},
			RequestedReviewers: []github.User{
				{
					Login: "user1",
				},
			},
			Assignees: []github.User{
				{
					Login: "approver",
				},
			},
		},
	}

	botUser := &github.UserData{Login: "ci-robot", Email: "ci-robot@users.noreply.github.com"}

	getSecret := func() []byte {
		return []byte("sha=abcdefg")
	}

	cfg := &externalplugins.Configuration{}
	cfg.TiCommunityCherrypicker = []externalplugins.TiCommunityCherrypicker{
		{
			Repos:       []string{"foo/bar"},
			LabelPrefix: "cherrypick/",
		},
	}
	ca := &externalplugins.ConfigAgent{}
	ca.Set(cfg)

	s := &Server{
		BotUser:        botUser,
		GitClient:      c,
		ConfigAgent:    ca,
		Push:           func(forkName, newBranch string, force bool) error { return nil },
		GitHubClient:   ghc,
		TokenGenerator: getSecret,
		Log:            logrus.StandardLogger().WithField("client", "cherrypicker"),
		Repos:          []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},
	}

	if err := s.handlePullRequest(logrus.NewEntry(logrus.StandardLogger()), pr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var expectedFn = func(branch string) string {
		expectedTitle := "This is a fix for Y (#2)"
		expectedBody := "This is an automated cherry-pick of #2"
		expectedHead := fmt.Sprintf(botUser.Login+":"+cherryPickBranchFmt, 2, branch)
		var expectedLabels []string
		for _, label := range pr.PullRequest.Labels {
			expectedLabels = append(expectedLabels, label.Name)
		}
		var expectedReviewers []string
		for _, reviewer := range pr.PullRequest.RequestedReviewers {
			expectedReviewers = append(expectedReviewers, reviewer.Login)
		}
		expectedAssignees := []string{"approver"}
		return fmt.Sprintf(prFmt, expectedTitle, expectedBody, expectedHead,
			branch, expectedLabels, expectedReviewers, expectedAssignees)
	}

	if len(ghc.prs) != len(expectedBranches) {
		t.Fatalf("Expected %d PRs, got %d", len(expectedBranches), len(ghc.prs))
	}

	expectedPrs := make(map[string]string)
	for _, branch := range expectedBranches {
		expectedPrs[expectedFn(branch)] = branch
	}
	seenBranches := make(map[string]bool)
	for _, p := range ghc.prs {
		pr := prToString(p)
		branch, present := expectedPrs[pr]
		if !present {
			t.Errorf("Unexpected PR:\n%s\nExpected to target one of the following branches: %v\n",
				pr, expectedBranches)
		} else {
			seenBranches[branch] = present
		}
	}
	if len(seenBranches) != len(expectedBranches) {
		t.Fatalf("Expected to see PRs for %d branches, got %d (%v)", len(expectedBranches), len(seenBranches), seenBranches)
	}
}

func TestCherryPickPRWithLabels(t *testing.T) {
	t.Parallel()
	testCherryPickPRWithLabels(localgit.New, t)
}

func TestCherryPickPRWithLabelsV2(t *testing.T) {
	t.Parallel()
	testCherryPickPRWithLabels(localgit.NewV2, t)
}

func testCherryPickPRWithLabels(clients localgit.Clients, t *testing.T) {
	lg, c, err := clients()
	if err != nil {
		t.Fatalf("Making localgit: %v", err)
	}
	defer func() {
		if err := lg.Clean(); err != nil {
			t.Errorf("Cleaning up localgit: %v", err)
		}
		if err := c.Clean(); err != nil {
			t.Errorf("Cleaning up client: %v", err)
		}
	}()
	if err := lg.MakeFakeRepo("foo", "bar"); err != nil {
		t.Fatalf("Making fake repo: %v", err)
	}
	if err := lg.AddCommit("foo", "bar", initialFiles); err != nil {
		t.Fatalf("Adding initial commit: %v", err)
	}

	expectedBranches := []string{"release-1.5", "release-1.6", "release-1.7"}
	for _, branch := range expectedBranches {
		if err := lg.CheckoutNewBranch("foo", "bar", branch); err != nil {
			t.Fatalf("Checking out pull branch: %v", err)
		}
	}

	pr := func(label github.Label) github.PullRequestEvent {
		return github.PullRequestEvent{
			Action: github.PullRequestActionLabeled,
			PullRequest: github.PullRequest{
				User: github.User{
					Login: "developer",
				},
				Base: github.PullRequestBranch{
					Ref: "master",
					Repo: github.Repo{
						Owner: github.User{
							Login: "foo",
						},
						Name: "bar",
					},
				},
				Number:   2,
				Merged:   true,
				MergeSHA: new(string),
				Title:    "This is a fix for Y",
				RequestedReviewers: []github.User{
					{
						Login: "user1",
					},
				},
			},
			Label: label,
		}
	}

	botUser := &github.UserData{Login: "ci-robot", Email: "ci-robot@users.noreply.github.com"}

	getSecret := func() []byte {
		return []byte("sha=abcdefg")
	}

	testCases := []struct {
		name         string
		labelPrefix  string
		prLabels     []github.Label
		prComments   []github.IssueComment
		shouldToggle bool
	}{
		{
			name:        "Default label prefix",
			labelPrefix: externalplugins.DefaultCherryPickLabelPrefix,
			prLabels: []github.Label{
				{
					Name: "cherrypick/release-1.5",
				},
				{
					Name: "cherrypick/release-1.6",
				},
				{
					Name: "cherrypick/release-1.7",
				},
			},
			shouldToggle: true,
		},
		{
			name:        "Custom label prefix",
			labelPrefix: "needs-cherry-pick-",
			prLabels: []github.Label{
				{
					Name: "needs-cherry-pick-release-1.5",
				},
				{
					Name: "needs-cherry-pick-release-1.6",
				},
				{
					Name: "needs-cherry-pick-release-1.7",
				},
			},
			shouldToggle: true,
		},
		{
			name:        "Random labels",
			labelPrefix: "needs-cherry-pick-",
			prLabels: []github.Label{
				{
					Name: "cherrypick/release-1.5",
				},
				{
					Name: "random-label",
				},
			},
			shouldToggle: false,
		},
	}

	for _, test := range testCases {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			for _, prLabel := range tc.prLabels {
				lb := prLabel
				t.Run(lb.Name, func(t *testing.T) {
					ghc := &fghc{
						orgMembers: []github.TeamMember{
							{
								Login: "approver",
							},
							{
								Login: "merge-bot",
							},
							{
								Login: "developer",
							},
						},
						prComments: []github.IssueComment{
							{
								User: github.User{
									Login: "developer",
								},
								Body: "a review comment",
							},
							{
								User: github.User{
									Login: "developer",
								},
								Body: "/cherrypick release-1.5\r",
							},
						},
						prLabels: tc.prLabels,
						isMember: true,
						patch:    patch,
					}

					cfg := &externalplugins.Configuration{}
					cfg.TiCommunityCherrypicker = []externalplugins.TiCommunityCherrypicker{
						{
							Repos:             []string{"foo/bar"},
							LabelPrefix:       tc.labelPrefix,
							PickedLabelPrefix: "type/cherrypick-for-",
						},
					}
					ca := &externalplugins.ConfigAgent{}
					ca.Set(cfg)

					s := &Server{
						BotUser:        botUser,
						GitClient:      c,
						ConfigAgent:    ca,
						Push:           func(forkName, newBranch string, force bool) error { return nil },
						GitHubClient:   ghc,
						TokenGenerator: getSecret,
						Log:            logrus.StandardLogger().WithField("client", "cherrypicker"),
						Repos:          []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},
					}

					if err := s.handlePullRequest(logrus.NewEntry(logrus.StandardLogger()), pr(lb)); err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					expectedFn := func(branch string) string {
						expectedTitle := "This is a fix for Y (#2)"
						expectedBody := "This is an automated cherry-pick of #2"
						expectedHead := fmt.Sprintf(botUser.Login+":"+cherryPickBranchFmt, 2, branch)
						var expectedLabels []string
						for _, label := range pr(lb).PullRequest.Labels {
							expectedLabels = append(expectedLabels, label.Name)
						}
						expectedLabels = append(expectedLabels, "type/cherrypick-for-"+branch)
						var expectedReviewers []string
						for _, reviewer := range pr(lb).PullRequest.RequestedReviewers {
							expectedReviewers = append(expectedReviewers, reviewer.Login)
						}
						expectedAssignees := []string{"developer"}
						return fmt.Sprintf(prFmt, expectedTitle, expectedBody, expectedHead,
							branch, expectedLabels, expectedReviewers, expectedAssignees)
					}

					if tc.shouldToggle {
						expectedPRs := 1
						if len(ghc.prs) != expectedPRs {
							t.Errorf("Expected %d PRs, got %d", expectedPRs, len(ghc.prs))
						}

						expectedPrs := make(map[string]string)
						for _, branch := range expectedBranches {
							expectedPrs[expectedFn(branch)] = branch
						}

						seenBranches := make(map[string]bool)
						for _, p := range ghc.prs {
							pr := prToString(p)
							branch, present := expectedPrs[pr]
							if !present {
								t.Errorf("Unexpected PR:\n%s\nExpected to target one of the following branches: %v\n",
									pr, expectedBranches)
							} else {
								seenBranches[branch] = present
							}
						}

						if len(seenBranches) != expectedPRs {
							t.Fatalf("Expected to see PRs for %d branches, got %d (%v)", expectedPRs, len(seenBranches), seenBranches)
						}
					} else if len(ghc.prs) > 0 {
						t.Error("PRs should not be created.")
					}
				})
			}
		})
	}
}

func TestCherryPickCreateIssue(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		org       string
		repo      string
		title     string
		body      string
		prNum     int
		labels    []string
		assignees []string
	}{
		{
			org:       "istio",
			repo:      "istio",
			title:     "brand new feature",
			body:      "automated cherry-pick",
			prNum:     2190,
			labels:    nil,
			assignees: []string{"clarketm"},
		},
		{
			org:       "kubernetes",
			repo:      "kubernetes",
			title:     "alpha feature",
			body:      "automated cherry-pick",
			prNum:     3444,
			labels:    []string{"new", "1.18"},
			assignees: nil,
		},
	}

	errMsg := func(field string) string {
		return fmt.Sprintf("GH issue %q does not match: \nexpected: \"%%v\" \nactual: \"%%v\"", field)
	}

	for _, tc := range testCases {
		ghc := &fghc{}

		s := &Server{
			GitHubClient: ghc,
		}

		if err := s.createIssue(logrus.WithField("test", t.Name()), tc.org, tc.repo, tc.title, tc.body, tc.prNum,
			nil, tc.labels, tc.assignees); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(ghc.issues) < 1 {
			t.Fatalf("Expected 1 GH issue to be created but got: %d", len(ghc.issues))
		}

		ghIssue := ghc.issues[len(ghc.issues)-1]

		if tc.title != ghIssue.Title {
			t.Fatalf(errMsg("title"), tc.title, ghIssue.Title)
		}

		if tc.body != ghIssue.Body {
			t.Fatalf(errMsg("body"), tc.title, ghIssue.Title)
		}

		if len(ghc.issues) != ghIssue.Number {
			t.Fatalf(errMsg("number"), len(ghc.issues), ghIssue.Number)
		}

		var actualAssignees []string
		for _, assignee := range ghIssue.Assignees {
			actualAssignees = append(actualAssignees, assignee.Login)
		}

		if !reflect.DeepEqual(tc.assignees, actualAssignees) {
			t.Fatalf(errMsg("assignees"), tc.assignees, actualAssignees)
		}

		var actualLabels []string
		for _, label := range ghIssue.Labels {
			actualLabels = append(actualLabels, label.Name)
		}

		if !reflect.DeepEqual(tc.labels, actualLabels) {
			t.Fatalf(errMsg("labels"), tc.labels, actualLabels)
		}

		cpFormat := fmt.Sprintf(commentFormat, tc.org, tc.repo, tc.prNum, "In response to a cherrypick label: %s")
		expectedComment := fmt.Sprintf(cpFormat, fmt.Sprintf("new issue created for failed cherrypick: #%d", ghIssue.Number))
		actualComment := ghc.comments[len(ghc.comments)-1]

		if expectedComment != actualComment {
			t.Fatalf(errMsg("comment"), expectedComment, actualComment)
		}
	}
}

func TestHandleLocks(t *testing.T) {
	t.Parallel()
	cfg := &externalplugins.Configuration{}
	cfg.TiCommunityCherrypicker = []externalplugins.TiCommunityCherrypicker{
		{
			Repos: []string{"org/repo"},
		},
	}
	ca := &externalplugins.ConfigAgent{}
	ca.Set(cfg)

	s := &Server{
		ConfigAgent:  ca,
		GitHubClient: &threadUnsafeFGHC{fghc: &fghc{}},
		BotUser:      &github.UserData{},
	}

	routine1Done := make(chan struct{})
	routine2Done := make(chan struct{})

	l := logrus.WithField("test", t.Name())
	pr := &github.PullRequest{
		Title:  "title",
		Body:   "body",
		Number: 0,
	}

	go func() {
		defer close(routine1Done)
		if err := s.handle(l, "", &github.IssueComment{}, "org", "repo", "targetBranch", pr); err != nil {
			t.Errorf("routine failed: %v", err)
		}
	}()
	go func() {
		defer close(routine2Done)
		if err := s.handle(l, "", &github.IssueComment{}, "org", "repo", "targetBranch", pr); err != nil {
			t.Errorf("routine failed: %v", err)
		}
	}()

	<-routine1Done
	<-routine2Done

	if actual := s.GitHubClient.(*threadUnsafeFGHC).orgRepoCountCalled; actual != 2 {
		t.Errorf("expected two EnsureFork calls, got %d", actual)
	}
}

func TestEnsureForkExists(t *testing.T) {
	botUser := &github.UserData{Login: "ci-robot", Email: "ci-robot@users.noreply.github.com"}

	ghc := &fghc{}

	cfg := &externalplugins.Configuration{}
	cfg.TiCommunityCherrypicker = []externalplugins.TiCommunityCherrypicker{
		{
			Repos: []string{"org/repo"},
		},
	}
	ca := &externalplugins.ConfigAgent{}
	ca.Set(cfg)

	s := &Server{
		BotUser:      botUser,
		ConfigAgent:  ca,
		GitHubClient: ghc,
		Repos:        []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},
	}

	testCases := []struct {
		name     string
		org      string
		repo     string
		expected string
		errors   bool
	}{
		{
			name:     "Repo name does not change after ensured",
			org:      "whatever",
			repo:     "repo",
			expected: "repo",
			errors:   false,
		},
		{
			name:     "EnsureFork changes repo name",
			org:      "whatever",
			repo:     "changeme",
			expected: "changed",
			errors:   false,
		},
		{
			name:     "EnsureFork errors",
			org:      "whatever",
			repo:     "error",
			expected: "error",
			errors:   true,
		},
	}

	for _, test := range testCases {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			res, err := s.ensureForkExists(tc.org, tc.repo)
			if tc.errors && err == nil {
				t.Errorf("expected error, but did not get one")
			}
			if !tc.errors && err != nil {
				t.Errorf("expected no error, but got one")
			}
			if res != tc.expected {
				t.Errorf("expected %s but got %s", tc.expected, res)
			}
		})
	}
}

type threadUnsafeFGHC struct {
	*fghc
	orgRepoCountCalled int
}

func (tuf *threadUnsafeFGHC) EnsureFork(login, org, repo string) (string, error) {
	tuf.orgRepoCountCalled++
	return "", errors.New("that is enough")
}

func TestServeHTTPErrors(t *testing.T) {
	pa := &plugins.ConfigAgent{}
	pa.Set(&plugins.Configuration{})

	getSecret := func() []byte {
		var repoLevelSec = `
'*':
  - value: abc
    created_at: 2019-10-02T15:00:00Z
  - value: key2
    created_at: 2020-10-02T15:00:00Z
foo/bar:
  - value: 123abc
    created_at: 2019-10-02T15:00:00Z
  - value: key6
    created_at: 2020-10-02T15:00:00Z
`
		return []byte(repoLevelSec)
	}

	// This is the SHA1 signature for payload "{}" and signature "abc"
	// echo -n '{}' | openssl dgst -sha1 -hmac abc
	const hmac string = "sha1=db5c76f4264d0ad96cf21baec394964b4b8ce580"
	const body string = "{}"
	var testcases = []struct {
		name string

		Method string
		Header map[string]string
		Body   string
		Code   int
	}{
		{
			name: "Delete",

			Method: http.MethodDelete,
			Header: map[string]string{
				"X-GitHub-Event":    "ping",
				"X-GitHub-Delivery": "I am unique",
				"X-Hub-Signature":   hmac,
				"content-type":      "application/json",
			},
			Body: body,
			Code: http.StatusMethodNotAllowed,
		},
		{
			name: "No event",

			Method: http.MethodPost,
			Header: map[string]string{
				"X-GitHub-Delivery": "I am unique",
				"X-Hub-Signature":   hmac,
				"content-type":      "application/json",
			},
			Body: body,
			Code: http.StatusBadRequest,
		},
		{
			name: "No content type",

			Method: http.MethodPost,
			Header: map[string]string{
				"X-GitHub-Event":    "ping",
				"X-GitHub-Delivery": "I am unique",
				"X-Hub-Signature":   hmac,
			},
			Body: body,
			Code: http.StatusBadRequest,
		},
		{
			name: "No event guid",

			Method: http.MethodPost,
			Header: map[string]string{
				"X-GitHub-Event":  "ping",
				"X-Hub-Signature": hmac,
				"content-type":    "application/json",
			},
			Body: body,
			Code: http.StatusBadRequest,
		},
		{
			name: "No signature",

			Method: http.MethodPost,
			Header: map[string]string{
				"X-GitHub-Event":    "ping",
				"X-GitHub-Delivery": "I am unique",
				"content-type":      "application/json",
			},
			Body: body,
			Code: http.StatusForbidden,
		},
		{
			name: "Bad signature",

			Method: http.MethodPost,
			Header: map[string]string{
				"X-GitHub-Event":    "ping",
				"X-GitHub-Delivery": "I am unique",
				"X-Hub-Signature":   "this doesn't work",
				"content-type":      "application/json",
			},
			Body: body,
			Code: http.StatusForbidden,
		},
		{
			name: "Good",

			Method: http.MethodPost,
			Header: map[string]string{
				"X-GitHub-Event":    "ping",
				"X-GitHub-Delivery": "I am unique",
				"X-Hub-Signature":   hmac,
				"content-type":      "application/json",
			},
			Body: body,
			Code: http.StatusOK,
		},
		{
			name: "Good, again",

			Method: http.MethodGet,
			Header: map[string]string{
				"content-type": "application/json",
			},
			Body: body,
			Code: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range testcases {
		t.Logf("Running scenario %q", tc.name)

		w := httptest.NewRecorder()
		r, err := http.NewRequest(tc.Method, "", strings.NewReader(tc.Body))
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range tc.Header {
			r.Header.Set(k, v)
		}

		s := Server{
			TokenGenerator: getSecret,
		}

		s.ServeHTTP(w, r)
		if w.Code != tc.Code {
			t.Errorf("For test case: %+v\nExpected code %v, got code %v", tc, tc.Code, w.Code)
		}
	}
}

func TestServeHTTP(t *testing.T) {
	pa := &plugins.ConfigAgent{}
	pa.Set(&plugins.Configuration{})

	getSecret := func() []byte {
		var repoLevelSec = `
'*':
  - value: abc
    created_at: 2019-10-02T15:00:00Z
  - value: key2
    created_at: 2020-10-02T15:00:00Z
foo/bar:
  - value: 123abc
    created_at: 2019-10-02T15:00:00Z
  - value: key6
    created_at: 2020-10-02T15:00:00Z
`
		return []byte(repoLevelSec)
	}

	lgtmComment, err := ioutil.ReadFile("../../../../test/testdata/lgtm_comment.json")
	if err != nil {
		t.Fatalf("read lgtm comment file failed: %v", err)
	}

	openedPR, err := ioutil.ReadFile("../../../../test/testdata/opened_pr.json")
	if err != nil {
		t.Fatalf("read opened PR file failed: %v", err)
	}

	// This is the SHA1 signature for payload "{}" and signature "abc"
	// echo -n '{}' | openssl dgst -sha1 -hmac abc
	var testcases = []struct {
		name string

		Method string
		Header map[string]string
		Body   string
		Code   int
	}{
		{
			name: "Issue comment event",

			Method: http.MethodPost,
			Header: map[string]string{
				"X-GitHub-Event":    "issue_comment",
				"X-GitHub-Delivery": "I am unique",
				"X-Hub-Signature":   "sha1=f3fee26b22d3748f393f7e37f71baa467495971a",
				"content-type":      "application/json",
			},
			Body: string(lgtmComment),
			Code: http.StatusOK,
		},
		{
			name: "Pull request event",

			Method: http.MethodPost,
			Header: map[string]string{
				"X-GitHub-Event":    "pull_request",
				"X-GitHub-Delivery": "I am unique",
				"X-Hub-Signature":   "sha1=9a62c443a5ab561e023e64610dc467523188defc",
				"content-type":      "application/json",
			},
			Body: string(openedPR),
			Code: http.StatusOK,
		},
	}

	for _, tc := range testcases {
		t.Logf("Running scenario %q", tc.name)

		w := httptest.NewRecorder()
		r, err := http.NewRequest(tc.Method, "", strings.NewReader(tc.Body))
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range tc.Header {
			r.Header.Set(k, v)
		}

		cfg := &externalplugins.Configuration{}
		cfg.TiCommunityCherrypicker = []externalplugins.TiCommunityCherrypicker{
			{
				Repos: []string{"foo/bar"},
			},
		}
		ca := &externalplugins.ConfigAgent{}
		ca.Set(cfg)

		s := Server{
			TokenGenerator: getSecret,
			ConfigAgent:    ca,
		}

		s.ServeHTTP(w, r)
		if w.Code != tc.Code {
			t.Errorf("For test case: %+v\nExpected code %v, got code %v", tc, tc.Code, w.Code)
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
			name: "Empty config",
			config: &externalplugins.Configuration{
				TiCommunityCherrypicker: []externalplugins.TiCommunityCherrypicker{
					{
						Repos: []string{"org2/repo"},
					},
				},
			},
			enabledRepos: enabledRepos,
			configInfoIncludes: []string{"For this repository, only organization members are allowed to do cherry-pick.",
				"When a cherry-pick PR conflicts, cherrypicker will create the PR with conflicts."},
			configInfoExcludes: []string{"The current label prefix for cherrypicker is: ",
				"The current picked label prefix for cherrypicker is: ",
				"For this repository, cherry-pick is available to all.",
				"When a cherry-pick PR conflicts, an issue will be created to track it."},
		},
		{
			name: "All configs enabled",
			config: &externalplugins.Configuration{
				TiCommunityCherrypicker: []externalplugins.TiCommunityCherrypicker{
					{
						Repos:             []string{"org2/repo"},
						LabelPrefix:       "cherrypick/",
						PickedLabelPrefix: "type/cherrypick-for-",
						AllowAll:          true,
						IssueOnConflict:   true,
					},
				},
			},
			enabledRepos: enabledRepos,
			configInfoIncludes: []string{"The current label prefix for cherrypicker is: ",
				"The current picked label prefix for cherrypicker is: ",
				"For this repository, cherry-pick is available to all.",
				"When a cherry-pick PR conflicts, an issue will be created to track it."},
			configInfoExcludes: []string{"For this repository, only organization members are allowed to do cherry-pick.",
				"When a cherry-pick PR conflicts, cherrypicker will create the PR with conflicts."},
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
