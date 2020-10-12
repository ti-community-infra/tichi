package externalplugins

import "testing"

func TestGetConfig(t *testing.T) {
	var testcases = []struct {
		lgtm                     *TiCommunityLgtm
		expectedPullReviewersURL string
	}{
		{
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullReviewersURL: "https://bots.tidb.io/ti-community-bot",
			},
			expectedPullReviewersURL: "https://bots.tidb.io/ti-community-bot",
		},
		{
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-live"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullReviewersURL: "https://bots.tidb.io/ti-community-bot",
			},
			expectedPullReviewersURL: "https://bots.tidb.io/ti-community-bot",
		},
	}
	for _, tc := range testcases {
		pa := ConfigAgent{configuration: &Configuration{TiCommunityLgtm: []TiCommunityLgtm{*tc.lgtm}}}

		config := pa.Config()
		for _, lgtm := range config.TiCommunityLgtm {
			if lgtm.PullReviewersURL != tc.expectedPullReviewersURL {
				t.Errorf("Different URL: Got \"%s\" expected \"%s\"", lgtm.PullReviewersURL, tc.expectedPullReviewersURL)
			}
		}
	}
}
