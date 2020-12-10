//nolint:gocritic
package merge

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/externalplugins"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/ownersclient"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

// PluginName will register into prow.
const PluginName = "ti-community-merge"

const githubUpdateCommitter = "web-flow"

var (
	addCanMergeLabelNotification   = "Can merge label has been added.  <details>Git tree hash: %s</details>"
	addCanMergeLabelNotificationRe = regexp.MustCompile(fmt.Sprintf(addCanMergeLabelNotification, "(.*)"))
	configInfoStoreTreeHash        = `Guaranteed commits does not remove can merge label.`

	// CanMergeRe is the regex that matches merge comments
	CanMergeRe = regexp.MustCompile(`(?mi)^/merge\s*$`)
	// CanMergeCancelRe is the regex that matches merge cancel comments
	CanMergeCancelRe        = regexp.MustCompile(`(?mi)^/merge cancel\s*$`)
	removeCanMergeLabelNoti = "New changes are detected. Can merge label has been removed."
)

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *externalplugins.ConfigAgent) func(
	enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()
		for _, repo := range enabledRepos {
			opts := cfg.MergeFor(repo.Org, repo.Repo)
			var isConfigured bool
			var configInfoStrings []string
			configInfoStrings = append(configInfoStrings, "The plugin has the following configuration:<ul>")
			if opts.StoreTreeHash {
				configInfoStrings = append(configInfoStrings, "<li>"+configInfoStoreTreeHash+"</li>")
				isConfigured = true
			}
			configInfoStrings = append(configInfoStrings, "</ul>")
			if isConfigured {
				configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
			}
		}
		pluginHelp := &pluginhelp.PluginHelp{
			Description: "The ti-community-merge plugin manages the application and " +
				"removal of the '" + externalplugins.CanMergeLabel + "' label which is typically used to gate merging.",
			Config: configInfo,
		}

		pluginHelp.AddCommand(pluginhelp.Command{
			Usage: "/merge [cancel] or GitHub Review action",
			Description: "Adds or removes the '" + externalplugins.CanMergeLabel + "' label which is typically " +
				"used to gate merging.",
			Featured:  true,
			WhoCanUse: "Collaborators on the repository. '/merge cancel' can be used additionally by the PR author.",
			Examples: []string{
				"/merge",
				"/merge cancel"},
		})
		return pluginHelp, nil
	}
}

type githubClient interface {
	AddLabel(owner, repo string, number int, label string) error
	CreateComment(owner, repo string, number int, comment string) error
	RemoveLabel(owner, repo string, number int, label string) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	ListIssueComments(org, repo string, number int) ([]github.IssueComment, error)
	DeleteComment(org, repo string, ID int) error
	BotName() (string, error)
	ListPRCommits(org, repo string, number int) ([]github.RepositoryCommit, error)
}

// reviewCtx contains information about each review event.
type reviewCtx struct {
	author, issueAuthor, body, htmlURL string
	repo                               github.Repo
	number                             int
}

// commentPruner used to delete bot comment.
type commentPruner interface {
	PruneComments(shouldPrune func(github.IssueComment) bool)
}

// HandleIssueCommentEvent handles a GitHub issue comment event and adds or removes a
// "status/can-merge" label.
func HandleIssueCommentEvent(gc githubClient, ice *github.IssueCommentEvent, cfg *externalplugins.Configuration,
	ol ownersclient.OwnersLoader, cp commentPruner, log *logrus.Entry) error {
	// Only consider open PRs and new comments.
	if !ice.Issue.IsPullRequest() || ice.Issue.State != "open" || ice.Action != github.IssueCommentActionCreated {
		return nil
	}

	rc := reviewCtx{
		author:      ice.Comment.User.Login,
		issueAuthor: ice.Issue.User.Login,
		body:        ice.Comment.Body,
		htmlURL:     ice.Comment.HTMLURL,
		repo:        ice.Repo,
		number:      ice.Issue.Number,
	}

	// If we create an "/merge" comment, add status/can-merge if necessary.
	// If we create a "/merge cancel" comment, remove status/can-merge if necessary.
	wantMerge := false
	if CanMergeRe.MatchString(rc.body) {
		wantMerge = true
	} else if CanMergeCancelRe.MatchString(rc.body) {
		wantMerge = false
	} else {
		return nil
	}

	// Use common handler to do the rest.
	return handle(wantMerge, cfg, rc, gc, ol, cp, log)
}

func HandlePullReviewCommentEvent(gc githubClient, pullReviewCommentEvent *github.ReviewCommentEvent,
	cfg *externalplugins.Configuration, ol ownersclient.OwnersLoader, cp commentPruner, log *logrus.Entry) error {
	// Only consider open PRs and new comments.
	if pullReviewCommentEvent.PullRequest.State != "open" ||
		pullReviewCommentEvent.Action != github.ReviewCommentActionCreated {
		return nil
	}

	rc := reviewCtx{
		author:      pullReviewCommentEvent.Comment.User.Login,
		issueAuthor: pullReviewCommentEvent.PullRequest.User.Login,
		body:        pullReviewCommentEvent.Comment.Body,
		htmlURL:     pullReviewCommentEvent.Comment.HTMLURL,
		repo:        pullReviewCommentEvent.Repo,
		number:      pullReviewCommentEvent.PullRequest.Number,
	}

	// If we create an "/merge" comment, add status/can-merge if necessary.
	// If we create a "/merge cancel" comment, remove status/can-merge if necessary.
	wantMerge := false
	if CanMergeRe.MatchString(rc.body) {
		wantMerge = true
	} else if CanMergeCancelRe.MatchString(rc.body) {
		wantMerge = false
	} else {
		return nil
	}

	// Use common handler to do the rest.
	return handle(wantMerge, cfg, rc, gc, ol, cp, log)
}

func HandlePullRequestEvent(gc githubClient, pe *github.PullRequestEvent,
	cfg *externalplugins.Configuration, log *logrus.Entry) error {
	if pe.PullRequest.Merged {
		return nil
	}

	if pe.Action != github.PullRequestActionSynchronize {
		return nil
	}

	org := pe.PullRequest.Base.Repo.Owner.Login
	repo := pe.PullRequest.Base.Repo.Name
	number := pe.PullRequest.Number

	opts := cfg.MergeFor(org, repo)

	// If we don't have the 'status/can-merge' label, we don't need to check anything.
	labels, err := gc.GetIssueLabels(org, repo, number)
	if err != nil {
		log.WithError(err).Error("Failed to get labels.")
	}
	hasCanMerge := false
	for _, label := range labels {
		if label.Name == externalplugins.CanMergeLabel {
			hasCanMerge = true
		}
	}
	if !hasCanMerge {
		return nil
	}

	if opts.StoreTreeHash {
		// Check if we have a tree-hash comment.
		var lastCanMergeTreeHash string
		botName, err := gc.BotName()
		if err != nil {
			return err
		}
		comments, err := gc.ListIssueComments(org, repo, number)
		if err != nil {
			log.WithError(err).Error("Failed to get issue comments.")
		}
		// Older comments are still present
		// iterate backwards to find the last can merge tree-hash.
		for i := len(comments) - 1; i >= 0; i-- {
			comment := comments[i]
			m := addCanMergeLabelNotificationRe.FindStringSubmatch(comment.Body)
			if comment.User.Login == botName && m != nil && comment.UpdatedAt.Equal(comment.CreatedAt) {
				lastCanMergeTreeHash = m[1]
				break
			}
		}
		if lastCanMergeTreeHash != "" {
			prCommits, err := gc.ListPRCommits(org, repo, number)
			if err != nil {
				log.WithField("sha", pe.PullRequest.Head.SHA).WithError(err).Error("Failed to get PR's commits.")
			}

			// Don't remove the label, PR code hasn't changed.
			if isAllGuaranteed(prCommits, lastCanMergeTreeHash, log) {
				return nil
			}
		}
	}

	if err := gc.RemoveLabel(org, repo, number, externalplugins.CanMergeLabel); err != nil {
		return fmt.Errorf("failed removing can merge label: %v", err)
	}

	// Create a comment to inform participants that can merge label is removed due to new
	// pull request changes.
	log.Infof("Commenting with an can merge removed notification to %s/%s#%d with a message: %s",
		org, repo, number, removeCanMergeLabelNoti)
	return gc.CreateComment(org, repo, number, removeCanMergeLabelNoti)
}

func handle(wantMerge bool, config *externalplugins.Configuration, rc reviewCtx,
	gc githubClient, ol ownersclient.OwnersLoader, cp commentPruner, log *logrus.Entry) error {
	author := rc.author
	issueAuthor := rc.issueAuthor
	number := rc.number
	body := rc.body
	htmlURL := rc.htmlURL
	org := rc.repo.Owner.Login
	repoName := rc.repo.Name

	// Author cannot merge own PR, comment and abort.
	isAuthor := author == issueAuthor
	if isAuthor && wantMerge {
		resp := "you cannot merge your own PR."
		log.Infof("Commenting with \"%s\".", resp)
		return gc.CreateComment(rc.repo.Owner.Login, rc.repo.Name, rc.number,
			externalplugins.FormatResponseRaw(rc.body, rc.htmlURL, rc.author, resp))
	}

	// Get ti-community-merge config.
	opts := config.MergeFor(rc.repo.Owner.Login, rc.repo.Name)
	url := fmt.Sprintf(ownersclient.OwnersURLFmt, opts.PullOwnersEndpoint, org, repoName, number)
	owners, err := ol.LoadOwners(opts.PullOwnersEndpoint, org, repoName, number)
	if err != nil {
		return err
	}

	committers := sets.String{}
	for _, committer := range owners.Committers {
		committers.Insert(committer)
	}

	// Not committers but want merge.
	if !committers.Has(author) && wantMerge {
		resp := "adding 'status/cam-merge' is restricted to committers in [list](" + url + ")."
		log.Infof("Reply to /merge request with comment: \"%s\"", resp)
		return gc.CreateComment(org, repoName, number, externalplugins.FormatResponseRaw(body, htmlURL, author, resp))
	}

	// Not author or committers but want remove merge.
	if !committers.Has(author) && !isAuthor && !wantMerge {
		resp := "removing 'status/cam-merge' is restricted to committers in [list](" + url + ") or PR author."
		log.Infof("Reply to /merge cancel request with comment: \"%s\"", resp)
		return gc.CreateComment(org, repoName, number, externalplugins.FormatResponseRaw(body, htmlURL, author, resp))
	}

	// Now we update the 'status/cam-merge' labels, having checked all cases where changing.
	// Only add the label if it doesn't have it, and vice versa.
	labels, err := gc.GetIssueLabels(org, repoName, number)
	if err != nil {
		log.WithError(err).Error("Failed to get issue labels.")
	}
	hasCanMerge := false
	for _, label := range labels {
		if label.Name == externalplugins.CanMergeLabel {
			hasCanMerge = true
		}
	}

	isSatisfy := isLGTMSatisfy(externalplugins.LgtmLabelPrefix, labels, owners.NeedsLgtm)

	// Remove the label if necessary, we're done after this.
	if hasCanMerge && !wantMerge {
		log.Info("Removing '" + externalplugins.CanMergeLabel + "' label.")
		if err := gc.RemoveLabel(org, repoName, number, externalplugins.CanMergeLabel); err != nil {
			return err
		}
		if opts.StoreTreeHash {
			cp.PruneComments(func(comment github.IssueComment) bool {
				return addCanMergeLabelNotificationRe.MatchString(comment.Body)
			})
		}
	} else if !hasCanMerge && wantMerge {
		if isSatisfy {
			log.Info("Adding '" + externalplugins.CanMergeLabel + "' label.")
			if err := gc.AddLabel(org, repoName, number, externalplugins.CanMergeLabel); err != nil {
				return err
			}
			if opts.StoreTreeHash {
				pr, err := gc.GetPullRequest(org, repoName, number)
				if err != nil {
					log.WithError(err).Error("Failed to get pull request.")
				}
				prCommits, err := gc.ListPRCommits(org, repoName, number)
				if err != nil {
					log.WithField("sha", pr.Head.SHA).WithError(err).Error("Failed to get commit.")
				}
				// Store the last commit hash.
				treeHash := prCommits[len(prCommits)-1].SHA
				log.WithField("tree", treeHash).Info("Adding comment to store tree-hash.")
				if err := gc.CreateComment(org, repoName, number, fmt.Sprintf(addCanMergeLabelNotification, treeHash)); err != nil {
					log.WithError(err).Error("Failed to add comment.")
				}
			}
			// Delete the 'status/can-merge' removed noti after the 'status/can-merge' label is added.
			cp.PruneComments(func(comment github.IssueComment) bool {
				return strings.Contains(comment.Body, removeCanMergeLabelNoti)
			})
		} else {
			resp := fmt.Sprintf("adding '"+externalplugins.CanMergeLabel+"' to this PR must have %d LGTMs", owners.NeedsLgtm)
			log.Infof("Reply to /merge request with comment: \"%s\"", resp)
			return gc.CreateComment(org, repoName, number, externalplugins.FormatResponseRaw(body, htmlURL, author, resp))
		}
	}

	return nil
}

// isLGTMSatisfy returns pull request current label number.
func isLGTMSatisfy(prefix string, labels []github.Label, needsLgtm int) bool {
	currentLgtmNumber := 0
	for _, label := range labels {
		if strings.Contains(label.Name, prefix) {
			currentLgtmNumber, _ = strconv.Atoi(strings.Trim(label.Name, prefix))
		}
	}

	return needsLgtm == currentLgtmNumber
}

func isAllGuaranteed(prCommits []github.RepositoryCommit, lastCanMergeTreeHash string, log *logrus.Entry) bool {
	guaranteed := true

	lastGuaranteedIndex := 0
	for i, commit := range prCommits {
		if commit.SHA == lastCanMergeTreeHash {
			lastGuaranteedIndex = i
		}
	}

	unCheckCommits := prCommits[lastGuaranteedIndex+1:]

	for _, commit := range unCheckCommits {
		if commit.Committer.Login != githubUpdateCommitter {
			guaranteed = false
		} else {
			log.Infof("Remove can merge label the tree-hash is: %s", commit.SHA)
		}
	}

	return guaranteed
}
