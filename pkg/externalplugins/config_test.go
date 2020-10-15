//nolint:scopelint
package externalplugins

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func TestValidateTiCommunityLgtmConfig(t *testing.T) {
	testCases := []struct {
		name     string
		lgtm     *TiCommunityLgtm
		expected error
	}{
		{
			name: "https pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "https://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "http pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "http://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "invalid pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "http/bots.tidb.io/ti-community-bot",
			},
			expected: fmt.Errorf("parse \"http/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := validateLgtm([]TiCommunityLgtm{*tc.lgtm})

			if tc.expected == nil && actual != nil {
				t.Errorf("unexpected error: '%v'", actual)
			}
			if tc.expected != nil && actual == nil {
				t.Errorf("expected error '%v'', but it is nil", tc.expected)
			}
			if tc.expected != nil && actual != nil && tc.expected.Error() != actual.Error() {
				t.Errorf("expected error '%v', but it is '%v'", tc.expected, actual)
			}
		})
	}
}

func TestValidateApproveConfig(t *testing.T) {
	testCases := []struct {
		name     string
		approve  *Approve
		expected error
	}{
		{
			name: "https pull owners URL",
			approve: &Approve{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "https://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "http pull owners URL",
			approve: &Approve{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "http://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "invalid pull owners URL",
			approve: &Approve{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "http/bots.tidb.io/ti-community-bot",
			},
			expected: fmt.Errorf("parse \"http/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := validateApprove([]Approve{*tc.approve})

			if tc.expected == nil && actual != nil {
				t.Errorf("unexpected error: '%v'", actual)
			}
			if tc.expected != nil && actual == nil {
				t.Errorf("expected error '%v'', but it is nil", tc.expected)
			}
			if tc.expected != nil && actual != nil && tc.expected.Error() != actual.Error() {
				t.Errorf("expected error '%v', but it is '%v'", tc.expected, actual)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name     string
		lgtm     *TiCommunityLgtm
		approve  *Approve
		expected error
	}{
		{
			name: "https pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "https://bots.tidb.io/ti-community-bot",
			},
			approve: &Approve{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "https://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "http pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "http://bots.tidb.io/ti-community-bot",
			},
			approve: &Approve{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "http://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "invalid lgtm pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "http/bots.tidb.io/ti-community-bot",
			},
			approve: &Approve{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "https://bots.tidb.io/ti-community-bot",
			},
			expected: fmt.Errorf("parse \"http/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
		{
			name: "invalid approve pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "https://bots.tidb.io/ti-community-bot",
			},
			approve: &Approve{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "http/bots.tidb.io/ti-community-bot",
			},
			expected: fmt.Errorf("parse \"http/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityLgtm: []TiCommunityLgtm{
				*tc.lgtm,
			}, Approve: []Approve{
				*tc.approve,
			}}
			actual := config.Validate()

			if tc.expected == nil && actual != nil {
				t.Errorf("unexpected error: '%v'", actual)
			}
			if tc.expected != nil && actual == nil {
				t.Errorf("expected error '%v'', but it is nil", tc.expected)
			}
			if tc.expected != nil && actual != nil && tc.expected.Error() != actual.Error() {
				t.Errorf("expected error '%v', but it is '%v'", tc.expected, actual)
			}
		})
	}
}

func TestLgtmFor(t *testing.T) {
	testCases := []struct {
		name string
		lgtm *TiCommunityLgtm
		org  string
		repo string
	}{
		{
			name: "Full name",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "https://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name: "Only org",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "http://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityLgtm: []TiCommunityLgtm{
				*tc.lgtm,
			}}
			lgtm := config.LgtmFor(tc.org, tc.repo)
			assert.DeepEqual(t, lgtm.Repos, tc.lgtm.Repos)
		})
	}
}

func TestApproveFor(t *testing.T) {
	testCases := []struct {
		name        string
		approve     *Approve
		org         string
		repo        string
		expectEmpty *Approve
	}{
		{
			name: "Full name",
			approve: &Approve{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "https://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name: "Only org",
			approve: &Approve{
				Repos:         []string{"tidb-community-bots"},
				PullOwnersURL: "http://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name:        "Can not find",
			approve:     &Approve{},
			org:         "tidb-community-bots1",
			repo:        "test-dev1",
			expectEmpty: &Approve{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{Approve: []Approve{
				*tc.approve,
			}}

			approve := config.ApproveFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, approve, &Approve{CommandHelpLink: "https://go.k8s.io/bot-commands",
					PrProcessLink: "https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process"})
			} else {
				assert.DeepEqual(t, approve.Repos, tc.approve.Repos)
			}
		})
	}
}

func TestHasSelfApproval(t *testing.T) {
	testCases := []struct {
		name                string
		requireSelfApproval bool
		approve             *Approve
		org                 string
		repo                string
	}{
		{
			name: "default self approval",
			approve: &Approve{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "https://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name:                "self approval",
			requireSelfApproval: true,
			approve: &Approve{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "https://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name:                "do not self approval",
			requireSelfApproval: false,
			approve: &Approve{
				Repos:         []string{"tidb-community-bots"},
				PullOwnersURL: "http://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.approve.RequireSelfApproval = &tc.requireSelfApproval

			actual := tc.approve.HasSelfApproval()
			if !tc.requireSelfApproval != actual {
				t.Errorf("expected '%v', but it is '%v'", !tc.requireSelfApproval, actual)
			}
		})
	}
}

func TestConsiderReviewState(t *testing.T) {
	testCases := []struct {
		name              string
		ignoreReviewState bool
		approve           *Approve
		org               string
		repo              string
	}{
		{
			name:    "default consider review",
			approve: &Approve{},
			org:     "tidb-community-bots",
			repo:    "test-dev",
		},
		{
			name:              "ignore review",
			ignoreReviewState: true,
			approve:           &Approve{},
			org:               "tidb-community-bots",
			repo:              "test-dev",
		},
		{
			name:              "do not ignore review",
			ignoreReviewState: false,
			approve:           &Approve{},
			org:               "tidb-community-bots",
			repo:              "test-dev",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.approve.IgnoreReviewState = &tc.ignoreReviewState

			actual := tc.approve.ConsiderReviewState()
			if !tc.ignoreReviewState != actual {
				t.Errorf("expected '%v', but it is '%v'", !tc.ignoreReviewState, actual)
			}
		})
	}
}
