package utils

import (
	"testing"
)

func TestNormalizeIssueNumbers(t *testing.T) {
	testcases := []struct {
		name      string
		content   string
		currOrg   string
		currRepo  string
		delimiter string

		expectNumbers string
	}{
		{
			name:     "issue number with short prefix",
			content:  "Issue Number: close #123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: "close #123",
		},
		{
			name:     "issue number with full prefix in the same repo",
			content:  "Issue Number: close pingcap/tidb#123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: "close #123",
		},
		{
			name:     "issue number with full prefix in the another repo",
			content:  "Issue Number: close pingcap/ticdc#123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: "close pingcap/ticdc#123",
		},
		{
			name:     "issue number with full prefix in the another org",
			content:  "Issue Number: close tikv/tikv#123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: "close tikv/tikv#123",
		},
		{
			name:     "issue number with link prefix in the same repo",
			content:  "Issue Number: close https://github.com/pingcap/tidb/issues/123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: "close #123",
		},
		{
			name:     "issue number with link prefix in the another repo",
			content:  "Issue Number: close https://github.com/pingcap/ticdc/issues/123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: "close pingcap/ticdc#123",
		},
		{
			name:     "issue number with link prefix in the another org",
			content:  "Issue Number: close https://github.com/tikv/tikv/issues/123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: "close tikv/tikv#123",
		},
		{
			name:     "duplicate issue numbers with same associate prefix",
			content:  "Issue Number: close #123, close https://github.com/pingcap/tidb/issues/123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: "close #123",
		},
		{
			name:     "multiple issue numbers with same associate prefix",
			content:  "Issue Number: close #456, close https://github.com/pingcap/tidb/issues/123",
			currOrg:  "pingcap",
			currRepo: "tidb",

			expectNumbers: "close #123, close #456",
		},
		{
			name:      "multiple issue numbers and custom delimiter",
			content:   "Issue Number: ref #123, close #456",
			currOrg:   "pingcap",
			currRepo:  "tidb",
			delimiter: "\n",

			expectNumbers: "ref #123\nclose #456",
		},
		{
			name:      "multiple issue numbers and custom delimiter",
			content:   "Issue Number: ref #123\nclose #456",
			currOrg:   "pingcap",
			currRepo:  "tidb",
			delimiter: "\n",

			expectNumbers: "ref #123\nclose #456",
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		actualNumbers := NormalizeIssueNumbers(tc.content, tc.currOrg, tc.currRepo, tc.delimiter)
		if tc.expectNumbers != actualNumbers {
			t.Errorf("For case \"%s\": \nexpect issue numbers are: \n%s\nbut got: \n%s",
				tc.name, tc.expectNumbers, actualNumbers)
		}
	}
}
