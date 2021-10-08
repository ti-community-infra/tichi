package matchchecker

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"
)

const (
	// PluginName will register into prow.
	PluginName = "ti-community-matching-checker"
	// CheckerNotificationIdentifier defines the identifier for the review notifications.
	CheckerNotificationIdentifier = "Checker Notification Identifier"
)

var (
	// notificationRegex is the regex that matches the notifications.
	notificationRegex = regexp.MustCompile("<!--" + CheckerNotificationIdentifier + "-->$")
)

type githubClient interface {
	AddLabels(org, repo string, number int, labels ...string) error
	RemoveLabel(org, repo string, number int, label string) error
	CreateComment(owner, repo string, number int, comment string) error
	DeleteComment(org, repo string, ID int) error
	ListIssueComments(org, repo string, number int) ([]github.IssueComment, error)
	ListPRCommits(org, repo string, number int) ([]github.RepositoryCommit, error)
	BotUserChecker() (func(candidate string) bool, error)
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.MatchCheckerFor(repo.Org, repo.Repo)
			var isConfigured bool
			var configInfoStrings []string

			configInfoStrings = append(configInfoStrings, "The plugin has these configurations: <ul>")

			if len(opts.RequiredMatchRules) != 0 {
				isConfigured = true
			}

			for _, rule := range opts.RequiredMatchRules {
				scopes := make([]string, 0)
				if rule.Title {
					scopes = append(scopes, "title")
				}
				if rule.Body {
					scopes = append(scopes, "body")
				}
				if rule.CommitMessage {
					scopes = append(scopes, "commit message")
				}

				configInfoString := fmt.Sprintf(
					"<li>check if %s (at least one) contains the content can be matched by regex: %s</li>",
					strings.Join(scopes, ", "), rule.Regexp)
				configInfoStrings = append(configInfoStrings, configInfoString)
			}

			configInfoStrings = append(configInfoStrings, "</ul>")
			if isConfigured {
				configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
			}
		}

		yamlSnippet, err := plugins.CommentMap.GenYaml(&tiexternalplugins.Configuration{
			TiCommunityMatchChecker: []tiexternalplugins.TiCommunityMatchChecker{
				{
					Repos: []string{"ti-community-infra/test-dev"},
					RequiredMatchRules: []tiexternalplugins.RequiredMatchRule{
						{
							Title:          true,
							Regexp:         "^(\\[TI-[1-9]\\d*\\])+.+: .{10,160}$",
							MissingMessage: "Please fill in the PR title in the correct format.",
							MissingLabel:   "do-not-merge/invalid-title",
						},
					},
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
		}

		pluginHelp := &pluginhelp.PluginHelp{
			Description: fmt.Sprintf("The %s plugin will check the title, body or commits message of "+
				"the issue or PR whether it matches the required rule.", PluginName),
			Config:  configInfo,
			Snippet: yamlSnippet,
			Events:  []string{tiexternalplugins.PullRequestEvent},
		}

		return pluginHelp, nil
	}
}

func HandlePullRequestEvent(gc githubClient, pe *github.PullRequestEvent,
	cfg *tiexternalplugins.Configuration, log *logrus.Entry) error {
	if pe.Action != github.PullRequestActionOpened && pe.Action != github.PullRequestActionEdited &&
		pe.Action != github.PullRequestActionReopened && pe.Action != github.PullRequestActionSynchronize {
		log.Debug("Not a pull request opened action, skipping...")
		return nil
	}

	org := pe.Repo.Owner.Login
	repo := pe.Repo.Name
	num := pe.Number

	blocker := cfg.MatchCheckerFor(org, repo)
	needCheckCommits := false
	rulesForPullRequest := make([]tiexternalplugins.RequiredMatchRule, 0)
	for _, rule := range blocker.RequiredMatchRules {
		if rule.PullRequest {
			rulesForPullRequest = append(rulesForPullRequest, rule)
		}
		if rule.CommitMessage {
			needCheckCommits = true
		}
	}

	// Notice: You need to get the list of commits through the API when you need to check the commit messages.
	commitMessages := make([]string, 0)
	if needCheckCommits {
		commits, err := gc.ListPRCommits(org, repo, num)
		if err != nil {
			log.WithError(err).Errorf("Failed to list PR commits.")
			return err
		}
		for _, commit := range commits {
			commitMessages = append(commitMessages, commit.Commit.Message)
		}
	}

	err := handle(
		gc, log, org, repo, num, pe.PullRequest.Title, pe.PullRequest.Body, commitMessages,
		pe.PullRequest.Labels, rulesForPullRequest,
	)
	if err != nil {
		return err
	}

	return nil
}

func HandleIssueEvent(gc githubClient, ie *github.IssueEvent,
	cfg *tiexternalplugins.Configuration, log *logrus.Entry) error {
	if ie.Action != github.IssueActionOpened && ie.Action != github.IssueActionEdited &&
		ie.Action != github.IssueActionReopened && ie.Issue.PullRequest != nil {
		log.Debug("Not a issue opened or edited action, skipping...")
		return nil
	}

	org := ie.Repo.Owner.Login
	repo := ie.Repo.Name
	num := ie.Issue.Number

	blocker := cfg.MatchCheckerFor(org, repo)
	rulesForIssue := make([]tiexternalplugins.RequiredMatchRule, 0)
	for _, rule := range blocker.RequiredMatchRules {
		if rule.Issue {
			rulesForIssue = append(rulesForIssue, rule)
		}
	}

	err := handle(
		gc, log, org, repo, num, ie.Issue.Title, ie.Issue.Body, nil,
		ie.Issue.Labels, rulesForIssue,
	)
	if err != nil {
		return err
	}

	return nil
}

func handle(
	gc githubClient, log *logrus.Entry, org, repo string, num int, title, body string, commitMessages []string,
	labels []github.Label, rules []tiexternalplugins.RequiredMatchRule,
) error {
	var errs []error
	messages := sets.NewString()
	labelsExisted := sets.NewString()
	labelsNeedDeleted := sets.NewString()
	labelsNeedAdded := sets.NewString()

	for _, label := range labels {
		labelsExisted.Insert(label.Name)
	}

	for _, rule := range rules {
		regex, err := regexp.Compile(rule.Regexp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		titleMatch := false
		if rule.Title {
			titleMatch = regex.MatchString(title)
		}
		bodyMatch := false
		if rule.Body {
			bodyMatch = regex.MatchString(body)
		}
		commitMessageMatch := false
		if rule.CommitMessage {
			for _, message := range commitMessages {
				if regex.MatchString(message) {
					commitMessageMatch = true
					break
				}
			}
		}

		noMatch := !titleMatch && !bodyMatch && !commitMessageMatch
		if noMatch && len(rule.MissingLabel) != 0 && !labelsExisted.Has(rule.MissingLabel) {
			labelsNeedAdded.Insert(rule.MissingLabel)
		} else if !noMatch && len(rule.MissingLabel) != 0 && labelsExisted.Has(rule.MissingLabel) {
			labelsNeedDeleted.Insert(rule.MissingLabel)
		}

		if noMatch && len(rule.MissingMessage) != 0 {
			messages.Insert(rule.MissingMessage)
		}
	}

	// Notice: If a label needs to be added or deleted at the same time, no operation is performed.
	labelsOnConflict := labelsNeedAdded.Intersection(labelsNeedDeleted)

	labelsDeleted := labelsNeedDeleted.Difference(labelsOnConflict).List()
	if len(labelsDeleted) != 0 {
		for _, label := range labelsDeleted {
			err := gc.RemoveLabel(org, repo, num, label)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	labelsAdded := labelsNeedAdded.Difference(labelsOnConflict).List()
	if len(labelsAdded) != 0 {
		err := gc.AddLabels(org, repo, num, labelsAdded...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	// Clean up the old notifications.
	err := cleanUpOldNotifications(gc, log, org, repo, num)
	if err != nil {
		return err
	}

	// Add the new notification comment.
	if len(messages) != 0 {
		notification, err := generateNotification(messages.List())
		if err != nil {
			return err
		}
		err = gc.CreateComment(org, repo, num, notification)
		if err != nil {
			return err
		}
	}

	return utilerrors.NewAggregate(errs)
}

// cleanUpOldNotifications used to clean up old Notifications.
func cleanUpOldNotifications(gc githubClient, log *logrus.Entry, org, repo string, num int) error {
	botUserChecker, err := gc.BotUserChecker()
	if err != nil {
		return fmt.Errorf("failed to get bot name: %v", err)
	}
	issueComments, err := gc.ListIssueComments(org, repo, num)
	if err != nil {
		return fmt.Errorf("failed to issue comments: %v", err)
	}
	notifications := filterComments(issueComments, notificationMatcher(botUserChecker))
	if len(notifications) != 0 {
		for _, notification := range notifications {
			notif := notification
			if err := gc.DeleteComment(org, repo, notif.ID); err != nil {
				log.WithError(err).Errorf("Failed to delete comment from %s/%s#%d, ID: %d.", org, repo, num, notif.ID)
			}
		}
	}
	return nil
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

// generateNotification returns the comment body that we want the checker plugin to display on Issue / PR.
func generateNotification(messages []string) (string, error) {
	msg := strings.Join(messages, "\n<hr>\n")
	notification, err := generateTemplate(`
[Checker Notification]

{{ .msg }}

<!--{{ .checkerNotificationIdentifier }}-->
`, "message", map[string]interface{}{
		"msg":                           msg,
		"checkerNotificationIdentifier": CheckerNotificationIdentifier,
	})
	if err != nil {
		return "", err
	}

	return notification, nil
}
