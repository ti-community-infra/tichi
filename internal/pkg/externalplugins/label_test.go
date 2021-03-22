package externalplugins

import (
	"reflect"
	"sort"
	"testing"
)

func TestFormatTestLabels(t *testing.T) {
	testcases := []struct {
		name   string
		org    string
		repo   string
		number int
		labels []string

		expectedLabels []string
	}{
		{
			name:   "A single label",
			org:    "org",
			repo:   "repo",
			number: 1,
			labels: []string{"foo"},

			expectedLabels: []string{"org/repo#1:foo"},
		},
		{
			name:   "Many labels",
			org:    "org",
			repo:   "repo",
			number: 1,
			labels: []string{"foo", "bar"},

			expectedLabels: []string{"org/repo#1:foo", "org/repo#1:bar"},
		},
		{
			name:   "Different args",
			org:    "org1",
			repo:   "repo1",
			number: 2,
			labels: []string{"foo", "bar"},

			expectedLabels: []string{"org1/repo1#2:foo", "org1/repo1#2:bar"},
		},
	}

	for _, tc := range testcases {
		t.Logf("Running scenario %q", tc.name)
		labels := FormatTestLabels(tc.org, tc.repo, tc.number, tc.labels...)

		sort.Strings(tc.expectedLabels)
		sort.Strings(labels)
		if !reflect.DeepEqual(tc.expectedLabels, labels) {
			t.Errorf("expected the labels %q to be added, but %q were added.",
				tc.expectedLabels, labels)
		}
	}
}
