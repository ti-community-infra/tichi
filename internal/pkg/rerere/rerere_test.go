package rerere

import (
	"errors"
	"fmt"
	"os"
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

func (f *fgc) CheckoutNewBranch(_ string) error {
	f.checkoutTimes++
	return nil
}

func (f *fgc) Commit(_, _ string) error {
	f.commitNum++
	return nil
}

func (f *fgc) PushToCentral(_ string, _ bool) error {
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

		expectAllPassed bool
		expectError     string
	}{
		{
			name:            "non passed statuses",
			org:             "org",
			repo:            "repo",
			requireContexts: []string{"test1", "test2"},
			statuses: []github.Status{
				{Context: "test1", State: github.StatusPending},
				{Context: "test2", State: github.StatusPending},
			},
			checkRun: github.CheckRunList{
				Total:     0,
				CheckRuns: []github.CheckRun{},
			},
			expectAllPassed: false,
		},
		{
			name:            "non passed check runs",
			org:             "org",
			repo:            "repo",
			requireContexts: []string{"test1", "test2"},
			statuses:        []github.Status{},
			checkRun: github.CheckRunList{
				Total:     0,
				CheckRuns: []github.CheckRun{{Name: "test1", Status: checkRunStatusCompleted, Conclusion: "skipped"}},
			},
			expectAllPassed: false,
			expectError:     "require context test1 failed",
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
			expectAllPassed: false,
			expectError:     "require context test2 failed",
		},
		{
			name:            "one passed check run",
			org:             "org",
			repo:            "repo",
			requireContexts: []string{"test1", "test2"},
			statuses:        []github.Status{{Context: "test2", State: github.StatusFailure}},
			checkRun: github.CheckRunList{
				Total: 0,
				CheckRuns: []github.CheckRun{
					{Name: "test1", Status: checkRunStatusCompleted, Conclusion: checkRunConclusionSuccess},
				},
			},
			expectAllPassed: false,
			expectError:     "require context test2 failed",
		},
		{
			name:            "all statuses passed",
			org:             "org",
			repo:            "repo",
			requireContexts: []string{"test1", "test2"},
			statuses: []github.Status{
				{Context: "test1", State: github.StatusSuccess},
				{Context: "test2", State: github.StatusSuccess},
			},
			checkRun: github.CheckRunList{
				Total: 0,
				CheckRuns: []github.CheckRun{
					{Name: "test3", Status: checkRunStatusCompleted, Conclusion: checkRunConclusionNeutral},
				},
			},
			expectAllPassed: true,
		},
		{
			name:            "all check runs passed",
			org:             "org",
			repo:            "repo",
			requireContexts: []string{"test1", "test2"},
			statuses: []github.Status{
				{Context: "test3", State: github.StatusSuccess},
			},
			checkRun: github.CheckRunList{
				Total: 0,
				CheckRuns: []github.CheckRun{
					{Name: "test1", Status: checkRunStatusCompleted, Conclusion: checkRunConclusionNeutral},
					{Name: "test2", Status: checkRunStatusCompleted, Conclusion: checkRunConclusionSuccess},
				},
			},
			expectAllPassed: true,
		},
	}

	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"

	for _, test := range tests {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			// Init the fake github client.
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

			isAllPassed, err := checkContexts(logrus.WithField("rerere", "testing"),
				&ghc, prowflagutil.NewStrings(tc.requireContexts...), tc.branch, tc.org, tc.repo)
			if err != nil {
				if len(tc.expectError) == 0 {
					t.Errorf("unexpected error: '%v'", err)
				} else if err.Error() != tc.expectError {
					t.Errorf("expected error '%v', but it is '%v'", tc.expectError, err)
				}
			} else {
				if len(tc.expectError) != 0 {
					t.Errorf("expected error: '%v', but it is nil", tc.expectError)
				}
			}

			if isAllPassed != tc.expectAllPassed {
				t.Errorf("expected all passed: '%v', but it is '%v'", tc.expectAllPassed, isAllPassed)
			}
		})
	}
}

func TestRetesting(t *testing.T) {
	tests := []struct {
		name                string
		options             RetestingOptions
		run                 func() mockCheck
		expectCheckoutTimes int
		expectCommitNum     int
		expectPushTimes     int
		expectError         string
	}{
		{
			name: "once pass",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           3,
				Contexts:        prowflagutil.NewStrings("test"),
				Timeout:         time.Nanosecond * 5,
			},
			run: func() mockCheck {
				return func(log *logrus.Entry, ghc githubClient, contexts prowflagutil.Strings,
					retestingBranch string, org string, repo string) (bool, error) {
					return true, nil
				}
			},
			expectCheckoutTimes: 1,
			expectCommitNum:     1,
			expectPushTimes:     1,
		},
		{
			name: "retry twice to pass",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           3,
				Contexts:        prowflagutil.NewStrings("test"),
				Timeout:         time.Nanosecond * 1,
			},
			run: func() mockCheck {
				result := []error{errors.New("one"), nil}
				i := 0
				next := func() (bool, error) {
					if i > 0 {
						time.Sleep(time.Nanosecond * 2)
					}
					err := result[i]
					i++
					return err == nil, err
				}
				return func(log *logrus.Entry, ghc githubClient, contexts prowflagutil.Strings,
					retestingBranch string, org string, repo string) (bool, error) {
					return next()
				}
			},
			expectCheckoutTimes: 1,
			expectCommitNum:     2,
			expectPushTimes:     2,
		},
		{
			name: "retry three times to pass",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           3,
				Contexts:        prowflagutil.NewStrings("test"),
				Timeout:         time.Nanosecond * 1,
			},
			run: func() mockCheck {
				result := []error{errors.New("one"), errors.New("two"), nil}
				i := 0
				next := func() (bool, error) {
					if i > 0 {
						time.Sleep(time.Nanosecond * 2)
					}
					err := result[i]
					i++
					return err == nil, err
				}
				return func(log *logrus.Entry, ghc githubClient, contexts prowflagutil.Strings,
					retestingBranch string, org string, repo string) (bool, error) {
					return next()
				}
			},
			expectCheckoutTimes: 1,
			expectCommitNum:     3,
			expectPushTimes:     3,
		},
		{
			name: "all retries time out",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           3,
				Contexts:        prowflagutil.NewStrings("test"),
				Timeout:         time.Nanosecond * 1,
			},
			run: func() mockCheck {
				result := []error{errors.New("one"), errors.New("two"), errors.New("three")}
				i := 0
				next := func() (bool, error) {
					if i > 0 {
						time.Sleep(time.Nanosecond * 2)
					}
					err := result[i]
					i++
					return err == nil, err
				}
				return func(log *logrus.Entry, ghc githubClient, contexts prowflagutil.Strings,
					retestingBranch string, org string, repo string) (bool, error) {
					return next()
				}
			},
			expectCheckoutTimes: 1,
			expectCommitNum:     3,
			expectPushTimes:     3,
			expectError:         "retesting failed",
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
			// Setup the check period.
			defaultCheckPeriod = time.Nanosecond * 1

			err := Retesting(logrus.WithField("rerere", "testing"), nil, &gc, &tc.options, org, repo, nil)

			if err != nil {
				if len(tc.expectError) == 0 {
					t.Errorf("unexpected error: '%v'", err)
				} else if err.Error() != tc.expectError {
					t.Errorf("expected error '%v', but it is '%v'", tc.expectError, err)
				}
			} else {
				if len(tc.expectError) != 0 {
					t.Errorf("expected error: '%v', but it is nil", tc.expectError)
				}
			}
			if gc.checkoutTimes != tc.expectCheckoutTimes {
				t.Errorf("expected checkout '%d' times, but it is '%d' times", tc.expectCheckoutTimes, gc.checkoutTimes)
			}
			if gc.commitNum != tc.expectCommitNum {
				t.Errorf("expected commit '%d' times, but it is '%d' times", tc.expectCommitNum, gc.commitNum)
			}
			if gc.pushTimes != tc.expectPushTimes {
				t.Errorf("expected push '%d' times, but it is '%d' times", tc.expectPushTimes, gc.pushTimes)
			}
		})
	}

	// Remove the log file after testing.
	_ = os.Remove(defaultRetestingLogFileName)
}

func TestRetestingOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		options RetestingOptions

		expectError string
	}{
		{
			name: "invalid retry times",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           -1,
				Contexts:        prowflagutil.NewStrings("test"),
				Timeout:         0,
			},
			expectError: "--retry must more than zero",
		},
		{
			name: "invalid contexts",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           1,
				Contexts:        prowflagutil.NewStrings(),
				Timeout:         0,
			},
			expectError: "--requireContexts must contain at least one context",
		},
		{
			name: "valid options",
			options: RetestingOptions{
				RetestingBranch: "rerere",
				Retry:           1,
				Contexts:        prowflagutil.NewStrings("test"),
				Timeout:         0,
			},
		},
	}

	for _, test := range tests {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			err := tc.options.Validate(true)

			if err != nil {
				if len(tc.expectError) == 0 {
					t.Errorf("unexpected error: '%v'", err)
				} else if err.Error() != tc.expectError {
					t.Errorf("expected error '%v', but it is '%v'", tc.expectError, err)
				}
			} else {
				if len(tc.expectError) != 0 {
					t.Errorf("expected error: '%v', but it is nil", tc.expectError)
				}
			}
		})
	}
}
