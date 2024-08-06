package owners

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	githubql "github.com/shurcooL/githubv4"
	"github.com/shurcooL/graphql"
	"github.com/sirupsen/logrus"
	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/lib"
	"gotest.tools/assert"
	"k8s.io/test-infra/prow/github"
)

type fakegithub struct {
	PullRequests  map[int]*github.PullRequest
	Collaborators []RepositoryCollaboratorConnection
}

// GetPullRequest returns details about the PR.
func (f *fakegithub) GetPullRequest(_, _ string, number int) (*github.PullRequest, error) {
	val, exists := f.PullRequests[number]
	if !exists {
		return nil, fmt.Errorf("pull request number %d does not exist", number)
	}
	return val, nil
}

func (f *fakegithub) QueryWithGitHubAppsSupport(
	_ context.Context, q interface{}, vars map[string]interface{}, _ string) error {
	query, ok := q.(*collaboratorsQuery)
	if ok {
		query.Repository.Collaborators.Edges = f.Collaborators
		return nil
	}

	sq, ok := q.(*lib.TeamMembersQuery)
	if ok {
		var res lib.TeamMembersQuery
		members := make([]lib.MemberEdge, 0)
		logins := make([]string, 0)

		teamSlug, ok := vars["teamSlug"]
		if !ok {
			return errors.New("can not found variable teamSlug")
		}
		slug, ok := teamSlug.(githubql.String)
		if !ok {
			return errors.New("unexpected variable type")
		}

		switch string(slug) {
		case "Admins":
			logins = []string{"admin1", "admin2"}
		case "Leads":
			logins = []string{"sig-leader1", "sig-leader2"}
		case "Releasers":
			logins = []string{"admin1", "releaser1", "releaser2"}
		case "Reviewers":
			logins = []string{"reviewer1", "reviewer2"}
		case "Committers":
			logins = []string{"committer1", "committer2"}
		}

		for _, login := range logins {
			members = append(members, lib.MemberEdge{
				Node: lib.MemberNode{
					Login: graphql.String(login),
				},
			})
		}

		res.Organization.Team.Members.Edges = members
		sq.Organization = res.Organization
		sq.RateLimit = res.RateLimit
		return nil
	}

	return errors.New("unexpected query type")
}

// ListTeams return a list of fake teams that correspond to the fake team members returned by ListTeamMembers.
func (f *fakegithub) ListTeams(string) ([]github.Team, error) {
	return []github.Team{
		{
			ID:   0,
			Name: "Admins",
			Slug: "Admins",
		},
		{
			ID:   42,
			Name: "Leads",
			Slug: "Leads",
		},
		{
			ID:   60,
			Name: "Releasers",
			Slug: "Releasers",
		},
		{
			ID:   70,
			Name: "Reviewers",
			Slug: "Reviewers",
		},
		{
			ID:   80,
			Name: "Committers",
			Slug: "Committers",
		},
	}, nil
}

func TestListOwners(t *testing.T) {
	sig1Res := SigResponse{
		Data: SigInfo{
			Name: "sig1",
			Membership: SigMembership{
				TechLeaders: []MemberInfo{
					{
						GithubName: "leader1",
					}, {
						GithubName: "leader2",
					},
				},
				CoLeaders: []MemberInfo{
					{
						GithubName: "coLeader1",
					}, {
						GithubName: "coLeader2",
					},
				},
				Committers: []MemberInfo{
					{
						GithubName: "committer1",
					}, {
						GithubName: "committer2",
					},
				},
				Reviewers: []MemberInfo{
					{
						GithubName: "reviewer1",
					}, {
						GithubName: "reviewer2",
					},
				},
			},
			NeedsLgtm: defaultRequireLgtmNum,
		},
		Message: "Test sig1.",
	}

	sig2Res := SigResponse{
		Data: SigInfo{
			Name: "sig2",
			Membership: SigMembership{
				TechLeaders: []MemberInfo{
					{
						GithubName: "leader3",
					}, {
						GithubName: "leader4",
					},
				},
				CoLeaders: []MemberInfo{
					{
						GithubName: "coLeader3",
					}, {
						GithubName: "coLeader4",
					},
				},
				Committers: []MemberInfo{
					{
						GithubName: "committer3",
					}, {
						GithubName: "committer4",
					},
				},
				Reviewers: []MemberInfo{
					{
						GithubName: "reviewer3",
					}, {
						GithubName: "reviewer4",
					},
				},
			},
			NeedsLgtm: 1,
		},
		Message: "Test sig2.",
	}

	membersResponse := MembersResponse{
		Data: MembersInfo{
			Members: []MemberInfo{
				{
					Level:      activeContributorLevel,
					GithubName: "activeContributor1",
				},
				{
					Level:      activeContributorLevel,
					GithubName: "activeContributor2",
				},
				{
					Level:      reviewerLevel,
					GithubName: "reviewer1",
				},
				{
					Level:      reviewerLevel,
					GithubName: "reviewer2",
				},
				{
					Level:      committerLevel,
					GithubName: "committer1",
				},
				{
					Level:      committerLevel,
					GithubName: "committer2",
				},
				{
					Level:      coLeaderLevel,
					GithubName: "coLeader1",
				},
				{
					Level:      coLeaderLevel,
					GithubName: "coLeader2",
				},
				{
					Level:      leaderLevel,
					GithubName: "leader1",
				},
				{
					Level:      leaderLevel,
					GithubName: "leader2",
				},
			},
			Total: 10,
		},
		Message: "Test members.",
	}

	collaborators := []RepositoryCollaboratorConnection{
		{
			Permission: "",
			Node:       struct{ Login githubql.String }{Login: "passerby"},
		},
		{
			Permission: readPermission,
			Node:       struct{ Login githubql.String }{Login: "collab1"},
		},
		{
			Permission: triagePermission,
			Node:       struct{ Login githubql.String }{Login: "collab2"},
		},
		{
			Permission: writePermission,
			Node:       struct{ Login githubql.String }{Login: "collab3"},
		},
		{
			Permission: maintainPermission,
			Node:       struct{ Login githubql.String }{Login: "collab4"},
		},
		{
			Permission: adminPermission,
			Node:       struct{ Login githubql.String }{Login: "collab5"},
		},
	}

	org := "ti-community-infra"
	repoName := "test-dev"
	pullNumber := 1
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	testcases := []struct {
		name                   string
		sigResponses           []SigResponse
		membersResponse        *MembersResponse
		labels                 []github.Label
		defaultSigName         string
		reviewerTeams          []string
		committerTeams         []string
		defaultRequireLgtm     int
		requireLgtmLabelPrefix string
		useGitHubPermission    bool
		useGitHubTeam          bool
		branchesConfig         map[string]tiexternalplugins.TiCommunityOwnerBranchConfig

		expectCommitters []string
		expectReviewers  []string
		expectNeedsLgtm  int
	}{
		{
			name:         "has one sig label",
			sigResponses: []SigResponse{sig1Res},
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
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:         "has one sig label and require one lgtm",
			sigResponses: []SigResponse{sig1Res},
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
			name:         "have two sig labels",
			sigResponses: []SigResponse{sig1Res, sig2Res},
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
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:         "have two sig labels and require one lgtm",
			sigResponses: []SigResponse{sig1Res, sig2Res},
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
			name:            "non sig label",
			sigResponses:    []SigResponse{sig1Res},
			membersResponse: &membersResponse,
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2", "committer1", "committer2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2", "committer1", "committer2", "reviewer1", "reviewer2",
			},
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:            "non sig label and require one lgtm",
			sigResponses:    []SigResponse{sig1Res},
			membersResponse: &membersResponse,
			labels: []github.Label{
				{
					Name: "require-LGT1",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2", "committer1", "committer2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2", "committer1", "committer2", "reviewer1", "reviewer2",
			},
			expectNeedsLgtm: 1,
		},
		{
			name:           "non sig label but use default sig name",
			sigResponses:   []SigResponse{sig1Res},
			defaultSigName: "sig1",
			expectCommitters: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			expectReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:           "non sig label but use default sig name and require one lgtm",
			sigResponses:   []SigResponse{sig1Res},
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
			sigResponses:           []SigResponse{sig1Res},
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
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:         "has one sig label and a committer team",
			sigResponses: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
			},
			committerTeams: []string{"Leads"},
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
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:         "owners plugin config contains branch config",
			sigResponses: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
			},
			defaultRequireLgtm: 2,
			committerTeams:     []string{"Leads"},
			branchesConfig: map[string]tiexternalplugins.TiCommunityOwnerBranchConfig{
				"master": {
					DefaultRequireLgtm: 3,
					CommitterTeams:     []string{"Committers", "Admins"},
					ReviewerTeams:      []string{"Reviewers"},
				},
				"release": {
					DefaultRequireLgtm: 4,
					CommitterTeams:     []string{"Releasers"},
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
			name:         "has one sig label and multiple committers teams",
			sigResponses: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
			},
			committerTeams: []string{"Leads", "Admins", "Releasers"},
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
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:                "use GitHub permission",
			sigResponses:        []SigResponse{sig1Res},
			labels:              []github.Label{},
			useGitHubPermission: true,
			expectCommitters: []string{
				"collab3", "collab4", "collab5",
			},
			expectReviewers: []string{
				"collab2", "collab3", "collab4", "collab5",
			},
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:         "use GitHub permission and require one lgtm",
			sigResponses: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "require-LGT1",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			useGitHubPermission:    true,
			expectCommitters: []string{
				"collab3", "collab4", "collab5",
			},
			expectReviewers: []string{
				"collab2", "collab3", "collab4", "collab5",
			},
			expectNeedsLgtm: 1,
		},
		{
			name:                   "use GitHub permission and a committer team",
			sigResponses:           []SigResponse{sig1Res},
			labels:                 []github.Label{},
			committerTeams:         []string{"Leads"},
			requireLgtmLabelPrefix: "require-LGT",
			useGitHubPermission:    true,
			expectCommitters: []string{
				"collab3", "collab4", "collab5",
				// Team members.
				"sig-leader1", "sig-leader2",
			},
			expectReviewers: []string{
				"collab2", "collab3", "collab4", "collab5",
				// Team members.
				"sig-leader1", "sig-leader2",
			},
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:               "use GitHub permission and owners plugin config contains branch config",
			sigResponses:       []SigResponse{sig1Res},
			labels:             []github.Label{},
			defaultRequireLgtm: 2,
			committerTeams:     []string{"Leads"},
			branchesConfig: map[string]tiexternalplugins.TiCommunityOwnerBranchConfig{
				"master": {
					DefaultRequireLgtm:  3,
					UseGitHubPermission: true,
					CommitterTeams:      []string{"Admins"},
				},
				"release": {
					DefaultRequireLgtm: 4,
				},
			},
			expectCommitters: []string{
				"collab3", "collab4", "collab5",
				// Team members.
				"admin1", "admin2",
			},
			expectReviewers: []string{
				"collab2", "collab3", "collab4", "collab5",
				// Team members.
				"admin1", "admin2",
			},
			expectNeedsLgtm: 3,
		},
		{
			name:         "use GitHub permission and has one sig label",
			sigResponses: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
			},
			useGitHubPermission: true,
			expectCommitters: []string{
				"collab3", "collab4", "collab5",
			},
			expectReviewers: []string{
				"collab2", "collab3", "collab4", "collab5",
			},
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:                "use GitHub permission and config contains reviewer teams and committers teams",
			useGitHubPermission: true,
			reviewerTeams:       []string{"Reviewers"},
			committerTeams:      []string{"Committers"},
			expectCommitters: []string{
				"collab3", "collab4", "collab5",
				// Team members.
				"committer1", "committer2",
			},
			expectReviewers: []string{
				"collab2", "collab3", "collab4", "collab5",
				// Team members.
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:           "use GitHub teams",
			sigResponses:   []SigResponse{sig1Res},
			labels:         []github.Label{},
			useGitHubTeam:  true,
			committerTeams: []string{"Committers"},
			reviewerTeams:  []string{"Reviewers"},
			expectCommitters: []string{
				"committer1", "committer2",
			},
			expectReviewers: []string{
				"reviewer1", "reviewer2", "committer1", "committer2",
			},
			expectNeedsLgtm: defaultRequireLgtmNum,
		},
		{
			name:         "use GitHub teams and require one lgtm",
			sigResponses: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "require-LGT1",
				},
			},
			requireLgtmLabelPrefix: "require-LGT",
			useGitHubTeam:          true,
			committerTeams:         []string{"Committers"},
			reviewerTeams:          []string{"Reviewers"},
			expectCommitters: []string{
				"committer1", "committer2",
			},
			expectReviewers: []string{
				"reviewer1", "reviewer2", "committer1", "committer2",
			},
			expectNeedsLgtm: 1,
		},
		{
			name:               "use GitHub team and owners plugin config contains branch config",
			sigResponses:       []SigResponse{sig1Res},
			labels:             []github.Label{},
			defaultRequireLgtm: 2,
			useGitHubTeam:      true,
			reviewerTeams:      []string{"Reviewers"},
			committerTeams:     []string{"Committers"},
			branchesConfig: map[string]tiexternalplugins.TiCommunityOwnerBranchConfig{
				"master": {
					DefaultRequireLgtm: 3,
					UseGithubTeam:      true,
					CommitterTeams:     []string{"Admins"},
				},
				"release": {
					DefaultRequireLgtm: 4,
					UseGithubTeam:      true,
					CommitterTeams:     []string{"Leads"},
				},
			},
			expectCommitters: []string{
				// Team members.
				"admin1", "admin2",
			},
			expectReviewers: []string{
				"reviewer1", "reviewer2",
				// Team members.
				"admin1", "admin2",
			},
			expectNeedsLgtm: 3,
		},
		{
			name:         "use GitHub team and has one sig label",
			sigResponses: []SigResponse{sig1Res},
			labels: []github.Label{
				{
					Name: "sig/sig1",
				},
			},
			useGitHubTeam:  true,
			committerTeams: []string{"Committers"},
			reviewerTeams:  []string{"Reviewers"},
			expectCommitters: []string{
				"committer1", "committer2",
			},
			expectReviewers: []string{
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
			expectNeedsLgtm: defaultRequireLgtmNum,
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
				Repos:               []string{"ti-community-infra/test-dev"},
				SigEndpoint:         testServer.URL,
				DefaultRequireLgtm:  tc.defaultRequireLgtm,
				UseGitHubPermission: tc.useGitHubPermission,
				UseGithubTeam:       tc.useGitHubTeam,
			}

			if len(tc.defaultSigName) != 0 {
				repoConfig.DefaultSigName = tc.defaultSigName
			}

			if tc.reviewerTeams != nil {
				repoConfig.ReviewerTeams = tc.reviewerTeams
			}

			if tc.committerTeams != nil {
				repoConfig.CommitterTeams = tc.committerTeams
			}

			if len(tc.requireLgtmLabelPrefix) != 0 {
				repoConfig.RequireLgtmLabelPrefix = tc.requireLgtmLabelPrefix
			}

			if tc.branchesConfig != nil {
				repoConfig.Branches = tc.branchesConfig
			}

			config.TiCommunityOwners = []tiexternalplugins.TiCommunityOwners{
				repoConfig,
			}

			for _, res := range tc.sigResponses {
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

			if tc.membersResponse != nil {
				mux.HandleFunc(MembersEndpoint, func(res http.ResponseWriter, req *http.Request) {
					if req.Method != "GET" {
						t.Errorf("expect 'Get' got '%s'", req.Method)
					}
					reqBodyBytes := new(bytes.Buffer)
					err := json.NewEncoder(reqBodyBytes).Encode(tc.membersResponse)
					if err != nil {
						t.Errorf("Encoding data '%v' failed", tc.membersResponse)
					}

					_, err = res.Write(reqBodyBytes.Bytes())
					if err != nil {
						t.Errorf("Write data '%v' failed", tc.membersResponse)
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
	org := "ti-community-infra"
	repoName := "test-dev"
	sigName := "testing"
	pullNumber := 1
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	testcases := []struct {
		name               string
		labels             []github.Label
		invalidSigInfoData bool
		invalidMembersData bool
		expectError        string
	}{
		{
			name: "has one sig label",
			labels: []github.Label{
				{
					Name: "sig/testing",
				},
			},
			invalidSigInfoData: true,
			expectError:        "unexpected end of JSON input",
		},
		{
			name:               "non sig label",
			labels:             []github.Label{},
			invalidMembersData: true,
			expectError:        "unexpected end of JSON input",
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

			// SIG info URL pattern.
			pattern := fmt.Sprintf(SigEndpointFmt, sigName)
			mux.HandleFunc(pattern, func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("expect 'Get' got '%s'", req.Method)
				}

				if tc.invalidSigInfoData {
					_, err := res.Write([]byte{})
					if err != nil {
						t.Errorf("Write data sig info data failed")
					}
				} else {
					// Just http filed.
					res.WriteHeader(http.StatusInternalServerError)
				}
			})

			mux.HandleFunc(MembersEndpoint, func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("expect 'Get' got '%s'", req.Method)
				}

				if tc.invalidMembersData {
					_, err := res.Write([]byte{})
					if err != nil {
						t.Errorf("Write data members data failed")
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

func TestGetSigNamesByLabel(t *testing.T) {
	testLabel1 := "testLabel1"
	testLabel2 := "testLabel2"
	sig1Label := "sig/testing1"
	sig2Label := "sig/testing2"

	testcases := []struct {
		name           string
		labels         []github.Label
		expectSigNames []string
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
			expectSigNames: []string{"testing1"},
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
			expectSigNames: []string{"testing1", "testing2"},
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
			expectSigNames: nil,
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			sigNames := getSigNamesByLabels(tc.labels)
			// sort the name.
			sort.Strings(sigNames)
			sort.Strings(tc.expectSigNames)

			assert.DeepEqual(t, sigNames, tc.expectSigNames)
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
