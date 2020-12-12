package rerere

import (
	"errors"
	"flag"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"
)

const (
	DefaultRetestingBranch = "rerere"
	DefaultRetestingTimes  = 3
	DefaultTimeOut         = time.Minute * 15
)

const checkRunStatusCompleted = "completed"

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
	options *RetestingOptions, org string, repo string) error {
	err := gc.Clean()
	if err != nil {
		return err
	}
	for i := 0; i < options.Retry; i++ {
		// Init client form current dir.
		client, err := gc.ClientFromDir(org, repo, "")
		if err != nil {
			return err
		}
		// Force push to retesting branch.
		err = client.PushToCentral(options.RetestingBranch, true)
		if err != nil {
			return err
		}

		lastCommit, err := ghc.GetSingleCommit(org, repo, options.RetestingBranch)
		if err != nil {
			log.Warnf("Get %s last commit failed", options.RetestingBranch)
			continue
		}

		passedContexts := sets.String{}
		lastCommitRef := lastCommit.SHA
		// List all status.
		statuses, err := ghc.ListStatuses(org, repo, lastCommitRef)
		if err != nil {
			log.Warnf("List %s statuses failed", lastCommitRef)
			continue
		}
		for _, status := range statuses {
			if status.State == github.StatusSuccess {
				passedContexts.Insert(status.Context)
			}
		}
		// List all check runs.
		checkRun, err := ghc.ListCheckRuns(org, repo, lastCommitRef)
		if err != nil {
			log.Warnf("List %s check runs failed", lastCommitRef)
			continue
		}
		for _, runs := range checkRun.CheckRuns {
			if runs.Status == checkRunStatusCompleted {
				passedContexts.Insert(runs.Name)
			}
		}

		// All required contexts passed.
		if passedContexts.HasAll(options.Contexts.StringSet().List()...) {
			return nil
		}
	}
	return errors.New("retesting failed")
}
