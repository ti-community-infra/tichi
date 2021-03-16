package label

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
)

const PluginName = "ti-community-label"

var (
	labelRegexp                 = `(?m)^/(%s)\s*(.*)$`
	removeLabelRegexp           = `(?m)^/remove-(%s)\s*(.*)$`
	customLabelRegex            = regexp.MustCompile(`(?m)^/label\s*(.*)$`)
	customRemoveLabelRegex      = regexp.MustCompile(`(?m)^/remove-label\s*(.*)$`)
	nonExistentAdditionalLabels = "The label(s) `%s` cannot be applied. These labels are supported: `%s`."
	nonExistentLabelInRepo      = "The label(s) `%s` cannot be applied, because the repository doesn't have them."
	nonExistentLabelOnIssue     = "These labels are not set on the issue: `%v`."
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
func HelpProvider(epa *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		labelConfig := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.LabelFor(repo.Org, repo.Repo)

			var prefixConfigMsg, additionalLabelsConfigMsg, excludeLabelsConfigMsg string
			if opts.Prefixes != nil {
				prefixConfigMsg = fmt.Sprintf("The label plugin includes commands based on %v prefixes.\n", opts.Prefixes)
			}
			if opts.AdditionalLabels != nil {
				additionalLabelsConfigMsg = fmt.Sprintf("%v labels can be used with the `/[remove-]label` command.\n",
					opts.AdditionalLabels)
			}
			if opts.ExcludeLabels != nil {
				excludeLabelsConfigMsg = fmt.Sprintf("%v labels cannot be added by command.\n",
					opts.ExcludeLabels)
			}
			labelConfig[repo.String()] = prefixConfigMsg + additionalLabelsConfigMsg + excludeLabelsConfigMsg
		}

		pluginHelp := &pluginhelp.PluginHelp{
			Description: "The label plugin provides commands that add or remove certain types of labels. " +
				"For example, the labels like 'status/*', 'sig/*' and bare labels can be " +
				"managed by using `/status`, `/sig` and `/label`.",
			Config: labelConfig,
		}
		pluginHelp.AddCommand(pluginhelp.Command{
			Usage:       "/[remove-](status|sig|type|label|component) <target>",
			Description: "Add or remove a label of the given type.",
			Featured:    false,
			WhoCanUse:   "Everyone can trigger this command.",
			Examples:    []string{"/type bug", "/remove-sig engine", "/sig engine"},
		})
		return pluginHelp, nil
	}
}

func HandleIssueCommentEvent(gc githubClient, ice *github.IssueCommentEvent,
	cfg *tiexternalplugins.Configuration, log *logrus.Entry) error {
	opts := cfg.LabelFor(ice.Repo.Owner.Login, ice.Repo.Name)
	var additionalLabels []string
	var prefixes []string
	var excludeLabels []string

	if opts.AdditionalLabels != nil {
		additionalLabels = opts.AdditionalLabels
	}
	if opts.Prefixes != nil {
		prefixes = opts.Prefixes
	}
	if opts.ExcludeLabels != nil {
		excludeLabels = opts.ExcludeLabels
	}
	return handle(gc, log, additionalLabels, prefixes, excludeLabels, ice)
}

// Get labels from RegExp matches.
func getLabelsFromREMatches(matches [][]string) (labels []string) {
	for _, match := range matches {
		parts := strings.Split(strings.TrimSpace(match[0]), " ")
		for _, label := range parts[1:] {
			// Filter out invisible characters that may be matched.
			if len(strings.TrimSpace(label)) == 0 {
				continue
			}
			label = strings.ToLower(match[1] + "/" + strings.TrimSpace(label))
			labels = append(labels, label)
		}
	}
	return
}

// getLabelsFromGenericMatches returns label matches with extra labels if those
// have been configured in the plugin config.
func getLabelsFromGenericMatches(matches [][]string, additionalLabels []string, invalidLabels *[]string) []string {
	if len(additionalLabels) == 0 {
		return nil
	}

	var labels []string
	labelFilter := sets.String{}
	for _, l := range additionalLabels {
		labelFilter.Insert(strings.ToLower(l))
	}

	for _, match := range matches {
		// Use trim to filter out \r characters that may be matched.
		parts := strings.Split(strings.TrimSpace(match[0]), " ")
		if ((parts[0] != "/label") && (parts[0] != "/remove-label")) || len(parts) != 2 {
			continue
		}
		label := strings.ToLower(parts[1])
		if labelFilter.Has(label) {
			labels = append(labels, label)
		} else {
			*invalidLabels = append(*invalidLabels, label)
		}
	}
	return labels
}

func handle(gc githubClient, log *logrus.Entry, additionalLabels,
	prefixes, excludeLabels []string, e *github.IssueCommentEvent) error {
	// Arrange prefixes in the format "sig|kind|priority|...",
	// so that they can be used to create labelRegex and removeLabelRegex.
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
	issueLabels, err := gc.GetIssueLabels(org, repo, e.Issue.Number)
	if err != nil {
		return err
	}

	repoExistingLabels := map[string]string{}
	for _, l := range repoLabels {
		repoExistingLabels[strings.ToLower(l.Name)] = l.Name
	}

	excludeLabelsSet := sets.NewString()
	for _, l := range excludeLabels {
		excludeLabelsSet.Insert(strings.ToLower(l))
	}

	var (
		nonexistent         []string
		noSuchLabelsInRepo  []string
		noSuchLabelsOnIssue []string
		labelsToAdd         []string
		labelsToRemove      []string
	)

	// Get labels to add and labels to remove from the RegExp matches.
	// Notice: The returned label is lowercase.
	labelsToAdd = append(getLabelsFromREMatches(labelMatches),
		getLabelsFromGenericMatches(customLabelMatches, additionalLabels, &nonexistent)...)
	labelsToRemove = append(getLabelsFromREMatches(removeLabelMatches),
		getLabelsFromGenericMatches(customRemoveLabelMatches, additionalLabels, &nonexistent)...)

	// Add labels.
	for _, labelToAdd := range labelsToAdd {
		if github.HasLabel(labelToAdd, issueLabels) {
			continue
		}

		if _, ok := repoExistingLabels[labelToAdd]; !ok {
			noSuchLabelsInRepo = append(noSuchLabelsInRepo, labelToAdd)
			continue
		}

		// Ignore the exclude label.
		if excludeLabelsSet.Has(labelToAdd) {
			log.Infof("Ignore add exclude label: %s.", labelToAdd)
			continue
		}

		if err := gc.AddLabel(org, repo, e.Issue.Number, repoExistingLabels[labelToAdd]); err != nil {
			log.WithError(err).Errorf("Github failed to add the following label: %s", labelToAdd)
		}
	}

	// Remove labels.
	for _, labelToRemove := range labelsToRemove {
		if !github.HasLabel(labelToRemove, issueLabels) {
			noSuchLabelsOnIssue = append(noSuchLabelsOnIssue, labelToRemove)
			continue
		}

		if _, ok := repoExistingLabels[labelToRemove]; !ok {
			noSuchLabelsInRepo = append(noSuchLabelsInRepo, labelToRemove)
			continue
		}

		// Ignore the exclude label.
		if excludeLabelsSet.Has(labelToRemove) {
			log.Infof("Ignore remove exclude label: %s", labelToRemove)
			continue
		}

		if err := gc.RemoveLabel(org, repo, e.Issue.Number, labelToRemove); err != nil {
			log.WithError(err).Errorf("Github failed to remove the following label: %s", labelToRemove)
		}
	}

	// Tried to add/remove labels that were not in the configuration.
	if len(nonexistent) > 0 {
		log.Infof("Nonexistent labels: %v", nonexistent)
		msg := fmt.Sprintf(nonExistentAdditionalLabels, strings.Join(nonexistent, ", "),
			strings.Join(additionalLabels, ", "))
		msg = tiexternalplugins.FormatResponseRaw(e.Comment.Body, e.Comment.HTMLURL, e.Comment.User.Login, msg)
		return gc.CreateComment(org, repo, e.Issue.Number, msg)
	}

	// Tried to add labels that were not present in the repository.
	if len(noSuchLabelsInRepo) > 0 {
		log.Infof("Labels missing in repo: %v", noSuchLabelsInRepo)
		msg := fmt.Sprintf(nonExistentLabelInRepo, strings.Join(noSuchLabelsInRepo, ", "))
		msg = tiexternalplugins.FormatResponseRaw(e.Comment.Body, e.Comment.HTMLURL, e.Comment.User.Login, msg)
		return gc.CreateComment(org, repo, e.Issue.Number, msg)
	}

	// Tried to remove labels that were not present on the issue.
	if len(noSuchLabelsOnIssue) > 0 {
		msg := fmt.Sprintf(nonExistentLabelOnIssue, strings.Join(noSuchLabelsOnIssue, ", "))
		msg = tiexternalplugins.FormatResponseRaw(e.Comment.Body, e.Comment.HTMLURL, e.Comment.User.Login, msg)
		return gc.CreateComment(org, repo, e.Issue.Number, msg)
	}

	return nil
}
