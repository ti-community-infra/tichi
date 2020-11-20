package label

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

const PluginName = "ti-community-label"

var (
	labelRegexp             = `(?m)^/(%s)\s*(.*)$`
	removeLabelRegexp       = `(?m)^/remove-(%s)\s*(.*)$`
	customLabelRegex        = regexp.MustCompile(`(?m)^/label\s*(.*)$`)
	customRemoveLabelRegex  = regexp.MustCompile(`(?m)^/remove-label\s*(.*)$`)
	nonExistentLabelOnIssue = "Those labels are not set on the issue: `%v`"
)

type githubClient interface {
	CreateComment(owner, repo string, number int, comment string) error
	AddLabel(owner, repo string, number int, label string) error
	RemoveLabel(owner, repo string, number int, label string) error
	GetRepoLabels(owner, repo string) ([]github.Label, error)
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *externalplugins.ConfigAgent) func(
	enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		labelConfig := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.LabelFor(repo.Org, repo.Repo)

			var prefixConfigMsg, additionalLabelsConfigMsg string
			if opts.Prefixes != nil {
				prefixConfigMsg = fmt.Sprintf("The label plugin also includes commands based on %v prefixes.", opts.Prefixes)
			}
			if opts.AdditionalLabels != nil {
				additionalLabelsConfigMsg = fmt.Sprintf("%v labels can be used with the `/[remove-]label` command.",
					opts.AdditionalLabels)
			}
			labelConfig[repo.String()] = prefixConfigMsg + additionalLabelsConfigMsg
		}

		pluginHelp := &pluginhelp.PluginHelp{
			Description: "The label plugin provides commands that add or remove certain types of labels. " +
				"Labels of the following types can be manipulated: 'status/*'," +
				" 'sig/*', and 'type/*'. More labels can be configured to be used via the /label command.",
			Config: labelConfig,
		}
		pluginHelp.AddCommand(pluginhelp.Command{
			Usage:       "/[remove-](status|sig|kind|label) <target>",
			Description: "Applies or removes a label from one of the recognized types of labels.",
			Featured:    false,
			WhoCanUse:   "Anyone can trigger this command on a PR.",
			Examples:    []string{"/type bug", "/remove-sig engine", "/sig engine"},
		})
		return pluginHelp, nil
	}
}

func HandleIssueCommentEvent(gc githubClient, ice *github.IssueCommentEvent,
	cfg *externalplugins.Configuration, log *logrus.Entry) error {
	opts := cfg.LabelFor(ice.Repo.Owner.Login, ice.Repo.Name)
	var additionalLabels []string
	var prefixes []string

	if opts.AdditionalLabels != nil {
		additionalLabels = opts.AdditionalLabels
	}
	if opts.Prefixes != nil {
		prefixes = opts.Prefixes
	}
	return handle(gc, log, additionalLabels, prefixes, ice)
}

// Get Labels from Regexp matches
func getLabelsFromREMatches(matches [][]string) (labels []string) {
	for _, match := range matches {
		for _, label := range strings.Split(match[0], " ")[1:] {
			label = strings.ToLower(match[1] + "/" + strings.TrimSpace(label))
			labels = append(labels, label)
		}
	}
	return
}

// getLabelsFromGenericMatches returns label matches with extra labels if those
// have been configured in the plugin config.
func getLabelsFromGenericMatches(matches [][]string, additionalLabels []string) []string {
	if len(additionalLabels) == 0 {
		return nil
	}
	var labels []string
	for _, match := range matches {
		parts := strings.Split(match[0], " ")
		if ((parts[0] != "/label") && (parts[0] != "/remove-label")) || len(parts) != 2 {
			continue
		}
		for _, l := range additionalLabels {
			if l == parts[1] {
				labels = append(labels, parts[1])
			}
		}
	}
	return labels
}

func handle(gc githubClient, log *logrus.Entry, additionalLabels,
	prefixes []string, e *github.IssueCommentEvent) error {
	// arrange prefixes in the format "sig|kind|priority|..."
	// so that they can be used to create labelRegex and removeLabelRegex
	labelPrefixes := strings.Join(prefixes, "|")

	labelRegex, err := regexp.Compile(fmt.Sprintf(labelRegexp, labelPrefixes))
	if err != nil {
		return err
	}
	removeLabelRegex, err := regexp.Compile(fmt.Sprintf(removeLabelRegexp, labelPrefixes))
	if err != nil {
		return err
	}

	labelMatches := labelRegex.FindAllStringSubmatch(e.Comment.Body, -1)
	removeLabelMatches := removeLabelRegex.FindAllStringSubmatch(e.Comment.Body, -1)
	customLabelMatches := customLabelRegex.FindAllStringSubmatch(e.Comment.Body, -1)
	customRemoveLabelMatches := customRemoveLabelRegex.FindAllStringSubmatch(e.Comment.Body, -1)
	if len(labelMatches) == 0 && len(removeLabelMatches) == 0 &&
		len(customLabelMatches) == 0 && len(customRemoveLabelMatches) == 0 {
		return nil
	}

	org := e.Repo.Owner.Login
	repo := e.Repo.Name

	repoLabels, err := gc.GetRepoLabels(org, repo)
	if err != nil {
		return err
	}
	labels, err := gc.GetIssueLabels(org, repo, e.Issue.Number)
	if err != nil {
		return err
	}

	existingLabels := map[string]string{}
	for _, l := range repoLabels {
		existingLabels[strings.ToLower(l.Name)] = l.Name
	}
	var (
		nonexistent         []string
		noSuchLabelsOnIssue []string
		labelsToAdd         []string
		labelsToRemove      []string
	)

	// Get labels to add and labels to remove from regexp matches
	labelsToAdd = append(getLabelsFromREMatches(labelMatches),
		getLabelsFromGenericMatches(customLabelMatches, additionalLabels)...)
	labelsToRemove = append(getLabelsFromREMatches(removeLabelMatches),
		getLabelsFromGenericMatches(customRemoveLabelMatches, additionalLabels)...)

	// Add labels
	for _, labelToAdd := range labelsToAdd {
		if github.HasLabel(labelToAdd, labels) {
			continue
		}

		if _, ok := existingLabels[labelToAdd]; !ok {
			nonexistent = append(nonexistent, labelToAdd)
			continue
		}

		if err := gc.AddLabel(org, repo, e.Issue.Number, existingLabels[labelToAdd]); err != nil {
			log.WithError(err).Errorf("Github failed to add the following label: %s", labelToAdd)
		}
	}

	// Remove labels
	for _, labelToRemove := range labelsToRemove {
		if !github.HasLabel(labelToRemove, labels) {
			noSuchLabelsOnIssue = append(noSuchLabelsOnIssue, labelToRemove)
			continue
		}

		if _, ok := existingLabels[labelToRemove]; !ok {
			nonexistent = append(nonexistent, labelToRemove)
			continue
		}

		if err := gc.RemoveLabel(org, repo, e.Issue.Number, labelToRemove); err != nil {
			log.WithError(err).Errorf("Github failed to remove the following label: %s", labelToRemove)
		}
	}

	if len(nonexistent) > 0 {
		log.Infof("Nonexistent labels: %v", nonexistent)
	}

	// Tried to remove Labels that were not present on the Issue
	if len(noSuchLabelsOnIssue) > 0 {
		msg := fmt.Sprintf(nonExistentLabelOnIssue, strings.Join(noSuchLabelsOnIssue, ", "))
		return gc.CreateComment(org, repo, e.Issue.Number,
			externalplugins.FormatResponseRaw(e.Comment.Body, e.Comment.HTMLURL, e.Comment.User.Login, msg))
	}

	return nil
}
