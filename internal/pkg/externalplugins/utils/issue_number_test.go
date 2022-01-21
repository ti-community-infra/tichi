package utils

import (
	"reflect"
	"testing"
)

func TestNormalizeIssueNumbers(t *testing.T) {
	testcases := []struct {
		name     string
		content  string
		currOrg  string
		currRepo string

		expectNumbers []IssueNumberData
	}{
		{
			name:     "issue number with short prefix",
			content:  "Issue Number: close #123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: []IssueNumberData{
				{
					AssociatePrefix: "close",
					Org:             "pingcap",
					Repo:            "tidb",
					Number:          123,
				},
			},
		},
		{
			name:     "issue number with full prefix in the same repo",
			content:  "Issue Number: close pingcap/tidb#123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: []IssueNumberData{
				{
					AssociatePrefix: "close",
					Org:             "pingcap",
					Repo:            "tidb",
					Number:          123,
				},
			},
		},
		{
			name:     "issue number with full prefix in the another repo",
			content:  "Issue Number: close pingcap/ticdc#123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: []IssueNumberData{
				{
					AssociatePrefix: "close",
					Org:             "pingcap",
					Repo:            "ticdc",
					Number:          123,
				},
			},
		},
		{
			name:     "issue number with full prefix in the another org",
			content:  "Issue Number: close tikv/tikv#123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: []IssueNumberData{
				{
					AssociatePrefix: "close",
					Org:             "tikv",
					Repo:            "tikv",
					Number:          123,
				},
			},
		},
		{
			name:     "issue number with link prefix in the same repo",
			content:  "Issue Number: close https://github.com/pingcap/tidb/issues/123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: []IssueNumberData{
				{
					AssociatePrefix: "close",
					Org:             "pingcap",
					Repo:            "tidb",
					Number:          123,
				},
			},
		},
		{
			name:     "issue number with link prefix in the another repo",
			content:  "Issue Number: close https://github.com/pingcap/ticdc/issues/123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: []IssueNumberData{
				{
					AssociatePrefix: "close",
					Org:             "pingcap",
					Repo:            "ticdc",
					Number:          123,
				},
			},
		},
		{
			name:     "issue number with link prefix in the another org",
			content:  "Issue Number: close https://github.com/tikv/tikv/issues/123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: []IssueNumberData{
				{
					AssociatePrefix: "close",
					Org:             "tikv",
					Repo:            "tikv",
					Number:          123,
				},
			},
		},
		{
			name:     "duplicate issue numbers with same associate prefix",
			content:  "Issue Number: close #123, close https://github.com/pingcap/tidb/issues/123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: []IssueNumberData{
				{
					AssociatePrefix: "close",
					Org:             "pingcap",
					Repo:            "tidb",
					Number:          123,
				},
			},
		},
		{
			name:     "multiple issue numbers with same associate prefix",
			content:  "Issue Number: close #456, close https://github.com/pingcap/tidb/issues/123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: []IssueNumberData{
				{
					AssociatePrefix: "close",
					Org:             "pingcap",
					Repo:            "tidb",
					Number:          123,
				},
				{
					AssociatePrefix: "close",
					Org:             "pingcap",
					Repo:            "tidb",
					Number:          456,
				},
			},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		actualNumbers := NormalizeIssueNumbers(tc.content, tc.currOrg, tc.currRepo)

		if !reflect.DeepEqual(tc.expectNumbers, actualNumbers) {
			t.Errorf("For case [%s]: expect issue numbers are: \n%v\nbut got: \n%v\n",
				tc.name, tc.expectNumbers, actualNumbers)
		}
	}
}
