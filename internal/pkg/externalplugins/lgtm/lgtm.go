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
	"github.com/ti-community-infra/tichi/internal/pkg/ownersclient"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
)

const (
	// PluginName will register into prow.
	PluginName = "ti-community-lgtm"
	// ReviewNotificationName defines the name used in the title for the review notifications.
	ReviewNotificationName = "Review Notification"
	// ReviewNotificationIdentifier defines the identifier for the review notifications.
	ReviewNotificationIdentifier = "Review Notification Identifier"
)

var (
	// notificationRegex is the regex that matches the notifications.
	notificationRegex = regexp.MustCompile("<!--" + ReviewNotificationIdentifier + "-->$")
	// reviewersRegex is the regex that matches the reviewers, such as: - hi-rustin.
	reviewersRegex = regexp.MustCompile(`(?i)- [@]*([a-z0-9](?:-?[a-z0-9]){0,38})`)
)

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(_ *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
	return func(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		yamlSnippet, err := plugins.CommentMap.GenYaml(&tiexternalplugins.Configuration{
			TiCommunityLgtm: []tiexternalplugins.TiCommunityLgtm{
				{
					Repos:              []string{"ti-community-infra/test-dev"},
					PullOwnersEndpoint: "https://prow-dev.tidb.net/ti-community-owners",
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
		}
		pluginHelp := &pluginhelp.PluginHelp{
			Description: "The ti-community-lgtm plugin manages the 'status/LGT{number}' (Looks Good To Me) label.",
			Snippet:     yamlSnippet,
			Events:      []string{tiexternalplugins.PullRequestReviewEvent, tiexternalplugins.PullRequestEvent},
		}

		pluginHelp.AddCommand(pluginhelp.Command{
			Usage:       "Triggered by GitHub review action: 'Approve' or 'Request Changes'.",
			Description: "Add or remove the 'status/LGT{number}' label.",
			Featured:    true,
			WhoCanUse:   "Reviewers of this pull request.",
			Examples: []string{
				"<a href=\"https://help.github.com/articles/about-pull-request-reviews/\">'Approve' or 'Request Changes'</a>"},
		})
		return pluginHelp, nil
	}
}

type githubClient interface {
	AddLabel(owner, repo string, number int, label string) error
	RemoveLabel(owner, repo string, number int, label string) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	CreateComment(owner, repo string, number int, comment string) error
	EditComment(org, repo string, id int, comment string) error
	ListIssueComments(org, repo string, number int) ([]github.IssueComment, error)
	DeleteComment(org, repo string, ID int) error
	BotUserChecker() (func(candidate string) bool, error)
}

// reviewCtx contains information about each review event.
type reviewCtx struct {
	author, issueAuthor, body, htmlURL string
	repo                               github.Repo
	number                             int
}

func HandlePullReviewEvent(gc githubClient, pullReviewEvent *github.ReviewEvent,
	cfg *tiexternalplugins.Configuration, ol ownersclient.OwnersLoader, log *logrus.Entry) error {
	rc := reviewCtx{
		author:      pullReviewEvent.Review.User.Login,
		issueAuthor: pullReviewEvent.PullRequest.User.Login,
		repo:        pullReviewEvent.Repo,
		number:      pullReviewEvent.PullRequest.Number,
		body:        pullReviewEvent.Review.Body,
		htmlURL:     pullReviewEvent.Review.HTMLURL,
	}

	// Only react to reviews that are being submitted (not edited or dismissed).
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

func HandlePullRequestEvent(gc githubClient, pe *github.PullRequestEvent,
	config *tiexternalplugins.Configuration, log *logrus.Entry) error {
	if pe.Action != github.PullRequestActionOpened {
		log.Debug("Not a pull request opened action, skipping...")
		return nil
	}

	org := pe.PullRequest.Base.Repo.Owner.Login
	repo := pe.PullRequest.Base.Repo.Name
	number := pe.PullRequest.Number
	tichiURL := fmt.Sprintf(ownersclient.OwnersURLFmt, config.TichiWebURL, org, repo, number)

	reviewMsg, err := getMessage(nil, config.CommandHelpLink, config.PRProcessLink, tichiURL, org, repo)
	if err != nil {
		return err
	}

	return gc.CreateComment(org, repo, number, *reviewMsg)
}

func handle(wantLGTM bool, config *tiexternalplugins.Configuration, rc reviewCtx,
	gc githubClient, ol ownersclient.OwnersLoader, log *logrus.Entry) error {
	funcStart := time.Now()
	defer func() {
		log.WithField("duration", time.Since(funcStart).String()).Debug("Completed handle")
	}()

	currentReviewer := rc.author
	number := rc.number
	body := rc.body
	htmlURL := rc.htmlURL
	org := rc.repo.Owner.Login
	repo := rc.repo.Name
	fetchErr := func(context string, err error) error {
		return fmt.Errorf("failed to get %s for %s/%s#%d: %v", context, org, repo, number, err)
	}

	// Get ti-community-lgtm config.
	opts := config.LgtmFor(rc.repo.Owner.Login, rc.repo.Name)
	tichiURL := fmt.Sprintf(ownersclient.OwnersURLFmt, config.TichiWebURL, org, repo, number)
	reviewersAndNeedsLGTM, err := ol.LoadOwners(opts.PullOwnersEndpoint, org, repo, number)
	if err != nil {
		return fetchErr("owners info", err)
	}

	reviewers := sets.String{}
	for _, reviewer := range reviewersAndNeedsLGTM.Reviewers {
		reviewers.Insert(reviewer)
	}

	// Not reviewers but want to add LGTM.
	if !reviewers.Has(currentReviewer) && wantLGTM {
		resp := "Thanks for your review. "
		resp += "The bot only counts approvals from reviewers and higher roles in [list](" + tichiURL + "), "
		resp += "but you're still welcome to leave your comments."
		log.Infof("Reply approve pull request in comment: \"%s\"", resp)
		if !opts.IgnoreInvalidReviewPrompt {
			return gc.CreateComment(org, repo, number, tiexternalplugins.FormatResponseRaw(body, htmlURL, currentReviewer, resp))
		}
		return nil
	}

	// Not reviewers but want to remove LGTM.
	if !reviewers.Has(currentReviewer) && !wantLGTM {
		resp := "Request changes is only allowed for the reviewers in [list](" + tichiURL + ")."
		log.Infof("Reply request changes pull request in comment: \"%s\"", resp)
		if !opts.IgnoreInvalidReviewPrompt {
			return gc.CreateComment(org, repo, number, tiexternalplugins.FormatResponseRaw(body, htmlURL, currentReviewer, resp))
		}
		return nil
	}

	labels, err := gc.GetIssueLabels(org, repo, number)
	if err != nil {
		return fetchErr("issue labels", err)
	}
	botUserChecker, err := gc.BotUserChecker()
	if err != nil {
		return fetchErr("bot name", err)
	}
	issueComments, err := gc.ListIssueComments(org, repo, number)
	if err != nil {
		return fetchErr("issue comments", err)
	}
	notifications := filterComments(issueComments, notificationMatcher(botUserChecker))
	latestNotification := getLastComment(notifications)
	cleanupRedundantNotifications := func() {
		if len(notifications) != 0 {
			for _, notification := range notifications[:len(notifications)-1] {
				notif := notification
				if err := gc.DeleteComment(org, repo, notif.ID); err != nil {
					log.WithError(err).Errorf("Failed to delete comment from %s/%s#%d, ID: %d.", org, repo, number, notif.ID)
				}
			}
		}
	}

	// Now we update the LGTM labels, having checked all cases where changing.
	// Only add the label if it doesn't have it, and vice versa.
	currentLabel, nextLabel := getCurrentAndNextLabel(tiexternalplugins.LgtmLabelPrefix, labels,
		reviewersAndNeedsLGTM.NeedsLgtm)
	// Remove the label if necessary, we're done after this.
	if currentLabel != "" && !wantLGTM {
		newMsg, err := getMessage(nil, config.CommandHelpLink, config.PRProcessLink, tichiURL, org, repo)
		if err != nil {
			return err
		}

		// Create or update the review notification comment.
		if latestNotification == nil {
			err := gc.CreateComment(org, repo, number, *newMsg)
			if err != nil {
				return err
			}
		} else {
			err := gc.EditComment(org, repo, latestNotification.ID, *newMsg)
			if err != nil {
				return err
			}
		}

		log.Info("Removing LGTM label.")
		if err := gc.RemoveLabel(org, repo, number, currentLabel); err != nil {
			return err
		}

		// Clean up redundant notifications after we added the new notification.
		cleanupRedundantNotifications()
	} else if nextLabel != "" && wantLGTM {
		reviewedReviewers := getReviewersFromNotification(latestNotification)
		// Ignore already reviewed reviewer.
		if reviewedReviewers.Has(currentReviewer) {
			log.Infof("Ignore %s's multiple reviews.", currentReviewer)
			return nil
		}

		// Add currentReviewer as reviewers and create new notification.
		reviewedReviewers.Insert(currentReviewer)
		newMsg, err := getMessage(reviewedReviewers.List(), config.CommandHelpLink, config.PRProcessLink, tichiURL, org, repo)
		if err != nil {
			return err
		}

		// Create or update the review notification comment.
		if latestNotification == nil {
			err := gc.CreateComment(org, repo, number, *newMsg)
			if err != nil {
				return err
			}
		} else {
			err := gc.EditComment(org, repo, latestNotification.ID, *newMsg)
			if err != nil {
				return err
			}
		}

		if err := updateLabels(gc, log, org, repo, number, currentLabel, nextLabel); err != nil {
			return err
		}

		// Clean up redundant notifications after we added the new notification.
		cleanupRedundantNotifications()
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

func updateLabels(gc githubClient, log *logrus.Entry, org, repo string, number int,
	currentLabel, nextLabel string) error {
	log.Info("Adding LGTM label.")

	// Remove current label.
	if currentLabel != "" {
		if err := gc.RemoveLabel(org, repo, number, currentLabel); err != nil {
			return err
		}
	}

	return gc.AddLabel(org, repo, number, nextLabel)
}

// getReviewersFromNotification get the reviewers from latest notification.
func getReviewersFromNotification(latestNotification *github.IssueComment) sets.String {
	result := sets.String{}
	if latestNotification == nil {
		return result
	}

	reviewers := reviewersRegex.FindAllStringSubmatch(latestNotification.Body, -1)

	reviewerNameIndex := 1
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
//   - a list of reviewed reviewers
//   - how an approver can indicate their lgtm
//   - how an approver can cancel their lgtm
func getMessage(reviewedReviewers []string, commandHelpLink,
	prProcessLink, ownersLink, org, repo string) (*string, error) {
	//nolint:lll
	message, err := generateTemplate(`
{{if .reviewers}}
This pull request has been approved by:

{{range $index, $reviewer := .reviewers}}- {{$reviewer}}`+"\n"+`{{end}}

{{else}}
This pull request has not been approved.
{{end}}

To complete the [pull request process]({{ .prProcessLink }}), please ask the reviewers in the [list]({{ .ownersLink }}) to review by filling `+"`/cc @reviewer`"+` in the comment.
After your PR has acquired the required number of LGTMs, you can assign this pull request to the committer in the [list]({{ .ownersLink }}) by filling  `+"`/assign @committer`"+` in the comment to help you merge this pull request.

The full list of commands accepted by this bot can be found [here]({{ .commandHelpLink }}?repo={{ .org }}%2F{{ .repo }}).

<details>

Reviewer can indicate their review by submitting an approval review.
Reviewer can cancel approval by submitting a request changes review.
</details>

<!--{{ .reviewNotificationIdentifier }}-->
`, "message", map[string]interface{}{
		"reviewers":                    reviewedReviewers,
		"commandHelpLink":              commandHelpLink,
		"prProcessLink":                prProcessLink,
		"ownersLink":                   ownersLink,
		"org":                          org,
		"repo":                         repo,
		"reviewNotificationIdentifier": ReviewNotificationIdentifier,
	})
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
