//nolint:gocritic
package lgtm

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
const PluginName = "ti-community-lgtm"

var (
	configInfoReviewActsAsLgtm = `Reviews of "approve" or "request changes" act as adding or removing LGTM.`

	// LGTMRe is the regex that matches lgtm comments
	LGTMRe = regexp.MustCompile(`(?mi)^/lgtm\s*$`)
	// LGTMCancelRe is the regex that matches lgtm cancel comments
	LGTMCancelRe = regexp.MustCompile(`(?mi)^/lgtm cancel\s*$`)
)

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *externalplugins.ConfigAgent) func(
	enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()
		for _, repo := range enabledRepos {
			opts := cfg.LgtmFor(repo.Org, repo.Repo)
			var isConfigured bool
			var configInfoStrings []string
			configInfoStrings = append(configInfoStrings, "The plugin has the following configuration:<ul>")
			if opts.ReviewActsAsLgtm {
				configInfoStrings = append(configInfoStrings, "<li>"+configInfoReviewActsAsLgtm+"</li>")
				isConfigured = true
			}
			configInfoStrings = append(configInfoStrings, "</ul>")
			if isConfigured {
				configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
			}
		}
		pluginHelp := &pluginhelp.PluginHelp{
			Description: "The ti-community-lgtm plugin manages the application and " +
				"removal of the 'status/LGT{number}' (Looks Good To Me) label which is typically used to gate merging.",
			Config: configInfo,
		}

		pluginHelp.AddCommand(pluginhelp.Command{
			Usage:       "/lgtm [cancel] or GitHub Review action",
			Description: "Adds or removes the 'status/LGT{number}' label which is typically used to gate merging.",
			Featured:    true,
			WhoCanUse:   "Collaborators on the repository. '/lgtm cancel' can be used additionally by the PR author.",
			Examples: []string{
				"/lgtm",
				"/lgtm cancel",
				"<a href=\"https://help.github.com/articles/about-pull-request-reviews/\">'Approve' or 'Request Changes'</a>"},
		})
		return pluginHelp, nil
	}
}

type githubClient interface {
	IsCollaborator(owner, repo, login string) (bool, error)
	AddLabel(owner, repo string, number int, label string) error
	AssignIssue(owner, repo string, number int, assignees []string) error
	CreateComment(owner, repo string, number int, comment string) error
	RemoveLabel(owner, repo string, number int, label string) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
	ListIssueComments(org, repo string, number int) ([]github.IssueComment, error)
	DeleteComment(org, repo string, ID int) error
	BotName() (string, error)
	GetSingleCommit(org, repo, SHA string) (github.SingleCommit, error)
	ListTeams(org string) ([]github.Team, error)
	ListTeamMembers(id int, role string) ([]github.TeamMember, error)
}

// reviewCtx contains information about each review event.
type reviewCtx struct {
	author, issueAuthor, body, htmlURL string
	repo                               github.Repo
	number                             int
}

// HandleIssueCommentEvent handles a GitHub issue comment event and adds or removes a
// "status/LGT{number}" label.
func HandleIssueCommentEvent(gc githubClient, ice *github.IssueCommentEvent, cfg *externalplugins.Configuration,
	ol ownersclient.OwnersLoader, log *logrus.Entry) error {
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

	// If we create an "/lgtm" comment, add lgtm if necessary.
	// If we create a "/lgtm cancel" comment, remove lgtm if necessary.
	wantLGTM := false
	if LGTMRe.MatchString(rc.body) {
		wantLGTM = true
	} else if LGTMCancelRe.MatchString(rc.body) {
		wantLGTM = false
	} else {
		return nil
	}

	// Use common handler to do the rest.
	return handle(wantLGTM, cfg, rc, gc, ol, log)
}

func HandlePullReviewEvent(gc githubClient, pullReviewEvent *github.ReviewEvent,
	cfg *externalplugins.Configuration, ol ownersclient.OwnersLoader, log *logrus.Entry) error {
	// If ReviewActsAsLgtm is disabled, ignore review event.
	opts := cfg.LgtmFor(pullReviewEvent.Repo.Owner.Login, pullReviewEvent.Repo.Name)
	if !opts.ReviewActsAsLgtm {
		return nil
	}

	rc := reviewCtx{
		author:      pullReviewEvent.Review.User.Login,
		issueAuthor: pullReviewEvent.PullRequest.User.Login,
		repo:        pullReviewEvent.Repo,
		number:      pullReviewEvent.PullRequest.Number,
		body:        pullReviewEvent.Review.Body,
		htmlURL:     pullReviewEvent.Review.HTMLURL,
	}

	// Only react to reviews that are being submitted (not editted or dismissed).
	if pullReviewEvent.Action != github.ReviewActionSubmitted {
		return nil
	}

	// If the review event body contains an '/lgtm' or '/lgtm cancel' comment,
	// skip handling the review event
	if LGTMRe.MatchString(rc.body) || LGTMCancelRe.MatchString(rc.body) {
		return nil
	}

	// The review webhook returns state as lowercase, while the review API
	// returns state as uppercase. Uppercase the value here so it always
	// matches the constant.
	reviewState := github.ReviewState(strings.ToUpper(string(pullReviewEvent.Review.State)))

	// If we review with Approve, add lgtm if necessary.
	// If we review with Request Changes, remove lgtm if necessary.
	wantLGTM := false
	if reviewState == github.ReviewStateApproved {
		wantLGTM = true
	} else if reviewState == github.ReviewStateChangesRequested {
		wantLGTM = false
	} else {
		return nil
	}

	// Use common handler to do the rest.
	return handle(wantLGTM, cfg, rc, gc, ol, log)
}

func HandlePullReviewCommentEvent(gc githubClient, pullReviewCommentEvent *github.ReviewCommentEvent,
	cfg *externalplugins.Configuration, ol ownersclient.OwnersLoader, log *logrus.Entry) error {
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

	// If we create an "/lgtm" comment, add lgtm if necessary.
	// If we create a "/lgtm cancel" comment, remove lgtm if necessary.
	wantLGTM := false
	if LGTMRe.MatchString(rc.body) {
		wantLGTM = true
	} else if LGTMCancelRe.MatchString(rc.body) {
		wantLGTM = false
	} else {
		return nil
	}

	// Use common handler to do the rest.
	return handle(wantLGTM, cfg, rc, gc, ol, log)
}

func handle(wantLGTM bool, config *externalplugins.Configuration, rc reviewCtx,
	gc githubClient, ol ownersclient.OwnersLoader, log *logrus.Entry) error {
	author := rc.author
	issueAuthor := rc.issueAuthor
	number := rc.number
	body := rc.body
	htmlURL := rc.htmlURL
	org := rc.repo.Owner.Login
	repoName := rc.repo.Name

	// Author cannot LGTM own PR, comment and abort.
	isAuthor := author == issueAuthor
	if isAuthor && wantLGTM {
		resp := "you cannot LGTM your own PR."
		log.Infof("Commenting with \"%s\".", resp)
		return gc.CreateComment(rc.repo.Owner.Login, rc.repo.Name, rc.number,
			externalplugins.FormatResponseRaw(rc.body, rc.htmlURL, rc.author, resp))
	}

	// Get ti-community-lgtm config.
	opts := config.LgtmFor(rc.repo.Owner.Login, rc.repo.Name)
	url := fmt.Sprintf(ownersclient.OwnersURLFmt, opts.PullOwnersEndpoint, org, repoName, number)
	reviewersAndNeedsLGTM, err := ol.LoadOwners(opts.PullOwnersEndpoint, org, repoName, number)
	if err != nil {
		return err
	}

	reviewers := sets.String{}
	for _, reviewer := range reviewersAndNeedsLGTM.Reviewers {
		reviewers.Insert(reviewer)
	}

	// Not reviewers but want add LGTM.
	if !reviewers.Has(author) && wantLGTM {
		resp := "adding LGTM is restricted to reviewers in [list](" + url + ")."
		log.Infof("Reply to /lgtm request with comment: \"%s\"", resp)
		return gc.CreateComment(org, repoName, number, externalplugins.FormatResponseRaw(body, htmlURL, author, resp))
	}

	// Not author or reviewers but want remove LGTM.
	if !reviewers.Has(author) && !isAuthor && !wantLGTM {
		resp := "removing LGTM is restricted to reviewers in [list](" + url + ") or PR author."
		log.Infof("Reply to /lgtm cancel request with comment: \"%s\"", resp)
		return gc.CreateComment(org, repoName, number, externalplugins.FormatResponseRaw(body, htmlURL, author, resp))
	}

	// Now we update the LGTM labels, having checked all cases where changing.
	// Only add the label if it doesn't have it, and vice versa.
	labels, err := gc.GetIssueLabels(org, repoName, number)
	if err != nil {
		log.WithError(err).Error("Failed to get issue labels.")
	}

	currentLabel, nextLabel := getCurrentAndNextLabel(externalplugins.LgtmLabelPrefix, labels,
		reviewersAndNeedsLGTM.NeedsLgtm)

	// Remove the label if necessary, we're done after this.
	if currentLabel != "" && !wantLGTM {
		log.Info("Removing LGTM label.")
		if err := gc.RemoveLabel(org, repoName, number, currentLabel); err != nil {
			return err
		}
	} else if nextLabel != "" && wantLGTM {
		log.Info("Adding LGTM label.")
		// Remove current label.
		if currentLabel != "" {
			if err := gc.RemoveLabel(org, repoName, number, currentLabel); err != nil {
				return err
			}
		}
		if err := gc.AddLabel(org, repoName, number, nextLabel); err != nil {
			return err
		}
	}

	return nil
}

// getCurrentAndNextLabel returns pull request current label and next required label.
func getCurrentAndNextLabel(prefix string, labels []github.Label, needsLgtm int) (string, string) {
	currentLabel := ""
	nextLabel := ""
	for _, label := range labels {
		if strings.Contains(label.Name, prefix) {
			currentLabel = label.Name
			currentLgtmNumber, _ := strconv.Atoi(strings.Trim(label.Name, prefix))
			if currentLgtmNumber < needsLgtm {
				nextLabel = fmt.Sprintf("%s%d", prefix, currentLgtmNumber+1)
			}
		}
	}
	if currentLabel == "" {
		nextLabel = fmt.Sprintf("%s%d", prefix, 1)
	}

	return currentLabel, nextLabel
}
