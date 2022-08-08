package guard

import (
	"regexp"
	"strings"

	"github.com/ahmetb/go-linq/v3"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
)

type prBasicInfo struct {
	org  string
	repo string
	num  int
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
func HelpProvider(epa *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.GuardFor(repo.Org, repo.Repo)
			if opts == nil {
				continue
			}

			var configInfoStrings []string
			configInfoStrings = append(configInfoStrings, "The plugin has these configurations:<ul>")
			// TODO(wuhuizuo): fill it.
			configInfoStrings = append(configInfoStrings, "</ul>")
			configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
		}

		yamlSnippet, err := plugins.CommentMap.GenYaml(&tiexternalplugins.Configuration{
			TiCommunityGuard: []tiexternalplugins.TiCommunityGuard{
				{
					Repos:     []string{"ti-community-infra/test-dev"},
					Approvers: []string{"guard-a", "guard-b"},
					Patterns:  []string{`^config/.*\.go`, `^var/.*\.go`},
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
			return nil, err
		}

		pluginHelp := &pluginhelp.PluginHelp{
			Description: `The ti-community-guard plugin automatically requests approves to 
 			guards when changed file hit patterns`,
			Config:  configInfo,
			Snippet: yamlSnippet,
			Events:  []string{tiexternalplugins.PullRequestEvent, tiexternalplugins.IssueCommentEvent},
		}

		return pluginHelp, nil
	}
}

// HandlePullRequestEvent handles a GitHub pull request event and requests review.
func HandlePullRequestEvent(
	gc githubClient,
	event *github.PullRequestEvent,
	cfg *tiexternalplugins.Configuration,
	log *logrus.Entry,
) error {
	// skip for closed or draft state.
	if event.PullRequest.State == github.PullRequestStateClosed {
		return nil
	}
	if event.PullRequest.Draft {
		return nil
	}

	// skip for no mergeable: skip if mergeable is false.
	if event.PullRequest.Mergable != nil && !*event.PullRequest.Mergable {
		return nil
	}

	switch event.Action {
	case github.PullRequestActionOpened,
		github.PullRequestActionReopened,
		github.PullRequestActionReadyForReview,
		github.PullRequestActionSynchronize:
		return handleForOpened(gc, event, cfg)
	case github.PullRequestActionLabeled:
		return handleForLabelAdded(gc, event, cfg)
	case github.PullRequestActionReviewRequestRemoved:
		return handleForReviewerRemoved(gc, event, cfg)
	default:
		return nil
	}
}

func HandlePullRequestReviewEvent(
	gc githubClient,
	event *github.ReviewEvent,
	cfg *tiexternalplugins.Configuration,
	log *logrus.Entry,
) error {
	org := event.Repo.Owner.Login
	repo := event.Repo.Name
	opts := cfg.GuardFor(org, repo)

	// only deal with approves from `opts.Approves`
	if !sets.NewString(opts.Approvers...).Has(event.Review.User.Login) {
		return nil
	}

	switch event.Action {
	case github.ReviewActionSubmitted:
		pr := prBasicInfo{
			org:  org,
			repo: repo,
			num:  event.PullRequest.Number,
		}
		switch event.Review.State {
		case github.ReviewStateApproved:
			return handleForRevieweApproved(pr, gc, event.PullRequest.Labels, &opts.Label)
		case github.ReviewStateChangesRequested:
			return handleForRevieweChangesRequested(pr, gc, event.PullRequest.Labels, &opts.Label)
		default:
			return nil
		}
	default:
		// Q: is it need to deal other actions?
		return nil
	}
}

func handleForReviewerRemoved(
	gc githubClient,
	event *github.PullRequestEvent,
	cfg *tiexternalplugins.Configuration,
) error {
	org := event.Repo.Owner.Login
	repo := event.Repo.Name
	prNum := event.PullRequest.Number
	opts := cfg.GuardFor(org, repo)

	// only care for unapproved labeled pull request.
	if !matchLabels(event.PullRequest.Labels, opts.Label.Unapproved) {
		return nil
	}

	var currentReviewers []string
	for _, r := range event.PullRequest.RequestedReviewers {
		currentReviewers = append(currentReviewers, r.Login)
	}

	// add all approvers if none in current reviewers.
	if !sets.NewString(currentReviewers...).HasAny(opts.Approvers...) {
		// todo: comment to add required reviewer.
		return gc.RequestReview(org, repo, prNum, append(currentReviewers, opts.Approvers...))
	}

	return nil
}

func handleForOpened(
	gc githubClient,
	event *github.PullRequestEvent,
	cfg *tiexternalplugins.Configuration,
) error {
	org := event.Repo.Owner.Login
	repo := event.Repo.Name
	prNum := event.PullRequest.Number
	opts := cfg.GuardFor(org, repo)

	changeFiles, err := getPullRequestChangeFilenames(gc, org, repo, prNum)
	if err != nil {
		return err
	}

	matchedFiles := matchFiles(changeFiles, opts.Patterns)
	if len(matchedFiles) == 0 {
		if !matchLabels(event.PullRequest.Labels, opts.Label.Unapproved) {
			return nil
		}

		// remove unapproved label.
		return gc.RemoveLabel(org, repo, prNum, opts.Label.Unapproved)
	}

	// todo: comment on github.
	if matchLabels(event.PullRequest.Labels, opts.Label.Unapproved) {
		// keep it, no change.
		return nil
	}

	// remove approved label.
	if matchLabels(event.PullRequest.Labels, opts.Label.Approved) {
		if err := gc.RemoveLabel(org, repo, prNum, opts.Label.Approved); err != nil {
			// todo: send comment.
			return err
		}
	}

	// add unapproved label.
	return gc.AddLabel(org, repo, prNum, opts.Label.Unapproved)
}

func handleForLabelAdded(
	gc githubClient,
	event *github.PullRequestEvent,
	cfg *tiexternalplugins.Configuration,
) error {
	org := event.Repo.Owner.Login
	repo := event.Repo.Name
	prNum := event.PullRequest.Number
	opts := cfg.GuardFor(org, repo)

	switch event.Label.Name {
	case opts.Label.Unapproved:
		var reviewers []string
		for _, r := range event.PullRequest.RequestedReviewers {
			reviewers = append(reviewers, r.Login)
		}

		// ensure reviewer contained.
		if !sets.NewString(reviewers...).HasAny(opts.Approvers...) {
			reviewers = append(reviewers, opts.Approvers...)
		}

		// todo: send comment.
		return gc.RequestReview(org, repo, prNum, reviewers)
	case opts.Label.Approved:
		// todo: only send comment
		return nil
	default:
		return nil
	}
}

func handleForRevieweApproved(
	pr prBasicInfo,
	gc githubClient,
	labels []github.Label,
	guardLabel *tiexternalplugins.TiCommunityGuardLabel,
) error {
	// only care of labeled for need to approve.
	if !matchLabels(labels, guardLabel.Unapproved) {
		return nil
	}

	// add approve label, remove unapprove label
	if err := gc.AddLabel(pr.org, pr.repo, pr.num, guardLabel.Approved); err != nil {
		return err
	}
	if err := gc.RemoveLabel(pr.org, pr.repo, pr.num, guardLabel.Unapproved); err != nil {
		return nil
	}

	return nil
}

func handleForRevieweChangesRequested(
	pr prBasicInfo,
	gc githubClient,
	labels []github.Label,
	guardLabel *tiexternalplugins.TiCommunityGuardLabel,
) error {
	// only care of labeled approved.
	if !matchLabels(labels, guardLabel.Approved) {
		return nil
	}

	// add unapproved label, remove approved label
	if err := gc.AddLabel(pr.org, pr.repo, pr.num, guardLabel.Unapproved); err != nil {
		return err
	}
	if err := gc.RemoveLabel(pr.org, pr.repo, pr.num, guardLabel.Approved); err != nil {
		return nil
	}
	return nil
}

func getPullRequestChangeFilenames(gc githubClient, org, repo string, prNum int) ([]string, error) {
	changes, err := gc.GetPullRequestChanges(org, repo, prNum)
	if err != nil {
		return nil, err
	}

	var ret []string
	for _, c := range changes {
		ret = append(ret, c.Filename)
		if c.PreviousFilename != "" && c.PreviousFilename != c.Filename {
			ret = append(ret, c.PreviousFilename)
		}
	}

	return ret, nil
}

func matchFiles(files []string, patterns []string) []string {
	// compile all regexes.
	var regs []*regexp.Regexp
	for _, p := range patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			continue
		}

		regs = append(regs, r)
	}

	var ret []string
	regQ := linq.From(regs)
	linq.
		From(files).
		Where(func(f interface{}) bool {
			return regQ.AnyWith(func(q interface{}) bool {
				return q.(*regexp.Regexp).Match([]byte(f.(string)))
			})
		}).
		ToSlice(&ret)

	return ret
}

func matchLabels(labels []github.Label, label string) bool {
	for _, l := range labels {
		if l.Name == label {
			return true
		}
	}

	return false
}
