package guard

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
)

//go:generate mockgen -package=guard -source=types.go --destination=types_mock.go

var (
	prOrg         = "test-org"
	prRepo        = "test-repo"
	prNum         = 123
	fullRepo      = strings.Join([]string{prOrg, prRepo}, "/")
	changeFileSHA = `bbcd538c8e72b8c175046e27cc8f907076331401`

	pluginCfg = tiexternalplugins.TiCommunityGuard{
		Repos:    []string{fullRepo},
		Patterns: []string{`^config/a\.conf$`, `^config/b.*\.conf`},
		Label: tiexternalplugins.TiCommunityGuardLabel{
			Unapproved: "hold/need-approve",
			Approved:   "status/approved",
		},
		Approvers: []string{"approver-a"},
	}
	pluginCfgs = []tiexternalplugins.TiCommunityGuard{pluginCfg}

	basePullRequestEvent = github.PullRequestEvent{
		Number:      prNum,
		PullRequest: github.PullRequest{Number: prNum},
		Repo: github.Repo{
			Owner:    github.User{Login: prOrg},
			Name:     prRepo,
			FullName: fullRepo,
		},
	}

	baseReviewEvent = github.ReviewEvent{
		Action:      github.ReviewActionSubmitted,
		PullRequest: github.PullRequest{Number: prNum},
		Repo: github.Repo{
			Owner:    github.User{Login: prOrg},
			Name:     prRepo,
			FullName: fullRepo,
		},
		Review: github.Review{
			User:        github.User{Login: pluginCfg.Approvers[0]},
			State:       github.ReviewStateApproved,
			SubmittedAt: time.Now(),
		},
	}
)

func TestHandlePullRequestEvent(t *testing.T) {
	type args struct {
		gc    githubClient
		event *github.PullRequestEvent
		cfg   *tiexternalplugins.Configuration
		log   *logrus.Entry
	}

	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "PR is closed",
			args: args{
				gc: nil,
				event: &github.PullRequestEvent{
					Action: github.PullRequestActionSynchronize,
					Number: prNum,
					PullRequest: github.PullRequest{
						State: github.PullRequestStateClosed,
					},
				},
				cfg: &tiexternalplugins.Configuration{},
				log: &logrus.Entry{},
			},
			wantErr: false,
		},
		{
			name: "PR is in draft state",
			args: args{
				gc: nil,
				event: &github.PullRequestEvent{
					Action: github.PullRequestActionSynchronize,
					Number: prNum,
					PullRequest: github.PullRequest{
						State: github.PullRequestStateOpen,
						Draft: true,
					},
				},
				cfg: &tiexternalplugins.Configuration{},
				log: &logrus.Entry{},
			},
			wantErr: false,
		},
		{
			name: "PR not mergeable",
			args: args{
				gc: nil,
				event: &github.PullRequestEvent{
					Action: github.PullRequestActionSynchronize,
					Number: prNum,
					PullRequest: github.PullRequest{
						State:    github.PullRequestStateOpen,
						Mergable: func(ret bool) *bool { return &ret }(false),
					},
				},
				cfg: &tiexternalplugins.Configuration{},
				log: &logrus.Entry{},
			},
			wantErr: false,
		},
		{
			name: "PR opened event",
			args: args{
				gc: func() githubClient {
					mc := NewMockgithubClient(mockCtrl)
					getChangesCall := mc.EXPECT().
						GetPullRequestChanges(prOrg, prRepo, prNum).
						Return([]github.PullRequestChange{
							{
								SHA:       changeFileSHA,
								Filename:  "config/a.conf",
								Status:    string(github.PullRequestFileModified),
								Additions: 10,
								Deletions: 0,
								Changes:   0,
							},
						}, nil)

					mc.EXPECT().
						AddLabel(prOrg, prRepo, prNum, pluginCfg.Label.Unapproved).
						Return(nil).
						After(getChangesCall)

					return mc
				}(),
				event: func(e github.PullRequestEvent) *github.PullRequestEvent {
					e.Action = github.PullRequestActionOpened
					return &e
				}(basePullRequestEvent),
				cfg: &tiexternalplugins.Configuration{TiCommunityGuard: pluginCfgs},
				log: logrus.New().WithField("test", true),
			},
			wantErr: false,
		},
		{
			name: "PR ready for review from draft state",
			args: args{
				gc: func() githubClient {
					mc := NewMockgithubClient(mockCtrl)
					getChangesCall := mc.EXPECT().
						GetPullRequestChanges(prOrg, prRepo, prNum).
						Return([]github.PullRequestChange{
							{
								SHA:       changeFileSHA,
								Filename:  "config/a.conf",
								Status:    string(github.PullRequestFileModified),
								Additions: 10,
								Deletions: 0,
								Changes:   0,
							},
						}, nil)

					mc.EXPECT().
						AddLabel(prOrg, prRepo, prNum, pluginCfg.Label.Unapproved).
						Return(nil).
						After(getChangesCall)

					return mc
				}(),
				event: func(e github.PullRequestEvent) *github.PullRequestEvent {
					e.Action = github.PullRequestActionReadyForReview
					return &e
				}(basePullRequestEvent),
				cfg: &tiexternalplugins.Configuration{TiCommunityGuard: pluginCfgs},
				log: logrus.New().WithField("test", true),
			},
			wantErr: false,
		},
		{
			name: "PR reopend event",
			args: args{
				gc: func() githubClient {
					mc := NewMockgithubClient(mockCtrl)
					mc.EXPECT().
						GetPullRequestChanges(prOrg, prRepo, prNum).
						Return([]github.PullRequestChange{
							{
								SHA:       changeFileSHA,
								Filename:  "config/a.conf",
								Status:    string(github.PullRequestFileModified),
								Additions: 10,
								Deletions: 0,
								Changes:   0,
							},
						}, nil)

					return mc
				}(),
				event: func(e github.PullRequestEvent) *github.PullRequestEvent {
					e.Action = github.PullRequestActionReopened

					e.PullRequest.Labels = append(e.PullRequest.Labels, github.Label{
						Name: pluginCfg.Label.Unapproved,
					})
					return &e
				}(basePullRequestEvent),
				cfg: &tiexternalplugins.Configuration{TiCommunityGuard: pluginCfgs},
				log: logrus.New().WithField("test", true),
			},
			wantErr: false,
		},
		{
			name: "PR update with new commit and make change files not matched",
			args: args{
				gc: func() githubClient {
					mc := NewMockgithubClient(mockCtrl)
					getChangesCall := mc.EXPECT().
						GetPullRequestChanges(prOrg, prRepo, prNum).
						Return([]github.PullRequestChange{
							{
								SHA:       changeFileSHA,
								Filename:  "other-dir/a.conf",
								Status:    string(github.PullRequestFileModified),
								Additions: 10,
								Deletions: 10,
								Changes:   10,
							},
						}, nil)

					mc.EXPECT().
						RemoveLabel(prOrg, prRepo, prNum, pluginCfg.Label.Unapproved).
						Return(nil).
						After(getChangesCall)

					return mc
				}(),
				event: func(e github.PullRequestEvent) *github.PullRequestEvent {
					e.Action = github.PullRequestActionSynchronize

					e.PullRequest.Labels = append(e.PullRequest.Labels, github.Label{
						Name: pluginCfg.Label.Unapproved,
					})
					return &e
				}(basePullRequestEvent),
				cfg: &tiexternalplugins.Configuration{TiCommunityGuard: pluginCfgs},
				log: logrus.New().WithField("test", true),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := HandlePullRequestEvent(tt.args.gc, tt.args.event, tt.args.cfg, tt.args.log); (err != nil) != tt.wantErr {
				t.Errorf("HandlePullRequestEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlePullRequestReviewEvent(t *testing.T) {
	type args struct {
		gc    githubClient
		event *github.ReviewEvent
		cfg   *tiexternalplugins.Configuration
		log   *logrus.Entry
	}

	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)

	log := logrus.New().WithField("test", true)
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "guard approved",
			args: args{
				gc: func() githubClient {
					mc := NewMockgithubClient(mockCtrl)

					addLabelCall := mc.EXPECT().
						AddLabel(prOrg, prRepo, prNum, pluginCfg.Label.Approved).
						Return(nil)
					mc.EXPECT().
						RemoveLabel(prOrg, prRepo, prNum, pluginCfg.Label.Unapproved).
						Return(nil).
						After(addLabelCall)

					return mc
				}(),
				event: func(e github.ReviewEvent) *github.ReviewEvent {
					e.Action = github.ReviewActionSubmitted
					e.Review.State = github.ReviewStateApproved
					e.PullRequest = github.PullRequest{
						Number: prNum,
						Labels: []github.Label{{
							Name: pluginCfg.Label.Unapproved,
						}},
					}

					return &e
				}(baseReviewEvent),
				cfg: &tiexternalplugins.Configuration{TiCommunityGuard: pluginCfgs},
				log: log,
			},
			wantErr: false,
		},
		{
			name: "approved by person who is not a guard",
			args: args{
				gc: nil,
				event: func(e github.ReviewEvent) *github.ReviewEvent {
					e.Action = github.ReviewActionSubmitted
					e.Review.State = github.ReviewStateApproved
					e.Review.User.Login = "other-person"
					e.PullRequest = github.PullRequest{
						Number: prNum,
						Labels: []github.Label{{
							Name: pluginCfg.Label.Unapproved,
						}},
					}

					return &e
				}(baseReviewEvent),
				cfg: &tiexternalplugins.Configuration{TiCommunityGuard: pluginCfgs},
				log: log,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HandlePullRequestReviewEvent(tt.args.gc, tt.args.event, tt.args.cfg, tt.args.log)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandlePullRequestReviewEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getPullRequestChangeFilenames(t *testing.T) {
	type args struct {
		gc    githubClient
		org   string
		repo  string
		prNum int
	}

	prOrg := "test-org"
	prRepo := "test-repo"
	prNum := 123

	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "contain add/delete/update/rename",
			args: args{
				gc: func() githubClient {
					mc := NewMockgithubClient(mockCtrl)
					mc.EXPECT().
						GetPullRequestChanges(prOrg, prRepo, prNum).
						Return([]github.PullRequestChange{
							{
								SHA:       "bbcd538c8e72b8c175046e27cc8f907076331401",
								Filename:  "add.txt",
								Status:    github.PullRequestFileAdded,
								Additions: 10,
								Deletions: 0,
								Changes:   0,
							}, // add
							{
								SHA:       "bbcd538c8e72b8c175046e27cc8f907076331402",
								Filename:  "del.txt",
								Status:    github.PullRequestFileRemoved,
								Additions: 0,
								Deletions: 10,
								Changes:   0,
							}, // delete
							{
								SHA:       "bbcd538c8e72b8c175046e27cc8f907076331403",
								Filename:  "update.txt",
								Status:    string(github.PullRequestFileModified),
								Additions: 10,
								Deletions: 10,
								Changes:   10,
							}, // update
							{
								SHA:              "bbcd538c8e72b8c175046e27cc8f907076331404",
								PreviousFilename: "renamed-old.txt",
								Filename:         "renamed-new.txt",
								Status:           string(github.PullRequestFileRenamed),
								Additions:        0,
								Deletions:        0,
								Changes:          2,
							}, // rename
						}, nil)
					return mc
				}(),
				org:   prOrg,
				repo:  prRepo,
				prNum: prNum,
			},
			want:    []string{"add.txt", "del.txt", "update.txt", "renamed-new.txt", "renamed-old.txt"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPullRequestChangeFilenames(tt.args.gc, tt.args.org, tt.args.repo, tt.args.prNum)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPullRequestChangeFilenames() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPullRequestChangeFilenames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_matchFiles(t *testing.T) {
	type args struct {
		files    []string
		patterns []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "matched",
			args: args{
				files:    []string{`a1.txt`, `a2.txt`, `b1.txt`},
				patterns: []string{`^a\d+\.txt`},
			},
			want: []string{`a1.txt`, `a2.txt`},
		},
		{
			name: "none matched",
			args: args{
				files:    []string{`a1.txt`, `a2.txt`, `b1.txt`},
				patterns: []string{`^ab\d+\.txt`},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchFiles(tt.args.files, tt.args.patterns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("matchFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}
