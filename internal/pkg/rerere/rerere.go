package rerere

import (
	"errors"
	"flag"
	"time"

	"github.com/sirupsen/logrus"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/git/v2"
)

const (
	DefaultRetestingBranch = "rerere"
	DefaultRetestingTimes  = 3
	DefaultTimeOut         = time.Minute * 15
)

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
	AddLabel(org, repo string, number int, label string) error
}

func Retesting(log *logrus.Entry, ghc githubClient, gc git.ClientFactory, options *RetestingOptions) {

}
