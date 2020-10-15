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
				t.Errorf("expected error '%v', but it is nil", tc.expected)
			}
			if tc.expected != nil && actual != nil && tc.expected.Error() != actual.Error() {
				t.Errorf("expected error '%v', but it is '%v'", tc.expected, actual)
			}
		})
	}
}

func TestValidateMergeConfig(t *testing.T) {
	testCases := []struct {
		name     string
		merge    *TiCommunityMerge
		expected error
	}{
		{
			name: "https pull owners URL",
			merge: &TiCommunityMerge{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "https://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "http pull owners URL",
			merge: &TiCommunityMerge{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "http://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "invalid pull owners URL",
			merge: &TiCommunityMerge{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "http/bots.tidb.io/ti-community-bot",
			},
			expected: fmt.Errorf("parse \"http/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := validateMerge([]TiCommunityMerge{*tc.merge})

			if tc.expected == nil && actual != nil {
				t.Errorf("unexpected error: '%v'", actual)
			}
			if tc.expected != nil && actual == nil {
				t.Errorf("expected error '%v', but it is nil", tc.expected)
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
		merge    *TiCommunityMerge
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
			merge: &TiCommunityMerge{
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
			merge: &TiCommunityMerge{
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
			merge: &TiCommunityMerge{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "https://bots.tidb.io/ti-community-bot",
			},
			expected: fmt.Errorf("parse \"http/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
		{
			name: "invalid merge pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullOwnersURL:    "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
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
			}, TiCommunityMerge: []TiCommunityMerge{
				*tc.merge,
			}}
			actual := config.Validate()

			if tc.expected == nil && actual != nil {
				t.Errorf("unexpected error: '%v'", actual)
			}
			if tc.expected != nil && actual == nil {
				t.Errorf("expected error '%v', but it is nil", tc.expected)
			}
			if tc.expected != nil && actual != nil && tc.expected.Error() != actual.Error() {
				t.Errorf("expected error '%v', but it is '%v'", tc.expected, actual)
			}
		})
	}
}

func TestLgtmFor(t *testing.T) {
	testCases := []struct {
		name        string
		lgtm        *TiCommunityLgtm
		org         string
		repo        string
		expectEmpty *TiCommunityLgtm
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
		{
			name:        "Can not find",
			lgtm:        &TiCommunityLgtm{},
			org:         "tidb-community-bots1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityLgtm{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityLgtm: []TiCommunityLgtm{
				*tc.lgtm,
			}}
			lgtm := config.LgtmFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, lgtm, &TiCommunityLgtm{})
			} else {
				assert.DeepEqual(t, lgtm.Repos, tc.lgtm.Repos)
			}
		})
	}
}

func TestMergeFor(t *testing.T) {
	testCases := []struct {
		name        string
		merge       *TiCommunityMerge
		org         string
		repo        string
		expectEmpty *TiCommunityMerge
	}{
		{
			name: "Full name",
			merge: &TiCommunityMerge{
				Repos:         []string{"tidb-community-bots/test-dev"},
				PullOwnersURL: "https://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name: "Only org",
			merge: &TiCommunityMerge{
				Repos:         []string{"tidb-community-bots"},
				PullOwnersURL: "http://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name:        "Can not find",
			merge:       &TiCommunityMerge{},
			org:         "tidb-community-bots1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityMerge{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityMerge: []TiCommunityMerge{
				*tc.merge,
			}}

			merge := config.MergeFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, merge, &TiCommunityMerge{})
			} else {
				assert.DeepEqual(t, merge.Repos, tc.merge.Repos)
			}
		})
	}
}
