package cherrypicker

import (
	"net/http"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/git/localgit"
	git "k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"

	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
)

func TestInviteIC(t *testing.T) {
	lg, c, err := localgit.NewV2()
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
			Body: "/cherry-pick-invite",
		},
	}

	botUser := &github.UserData{Login: "ci-robot", Email: "ci-robot@users.noreply.github.com"}
	getSecret := func() []byte {
		return []byte("sha=abcdefg")
	}

	getGithubToken := func() []byte {
		return []byte("token")
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
		BotUser:                botUser,
		GitClient:              c,
		ConfigAgent:            ca,
		Push:                   func(forkName, newBranch string, force bool) error { return nil },
		GitHubClient:           ghc,
		WebhookSecretGenerator: getSecret,
		GitHubTokenGenerator:   getGithubToken,
		Log:                    logrus.StandardLogger().WithField("client", "cherrypicker"),
		Repos:                  []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},
	}

	if err := s.handleIssueComment(logrus.NewEntry(logrus.StandardLogger()), ic); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !sets.NewString(ghc.collaborators...).Has(ic.Comment.User.Login) {
		t.Fatalf("Expected collaborators has %s, got %v", ic.Comment.User.Login, ghc.collaborators)
	}
}

func TestServer_handleIssueComment(t *testing.T) {
	type fields struct {
		WebhookSecretGenerator func() []byte
		GitHubTokenGenerator   func() []byte
		BotUser                *github.UserData
		Email                  string
		GitClient              git.ClientFactory
		Push                   func(forkName, newBranch string, force bool) error
		GitHubClient           GithubClient
		Log                    *logrus.Entry
		ConfigAgent            *tiexternalplugins.ConfigAgent
		Bare                   *http.Client
		PatchURL               string
		GitHubURL              string
		repoLock               sync.Mutex
		Repos                  []github.Repo
		mapLock                sync.Mutex
		lockMap                map[cherryPickRequest]*sync.Mutex
	}
	type args struct {
		l  *logrus.Entry
		ic github.IssueCommentEvent
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				WebhookSecretGenerator: tt.fields.WebhookSecretGenerator,
				GitHubTokenGenerator:   tt.fields.GitHubTokenGenerator,
				BotUser:                tt.fields.BotUser,
				Email:                  tt.fields.Email,
				GitClient:              tt.fields.GitClient,
				Push:                   tt.fields.Push,
				GitHubClient:           tt.fields.GitHubClient,
				Log:                    tt.fields.Log,
				ConfigAgent:            tt.fields.ConfigAgent,
				Bare:                   tt.fields.Bare,
				PatchURL:               tt.fields.PatchURL,
				GitHubURL:              tt.fields.GitHubURL,
				repoLock:               tt.fields.repoLock,
				Repos:                  tt.fields.Repos,
				mapLock:                tt.fields.mapLock,
				lockMap:                tt.fields.lockMap,
			}
			if err := s.handleIssueComment(tt.args.l, tt.args.ic); (err != nil) != tt.wantErr {
				t.Errorf("Server.handleIssueComment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServer_inviteCollaborator(t *testing.T) {
	type fields struct {
		WebhookSecretGenerator func() []byte
		GitHubTokenGenerator   func() []byte
		BotUser                *github.UserData
		Email                  string
		GitClient              git.ClientFactory
		Push                   func(forkName, newBranch string, force bool) error
		GitHubClient           GithubClient
		Log                    *logrus.Entry
		ConfigAgent            *externalplugins.ConfigAgent
		Bare                   *http.Client
		PatchURL               string
		GitHubURL              string
		repoLock               sync.Mutex
		Repos                  []github.Repo
		mapLock                sync.Mutex
		lockMap                map[cherryPickRequest]*sync.Mutex
	}
	type args struct {
		org      string
		repo     string
		username string
		prNum    int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				WebhookSecretGenerator: tt.fields.WebhookSecretGenerator,
				GitHubTokenGenerator:   tt.fields.GitHubTokenGenerator,
				BotUser:                tt.fields.BotUser,
				Email:                  tt.fields.Email,
				GitClient:              tt.fields.GitClient,
				Push:                   tt.fields.Push,
				GitHubClient:           tt.fields.GitHubClient,
				Log:                    tt.fields.Log,
				ConfigAgent:            tt.fields.ConfigAgent,
				Bare:                   tt.fields.Bare,
				PatchURL:               tt.fields.PatchURL,
				GitHubURL:              tt.fields.GitHubURL,
				repoLock:               tt.fields.repoLock,
				Repos:                  tt.fields.Repos,
				mapLock:                tt.fields.mapLock,
				lockMap:                tt.fields.lockMap,
			}
			if err := s.inviteCollaborator(tt.args.org, tt.args.repo, tt.args.username, tt.args.prNum); (err != nil) != tt.wantErr {
				t.Errorf("Server.inviteIfNotCollaborator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
