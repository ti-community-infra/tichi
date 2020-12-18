//nolint:scopelint
package ownersclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testOwnersURLFmt = "/repos/%s/%s/pulls/%d/owners"

func TestLoadOwners(t *testing.T) {
	testCases := []struct {
		name             string
		ownersURL        string
		data             OwnersResponse
		expectCommitters []string
		expectReviewers  []string
		expectNeedsLgtm  int
		expectError      bool
	}{
		{
			name: "valid pull owners URL(use mock URL)",
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
			expectCommitters: []string{
				"Rustin-Liu",
			},
			expectReviewers: []string{
				"Rustin-Liu",
			},
			expectNeedsLgtm: 2,
		},
		{
			name:      "invalid pull owners URL",
			ownersURL: "not-found",
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
			expectCommitters: []string{
				"Rustin-Liu",
			},
			expectReviewers: []string{
				"Rustin-Liu",
			},
			expectNeedsLgtm: 2,
			expectError:     true,
		},
	}
	org := "ti-community-infra"
	repoName := "test-dev"
	number := 1

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Fake http client.
			mux := http.NewServeMux()
			testServer := httptest.NewServer(mux)

			// NOTICE: add pull owners URL.
			if testCase.ownersURL == "" {
				testCase.ownersURL = testServer.URL
			}

			// URL pattern.
			pattern := fmt.Sprintf(testOwnersURLFmt, org, repoName, number)
			mux.HandleFunc(pattern, func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("expect 'Get' got '%s'", req.Method)
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

			owners, err := client.LoadOwners(testCase.ownersURL, org, repoName, number)
			if err != nil {
				if !testCase.expectError {
					t.Errorf("unexpected error: '%v'", err)
				} else {
					// It should have a error, so skip follow assert.
					return
				}
			}

			if len(owners.Committers) != len(testCase.expectCommitters) {
				t.Errorf("Different committers: Got \"%v\" expected \"%v\"", owners.Committers, testCase.expectCommitters)
			}

			if len(owners.Reviewers) != len(testCase.expectReviewers) {
				t.Errorf("Different reviewers: Got \"%v\" expected \"%v\"", owners.Reviewers, testCase.expectReviewers)
			}

			if owners.NeedsLgtm != testCase.expectNeedsLgtm {
				t.Errorf("Different LGTM: Got \"%v\" expected \"%v\"", owners.NeedsLgtm, testCase.expectNeedsLgtm)
			}

			testServer.Close()
		})
	}
}

func TestLoadOwnersFailed(t *testing.T) {
	testCases := []struct {
		name        string
		ownersURL   string
		invalidData bool
		expectError string
	}{
		{
			name:        "get data form url failed(use mock URL)",
			expectError: "could not get a owners",
		},
		{
			name:        "parse data failed(use mock URL)",
			invalidData: true,
			expectError: "unexpected end of JSON input",
		},
	}
	org := "ti-community-infra"
	repoName := "test-dev"
	number := 1

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Fake http client.
			mux := http.NewServeMux()
			testServer := httptest.NewServer(mux)

			// Notice: use mock server URL.
			if testCase.ownersURL == "" {
				testCase.ownersURL = testServer.URL
			}

			// URL pattern.
			pattern := fmt.Sprintf(testOwnersURLFmt, org, repoName, number)
			mux.HandleFunc(pattern, func(res http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("expect 'Get' got '%s'", req.Method)
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

			_, err := client.LoadOwners(testCase.ownersURL, org, repoName, number)
			if err == nil {
				t.Errorf("expected error '%v', but it is nil", testCase.expectError)
			} else if err.Error() != testCase.expectError {
				t.Errorf("expected error '%v', but it is '%v'", testCase.expectError, err)
			}

			testServer.Close()
		})
	}
}
