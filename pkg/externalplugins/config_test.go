//nolint:scopelint
package externalplugins

import (
	"fmt"
	"testing"
)

func TestValidateConfigUpdater(t *testing.T) {
	testCases := []struct {
		name     string
		lgtm     *TiCommunityLgtm
		expected error
	}{
		{
			name: "https pull reviewers URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullReviewersURL: "https://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "http pull reviewers URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullReviewersURL: "http://bots.tidb.io/ti-community-bot",
			},
			expected: nil,
		},
		{
			name: "invalid pull reviewers URL",
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullReviewersURL: "http/bots.tidb.io/ti-community-bot",
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
