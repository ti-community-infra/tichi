//nolint:scopelint
package owners

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	tiexternalplugins "github.com/tidb-community-bots/ti-community-prow/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/github"
)

type fakegithub struct {
	PullRequests  map[int]*github.PullRequest
	Collaborators []github.User
}

// GetPullRequest returns details about the PR.
func (f *fakegithub) GetPullRequest(owner, repo string, number int) (*github.PullRequest, error) {
	val, exists := f.PullRequests[number]
	if !exists {
		return nil, fmt.Errorf("pull request number %d does not exist", number)
	}
	return val, nil
}

// ListCollaborators lists the collaborators.
func (f *fakegithub) ListCollaborators(org, repo string) ([]github.User, error) {
	return f.Collaborators, nil
}

// ListTeams return a list of fake teams that correspond to the fake team members returned by ListTeamMembers.
func (f *fakegithub) ListTeams(org string) ([]github.Team, error) {
	return []github.Team{
		{
			ID:   0,
			Name: "Admins",
		},
		{
			ID:   42,
			Name: "Leads",
		},
	}, nil
}

// ListTeamMembers return a fake team with a single "sig-lead" GitHub team member.
func (f *fakegithub) ListTeamMembers(_ string, teamID int, role string) ([]github.TeamMember, error) {
	if role != github.RoleAll {
		return nil, fmt.Errorf("unsupported role %v (only all supported)", role)
	}
	teams := map[int][]github.TeamMember{
		0:  {{Login: "default-sig-lead"}},
		42: {{Login: "sig-lead"}},
	}
	members, ok := teams[teamID]
	if !ok {
		return []github.TeamMember{}, nil
	}
	return members, nil
}

func TestListOwners(t *testing.T) {
	validSigRes := SigResponse{
		Data: SigInfo{
			Name: "test",
			Membership: SigMembership{
				TechLeaders: []ContributorInfo{
					{
						GithubName: "leader1",
					}, {
						GithubName: "leader2",
					},
				},
				CoLeaders: []ContributorInfo{
					{
						GithubName: "coLeader1",
					}, {
						GithubName: "coLeader2",
					},
				},
				Committers: []ContributorInfo{
					{
						GithubName: "committer1",
					}, {
						GithubName: "committer2",
					},
				},
				Reviewers: []ContributorInfo{
					{
						GithubName: "reviewer1",
					}, {
						GithubName: "reviewer2",
					},
				},
				ActiveContributors: []ContributorInfo{},
			},
			NeedsLgtm: lgtmTwo,
		},
		Message: listOwnersSuccessMessage,
	}

	collaborators := []github.User{
		{
			Login: "collab1",
			Permissions: github.RepoPermissions{
				Pull:  true,
				Push:  false,
				Admin: false,
			},
		},
		{
			Login: "collab2",
			Permissions: github.RepoPermissions{
				Pull:  true,
				Push:  true,
				Admin: false,
			},
		},
		{
			Login: "collab3",
			Permissions: github.RepoPermissions{
				Pull:  true,
				Push:  true,
				Admin: true,
			},
		},
	}

	org := "tidb-community-bots"
	repoName := "test-dev"
	sigName := "testing"
	pullNumber := 1
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	testCases := []struct {
		name                   string
		sigRes                 *SigResponse
		labels                 []github.Label
		useDefaultSigName      bool
		trustTeam              string
		defaultRequireLgtm     int
		requireLgtmLabelPrefix string

		exceptCommitters []string
		exceptReviewers  []string
		exceptNeedsLgtm  int
	}{
		{
			name:   "has one sig label",
			sigRes: &validSigRes,
			labels: []github.Label{
				{
					Name: "sig/testing",
				},
			},
			exceptCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			exceptReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			exceptNeedsLgtm: lgtmTwo,
		},
		{
			name:   "has one sig label and require one lgtm",
			sigRes: &validSigRes,
			labels: []github.Label{
				{
					Name: "sig/testing",
				}, {
					Name: "require-LGT1",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			exceptCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			exceptReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			exceptNeedsLgtm: 1,
		},
		{
			name:   "non sig label",
			sigRes: &validSigRes,
			exceptCommitters: []string{
				"collab2", "collab3",
			},
			exceptReviewers: []string{
				"collab2", "collab3",
			},
			exceptNeedsLgtm: lgtmTwo,
		},
		{
			name:   "non sig label and require one lgtm",
			sigRes: &validSigRes,
			labels: []github.Label{
				{
					Name: "require-LGT1",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			exceptCommitters: []string{
				"collab2", "collab3",
			},
			exceptReviewers: []string{
				"collab2", "collab3",
			},
			exceptNeedsLgtm: 1,
		},
		{
			name:              "non sig label but use default sig name",
			sigRes:            &validSigRes,
			useDefaultSigName: true,
			exceptCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			exceptReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			exceptNeedsLgtm: lgtmTwo,
		},
		{
			name:              "non sig label but use default sig name and require one lgtm",
			sigRes:            &validSigRes,
			useDefaultSigName: true,
			labels: []github.Label{
				{
					Name: "require-LGT1",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			exceptCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			exceptReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			exceptNeedsLgtm: 1,
		},
		{
			name:                   "non sig label but use default sig name and default require two lgtm",
			sigRes:                 &validSigRes,
			useDefaultSigName:      true,
			labels:                 []github.Label{},
			requireLgtmLabelPrefix: "require-LGT",
			defaultRequireLgtm:     2,
			exceptCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			exceptReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			exceptNeedsLgtm: lgtmTwo,
		},
		{
			name:   "has one sig label and a trust team",
			sigRes: &validSigRes,
			labels: []github.Label{
				{
					Name: "sig/testing",
				},
			},
			exceptCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
				// Team members.
				"sig-lead",
			},
			exceptReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
				// Team members.
				"sig-lead",
			},
			trustTeam:       "Leads",
			exceptNeedsLgtm: lgtmTwo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Fake http client.
			mux := http.NewServeMux()
			testServer := httptest.NewServer(mux)

			config := &tiexternalplugins.Configuration{}
			owners := tiexternalplugins.TiCommunityOwners{
				Repos:              []string{"tidb-community-bots/test-dev"},
				SigEndpoint:        testServer.URL,
				DefaultRequireLgtm: testCase.defaultRequireLgtm,
			}

			if testCase.useDefaultSigName {
				owners.DefaultSigName = sigName
			}

			if testCase.trustTeam != "" {
				owners.OwnersTrustTeam = testCase.trustTeam
			}

			if testCase.requireLgtmLabelPrefix != "" {
				owners.RequireLgtmLabelPrefix = testCase.requireLgtmLabelPrefix
			}

			config.TiCommunityOwners = []tiexternalplugins.TiCommunityOwners{
				owners,
			}

			// URL pattern.
			pattern := fmt.Sprintf(SigEndpointFmt, sigName)
			mux.HandleFunc(pattern, func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("Except 'Get' got '%s'", req.Method)
				}
				reqBodyBytes := new(bytes.Buffer)
				err := json.NewEncoder(reqBodyBytes).Encode(testCase.sigRes)
				if err != nil {
					t.Errorf("Encoding data '%v' failed", testCase.sigRes)
				}

				_, err = res.Write(reqBodyBytes.Bytes())
				if err != nil {
					t.Errorf("Write data '%v' failed", testCase.sigRes)
				}
			})

			fc := &fakegithub{
				PullRequests: map[int]*github.PullRequest{
					pullNumber: {
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
				Collaborators: collaborators,
			}

			// NOTICE: adds labels.
			if testCase.labels != nil {
				fc.PullRequests[pullNumber].Labels = testCase.labels
			}

			ownersServer := Server{
				Client: testServer.Client(),
				TokenGenerator: func() []byte {
					return []byte{}
				},
				Gc:  fc,
				Log: logrus.WithField("server", "testing"),
			}

			res, err := ownersServer.ListOwners(org, repoName, pullNumber, config)

			if err != nil {
				t.Errorf("unexpected error: '%v'", err)
			}

			if len(res.Data.Committers) != len(testCase.exceptCommitters) {
				t.Errorf("Different committers: Got \"%v\" expected \"%v\"", res.Data.Committers, testCase.exceptCommitters)
			}

			if len(res.Data.Reviewers) != len(testCase.exceptReviewers) {
				t.Errorf("Different reviewers: Got \"%v\" expected \"%v\"", res.Data.Reviewers, testCase.exceptReviewers)
			}

			if res.Data.NeedsLgtm != testCase.exceptNeedsLgtm {
				t.Errorf("Different LGTM: Got \"%v\" expected \"%v\"", res.Data.NeedsLgtm, testCase.exceptNeedsLgtm)
			}
		})
	}
}

func TestListOwnersFailed(t *testing.T) {
	collaborators := []github.User{
		{
			Login: "collab1",
		},
		{
			Login: "collab2",
		},
	}
	org := "tidb-community-bots"
	repoName := "test-dev"
	sigName := "testing"
	pullNumber := 1
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	testCases := []struct {
		name        string
		labels      []github.Label
		invalidData bool
		exceptError string
	}{
		{
			name: "has one sig label",
			labels: []github.Label{
				{
					Name: "sig/testing",
				},
			},
			invalidData: true,
			exceptError: "unexpected end of JSON input",
		},
		{
			name: "non sig label",
			labels: []github.Label{
				{
					Name: "sig/testing",
				},
			},
			invalidData: false,
			exceptError: "could not get a sig",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Fake http client.
			mux := http.NewServeMux()
			testServer := httptest.NewServer(mux)

			config := &tiexternalplugins.Configuration{}
			config.TiCommunityOwners = []tiexternalplugins.TiCommunityOwners{
				{
					Repos:       []string{"tidb-community-bots/test-dev"},
					SigEndpoint: testServer.URL,
				},
			}

			// URL pattern.
			pattern := fmt.Sprintf(SigEndpointFmt, sigName)
			mux.HandleFunc(pattern, func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("Except 'Get' got '%s'", req.Method)
				}

				if testCase.invalidData {
					_, err := res.Write([]byte{})
					if err != nil {
						t.Errorf("Write data invalidData failed")
					}
				} else {
					// Just http filed.
					res.WriteHeader(http.StatusInternalServerError)
				}
			})

			fc := &fakegithub{
				PullRequests: map[int]*github.PullRequest{
					pullNumber: {
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
				Collaborators: collaborators,
			}

			// NOTICE: adds labels.
			if testCase.labels != nil {
				fc.PullRequests[pullNumber].Labels = testCase.labels
			}

			ownersServer := Server{
				Client: testServer.Client(),
				TokenGenerator: func() []byte {
					return []byte{}
				},
				Gc:  fc,
				Log: logrus.WithField("server", "testing"),
			}

			_, err := ownersServer.ListOwners(org, repoName, pullNumber, config)
			if err == nil {
				t.Errorf("expected error '%v', but it is nil", testCase.exceptError)
			} else if err.Error() != testCase.exceptError {
				t.Errorf("expected error '%v', but it is '%v'", testCase.exceptError, err)
			}

			testServer.Close()
		})
	}
}

func TestGetSigNameByLabel(t *testing.T) {
	testLabel1 := "testLabel1"
	testLabel2 := "testLabel2"
	sigLabel := "sig/testing"

	testCases := []struct {
		name          string
		labels        []github.Label
		exceptSigName string
	}{
		{
			name: "has one sig label",
			labels: []github.Label{
				{
					Name: testLabel1,
				}, {
					Name: testLabel2,
				},
				{
					Name: sigLabel,
				},
			},
			exceptSigName: "testing",
		},
		{
			name: "non sig label",
			labels: []github.Label{
				{
					Name: testLabel1,
				}, {
					Name: testLabel2,
				},
			},
			exceptSigName: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sigName := getSigNameByLabel(testCase.labels)

			if sigName != testCase.exceptSigName {
				t.Errorf("expected sig '%s', but it is '%s'", testCase.exceptSigName, sigName)
			}
		})
	}
}

func TestGetRequireLgtmByLabel(t *testing.T) {
	testCases := []struct {
		name                   string
		labels                 []github.Label
		requireLgtmLabelPrefix string
		exceptLgtm             int
		exceptErr              string
	}{
		{
			name: "No require",
			labels: []github.Label{
				{
					Name: "sig/testing",
				},
			},
			requireLgtmLabelPrefix: "require/LGT",
			exceptLgtm:             0,
		}, {
			name: "Require one lgtm",
			labels: []github.Label{
				{
					Name: "sig/testing",
				}, {
					Name: "require/LGT1",
				},
			},
			requireLgtmLabelPrefix: "require/LGT",
			exceptLgtm:             1,
		}, {
			name: "Require two lgtm",
			labels: []github.Label{
				{
					Name: "sig/testing",
				}, {
					Name: "require-LGT2",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			exceptLgtm:             2,
		},
		{
			name: "Wrong require",
			labels: []github.Label{
				{
					Name: "sig/testing",
				}, {
					Name: "require-LGTabcde",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			exceptLgtm:             0,
			exceptErr:              "strconv.Atoi: parsing \"abcde\": invalid syntax",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			requireLgtm, err := getRequireLgtmByLabel(testCase.labels, testCase.requireLgtmLabelPrefix)

			if requireLgtm != testCase.exceptLgtm {
				t.Errorf("expected lgtm '%d', but it is '%d'", testCase.exceptLgtm, requireLgtm)
			}

			if err != nil && err.Error() != testCase.exceptErr {
				t.Errorf("expected err '%v', but it is '%v'", testCase.exceptErr, err)
			}
		})
	}
}
