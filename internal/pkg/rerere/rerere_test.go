package rerere

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/github"
)

type fghc struct {
	lastCommits  map[string]github.RepositoryCommit
	statuses     map[string][]github.Status
	checkRunList map[string]github.CheckRunList
}

func testRef(org, repo string, ref string) string {
	return fmt.Sprintf("%s/%s/%s", org, repo, ref)
}

func (f *fghc) ListStatuses(org, repo, ref string) ([]github.Status, error) {
	return f.statuses[testRef(org, repo, ref)], nil
}

func (f *fghc) GetSingleCommit(org, repo, ref string) (github.RepositoryCommit, error) {
	return f.lastCommits[testRef(org, repo, ref)], nil
}

func (f *fghc) ListCheckRuns(org, repo, ref string) (*github.CheckRunList, error) {
	run := f.checkRunList[testRef(org, repo, ref)]
	return &run, nil
}

func TestCheckContexts(t *testing.T) {
	tests := []struct {
		name            string
		org             string
		repo            string
		branch          string
		requireContexts []string
		statuses        []github.Status
		checkRun        github.CheckRunList

		exceptError string
	}{
		{
			name:            "non passed contexts",
			org:             "org",
			repo:            "repo",
			requireContexts: []string{"test1", "test2"},
			statuses: []github.Status{
				{Context: "test1", State: github.StatusPending},
				{Context: "test2", State: github.StatusFailure},
			},
			checkRun: github.CheckRunList{
				Total:     0,
				CheckRuns: []github.CheckRun{},
			},
			exceptError: "some of the required contexts are still not passed: [test1 test2]",
		},
		{
			name:            "one passed status",
			org:             "org",
			repo:            "repo",
			requireContexts: []string{"test1", "test2"},
			statuses: []github.Status{
				{Context: "test1", State: github.StatusSuccess},
				{Context: "test2", State: github.StatusFailure},
			},
			checkRun: github.CheckRunList{
				Total:     0,
				CheckRuns: []github.CheckRun{},
			},
			exceptError: "some of the required contexts are still not passed: [test2]",
		},
		{
			name:            "one passed check run",
			org:             "org",
			repo:            "repo",
			requireContexts: []string{"test1", "test2"},
			statuses:        []github.Status{{Context: "test2", State: github.StatusFailure}},
			checkRun: github.CheckRunList{
				Total:     0,
				CheckRuns: []github.CheckRun{{Name: "test1", Status: checkRunStatusCompleted}},
			},
			exceptError: "some of the required contexts are still not passed: [test2]",
		},
		{
			name:            "statuses all passed",
			org:             "org",
			repo:            "repo",
			requireContexts: []string{"test1", "test2"},
			statuses: []github.Status{
				{Context: "test1", State: github.StatusSuccess},
				{Context: "test2", State: github.StatusSuccess},
			},
			checkRun: github.CheckRunList{
				Total:     0,
				CheckRuns: []github.CheckRun{{Name: "test3", Status: checkRunStatusCompleted}},
			},
		},
		{
			name:            "check runs all passed",
			org:             "org",
			repo:            "repo",
			requireContexts: []string{"test1", "test2"},
			statuses: []github.Status{
				{Context: "test3", State: github.StatusSuccess},
			},
			checkRun: github.CheckRunList{
				Total: 0,
				CheckRuns: []github.CheckRun{
					{Name: "test1", Status: checkRunStatusCompleted},
					{Name: "test2", Status: checkRunStatusCompleted},
				},
			},
		},
	}

	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	for _, test := range tests {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			lastCommits := make(map[string]github.RepositoryCommit)
			lastCommits[testRef(tc.org, tc.repo, tc.branch)] = github.RepositoryCommit{SHA: SHA}

			refTestKey := testRef(tc.org, tc.repo, SHA)
			statuses := make(map[string][]github.Status)
			statuses[refTestKey] = tc.statuses
			checkRunList := make(map[string]github.CheckRunList)
			checkRunList[refTestKey] = tc.checkRun

			ghc := fghc{
				lastCommits:  lastCommits,
				statuses:     statuses,
				checkRunList: checkRunList,
			}
			err := checkContexts(logrus.WithField("rerere", "testing"),
				&ghc, prowflagutil.NewStrings(tc.requireContexts...), tc.branch, tc.org, tc.repo)
			if err != nil {
				if len(tc.exceptError) == 0 {
					t.Errorf("unexpected error: '%v'", err)
				} else if err.Error() != tc.exceptError {
					t.Errorf("expected error '%v', but it is '%v'", tc.exceptError, err)
				}
			}
		})
	}
}
