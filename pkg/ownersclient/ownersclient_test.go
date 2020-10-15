//nolint:scopelint
package ownersclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tidb-community-bots/ti-community-prow/pkg/externalplugins"
)

const testOwnersURLFmt = "/repos/%s/%s/pulls/%d/owners"

func TestLoadOwners(t *testing.T) {
	testCases := []struct {
		name             string
		lgtm             *externalplugins.TiCommunityLgtm
		data             OwnersResponse
		exceptCommitters []string
		exceptReviewers  []string
		exceptNeedsLgtm  int
		exceptError      bool
	}{
		{
			name: "valid pull owners URL(use mock URL)",
			lgtm: &externalplugins.TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
			},
			data: OwnersResponse{
				Data: Owners{
					Committers: []string{
						"Rustin-Liu",
					},
					Reviewers: []string{
						"Rustin-Liu",
					},
					NeedsLgtm: 2,
				},
				Message: "Test",
			},
			exceptCommitters: []string{
				"Rustin-Liu",
			},
			exceptReviewers: []string{
				"Rustin-Liu",
			},
			exceptNeedsLgtm: 2,
		},
		{
			name: "invalid pull owners URL",
			lgtm: &externalplugins.TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "not-found",
			},
			data: OwnersResponse{
				Data: Owners{
					Committers: []string{
						"Rustin-Liu",
					},
					Reviewers: []string{
						"Rustin-Liu",
					},
					NeedsLgtm: 2,
				},
				Message: "Test",
			},
			exceptCommitters: []string{
				"Rustin-Liu",
			},
			exceptReviewers: []string{
				"Rustin-Liu",
			},
			exceptNeedsLgtm: 2,
			exceptError:     true,
		},
	}
	org := "tidb-community-bots"
	repoName := "test-dev"
	number := 1

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Fake http client.
			mux := http.NewServeMux()
			testServer := httptest.NewServer(mux)

			// NOTICE: add pull owners URL.
			if testCase.lgtm.PullOwnersURL == "" {
				testCase.lgtm.PullOwnersURL = testServer.URL
			}

			// URL pattern.
			pattern := fmt.Sprintf(testOwnersURLFmt, org, repoName, number)
			mux.HandleFunc(pattern, func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("Except 'Get' got '%s'", req.Method)
				}
				reqBodyBytes := new(bytes.Buffer)
				err := json.NewEncoder(reqBodyBytes).Encode(testCase.data)
				if err != nil {
					t.Errorf("Encoding data '%v' failed", testCase.data)
				}

				_, err = res.Write(reqBodyBytes.Bytes())
				if err != nil {
					t.Errorf("Write data '%v' failed", testCase.data)
				}
			})

			client := OwnersClient{Client: testServer.Client()}

			owners, err := client.LoadOwners(testCase.lgtm, org, repoName, number)
			if err != nil {
				if !testCase.exceptError {
					t.Errorf("unexpected error: '%v'", err)
				} else {
					// It should have a error, so skip follow assert.
					return
				}
			}

			if len(owners.Committers) != len(testCase.exceptCommitters) {
				t.Errorf("Different committers: Got \"%v\" expected \"%v\"", owners.Committers, testCase.exceptCommitters)
			}

			if len(owners.Reviewers) != len(testCase.exceptReviewers) {
				t.Errorf("Different reviewers: Got \"%v\" expected \"%v\"", owners.Reviewers, testCase.exceptReviewers)
			}

			if owners.NeedsLgtm != testCase.exceptNeedsLgtm {
				t.Errorf("Different LGTM: Got \"%v\" expected \"%v\"", owners.NeedsLgtm, testCase.exceptNeedsLgtm)
			}

			testServer.Close()
		})
	}
}

func TestLoadOwnersFailed(t *testing.T) {
	testCases := []struct {
		name        string
		lgtm        *externalplugins.TiCommunityLgtm
		invalidData bool
		exceptError string
	}{
		{
			name: "get data form url failed(use mock URL)",
			lgtm: &externalplugins.TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
			},
			exceptError: "could not get a owners",
		},
		{
			name: "parse data failed(use mock URL)",
			lgtm: &externalplugins.TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
			},
			invalidData: true,
			exceptError: "unexpected end of JSON input",
		},
	}
	org := "tidb-community-bots"
	repoName := "test-dev"
	number := 1

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Fake http client.
			mux := http.NewServeMux()
			testServer := httptest.NewServer(mux)

			// Notice: use mock server URL.
			if testCase.lgtm.PullOwnersURL == "" {
				testCase.lgtm.PullOwnersURL = testServer.URL
			}

			// URL pattern.
			pattern := fmt.Sprintf(testOwnersURLFmt, org, repoName, number)
			mux.HandleFunc(pattern, func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("Except 'Get' got '%s'", req.Method)
				}
				// If set invalid data true, we need response a invalid data.
				if testCase.invalidData {
					_, err := res.Write([]byte{})
					if err != nil {
						t.Errorf("Write data '%v' failed", []byte{})
					}
				} else {
					// Just http filed.
					res.WriteHeader(http.StatusInternalServerError)
				}
			})

			client := OwnersClient{Client: testServer.Client()}

			_, err := client.LoadOwners(testCase.lgtm, org, repoName, number)
			if err == nil {
				t.Errorf("expected error '%v', but it is nil", testCase.exceptError)
			} else if err.Error() != testCase.exceptError {
				t.Errorf("expected error '%v', but it is '%v'", testCase.exceptError, err)
			}

			testServer.Close()
		})
	}
}
