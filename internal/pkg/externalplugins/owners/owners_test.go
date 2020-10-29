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
	"k8s.io/test-infra/prow/github/fakegithub"
)

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

	collaborators := []string{"collab1", "collab2"}
	org := "tidb-community-bots"
	repoName := "test-dev"
	sigName := "testing"
	pullNumber := 1
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	testCases := []struct {
		name              string
		sigRes            *SigResponse
		labels            []github.Label
		useDefaultSigName bool
		exceptApprovers   []string
		exceptReviewers   []string
		exceptNeedsLgtm   int
	}{
		{
			name:   "has one sig label",
			sigRes: &validSigRes,
			labels: []github.Label{
				{
					Name: "sig/testing",
				},
			},
			exceptApprovers: []string{
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
			name:            "non sig label",
			sigRes:          &validSigRes,
			exceptApprovers: collaborators,
			exceptReviewers: collaborators,
			exceptNeedsLgtm: lgtmTwo,
		},
		{
			name:              "non sig label but use default sig name",
			sigRes:            &validSigRes,
			useDefaultSigName: true,
			exceptApprovers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2",
			},
			exceptReviewers: []string{
				"leader1", "leader2", "coLeader1", "coLeader2",
				"committer1", "committer2", "reviewer1", "reviewer2",
			},
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
				Repos:       []string{"tidb-community-bots/test-dev"},
				SigEndpoint: testServer.URL,
			}

			if testCase.useDefaultSigName {
				owners.DefaultSigName = sigName
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

			fc := &fakegithub.FakeClient{
				IssueComments: make(map[int][]github.IssueComment),
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
				PullRequestChanges: map[int][]github.PullRequestChange{
					pullNumber: {
						{Filename: "doc/README.md"},
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

			if len(res.Data.Approvers) != len(testCase.exceptApprovers) {
				t.Errorf("Different approvers: Got \"%v\" expected \"%v\"", res.Data.Approvers, testCase.exceptApprovers)
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
	collaborators := []string{"collab1", "collab2"}
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

			fc := &fakegithub.FakeClient{
				IssueComments: make(map[int][]github.IssueComment),
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
				PullRequestChanges: map[int][]github.PullRequestChange{
					pullNumber: {
						{Filename: "doc/README.md"},
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
			sigName := GetSigNameByLabel(testCase.labels)

			if sigName != testCase.exceptSigName {
				t.Errorf("expected sig '%s', but it is '%s'", testCase.exceptSigName, sigName)
			}
		})
	}
}
