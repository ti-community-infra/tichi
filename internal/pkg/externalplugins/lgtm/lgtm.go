//nolint:gocritic
package lgtm

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/ownersclient"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

// PluginName will register into prow.
const (
	PluginName = "ti-community-lgtm"

	// ReviewNotificationName defines the name used in the title for the review notifications.
	ReviewNotificationName = "Review Notification"
	prProcessLink          = "https://book.prow.tidb.io/#/en/workflows/pr"
	commandHelpLink        = "https://prow-dev.tidb.io/command-help"
)

var (
	configInfoReviewActsAsLgtm = "'Approve' review action will add a LGTM " +
		"and 'Request Changes' review action will remove the LGTM."

	// lgtmRe is the regex that matches lgtm comments.
	lgtmRe = regexp.MustCompile(`(?mi)^/lgtm\s*$`)
	// lgtmCancelRe is the regex that matches lgtm cancel comments.
	lgtmCancelRe      = regexp.MustCompile(`(?mi)^/lgtm cancel\s*$`)
	notificationRegex = regexp.MustCompile(`(?is)^\[` + ReviewNotificationName + `\] *?([^\n]*)(?:\n\n(.*))?`)
	reviewersRegex    = regexp.MustCompile(`(?i)- [@]*([a-z0-9](?:-?[a-z0-9]){0,38})`)
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
			configInfoStrings = append(configInfoStrings, "The plugin has these configurations:<ul>")
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
			Description: "The ti-community-lgtm plugin manages the 'status/LGT{number}' (Looks Good To Me) label.",
			Config:      configInfo,
		}

		pluginHelp.AddCommand(pluginhelp.Command{
			Usage:       "/lgtm [cancel] or triggers by GitHub review action.",
			Description: "Add or remove the 'status/LGT{number}' label. Additionally, the PR author can use '/lgtm cancel'.",
			Featured:    true,
			WhoCanUse:   "Collaborators of this repository. Additionally, the PR author can use '/lgtm cancel'.",
			Examples: []string{
				"/lgtm",
				"/lgtm cancel",
				"<a href=\"https://help.github.com/articles/about-pull-request-reviews/\">'Approve' or 'Request Changes'</a>"},
		})
		return pluginHelp, nil
	}
}

type githubClient interface {
	AddLabel(owner, repo string, number int, label string) error
	CreateComment(owner, repo string, number int, comment string) error
	RemoveLabel(owner, repo string, number int, label string) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	BotUserChecker() (func(candidate string) bool, error)
	ListIssueComments(org, repo string, number int) ([]github.IssueComment, error)
	DeleteComment(org, repo string, ID int) error
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
	if lgtmRe.MatchString(rc.body) {
		wantLGTM = true
	} else if lgtmCancelRe.MatchString(rc.body) {
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
	if lgtmRe.MatchString(rc.body) {
		wantLGTM = true
	} else if lgtmCancelRe.MatchString(rc.body) {
		wantLGTM = false
	} else {
		return nil
	}

	// Use common handler to do the rest.
	return handle(wantLGTM, cfg, rc, gc, ol, log)
}

func HandlePullRequestEvent(gc githubClient, pe *github.PullRequestEvent,
	_ *externalplugins.Configuration, log *logrus.Entry) error {
	if pe.Action != github.PullRequestActionOpened && pe.Action != github.PullRequestActionReopened {
		log.Debug("Not a pull request opened action, skipping...")
		return nil
	}

	org := pe.PullRequest.Base.Repo.Owner.Login
	repo := pe.PullRequest.Base.Repo.Name
	number := pe.PullRequest.Number

	reviewMsg, err := getMessage(nil, commandHelpLink, prProcessLink, org, repo)
	if err != nil {
		return err
	}

	return gc.CreateComment(org, repo, number, *reviewMsg)
}

func handle(wantLGTM bool, config *externalplugins.Configuration, rc reviewCtx,
	gc githubClient, ol ownersclient.OwnersLoader, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handle")
	}()

	author := rc.author
	issueAuthor := rc.issueAuthor
	number := rc.number
	body := rc.body
	htmlURL := rc.htmlURL
	org := rc.repo.Owner.Login
	repoName := rc.repo.Name
	fetchErr := func(context string, err error) error {
		return fmt.Errorf("failed to get %s for %s/%s#%d: %v", context, org, repoName, number, err)
	}

	// Author cannot LGTM own PR, comment and abort.
	isAuthor := author == issueAuthor
	if isAuthor && wantLGTM {
		resp := "you cannot `/lgtm` your own PR."
		log.Infof("Commenting \"%s\".", resp)
		return gc.CreateComment(rc.repo.Owner.Login, rc.repo.Name, rc.number,
			externalplugins.FormatResponseRaw(rc.body, rc.htmlURL, rc.author, resp))
	}

	// Get ti-community-lgtm config.
	opts := config.LgtmFor(rc.repo.Owner.Login, rc.repo.Name)
	tichiURL := fmt.Sprintf(ownersclient.OwnersURLFmt, config.TichiWebURL, org, repoName, number)
	reviewersAndNeedsLGTM, err := ol.LoadOwners(opts.PullOwnersEndpoint, org, repoName, number)
	if err != nil {
		return fetchErr("owners info", err)
	}

	reviewers := sets.String{}
	for _, reviewer := range reviewersAndNeedsLGTM.Reviewers {
		reviewers.Insert(reviewer)
	}

	// Not reviewers but want to add LGTM.
	if !reviewers.Has(author) && wantLGTM {
		resp := "`/lgtm` is only allowed for the reviewers in [list](" + tichiURL + ")."
		log.Infof("Reply /lgtm request in comment: \"%s\"", resp)
		return gc.CreateComment(org, repoName, number, externalplugins.FormatResponseRaw(body, htmlURL, author, resp))
	}

	// Not author or reviewers but want to remove LGTM.
	if !reviewers.Has(author) && !isAuthor && !wantLGTM {
		resp := "`/lgtm cancel` is only allowed for the PR author or the reviewers in [list](" + tichiURL + ")."
		log.Infof("Reply /lgtm cancel request in comment: \"%s\"", resp)
		return gc.CreateComment(org, repoName, number, externalplugins.FormatResponseRaw(body, htmlURL, author, resp))
	}

	botUserChecker, err := gc.BotUserChecker()
	if err != nil {
		return fetchErr("bot name", err)
	}

	issueComments, err := gc.ListIssueComments(org, repoName, number)
	if err != nil {
		return fetchErr("issue comments", err)
	}
	notifications := filterComments(issueComments, notificationMatcher(botUserChecker))
	log.Infof("Get notifications: %v", notifications)

	// Now we update the LGTM labels, having checked all cases where changing.
	// Only add the label if it doesn't have it, and vice versa.
	labels, err := gc.GetIssueLabels(org, repoName, number)
	if err != nil {
		return fetchErr("issue labels", err)
	}

	currentLabel, nextLabel := getCurrentAndNextLabel(externalplugins.LgtmLabelPrefix, labels,
		reviewersAndNeedsLGTM.NeedsLgtm)
	// Remove the label if necessary, we're done after this.
	if currentLabel != "" && !wantLGTM {
		log.Info("Removing LGTM label.")
		if err := gc.RemoveLabel(org, repoName, number, currentLabel); err != nil {
			return err
		}
		reviewMsg, err := getMessage(nil, commandHelpLink, prProcessLink, org, repoName)
		if err != nil {
			return err
		}

		for _, notification := range notifications {
			notif := notification
			if err := gc.DeleteComment(org, repoName, notif.ID); err != nil {
				log.WithError(err).Errorf("Failed to delete comment from %s/%s#%d, ID: %d.", org, repoName, number, notif.ID)
			}
		}

		if err := gc.CreateComment(org, repoName, number, *reviewMsg); err != nil {
			log.WithError(err).Errorf("Failed to create comment on %s/%s#%d: %q.", org, repoName, number, *reviewMsg)
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

		latestNotification := getLastComment(notifications)
		reviewedReviewers := getReviewersFromNotification(latestNotification)
		// Ignore already reviewed reviewer.
		if reviewedReviewers.Has(author) {
			log.Infof("Ignore %s's multiple reviews.", author)
			return nil
		}

		// Add author as reviewers and get new notification.
		reviewedReviewers.Insert(author)
		reviewMsg, err := getMessage(reviewedReviewers.List(), commandHelpLink, prProcessLink, org, repoName)
		if err != nil {
			return err
		}

		for _, notification := range notifications {
			notif := notification
			if err := gc.DeleteComment(org, repoName, notif.ID); err != nil {
				log.WithError(err).Errorf("Failed to delete comment from %s/%s#%d, ID: %d.", org, repoName, number, notif.ID)
			}
		}

		if err := gc.CreateComment(org, repoName, number, *reviewMsg); err != nil {
			log.WithError(err).Errorf("Failed to create comment on %s/%s#%d: %q.", org, repoName, number, *reviewMsg)
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

// getReviewersFromNotification get the reviewers from latest notification.
func getReviewersFromNotification(latestNotification *github.IssueComment) sets.String {
	result := sets.String{}
	if latestNotification == nil {
		return result
	}

	reviewers := reviewersRegex.FindAllStringSubmatch(latestNotification.Body, -1)

	reviewerNameIndex := 2
	for _, reviewer := range reviewers {
		// Example: - a => [[- a a]]
		if len(reviewer) == reviewerNameIndex+1 {
			result.Insert(reviewer[reviewerNameIndex])
		}
	}
	return result
}

// filterComments will filtering the issue comments by filter.
func filterComments(comments []github.IssueComment,
	filter func(comment *github.IssueComment) bool) []*github.IssueComment {
	filtered := make([]*github.IssueComment, 0, len(comments))
	for _, comment := range comments {
		c := comment
		if filter(&c) {
			filtered = append(filtered, &c)
		}
	}
	return filtered
}

// getLastComment get the last issue comment.
func getLastComment(issueComments []*github.IssueComment) *github.IssueComment {
	if len(issueComments) == 0 {
		return nil
	}
	return issueComments[len(issueComments)-1]
}

// getMessage returns the comment body that we want the approve plugin to display on PRs
// The comment shows:
// 	- a list of reviewers
// 	- how an approver can indicate their lgtm
// 	- how an approver can cancel their lgtm
func getMessage(reviewers []string, commandHelpLink, prProcessLink, org, repo string) (*string, error) {
	// nolint:lll
	message, err := generateTemplate(`
{{if .reviewers}}
This pull request has been reviewed by:

{{range $index, $reviewer := .reviewers}}- {{$reviewer}}`+"\n"+`{{end}}

{{else}}
This pull request has not been reviewed.
{{end}}

To complete the [pull request process]({{ .prProcessLink }}), please ask the reviewers in the [list]() to review by filling `+"`/cc @reviewer`"+` in the comment.
After reviewing, you can assign this pull request to the committer in the [list]() by filling  `+"`/assign @committer`"+` in the comment to help you merge this pull request.

The full list of commands accepted by this bot can be found [here]({{ .commandHelpLink }}?repo={{ .org }}%2F{{ .repo }}).

<details>

Reviewer can indicate their review by writing `+"`/lgtm`"+` in a comment.
Reviewer can cancel approval by writing `+"`/lgtm cancel`"+` in a comment.
</details>`, "message", map[string]interface{}{"reviewers": reviewers, "commandHelpLink": commandHelpLink,
		"prProcessLink": prProcessLink, "org": org, "repo": repo})
	if err != nil {
		return nil, err
	}

	return notification(ReviewNotificationName, "", message), nil
}

// generateTemplate takes a template, name and data, and generates
// the corresponding string.
func generateTemplate(templ, name string, data interface{}) (string, error) {
	buf := bytes.NewBufferString("")
	if messageTemplate, err := template.New(name).Parse(templ); err != nil {
		return "", fmt.Errorf("failed to parse template for %s: %v", name, err)
	} else if err := messageTemplate.Execute(buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template for %s: %v", name, err)
	}
	return buf.String(), nil
}

// notification create a notification message.
func notification(name, arguments, context string) *string {
	str := "[" + strings.ToUpper(name) + "]"

	args := strings.TrimSpace(arguments)
	if args != "" {
		str += " " + args
	}

	ctx := strings.TrimSpace(context)
	if ctx != "" {
		str += "\n\n" + ctx
	}

	return &str
}

// notificationMatcher matches issue comments for notifications.
func notificationMatcher(isBot func(string) bool) func(comment *github.IssueComment) bool {
	return func(c *github.IssueComment) bool {
		// Only match robot's comment.
		if !isBot(c.User.Login) {
			return false
		}
		match := notificationRegex.FindStringSubmatch(c.Body)
		return len(match) > 0
	}
}
