//nolint:scopelint,dupl
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
				Repos:              []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "http pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "invalid pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "http/bots.tidb.io/ti-community-bot",
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
				Repos:              []string{"tidb-community-bots/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "http pull owners URL",
			merge: &TiCommunityMerge{
				Repos:              []string{"tidb-community-bots/test-dev"},
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "invalid pull owners URL",
			merge: &TiCommunityMerge{
				Repos:              []string{"tidb-community-bots/test-dev"},
				PullOwnersEndpoint: "http/bots.tidb.io/ti-community-bot",
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
		owners   *TiCommunityOwners
		expected error
	}{
		{
			name: "https pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"tidb-community-bots/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"tidb-community-bots/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot"},
			expected: nil,
		},
		{
			name: "http pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"tidb-community-bots/test-dev"},
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"tidb-community-bots/test-dev"},
				SigEndpoint: "http://bots.tidb.io/ti-community-bot"},
			expected: nil,
		},
		{
			name: "invalid lgtm pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "http/bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"tidb-community-bots/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"tidb-community-bots/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot"},
			expected: fmt.Errorf("parse \"http/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
		{
			name: "invalid merge pull owners URL",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"tidb-community-bots/test-dev"},
				PullOwnersEndpoint: "http/bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"tidb-community-bots/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot"},
			expected: fmt.Errorf("parse \"http/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
		{
			name: "invalid owners sig endpoint",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"tidb-community-bots/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"tidb-community-bots/test-dev"},
				SigEndpoint: "https/bots.tidb.io/ti-community-bot"},
			expected: fmt.Errorf("parse \"https/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityLgtm: []TiCommunityLgtm{
				*tc.lgtm,
			}, TiCommunityMerge: []TiCommunityMerge{
				*tc.merge,
			}, TiCommunityOwners: []TiCommunityOwners{
				*tc.owners,
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
				Repos:              []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name: "Only org",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"tidb-community-bots"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
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
				Repos:              []string{"tidb-community-bots/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name: "Only org",
			merge: &TiCommunityMerge{
				Repos:              []string{"tidb-community-bots"},
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
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

func TestOwnersFor(t *testing.T) {
	testCases := []struct {
		name        string
		owners      *TiCommunityOwners
		org         string
		repo        string
		expectEmpty *TiCommunityOwners
	}{
		{
			name: "Full name",
			owners: &TiCommunityOwners{
				Repos:       []string{"tidb-community-bots/test-dev"},
				SigEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name: "Only org",
			owners: &TiCommunityOwners{
				Repos:       []string{"tidb-community-bots"},
				SigEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name:        "Can not find",
			owners:      &TiCommunityOwners{},
			org:         "tidb-community-bots1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityOwners{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityOwners: []TiCommunityOwners{
				*tc.owners,
			}}

			owners := config.OwnersFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, owners, &TiCommunityOwners{})
			} else {
				assert.DeepEqual(t, owners.Repos, tc.owners.Repos)
			}
		})
	}
}

func TestLabelFor(t *testing.T) {
	testCases := []struct {
		name        string
		label       *TiCommunityLabel
		org         string
		repo        string
		expectEmpty *TiCommunityLabel
	}{
		{
			name: "Full name",
			label: &TiCommunityLabel{
				Repos:    []string{"tidb-community-bots/test-dev"},
				Prefixes: []string{"status"},
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name: "Only org",
			label: &TiCommunityLabel{
				Repos:    []string{"tidb-community-bots"},
				Prefixes: []string{"status"},
			},
			org:  "tidb-community-bots",
			repo: "test-dev",
		},
		{
			name:        "Can not find",
			label:       &TiCommunityLabel{},
			org:         "tidb-community-bots1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityLabel{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityLabel: []TiCommunityLabel{
				*tc.label,
			}}

			label := config.LabelFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, label, &TiCommunityLabel{})
			} else {
				assert.DeepEqual(t, label.Repos, tc.label.Repos)
			}
		})
	}
}
