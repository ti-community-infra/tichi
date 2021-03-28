//nolint:dupl
package externalplugins

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func TestValidateConfig(t *testing.T) {
	testcases := []struct {
		name            string
		tichiWebURL     string
		prProcessLink   string
		commandHelpLink string
		logLevel        string
		lgtm            *TiCommunityLgtm
		merge           *TiCommunityMerge
		owners          *TiCommunityOwners
		labelBlocker    *TiCommunityLabelBlocker
		autoresponders  *TiCommunityAutoresponder
		blunderbuss     *TiCommunityBlunderbuss

		expected error
	}{
		{
			name:            "https pull owners URL",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: nil,
		},
		{
			name:            "http pull owners URL",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: nil,
		},
		{
			name:            "invalid lgtm pull owners URL",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "http/bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("parse \"http/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
		{
			name:            "invalid merge pull owners URL",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "http/bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("parse \"http/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
		{
			name:            "invalid owners sig endpoint",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https/bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("parse \"https/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
		{
			name:            "invalid blunderbuss regex",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   "?[)",
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("error parsing regexp: missing argument to repetition operator: `?`"),
		},
		{
			name:            "invalid blunderbuss pull owners",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https/bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("parse \"https/bots.tidb.io/ti-community-bot\": invalid URI for request"),
		},
		{
			name:            "invalid blunderbuss max reviewer count",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   -1,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("max reviewer count must more than 0"),
		},
		{
			name:            "invalid blunderbuss grace period duration",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
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
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"tidb-community-bots/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:               []string{"tidb-community-bots/test-dev"},
				MaxReviewerCount:    2,
				ExcludeReviewers:    []string{},
				PullOwnersEndpoint:  "https://bots.tidb.io/ti-community-bot",
				GracePeriodDuration: -1,
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("grace period duration must not less than 0"),
		},
		{
			name:            "invalid tichiWebURL",
			tichiWebURL:     "https//tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("parse \"https//tichiWebURL\": invalid URI for request"),
		},
		{
			name:            "invalid prProcessLink",
			tichiWebURL:     "https://tichiWebURL",
			prProcessLink:   "https//prProcessLink",
			commandHelpLink: "https://commandHelpLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("parse \"https//prProcessLink\": invalid URI for request"),
		},
		{
			name:            "invalid commandHelpLink",
			tichiWebURL:     "https://tichiWebURL",
			prProcessLink:   "https://prProcessLink",
			commandHelpLink: "https//commandHelpLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("parse \"https//commandHelpLink\": invalid URI for request"),
		},
		{
			name:            "invalid label blocker regex",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:   "?[)",
						Actions: []string{"labeled", "unlabeled"},
					},
				},
			},
			expected: fmt.Errorf("error parsing regexp: missing argument to repetition operator: `?`"),
		},
		{
			name:            "invalid empty actions",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:   `^status/can-merge$`,
						Actions: []string{},
					},
				},
			},
			expected: fmt.Errorf("there must be at least one action"),
		},
		{
			name:            "invalid action value",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:   `^status/can-merge$`,
						Actions: []string{"nop"},
					},
				},
			},
			expected: fmt.Errorf("actions contain illegal value nop"),
		},
		{
			name:            "invalid log level",
			tichiWebURL:     "https://tichiWebURL",
			commandHelpLink: "https://commandHelpLink",
			prProcessLink:   "https://prProcessLink",
			logLevel:        "nop",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			autoresponders: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex:        `^status/can-merge$`,
						Actions:      []string{"labeled", "unlabeled"},
						TrustedTeams: []string{"release-team"},
						TrustedUsers: []string{"ti-chi-bot"},
					},
				},
			},
			expected: fmt.Errorf("not a valid logrus Level: \"nop\""),
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{
				TichiWebURL:     tc.tichiWebURL,
				PRProcessLink:   tc.prProcessLink,
				CommandHelpLink: tc.commandHelpLink,
				LogLevel:        tc.logLevel,
				TiCommunityLgtm: []TiCommunityLgtm{
					*tc.lgtm,
				},
				TiCommunityMerge: []TiCommunityMerge{
					*tc.merge,
				},
				TiCommunityOwners: []TiCommunityOwners{
					*tc.owners,
				},
				TiCommunityAutoresponder: []TiCommunityAutoresponder{
					*tc.autoresponders,
				},
				TiCommunityBlunderbuss: []TiCommunityBlunderbuss{
					*tc.blunderbuss,
				},
				TiCommunityLabelBlocker: []TiCommunityLabelBlocker{
					*tc.labelBlocker,
				},
			}
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
	testcases := []struct {
		name        string
		lgtm        *TiCommunityLgtm
		org         string
		repo        string
		expectEmpty *TiCommunityLgtm
	}{
		{
			name: "Full name",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Only org",
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra"},
				ReviewActsAsLgtm:   true,
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name:        "Can not find",
			lgtm:        &TiCommunityLgtm{},
			org:         "ti-community-infra1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityLgtm{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
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
	testcases := []struct {
		name        string
		merge       *TiCommunityMerge
		org         string
		repo        string
		expectEmpty *TiCommunityMerge
	}{
		{
			name: "Full name",
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Only org",
			merge: &TiCommunityMerge{
				Repos:              []string{"ti-community-infra"},
				PullOwnersEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name:        "Can not find",
			merge:       &TiCommunityMerge{},
			org:         "ti-community-infra1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityMerge{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
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
	testcases := []struct {
		name        string
		owners      *TiCommunityOwners
		org         string
		repo        string
		expectEmpty *TiCommunityOwners
	}{
		{
			name: "Full name",
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra/test-dev"},
				SigEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Only org",
			owners: &TiCommunityOwners{
				Repos:       []string{"ti-community-infra"},
				SigEndpoint: "http://bots.tidb.io/ti-community-bot",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name:        "Can not find",
			owners:      &TiCommunityOwners{},
			org:         "ti-community-infra1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityOwners{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
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
	testcases := []struct {
		name        string
		label       *TiCommunityLabel
		org         string
		repo        string
		expectEmpty *TiCommunityLabel
	}{
		{
			name: "Full name",
			label: &TiCommunityLabel{
				Repos:    []string{"ti-community-infra/test-dev"},
				Prefixes: []string{"status"},
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Only org",
			label: &TiCommunityLabel{
				Repos:    []string{"ti-community-infra"},
				Prefixes: []string{"status"},
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name:        "Can not find",
			label:       &TiCommunityLabel{},
			org:         "ti-community-infra1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityLabel{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
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

func TestAutoresponderFor(t *testing.T) {
	testcases := []struct {
		name          string
		autoresponder *TiCommunityAutoresponder
		org           string
		repo          string
		expectEmpty   *TiCommunityAutoresponder
	}{
		{
			name: "Full name",
			autoresponder: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra/test-dev"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Only org",
			autoresponder: &TiCommunityAutoresponder{
				Repos: []string{"ti-community-infra"},
				AutoResponds: []AutoRespond{
					{
						Regex:   `(?mi)^/merge\s*$`,
						Message: "/run-all-test",
					},
				},
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name:          "Can not find",
			autoresponder: &TiCommunityAutoresponder{},
			org:           "ti-community-infra1",
			repo:          "test-dev1",
			expectEmpty:   &TiCommunityAutoresponder{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityAutoresponder: []TiCommunityAutoresponder{
				*tc.autoresponder,
			}}

			autoresponder := config.AutoresponderFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, autoresponder, &TiCommunityAutoresponder{})
			} else {
				assert.DeepEqual(t, autoresponder.Repos, tc.autoresponder.Repos)
			}
		})
	}
}

func TestBlunderbussFor(t *testing.T) {
	testcases := []struct {
		name        string
		blunderbuss *TiCommunityBlunderbuss
		org         string
		repo        string
		expectEmpty *TiCommunityBlunderbuss
	}{
		{
			name: "Full name",
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra/test-dev"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Only org",
			blunderbuss: &TiCommunityBlunderbuss{
				Repos:              []string{"ti-community-infra"},
				MaxReviewerCount:   2,
				ExcludeReviewers:   []string{},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name:        "Can not find",
			blunderbuss: &TiCommunityBlunderbuss{},
			org:         "ti-community-infra1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityBlunderbuss{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityBlunderbuss: []TiCommunityBlunderbuss{
				*tc.blunderbuss,
			}}

			blunderbuss := config.BlunderbussFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, blunderbuss, &TiCommunityBlunderbuss{})
			} else {
				assert.DeepEqual(t, blunderbuss.Repos, tc.blunderbuss.Repos)
			}
		})
	}
}

func TestTarsFor(t *testing.T) {
	testcases := []struct {
		name        string
		tars        *TiCommunityTars
		org         string
		repo        string
		expectEmpty *TiCommunityTars
	}{
		{
			name: "Full name",
			tars: &TiCommunityTars{
				Repos:         []string{"ti-community-infra/test-dev"},
				Message:       "updated",
				OnlyWhenLabel: "trigger-update",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Only org",
			tars: &TiCommunityTars{
				Repos:         []string{"ti-community-infra"},
				Message:       "updated",
				OnlyWhenLabel: "trigger-update",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name:        "Can not find",
			tars:        &TiCommunityTars{},
			org:         "ti-community-infra1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityTars{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityTars: []TiCommunityTars{
				*tc.tars,
			}}

			tars := config.TarsFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, tars, &TiCommunityTars{})
			} else {
				assert.DeepEqual(t, tars.Repos, tc.tars.Repos)
			}
		})
	}
}

func TestSetBlunderbussDefaults(t *testing.T) {
	testcases := []struct {
		name                      string
		gracePeriodDuration       int
		expectGracePeriodDuration int
	}{
		{
			name:                      "default",
			gracePeriodDuration:       0,
			expectGracePeriodDuration: 5,
		},
		{
			name:                      "overwrite",
			gracePeriodDuration:       3,
			expectGracePeriodDuration: 3,
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			c := &Configuration{
				TiCommunityBlunderbuss: []TiCommunityBlunderbuss{
					{
						GracePeriodDuration: tc.gracePeriodDuration,
					},
				},
			}

			c.setDefaults()

			for _, blunderbuss := range c.TiCommunityBlunderbuss {
				if blunderbuss.GracePeriodDuration != tc.expectGracePeriodDuration {
					t.Errorf("unexpected grace_period_duration: %v, expected: %v",
						blunderbuss.GracePeriodDuration, tc.expectGracePeriodDuration)
				}
			}
		})
	}
}

func TestLabelBlockerFor(t *testing.T) {
	testcases := []struct {
		name         string
		labelBlocker *TiCommunityLabelBlocker
		org          string
		repo         string
		expectEmpty  *TiCommunityLabelBlocker
	}{
		{
			name: "Full name",
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra/test-dev"},
				BlockLabels: []BlockLabel{
					{
						Regex: `^status/can-merge$`,
					},
				},
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Only org",
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra"},
				BlockLabels: []BlockLabel{
					{
						Regex: `^status/can-merge$`,
					},
				},
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Can not find",
			labelBlocker: &TiCommunityLabelBlocker{
				Repos: []string{"ti-community-infra"},
				BlockLabels: []BlockLabel{
					{
						Regex: `^status/can-merge$`,
					},
				},
			},
			org:         "ti-community-infra1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityLabelBlocker{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityLabelBlocker: []TiCommunityLabelBlocker{
				*tc.labelBlocker,
			}}

			labelBlocker := config.LabelBlockerFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, labelBlocker, &TiCommunityLabelBlocker{})
			} else {
				assert.DeepEqual(t, labelBlocker.Repos, tc.labelBlocker.Repos)
			}
		})
	}
}

func TestContributionFor(t *testing.T) {
	testcases := []struct {
		name         string
		contribution *TiCommunityContribution
		org          string
		repo         string
		expectEmpty  *TiCommunityContribution
	}{
		{
			name: "Full name",
			contribution: &TiCommunityContribution{
				Repos:   []string{"ti-community-infra/test-dev"},
				Message: "message",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Only org",
			contribution: &TiCommunityContribution{
				Repos:   []string{"ti-community-infra"},
				Message: "message",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Can not find",
			contribution: &TiCommunityContribution{
				Repos:   []string{"ti-community-infra"},
				Message: "message",
			},
			org:         "ti-community-infra1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityContribution{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityContribution: []TiCommunityContribution{
				*tc.contribution,
			}}

			contribution := config.ContributionFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, contribution, &TiCommunityContribution{})
			} else {
				assert.DeepEqual(t, contribution.Repos, tc.contribution.Repos)
			}
		})
	}
}

func TestCherrypickerFor(t *testing.T) {
	testcases := []struct {
		name         string
		cherrypicker *TiCommunityCherrypicker
		org          string
		repo         string
		expectEmpty  *TiCommunityCherrypicker
	}{
		{
			name: "Full name",
			cherrypicker: &TiCommunityCherrypicker{
				Repos:       []string{"ti-community-infra/test-dev"},
				LabelPrefix: "cherrypick/",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Only org",
			cherrypicker: &TiCommunityCherrypicker{
				Repos:       []string{"ti-community-infra"},
				LabelPrefix: "cherrypick/",
			},
			org:  "ti-community-infra",
			repo: "test-dev",
		},
		{
			name: "Can not find",
			cherrypicker: &TiCommunityCherrypicker{
				Repos:       []string{"ti-community-infra"},
				LabelPrefix: "cherrypick/",
			},
			org:         "ti-community-infra1",
			repo:        "test-dev1",
			expectEmpty: &TiCommunityCherrypicker{},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			config := Configuration{TiCommunityCherrypicker: []TiCommunityCherrypicker{
				*tc.cherrypicker,
			}}

			cherrypicker := config.CherrypickerFor(tc.org, tc.repo)

			if tc.expectEmpty != nil {
				assert.DeepEqual(t, cherrypicker, &TiCommunityCherrypicker{})
			} else {
				assert.DeepEqual(t, cherrypicker.Repos, tc.cherrypicker.Repos)
			}
		})
	}
}

func TestSetCherrypickerDefaults(t *testing.T) {
	testcases := []struct {
		name              string
		labelPrefix       string
		expectLabelPrefix string
	}{
		{
			name:              "default",
			labelPrefix:       "",
			expectLabelPrefix: "cherrypick/",
		},
		{
			name:              "overwrite",
			labelPrefix:       "needs-cherry-pick-",
			expectLabelPrefix: "needs-cherry-pick-",
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			c := &Configuration{
				TiCommunityCherrypicker: []TiCommunityCherrypicker{
					{
						LabelPrefix: tc.labelPrefix,
					},
				},
			}

			c.setDefaults()

			for _, cherrypicker := range c.TiCommunityCherrypicker {
				if cherrypicker.LabelPrefix != tc.expectLabelPrefix {
					t.Errorf("unexpected labelPrefix: %v, expected: %v",
						cherrypicker.LabelPrefix, tc.expectLabelPrefix)
				}
			}
		})
	}
}
