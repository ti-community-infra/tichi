package rerere

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pod-utils/downwardapi"
)

const (
	DefaultRetestingBranch      = "rerere"
	DefaultRetestingTimes       = 3
	DefaultTimeOut              = time.Minute * 15
	DefaultCheckPeriod          = time.Minute * 5
	DefaultRetestingLogFileName = ".rerere.json"
)

const checkRunStatusCompleted = "completed"

var check = checkContexts

type RetestingLog struct {
	Job               *downwardapi.JobSpec `json:"job,omitempty"`
	Options           *RetestingOptions    `json:"options,omitempty"`
	CurrentRetryTimes int                  `json:"current_retry_times,omitempty"`
	Time              time.Time            `json:"time,omitempty"`
}

// RetestingOptions holds options for retesting.
type RetestingOptions struct {
	RetestingBranch string
	Retry           int
	Contexts        prowflagutil.Strings
	Timeout         time.Duration
}

// AddFlags injects retesting options into the given FlagSet.
func (o *RetestingOptions) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.RetestingBranch, "retesting-branch", DefaultRetestingBranch, "Retesting target branch.")
	fs.IntVar(&o.Retry, "retry", DefaultRetestingTimes, "Retry testing times.")
	fs.Var(&o.Contexts, "contexts", "Required contexts that must be green to merge.")
	fs.DurationVar(&o.Timeout, "timeout", DefaultTimeOut, "Test timeout time.")
}

func (o *RetestingOptions) Validate(bool) error {
	if o.Retry <= 0 {
		return errors.New("--retry must more than zero")
	}
	contexts := o.Contexts.Strings()
	if len(contexts) == 0 {
		return errors.New("--contexts must contain at least one context")
	}
	return nil
}

type githubClient interface {
	ListStatuses(org, repo, ref string) ([]github.Status, error)
	GetSingleCommit(org, repo, SHA string) (github.RepositoryCommit, error)
	ListCheckRuns(org, repo, ref string) (*github.CheckRunList, error)
}

func Retesting(log *logrus.Entry, ghc githubClient, gc git.ClientFactory,
	options *RetestingOptions, org string, repo string, spec *downwardapi.JobSpec) error {
	log.Infof("String resting on %s/%s/branches/%s", org, repo, options.RetestingBranch)
	for i := 0; i < options.Retry; i++ {
		// Init client form current dir.
		client, err := gc.ClientFromDir(org, repo, "")
		if err != nil {
			return err
		}

		// First time retesting we need to checkout the retesting branch.
		if i == 0 {
			err = client.CheckoutNewBranch(options.RetestingBranch)
			if err != nil {
				return err
			}
		}

		// Log with commit.
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
		err = ioutil.WriteFile(DefaultRetestingLogFileName, rawLog, 0600)
		if err != nil {
			return err
		}
		// TODO: waiting bug fix See: https://github.com/kubernetes/test-infra/pull/20222.
		//err = client.Commit(fmt.Sprintf("Retesing %v", options.Contexts), string(rawLog))
		//if err != nil {
		//	return err
		//}

		// Force push to retesting branch.
		err = client.PushToCentral(options.RetestingBranch, true)
		if err != nil {
			return err
		}

		// Start retesting.
		startTime := time.Now()
		ticker := time.NewTicker(DefaultCheckPeriod)
		for t := range ticker.C {
			log.Infof("Check contexts at %v", t)
			err = check(log, ghc, options.Contexts, options.RetestingBranch, org, repo)
			if err == nil {
				ticker.Stop()
				return nil
			}
			log.WithError(err).Warn("Retesting failed")
			if t.Sub(startTime) > DefaultTimeOut {
				log.WithError(err).Warnf("Retesting timeout at %v", t)
				ticker.Stop()
				break
			}
		}
	}
	log.Warnf("Retry %d times failed", options.Retry)

	return errors.New("retesting failed")
}

func checkContexts(log *logrus.Entry, ghc githubClient, contexts prowflagutil.Strings,
	retestingBranch string, org string, repo string) error {
	lastCommit, err := ghc.GetSingleCommit(org, repo, retestingBranch)
	if err != nil {
		return fmt.Errorf("get %s last commit failed: %v", retestingBranch, err)
	}

	passedContexts := sets.String{}
	lastCommitRef := lastCommit.SHA
	// List all status.
	statuses, err := ghc.ListStatuses(org, repo, lastCommitRef)
	if err != nil {
		return fmt.Errorf("list %s statuses failed: %v", retestingBranch, err)
	}
	for _, status := range statuses {
		if status.State == github.StatusSuccess {
			log.Infof("%s context passed", status.Context)
			passedContexts.Insert(status.Context)
		}
	}
	// List all check runs.
	checkRun, err := ghc.ListCheckRuns(org, repo, lastCommitRef)
	if err != nil {
		return fmt.Errorf("list %s check runs failed: %v", retestingBranch, err)
	}
	for _, runs := range checkRun.CheckRuns {
		if runs.Status == checkRunStatusCompleted {
			log.Infof("%s runs passed", runs.Name)
			passedContexts.Insert(runs.Name)
		}
	}

	// All required contexts passed.
	if passedContexts.HasAll(contexts.StringSet().List()...) {
		return nil
	}
	return fmt.Errorf("some contexts still not passed %v",
		contexts.StringSet().Difference(passedContexts).List())
}
