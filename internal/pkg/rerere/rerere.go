package rerere

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pod-utils/downwardapi"
)

const (
	// defaultRetestingBranch specifies the default branch used for retesting.
	defaultRetestingBranch = "rerere"
	// defaultRetestingTimes specifies the default number of retries.
	defaultRetestingTimes = 3
	// defaultTimeOut specifies the default timeout of test.
	defaultTimeOut = time.Minute * 15
	// defaultRetestingLogFileName specifies the default retry log file name.
	defaultRetestingLogFileName = ".rerere.json"
)

// defaultCheckPeriod specifies the default period for test ticker.
var defaultCheckPeriod = time.Minute * 1

// Ref: https://docs.github.com/en/rest/reference/checks#create-a-check-run--parameters.
const (
	// checkRunStatusCompleted means the check run passed.
	checkRunStatusCompleted = "completed"

	// checkRunConclusionNeutral means the check run conclusion is neutral.
	checkRunConclusionNeutral = "neutral"

	// checkRunConclusionSuccess means the check run conclusion is success.
	checkRunConclusionSuccess = "success"
)

// Mock check for test.
type mockCheck = func(log *logrus.Entry, ghc githubClient,
	contexts prowflagutil.Strings, retestingBranch string, org string, repo string) (bool, error)

var check = checkContexts

// RetestingLog specifies the details of this test.
type RetestingLog struct {
	Job               *downwardapi.JobSpec `json:"job,omitempty"`
	Options           *RetestingOptions    `json:"options,omitempty"`
	CurrentRetryTimes int                  `json:"current_retry_times,omitempty"`
	Time              time.Time            `json:"time,omitempty"`
}

// RetestingOptions holds options for retesting.
type RetestingOptions struct {
	RetestingBranch string               `json:"retesting-branch,omitempty"`
	Retry           int                  `json:"retry,omitempty"`
	Contexts        prowflagutil.Strings `json:"require_contexts,omitempty"`
	Timeout         time.Duration        `json:"timeout,omitempty"`
}

// AddFlags injects retesting options into the given FlagSet.
func (o *RetestingOptions) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.RetestingBranch, "retesting-branch", defaultRetestingBranch, "Retesting target branch.")
	fs.IntVar(&o.Retry, "retry", defaultRetestingTimes, "Retry testing times.")
	fs.Var(&o.Contexts, "require-contexts", "Required requireContexts that must be green to merge.")
	fs.DurationVar(&o.Timeout, "timeout", defaultTimeOut, "Test timeout time.")
}

// Validate validates retry times and contexts.
func (o *RetestingOptions) Validate(bool) error {
	if o.Retry <= 0 {
		return errors.New("--retry must more than zero")
	}
	contexts := o.Contexts.Strings()
	if len(contexts) == 0 {
		return errors.New("--requireContexts must contain at least one context")
	}
	return nil
}

type githubClient interface {
	ListStatuses(org, repo, ref string) ([]github.Status, error)
	GetSingleCommit(org, repo, SHA string) (github.RepositoryCommit, error)
	ListCheckRuns(org, repo, ref string) (*github.CheckRunList, error)
}

type gitRepoClient interface {
	CheckoutNewBranch(branch string) error
	Commit(title, body string) error
	PushToCentral(branch string, force bool) error
}

// Retesting drives the current code to the test branch and keeps checking the test results.
func Retesting(log *logrus.Entry, ghc githubClient, client gitRepoClient,
	options *RetestingOptions, org string, repo string, spec *downwardapi.JobSpec) error {
	log.Infof("String resting on %s/%s/branches/%s.", org, repo, options.RetestingBranch)
	for i := 0; i < options.Retry; i++ {
		// First time retesting we need to checkout the retesting branch.
		if i == 0 {
			err := client.CheckoutNewBranch(options.RetestingBranch)
			if err != nil {
				return err
			}
		}

		// Commit the retry log file.
		retestingLog := RetestingLog{
			Job:               spec,
			Options:           options,
			CurrentRetryTimes: i + 1,
			Time:              time.Now(),
		}
		rawLog, err := json.Marshal(retestingLog)
		if err != nil {
			return err
		}
		err = os.WriteFile(defaultRetestingLogFileName, rawLog, 0600)
		if err != nil {
			return err
		}

		var prs string
		for _, pull := range spec.Refs.Pulls {
			prs += fmt.Sprintf("#%d ", pull.Number)
		}
		log.Infof("Retesting %s.", prs)

		err = client.Commit(fmt.Sprintf("Retesting %s", prs), string(rawLog))
		if err != nil {
			return err
		}

		// Force push to retesting branch.
		log.Infof("Push to %v.", options.RetestingBranch)
		err = client.PushToCentral(options.RetestingBranch, true)
		if err != nil {
			return err
		}

		// Start retesting.
		startTime := time.Now()
		ticker := time.NewTicker(defaultCheckPeriod)
		for t := range ticker.C {
			log.Infof("Check requireContexts at %v.", t)

			isAllPassed, err := check(log, ghc, options.Contexts, options.RetestingBranch, org, repo)
			if err != nil {
				log.WithError(err).Warn("Retesting failed.")
				ticker.Stop()
				break
			}

			if isAllPassed {
				log.Infof("All required contexts passed.")
				ticker.Stop()
				return nil
			}

			// If timeout.
			if t.Sub(startTime) > options.Timeout {
				log.WithError(err).Warnf("Retesting timeout at %v.", t)
				ticker.Stop()
				break
			}
		}
	}
	log.Warnf("Retry %d times failed.", options.Retry)

	return errors.New("retesting failed")
}

// checkContexts checks if all the tests have passed.
func checkContexts(log *logrus.Entry, ghc githubClient, contexts prowflagutil.Strings,
	retestingBranch string, org string, repo string) (bool, error) {
	lastCommit, err := ghc.GetSingleCommit(org, repo, retestingBranch)
	if err != nil {
		return false, fmt.Errorf("get %s last commit failed: %v", retestingBranch, err)
	}

	passedContexts := sets.String{}
	lastCommitRef := lastCommit.SHA
	// List all status.
	statuses, err := ghc.ListStatuses(org, repo, lastCommitRef)
	if err != nil {
		return false, fmt.Errorf("list %s statuses failed: %v", retestingBranch, err)
	}

	requireContextSet := contexts.StringSet()

	for _, status := range statuses {
		if status.State == github.StatusFailure && requireContextSet.Has(status.Context) {
			return false, fmt.Errorf("require context %s failed", status.Context)
		}
		if status.State == github.StatusSuccess {
			log.Infof("%s context passed.", status.Context)
			passedContexts.Insert(status.Context)
		}
	}
	// List all check runs.
	checkRun, err := ghc.ListCheckRuns(org, repo, lastCommitRef)
	if err != nil {
		return false, fmt.Errorf("list %s check runs failed: %v", retestingBranch, err)
	}
	for _, run := range checkRun.CheckRuns {
		if run.Status == checkRunStatusCompleted {
			if run.Conclusion == checkRunConclusionNeutral || run.Conclusion == checkRunConclusionSuccess {
				log.Infof("%s runs passed.", run.Name)
				passedContexts.Insert(run.Name)
			} else if requireContextSet.Has(run.Name) {
				return false, fmt.Errorf("require context %s failed", run.Name)
			}
		}
	}

	// All required requireContexts passed.
	if passedContexts.HasAll(requireContextSet.List()...) {
		return true, nil
	}

	log.Infof("Some of the required contexts are still not passed: %v.",
		contexts.StringSet().Difference(passedContexts).List())
	return false, nil
}
