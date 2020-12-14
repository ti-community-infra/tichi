package rerere

import (
	"errors"
	"fmt"
	"testing"
	"time"

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

type fgc struct {
	checkoutTimes int
	commitNum     int
	pushTimes     int
}

func (f *fgc) CheckoutNewBranch(branch string) error {
	f.checkoutTimes++
	return nil
}

func (f *fgc) Commit(title, body string) error {
	f.commitNum++
	return nil
}

func (f *fgc) PushToCentral(branch string, force bool) error {
	f.pushTimes++
	return nil
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

func TestRetesting(t *testing.T) {
	tests := []struct {
		name                string
		options             RetestingOptions
		run                 func() mockCheck
		exceptCheckoutTimes int
		exceptCommitNum     int
		exceptPushTimes     int
		exceptError         string
	}{
		{
			name: "once",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           3,
				Contexts:        prowflagutil.NewStrings("test"),
				Timeout:         time.Nanosecond * 5,
			},
			run: func() mockCheck {
				return func(log *logrus.Entry, ghc githubClient, contexts prowflagutil.Strings,
					retestingBranch string, org string, repo string) error {
					return nil
				}
			},
			exceptCheckoutTimes: 1,
			exceptCommitNum:     1,
			exceptPushTimes:     1,
		},
		{
			name: "two times",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           3,
				Contexts:        prowflagutil.NewStrings("test"),
				Timeout:         time.Nanosecond * 1,
			},
			run: func() mockCheck {
				result := []error{errors.New("one"), nil}
				i := 0
				next := func() error {
					if i > 0 {
						time.Sleep(time.Nanosecond * 2)
					}
					err := result[i]
					i++
					return err
				}
				return func(log *logrus.Entry, ghc githubClient, contexts prowflagutil.Strings,
					retestingBranch string, org string, repo string) error {
					return next()
				}
			},
			exceptCheckoutTimes: 1,
			exceptCommitNum:     2,
			exceptPushTimes:     2,
		},
		{
			name: "three times",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           3,
				Contexts:        prowflagutil.NewStrings("test"),
				Timeout:         time.Nanosecond * 1,
			},
			run: func() mockCheck {
				result := []error{errors.New("one"), errors.New("two"), nil}
				i := 0
				next := func() error {
					if i > 0 {
						time.Sleep(time.Nanosecond * 2)
					}
					err := result[i]
					i++
					return err
				}
				return func(log *logrus.Entry, ghc githubClient, contexts prowflagutil.Strings,
					retestingBranch string, org string, repo string) error {
					return next()
				}
			},
			exceptCheckoutTimes: 1,
			exceptCommitNum:     3,
			exceptPushTimes:     3,
		},
		{
			name: "all timeout",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           3,
				Contexts:        prowflagutil.NewStrings("test"),
				Timeout:         time.Nanosecond * 1,
			},
			run: func() mockCheck {
				result := []error{errors.New("one"), errors.New("two"), errors.New("three")}
				i := 0
				next := func() error {
					if i > 0 {
						time.Sleep(time.Nanosecond * 2)
					}
					err := result[i]
					i++
					return err
				}
				return func(log *logrus.Entry, ghc githubClient, contexts prowflagutil.Strings,
					retestingBranch string, org string, repo string) error {
					return next()
				}
			},
			exceptCheckoutTimes: 1,
			exceptCommitNum:     3,
			exceptPushTimes:     3,
			exceptError:         "retesting failed",
		},
	}

	org := "org"
	repo := "repo"

	for _, test := range tests {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			// Mock the check.
			check = tc.run()
			gc := fgc{}
			defaultCheckPeriod = time.Nanosecond * 1

			err := Retesting(logrus.WithField("rerere", "testing"), nil, &gc, &tc.options, org, repo, nil)

			if err != nil {
				if len(tc.exceptError) == 0 {
					t.Errorf("unexpected error: '%v'", err)
				} else if err.Error() != tc.exceptError {
					t.Errorf("expected error '%v', but it is '%v'", tc.exceptError, err)
				}
			}
			if gc.checkoutTimes != tc.exceptCheckoutTimes {
				t.Errorf("expected checkout '%d' times, but it is '%d' times", tc.exceptCheckoutTimes, gc.checkoutTimes)
			}
			if gc.commitNum != tc.exceptCommitNum {
				t.Errorf("expected commit '%d' times, but it is '%d' times", tc.exceptCommitNum, gc.commitNum)
			}
			if gc.pushTimes != tc.exceptPushTimes {
				t.Errorf("expected push '%d' times, but it is '%d' times", tc.exceptPushTimes, gc.pushTimes)
			}
		})
	}
}
