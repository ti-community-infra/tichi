package cherrypicker

import (
	"testing"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/git/localgit"
	"k8s.io/test-infra/prow/github"

	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
)

func TestInviteIC(t *testing.T) {
	lg, c, err := localgit.NewV2()
	if err != nil {
		t.Fatalf("Making localgit: %v", err)
	}
	t.Cleanup(func() {
		if err := lg.Clean(); err != nil {
			t.Errorf("Cleaning up localgit: %v", err)
		}
		if err := c.Clean(); err != nil {
			t.Errorf("Cleaning up client: %v", err)
		}
	})
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

	tests := []struct {
		name          string
		members       []string
		isMember      bool
		collaborators []string
		commentUser   string
		commentBody   string
		wantError     bool
		wantHad       bool
	}{
		{
			name:          "invite using invalid comment",
			collaborators: []string{},
			commentUser:   "wiseguy",
			commentBody:   "/cherry-pick-wrong-invite",
			isMember:      false,
			wantError:     false,
			wantHad:       false,
		},
		{
			name:          "invite when he is a collaborator",
			collaborators: []string{"wiseguy"},
			commentUser:   "wiseguy",
			commentBody:   "/cherry-pick-invite",
			isMember:      false,
			wantError:     false,
			wantHad:       true,
		},
		{
			name:          "invite when he is a collaborator, but comment failed",
			collaborators: []string{patternErrWhenCreateComment},
			commentUser:   patternErrWhenCreateComment,
			commentBody:   "/cherry-pick-invite",
			isMember:      false,
			wantError:     true,
			wantHad:       true,
		},
		{
			name:        "invite when he is not a collaborator, but a org member",
			commentUser: "wiseguy",
			commentBody: "/cherry-pick-invite",
			wantError:   false,
			isMember:    true,
			wantHad:     true,
		},
		{
			name:        "invite when he is not a collaborator, but a org member, notify comment failed",
			commentUser: patternErrWhenCreateComment,
			commentBody: "/cherry-pick-invite",
			wantError:   true,
			isMember:    true,
			wantHad:     true,
		},
		{
			name:        "invite when he is not a collaborator, error when judge member",
			commentUser: patternErrWhenJudgeMember,
			commentBody: "/cherry-pick-invite",
			wantError:   true,
			isMember:    false,
			wantHad:     false,
		},
		{
			name:        "invite when he is not a member of our ORG using cherry-pick-invite command",
			isMember:    false,
			commentUser: "wiseguy",
			commentBody: "/cherry-pick-invite",
			wantError:   false,
			wantHad:     false,
		},
		{
			name:        "invite when he is not a member of our ORG using cherrypick-invite command",
			isMember:    false,
			commentUser: "wiseguy",
			commentBody: "/cherrypick-invite",
			wantError:   false,
			wantHad:     false,
		},
		{
			name:        "invite when he is not a member of our ORG, error when comment",
			isMember:    false,
			commentUser: patternErrWhenCreateComment,
			commentBody: "/cherry-pick-invite",
			wantError:   true,
			wantHad:     false,
		},
		{
			name:          "invite error when judge",
			collaborators: nil,
			commentUser:   patternErrWhenIsCollaborator,
			commentBody:   "/cherry-pick-invite",
			wantError:     true,
			wantHad:       false,
		},
		{
			name:          "invite error when add",
			collaborators: nil,
			isMember:      true,
			commentUser:   patternErrWhenAddCollaborator,
			commentBody:   "/cherry-pick-invite",
			wantError:     true,
			wantHad:       false,
		},
		{
			name:          "invite error when add, and comment failed",
			collaborators: nil,
			isMember:      true,
			commentUser:   patternErrWhenAddCollaborator + patternErrWhenCreateComment,
			commentBody:   "/cherry-pick-invite",
			wantError:     true,
			wantHad:       false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ghc := &fghc{
				pr: &github.PullRequest{
					Base:      github.PullRequestBranch{Ref: "master"},
					Number:    2,
					Merged:    true,
					Title:     "This is a fix for X",
					Body:      body,
					Assignees: []github.User{{Login: "user2"}},
				},
				isMember:      tc.isMember,
				members:       tc.members,
				patch:         patch,
				collaborators: tc.collaborators,
			}
			ic := github.IssueCommentEvent{
				Action: github.IssueCommentActionCreated,
				Repo: github.Repo{
					Owner:    github.User{Login: "foo"},
					Name:     "bar",
					FullName: "foo/bar",
				},
				Issue: github.Issue{Number: 2, State: "closed", PullRequest: &struct{}{}},
				Comment: github.IssueComment{
					User: github.User{Login: tc.commentUser},
					Body: tc.commentBody,
				},
			}
			s := &Server{
				BotUser:                botUser,
				GitClient:              c,
				ConfigAgent:            ca,
				Push:                   func(_, _ string, _ bool) error { return nil },
				GitHubClient:           ghc,
				WebhookSecretGenerator: getSecret,
				GitHubTokenGenerator:   getGithubToken,
				Log:                    logrus.StandardLogger().WithField("client", "cherrypicker"),
				Repos:                  []github.Repo{{Fork: true, FullName: "ci-robot/bar"}},
			}

			if err := s.handleIssueComment(logrus.NewEntry(logrus.StandardLogger()),
				&ic); (err != nil) != tc.wantError {
				t.Fatalf("got error: %v, expected error: %v", err, tc.wantError)
			}

			hasErrTpl := "Expected collaborators contains %s, got %v"
			if !tc.wantHad {
				hasErrTpl = "Expected collaborators not contains %s, got %v"
			}
			if sets.NewString(ghc.collaborators...).Has(ic.Comment.User.Login) != tc.wantHad {
				t.Fatalf(hasErrTpl, ic.Comment.User.Login, ghc.collaborators)
			}
		})
	}
}
