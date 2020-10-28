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
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
)

const testOwnersURLFmt = "/%s/community/blob/master/sig/%s/membership.json"

func TestListOwners(t *testing.T) {
	validSigInfo := SigMembersInfo{
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
	}
	collaborators := []string{"collab1", "collab2"}
	org := "tidb-community-bots"
	repoName := "test-dev"
	sigName := "testing"
	pullNumber := 1
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	testCases := []struct {
		name            string
		sigInfo         *SigMembersInfo
		labels          []github.Label
		exceptApprovers []string
		exceptReviewers []string
		exceptNeedsLgtm int
	}{
		{
			name:    "has one sig label",
			sigInfo: &validSigInfo,
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
			sigInfo:         &validSigInfo,
			exceptApprovers: collaborators,
			exceptReviewers: collaborators,
			exceptNeedsLgtm: lgtmTwo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Fake http client.
			mux := http.NewServeMux()
			testServer := httptest.NewServer(mux)
			sigInfoURL = testServer.URL

			// URL pattern.
			pattern := fmt.Sprintf(testOwnersURLFmt, org, sigName)
			mux.HandleFunc(pattern, func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("Except 'Get' got '%s'", req.Method)
				}
				reqBodyBytes := new(bytes.Buffer)
				err := json.NewEncoder(reqBodyBytes).Encode(testCase.sigInfo)
				if err != nil {
					t.Errorf("Encoding data '%v' failed", testCase.sigInfo)
				}

				_, err = res.Write(reqBodyBytes.Bytes())
				if err != nil {
					t.Errorf("Write data '%v' failed", testCase.sigInfo)
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

			res, err := ownersServer.ListOwners(org, repoName, pullNumber)

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
