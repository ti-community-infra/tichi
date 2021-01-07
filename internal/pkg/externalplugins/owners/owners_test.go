package owners

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	"github.com/sirupsen/logrus"
	tiexternalplugins "github.com/ti-community-infra/ti-community-prow/internal/pkg/externalplugins"
	"gotest.tools/assert"
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
		{
			ID:   60,
			Name: "Releasers",
		},
	}, nil
}

// ListTeamMembers return a fake team with a single "sig-lead" GitHub team member.
func (f *fakegithub) ListTeamMembers(_ string, teamID int, role string) ([]github.TeamMember, error) {
	if role != github.RoleAll {
		return nil, fmt.Errorf("unsupported role %v (only all supported)", role)
	}
	teams := map[int][]github.TeamMember{
		0: {
			{Login: "admin1"},
			{Login: "admin2"},
		},
		42: {
			{Login: "sig-leader1"},
			{Login: "sig-leader2"},
		},
		60: {
			{Login: "admin1"},
			{Login: "releaser1"},
			{Login: "releaser2"},
		},
	}
	members, ok := teams[teamID]
	if !ok {
		return []github.TeamMember{}, nil
	}
	return members, nil
}

func TestListOwners(t *testing.T) {
	sig1Res := SigResponse{
		Data: SigInfo{
			Name: "sig1",
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

	sig2Res := SigResponse{
		Data: SigInfo{
			Name: "sig2",
			Membership: SigMembership{
				TechLeaders: []ContributorInfo{
					{
						GithubName: "leader3",
					}, {
						GithubName: "leader4",
					},
				},
				CoLeaders: []ContributorInfo{
					{
						GithubName: "coLeader3",
					}, {
						GithubName: "coLeader4",
					},
				},
				Committers: []ContributorInfo{
					{
						GithubName: "committer3",
					}, {
						GithubName: "committer4",
					},
				},
				Reviewers: []ContributorInfo{
					{
						GithubName: "reviewer3",
					}, {
						GithubName: "reviewer4",
					},
				},
				ActiveContributors: []ContributorInfo{},
			},
			NeedsLgtm: 1,
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

	org := "ti-community-infra"
	repoName := "test-dev"
	pullNumber := 1
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	testcases := []struct {
		name                   string
		sigsRes                []SigResponse
		labels                 []github.Label
		defaultSigName         string
		trustTeams             []string
		defaultRequireLgtm     int
		requireLgtmLabelPrefix string
		branchesConfig         map[string]tiexternalplugins.TiCommunityOwnerBranchConfig

		expectCommitters []string
		expectReviewers  []string
		expectNeedsLgtm  int
	}{
		{
			name:    "has one sig label",
			sigsRes: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
			},
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			expectNeedsLgtm: lgtmTwo,
		},
		{
			name:    "has one sig label and require one lgtm",
			sigsRes: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
				{
					Name: "require-LGT1",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			expectNeedsLgtm: 1,
		},
		{
			name:    "has two sigs label",
			sigsRes: []SigResponse{sig1Res, sig2Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
				{
					Name: "sig/sig2",
				},
			},
			expectCommitters: []string{
				"leader1", "leader2", "leader3", "leader4", "coLeader1", "coLeader2", "coLeader3", "coLeader4",
				"committer1", "committer2", "committer3", "committer4",
			},
			expectReviewers: []string{
				"leader1", "leader2", "leader3", "leader4", "coLeader1", "coLeader2", "coLeader3", "coLeader4",
				"committer1", "committer2", "committer3", "committer4",
				"reviewer1", "reviewer2", "reviewer3", "reviewer4",
			},
			expectNeedsLgtm: lgtmTwo,
		},
		{
			name:    "has two sigs label and require one lgtm",
			sigsRes: []SigResponse{sig1Res, sig2Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
				{
					Name: "sig/sig2",
				},
				{
					Name: "require-LGT1",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			expectCommitters: []string{
				"leader1", "leader2", "leader3", "leader4", "coLeader1", "coLeader2", "coLeader3", "coLeader4",
				"committer1", "committer2", "committer3", "committer4",
			},
			expectReviewers: []string{
				"leader1", "leader2", "leader3", "leader4", "coLeader1", "coLeader2", "coLeader3", "coLeader4",
				"committer1", "committer2", "committer3", "committer4",
				"reviewer1", "reviewer2", "reviewer3", "reviewer4",
			},
			expectNeedsLgtm: 1,
		},
		{
			name:    "non sig label",
			sigsRes: []SigResponse{sig1Res},
			expectCommitters: []string{
				"collab2", "collab3",
			},
			expectReviewers: []string{
				"collab2", "collab3",
			},
			expectNeedsLgtm: lgtmTwo,
		},
		{
			name:    "non sig label and require one lgtm",
			sigsRes: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "require-LGT1",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			expectCommitters: []string{
				"collab2", "collab3",
			},
			expectReviewers: []string{
				"collab2", "collab3",
			},
			expectNeedsLgtm: 1,
		},
		{
			name:           "non sig label but use default sig name",
			sigsRes:        []SigResponse{sig1Res},
			defaultSigName: "sig1",
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			expectNeedsLgtm: lgtmTwo,
		},
		{
			name:           "non sig label but use default sig name and require one lgtm",
			sigsRes:        []SigResponse{sig1Res},
			defaultSigName: "sig1",
			labels: []github.Label{
				{
					Name: "require-LGT1",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			expectNeedsLgtm: 1,
		},
		{
			name:                   "non sig label but use default sig name and default require two lgtm",
			sigsRes:                []SigResponse{sig1Res},
			defaultSigName:         "sig1",
			labels:                 []github.Label{},
			requireLgtmLabelPrefix: "require-LGT",
			defaultRequireLgtm:     2,
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			expectNeedsLgtm: lgtmTwo,
		},
		{
			name:    "has one sig label and a trust team",
			sigsRes: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
			},
			trustTeams: []string{"Leads"},
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
				// Team members.
				"sig-leader1", "sig-leader2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
				// Team members.
				"sig-leader1", "sig-leader2",
			},
			expectNeedsLgtm: lgtmTwo,
		},
		{
			name:    "owners plugin config contains branch config",
			sigsRes: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
			},
			defaultRequireLgtm: 2,
			trustTeams:         []string{"Leads"},
			branchesConfig: map[string]tiexternalplugins.TiCommunityOwnerBranchConfig{
				"master": {
					DefaultRequireLgtm: 3,
					TrustTeams:         []string{"Admins"},
				},
				"release": {
					DefaultRequireLgtm: 4,
					TrustTeams:         []string{"Releasers"},
				},
			},
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
				// Team members.
				"admin1", "admin2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
				// Team members.
				"admin1", "admin2",
			},
			expectNeedsLgtm: 3,
		},
		{
			name:    "owners plugin config contains multiple trusted teams",
			sigsRes: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
			},
			trustTeams: []string{"Leads", "Admins", "Releasers"},
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
				// Team members.
				"admin1", "admin2", "sig-leader1", "sig-leader2",
				"releaser1", "releaser2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
				// Team members.
				"admin1", "admin2", "sig-leader1", "sig-leader2",
				"releaser1", "releaser2",
			},
			expectNeedsLgtm: 2,
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			// Fake http client.
			mux := http.NewServeMux()
			testServer := httptest.NewServer(mux)

			config := &tiexternalplugins.Configuration{}
			repoConfig := tiexternalplugins.TiCommunityOwners{
				Repos:              []string{"ti-community-infra/test-dev"},
				SigEndpoint:        testServer.URL,
				DefaultRequireLgtm: tc.defaultRequireLgtm,
			}

			if len(tc.defaultSigName) != 0 {
				repoConfig.DefaultSigName = tc.defaultSigName
			}

			if tc.trustTeams != nil {
				repoConfig.TrustTeams = tc.trustTeams
			}

			if tc.requireLgtmLabelPrefix != "" {
				repoConfig.RequireLgtmLabelPrefix = tc.requireLgtmLabelPrefix
			}

			if tc.branchesConfig != nil {
				repoConfig.Branches = tc.branchesConfig
			}

			config.TiCommunityOwners = []tiexternalplugins.TiCommunityOwners{
				repoConfig,
			}

			for _, res := range tc.sigsRes {
				sigRes := res
				// URL pattern.
				pattern := fmt.Sprintf(SigEndpointFmt, sigRes.Data.Name)
				mux.HandleFunc(pattern, func(res http.ResponseWriter, req *http.Request) {
					if req.Method != "GET" {
						t.Errorf("expect 'Get' got '%s'", req.Method)
					}
					reqBodyBytes := new(bytes.Buffer)
					err := json.NewEncoder(reqBodyBytes).Encode(sigRes)
					if err != nil {
						t.Errorf("Encoding data '%v' failed", sigRes)
					}

					_, err = res.Write(reqBodyBytes.Bytes())
					if err != nil {
						t.Errorf("Write data '%v' failed", sigRes)
					}
				})
			}

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
			if tc.labels != nil {
				fc.PullRequests[pullNumber].Labels = tc.labels
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

			sort.Strings(res.Data.Committers)
			sort.Strings(tc.expectCommitters)

			if len(res.Data.Committers) != len(tc.expectCommitters) ||
				!reflect.DeepEqual(res.Data.Committers, tc.expectCommitters) {
				t.Errorf("Different committers: Got \"%v\" expected \"%v\"", res.Data.Committers, tc.expectCommitters)
			}

			sort.Strings(res.Data.Reviewers)
			sort.Strings(tc.expectReviewers)

			if len(res.Data.Reviewers) != len(tc.expectReviewers) ||
				!reflect.DeepEqual(res.Data.Reviewers, tc.expectReviewers) {
				t.Errorf("Different reviewers: Got \"%v\" expected \"%v\"", res.Data.Reviewers, tc.expectReviewers)
			}

			if res.Data.NeedsLgtm != tc.expectNeedsLgtm {
				t.Errorf("Different LGTM: Got \"%v\" expected \"%v\"", res.Data.NeedsLgtm, tc.expectNeedsLgtm)
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
	org := "ti-community-infra"
	repoName := "test-dev"
	sigName := "testing"
	pullNumber := 1
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	testcases := []struct {
		name        string
		labels      []github.Label
		invalidData bool
		expectError string
	}{
		{
			name: "has one sig label",
			labels: []github.Label{
				{
					Name: "sig/testing",
				},
			},
			invalidData: true,
			expectError: "unexpected end of JSON input",
		},
		{
			name: "non sig label",
			labels: []github.Label{
				{
					Name: "sig/testing",
				},
			},
			invalidData: false,
			expectError: "could not get a sig",
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			// Fake http client.
			mux := http.NewServeMux()
			testServer := httptest.NewServer(mux)

			config := &tiexternalplugins.Configuration{}
			config.TiCommunityOwners = []tiexternalplugins.TiCommunityOwners{
				{
					Repos:       []string{"ti-community-infra/test-dev"},
					SigEndpoint: testServer.URL,
				},
			}

			// URL pattern.
			pattern := fmt.Sprintf(SigEndpointFmt, sigName)
			mux.HandleFunc(pattern, func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("expect 'Get' got '%s'", req.Method)
				}

				if tc.invalidData {
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
			if tc.labels != nil {
				fc.PullRequests[pullNumber].Labels = tc.labels
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
				t.Errorf("expected error '%v', but it is nil", tc.expectError)
			} else if err.Error() != tc.expectError {
				t.Errorf("expected error '%v', but it is '%v'", tc.expectError, err)
			}

			testServer.Close()
		})
	}
}

func TestGetSigsNameByLabel(t *testing.T) {
	testLabel1 := "testLabel1"
	testLabel2 := "testLabel2"
	sig1Label := "sig/testing1"
	sig2Label := "sig/testing2"

	testcases := []struct {
		name           string
		labels         []github.Label
		expectSigsName []string
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
					Name: sig1Label,
				},
			},
			expectSigsName: []string{"testing1"},
		},
		{
			name: "has two sig labels",
			labels: []github.Label{
				{
					Name: testLabel1,
				}, {
					Name: sig2Label,
				},
				{
					Name: sig1Label,
				},
			},
			expectSigsName: []string{"testing1", "testing2"},
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
			expectSigsName: nil,
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			sigsName := getSigsNameByLabels(tc.labels)
			// sort the name.
			sort.Strings(sigsName)
			sort.Strings(tc.expectSigsName)

			assert.DeepEqual(t, sigsName, tc.expectSigsName)
		})
	}
}

func TestGetRequireLgtmByLabel(t *testing.T) {
	testcases := []struct {
		name                   string
		labels                 []github.Label
		requireLgtmLabelPrefix string
		expectLgtm             int
		expectErr              string
	}{
		{
			name: "No require",
			labels: []github.Label{
				{
					Name: "sig/testing",
				},
			},
			requireLgtmLabelPrefix: "require/LGT",
			expectLgtm:             0,
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
			expectLgtm:             1,
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
			expectLgtm:             2,
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
			expectLgtm:             0,
			expectErr:              "strconv.Atoi: parsing \"abcde\": invalid syntax",
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			requireLgtm, err := getRequireLgtmByLabel(tc.labels, tc.requireLgtmLabelPrefix)

			if requireLgtm != tc.expectLgtm {
				t.Errorf("expected lgtm '%d', but it is '%d'", tc.expectLgtm, requireLgtm)
			}

			if err != nil && err.Error() != tc.expectErr {
				t.Errorf("expected err '%v', but it is '%v'", tc.expectErr, err)
			}
		})
	}
}
