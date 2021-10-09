package externalplugins

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/labels"
)

const (
	// defaultGracePeriodDuration define the time for blunderbuss plugin to wait
	// before requesting a review (default five seconds).
	defaultGracePeriodDuration = 5
	// defaultLogLevel defines the default log level of all ti community plugins.
	defaultLogLevel = logrus.InfoLevel
)

// Allowed value of the action configuration of the label blocker plugin.
const (
	LabeledAction   = "labeled"
	UnlabeledAction = "unlabeled"
)

// Configuration is the top-level serialization target for external plugin Configuration.
type Configuration struct {
	TichiWebURL     string `json:"tichi_web_url,omitempty"`
	PRProcessLink   string `json:"pr_process_link,omitempty"`
	CommandHelpLink string `json:"command_help_link,omitempty"`

	// LogLevel enables dynamically updating the log level of the
	// standard logger that is used by all ti community plugin.
	//
	// Valid values:
	//
	// "trace" "debug", "info", "warn", "warning", "error", "fatal", "panic"
	//
	// Defaults to "info".
	LogLevel string `json:"log_level,omitempty"`

	TiCommunityLgtm          []TiCommunityLgtm          `json:"ti-community-lgtm,omitempty"`
	TiCommunityMerge         []TiCommunityMerge         `json:"ti-community-merge,omitempty"`
	TiCommunityOwners        []TiCommunityOwners        `json:"ti-community-owners,omitempty"`
	TiCommunityLabel         []TiCommunityLabel         `json:"ti-community-label,omitempty"`
	TiCommunityAutoresponder []TiCommunityAutoresponder `json:"ti-community-autoresponder,omitempty"`
	TiCommunityBlunderbuss   []TiCommunityBlunderbuss   `json:"ti-community-blunderbuss,omitempty"`
	TiCommunityTars          []TiCommunityTars          `json:"ti-community-tars,omitempty"`
	TiCommunityLabelBlocker  []TiCommunityLabelBlocker  `json:"ti-community-label-blocker,omitempty"`
	TiCommunityContribution  []TiCommunityContribution  `json:"ti-community-contribution,omitempty"`
	TiCommunityCherrypicker  []TiCommunityCherrypicker  `json:"ti-community-cherrypicker,omitempty"`
	TiCommunityMatchChecker  []TiCommunityMatchChecker  `json:"ti-community-matching-checker,omitempty"`
}

// TiCommunityLgtm specifies a configuration for a single ti community lgtm.
// The configuration for the ti community lgtm plugin is defined as a list of these structures.
type TiCommunityLgtm struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// PullOwnersEndpoint specifies the URL of the reviewer of pull request.
	PullOwnersEndpoint string `json:"pull_owners_endpoint,omitempty"`
}

// TiCommunityMerge specifies a configuration for a single merge.
//
// The configuration for the merge plugin is defined as a list of these structures.
type TiCommunityMerge struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// StoreTreeHash indicates if tree_hash should be stored inside a comment to detect
	// guaranteed commits before removing can merge labels.
	StoreTreeHash bool `json:"store_tree_hash,omitempty"`
	// PullOwnersEndpoint specifies the URL of the reviewer of pull request.
	PullOwnersEndpoint string `json:"pull_owners_endpoint,omitempty"`
}

// TiCommunityOwners specifies a configuration for a single ti community owners plugin.
//
// The configuration for the owners plugin is defined as a list of these structures.
type TiCommunityOwners struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// SigEndpoint specifies the URL of the sig info.
	SigEndpoint string `json:"sig_endpoint,omitempty"`
	// DefaultSigName specifies the default sig name of this repo's PR.
	DefaultSigName string `json:"default_sig_name,omitempty"`
	// DefaultRequireLgtm specifies the default require lgtm number.
	DefaultRequireLgtm int `json:"default_require_lgtm,omitempty"`
	// RequireLgtmLabelPrefix specifies the prefix of require lgtm label.
	RequireLgtmLabelPrefix string `json:"require_lgtm_label_prefix,omitempty"`
	// ReviewerTeams specifies the GitHub teams whose members can review pull request.
	ReviewerTeams []string `json:"reviewer_teams,omitempty"`
	// CommitterTeams specifies the GitHub teams whose members can merge pull request.
	CommitterTeams []string `json:"committer_teams,omitempty"`
	// UseGitHubPermission specifies the permissions to use GitHub.
	// People with write and admin permissions have reviewer and committer permissions.
	UseGitHubPermission bool `json:"use_github_permission,omitempty"`
	// UseGithubTeam specifies the permissions to use specified GitHub team as committer teams or reviewer teams.
	UseGithubTeam bool `json:"use_github_team,omitempty"`
	// Branches specifies the branch level configuration that will override the repository
	// level configuration.
	Branches map[string]TiCommunityOwnerBranchConfig `json:"branches,omitempty"`
}

// TiCommunityOwnerBranchConfig is the branch level configuration of the owners plugin.
type TiCommunityOwnerBranchConfig struct {
	// DefaultRequireLgtm specifies the default require lgtm number of the branch.
	DefaultRequireLgtm int `json:"default_require_lgtm,omitempty"`
	// ReviewerTeams specifies the GitHub teams whose members can review pull request.
	ReviewerTeams []string `json:"reviewer_teams,omitempty"`
	// CommitterTeams specifies the GitHub teams whose members can merge pull request.
	CommitterTeams []string `json:"committer_teams,omitempty"`
	// UseGitHubPermission specifies the permissions to use GitHub.
	// People with write and admin permissions have reviewer and committer permissions.
	UseGitHubPermission bool `json:"use_github_permission,omitempty"`
	// UseGithubTeam specifies the permissions to use specified GitHub team as committer teams or reviewer teams.
	UseGithubTeam bool `json:"use_github_team,omitempty"`
}

// TiCommunityLabel is the config for the label plugin.
type TiCommunityLabel struct {
	// Repos is either of the form org/repos or just org.
	// The AdditionalLabels and Prefixes values are applicable
	// to these repos.
	Repos []string `json:"repos,omitempty"`
	// AdditionalLabels is a set of additional labels enabled for use
	// on top of the existing "status/*", "priority/*"
	// and "sig/*" labels.
	// Labels can be used with `/[remove-]label <additionalLabel>` commands.
	AdditionalLabels []string `json:"additional_labels,omitempty"`
	// Prefixes is a set of label prefixes which replaces the existing
	// "status", "priority"," and "sig" label prefixes.
	// Labels can be used with `/[remove-]<prefix> <target>` commands.
	Prefixes []string `json:"prefixes,omitempty"`
	// ExcludeLabels specifies labels that cannot be added by TiCommunityLabel.
	ExcludeLabels []string `json:"exclude_labels,omitempty"`
}

// TiCommunityAutoresponder is the config for the blunderbuss plugin.
type TiCommunityAutoresponder struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// AutoResponds is a set of responds.
	AutoResponds []AutoRespond `json:"auto_responds,omitempty"`
}

// AutoRespond is the config for auto respond.
type AutoRespond struct {
	// Regex specifies the conditions for the trigger to respond automatically.
	Regex string `json:"regex,omitempty"`
	// Message specifies the content of the automatic respond.
	Message string `json:"message,omitempty"`
}

// TiCommunityBlunderbuss is the config for the blunderbuss plugin.
type TiCommunityBlunderbuss struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// MaxReviewerCount is the maximum number of reviewers to request
	// reviews from. Defaults to 0 meaning no limit.
	MaxReviewerCount int `json:"max_request_count,omitempty"`
	// IncludeReviewers specifies which reviewers are the only ones involved in the code review.
	IncludeReviewers []string `json:"include_reviewers,omitempty"`
	// ExcludeReviewers specifies which reviewers do not participate in code review.
	ExcludeReviewers []string `json:"exclude_reviewers,omitempty"`
	// PullOwnersEndpoint specifies the URL of the reviewer of pull request.
	PullOwnersEndpoint string `json:"pull_owners_endpoint,omitempty"`
	// GracePeriodDuration specifies the waiting time before the plugin requests a review,
	// defaults to 5 means that the plugin will wait 5 seconds for the sig label to be added.
	GracePeriodDuration int `json:"grace_period_duration,omitempty"`
	// RequireSigLabel specifies whether the PR is required to have a sig label before requesting reviewers.
	RequireSigLabel bool `json:"require_sig_label,omitempty"`
}

// setDefaults will set the default value for the config of blunderbuss plugin.
func (c *TiCommunityBlunderbuss) setDefaults() {
	if c.GracePeriodDuration == 0 {
		c.GracePeriodDuration = defaultGracePeriodDuration
	}
}

// TiCommunityTars is the config for the tars plugin.
type TiCommunityTars struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// Message specifies the message when the PR is automatically updated.
	Message string `json:"message,omitempty"`
	// OnlyWhenLabel specifies that the automatic update is triggered only when the PR has this label.
	OnlyWhenLabel string `json:"only_when_label,omitempty"`
	// ExcludeLabels specifies that the automatic update are not triggered when the PR has these labels.
	ExcludeLabels []string `json:"exclude_labels,omitempty"`
}

// setDefaults will set the default label for the config of tars plugin.
func (c *TiCommunityTars) setDefaults() {
	if len(c.OnlyWhenLabel) == 0 {
		c.OnlyWhenLabel = CanMergeLabel
	}

	if len(c.ExcludeLabels) == 0 {
		// Label: needs-rebase.
		c.ExcludeLabels = append(c.ExcludeLabels, labels.NeedsRebase)
		// Label: do-not-merge/hold.
		c.ExcludeLabels = append(c.ExcludeLabels, labels.Hold)
		// Label: do-not-merge/work-in-progress.
		c.ExcludeLabels = append(c.ExcludeLabels, labels.WorkInProgress)
	}
}

// TiCommunityLabelBlocker is the config for the label blocker plugin.
type TiCommunityLabelBlocker struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// BlockLabels is a set of label block rules.
	BlockLabels []BlockLabel `json:"block_labels,omitempty"`
}

// BlockLabel is the config for label blocking.
type BlockLabel struct {
	// Regex specifies the regular expression for match the labels that need to be intercepted.
	Regex string `json:"regex,omitempty"`
	// Actions specifies the label actions that will trigger interception, you can fill in `labeled` or `unlabelled`.
	Actions []string `json:"actions,omitempty"`
	// TrustedTeams specifies the teams allowed adding/removing label.
	TrustedTeams []string `json:"trusted_teams,omitempty"`
	// TrustedUsers specifies the github login of the account allowed adding/removing label.
	TrustedUsers []string `json:"trusted_users,omitempty"`
	// Message specifies the message feedback to the user after blocking the label.
	Message string `json:"message,omitempty"`
}

// TiCommunityContribution is the config for the contribution plugin.
type TiCommunityContribution struct {
	// Repos is either of the form org/repo or just org.
	Repos []string `json:"repos,omitempty"`
	// Message specifies the tips for the contributor's PR.
	Message string `json:"message,omitempty"`
}

// TiCommunityCherrypicker is the config for the cherrypicker plugin.
type TiCommunityCherrypicker struct {
	// Repos is either of the form org/repo or just org.
	Repos []string `json:"repos,omitempty"`
	// AllowAll specifies whether everyone is allowed to cherry pick.
	AllowAll bool `json:"allow_all,omitempty"`
	// IssueOnConflict specifies whether to create an Issue when there is a PR conflict.
	IssueOnConflict bool `json:"create_issue_on_conflict,omitempty"`
	// LabelPrefix specifies the label prefix for cherrypicker.
	LabelPrefix string `json:"label_prefix,omitempty"`
	// PickedLabelPrefix specifies the label prefix after picked.
	PickedLabelPrefix string `json:"picked_label_prefix,omitempty"`
	// ExcludeLabels specifies the labels that need to be excluded when copying the labels of the original PR.
	ExcludeLabels []string `json:"excludeLabels,omitempty"`
}

// setDefaults will set the default value for the config of blunderbuss plugin.
func (c *TiCommunityCherrypicker) setDefaults() {
	if len(c.LabelPrefix) == 0 {
		c.LabelPrefix = DefaultCherryPickLabelPrefix
	}
}

// TiCommunityMatchChecker is the config for the matching-checker plugin.
type TiCommunityMatchChecker struct {
	// Repos are either of the form org/repo or just org.
	Repos []string `json:"repos,omitempty"`
	// RequiredMatchRules specifies rules required to match.
	RequiredMatchRules []RequiredMatchRule `json:"required_matching_rules,omitempty"`
}

// RequiredMatchRule is config about match rules for checking issue or PR content.
type RequiredMatchRule struct {
	// PullRequest specifies whether check for pull request.
	PullRequest bool `json:"pull_request,omitempty"`
	// Issue specifies whether check for issue.
	Issue bool `json:"issue,omitempty"`
	// Title specifies whether check the issue or pull request title.
	Title bool `json:"title,omitempty"`
	// Body specifies whether check the issue or pull request body.
	Body bool `json:"body,omitempty"`
	// CommitMessage specifies whether check the message of commits in the pull request.
	CommitMessage bool `json:"commit_message,omitempty"`
	// Regexp specifies the regular expression used for text match.
	Regexp string `json:"regexp"`
	// MissingMessage specifies the content commented by bot when there is no match.
	MissingMessage string `json:"missing_message,omitempty"`
	// MissingLabel specifies the label added by the bot when there is no match.
	MissingLabel string `json:"missing_label,omitempty"`
}

// LgtmFor finds the Lgtm for a repo, if one exists
// a trigger can be listed for the repo itself or for the
// owning organization
func (c *Configuration) LgtmFor(org, repo string) *TiCommunityLgtm {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, lgtm := range c.TiCommunityLgtm {
		if !sets.NewString(lgtm.Repos...).Has(fullName) {
			continue
		}
		return &lgtm
	}
	// If you don't find anything, loop again looking for an org config
	for _, lgtm := range c.TiCommunityLgtm {
		if !sets.NewString(lgtm.Repos...).Has(org) {
			continue
		}
		return &lgtm
	}
	return &TiCommunityLgtm{}
}

// MergeFor finds the TiCommunityMerge for a repo, if one exists.
// TiCommunityMerge configuration can be listed for a repository
// or an organization.
func (c *Configuration) MergeFor(org, repo string) *TiCommunityMerge {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, merge := range c.TiCommunityMerge {
		if !sets.NewString(merge.Repos...).Has(fullName) {
			continue
		}
		return &merge
	}
	// If you don't find anything, loop again looking for an org config
	for _, merge := range c.TiCommunityMerge {
		if !sets.NewString(merge.Repos...).Has(org) {
			continue
		}
		return &merge
	}
	return &TiCommunityMerge{}
}

// OwnersFor finds the TiCommunityOwners for a repo, if one exists.
// TiCommunityOwners configuration can be listed for a repository
// or an organization.
func (c *Configuration) OwnersFor(org, repo string) *TiCommunityOwners {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, owners := range c.TiCommunityOwners {
		if !sets.NewString(owners.Repos...).Has(fullName) {
			continue
		}
		return &owners
	}
	// If you don't find anything, loop again looking for an org config
	for _, owners := range c.TiCommunityOwners {
		if !sets.NewString(owners.Repos...).Has(org) {
			continue
		}
		return &owners
	}
	return &TiCommunityOwners{}
}

// LabelFor finds the TiCommunityLabel for a repo, if one exists.
// TiCommunityLabel configuration can be listed for a repository
// or an organization.
func (c *Configuration) LabelFor(org, repo string) *TiCommunityLabel {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, label := range c.TiCommunityLabel {
		if !sets.NewString(label.Repos...).Has(fullName) {
			continue
		}
		return &label
	}
	// If you don't find anything, loop again looking for an org config
	for _, label := range c.TiCommunityLabel {
		if !sets.NewString(label.Repos...).Has(org) {
			continue
		}
		return &label
	}
	return &TiCommunityLabel{}
}

// AutoresponderFor finds the TiCommunityAutoresponder for a repo, if one exists.
// TiCommunityAutoresponder configuration can be listed for a repository
// or an organization.
func (c *Configuration) AutoresponderFor(org, repo string) *TiCommunityAutoresponder {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, autoresponder := range c.TiCommunityAutoresponder {
		if !sets.NewString(autoresponder.Repos...).Has(fullName) {
			continue
		}
		return &autoresponder
	}
	// If you don't find anything, loop again looking for an org config
	for _, autoresponder := range c.TiCommunityAutoresponder {
		if !sets.NewString(autoresponder.Repos...).Has(org) {
			continue
		}
		return &autoresponder
	}
	return &TiCommunityAutoresponder{}
}

// BlunderbussFor finds the TiCommunityBlunderbuss for a repo, if one exists.
// TiCommunityBlunderbuss configuration can be listed for a repository
// or an organization.
func (c *Configuration) BlunderbussFor(org, repo string) *TiCommunityBlunderbuss {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, blunderbuss := range c.TiCommunityBlunderbuss {
		if !sets.NewString(blunderbuss.Repos...).Has(fullName) {
			continue
		}
		return &blunderbuss
	}
	// If you don't find anything, loop again looking for an org config
	for _, blunderbuss := range c.TiCommunityBlunderbuss {
		if !sets.NewString(blunderbuss.Repos...).Has(org) {
			continue
		}
		return &blunderbuss
	}
	return &TiCommunityBlunderbuss{}
}

// TarsFor finds the TiCommunityTars for a repo, if one exists.
// TiCommunityTars configuration can be listed for a repository
// or an organization.
func (c *Configuration) TarsFor(org, repo string) *TiCommunityTars {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, tars := range c.TiCommunityTars {
		if !sets.NewString(tars.Repos...).Has(fullName) {
			continue
		}
		return &tars
	}
	// If you don't find anything, loop again looking for an org config
	for _, tars := range c.TiCommunityTars {
		if !sets.NewString(tars.Repos...).Has(org) {
			continue
		}
		return &tars
	}
	return &TiCommunityTars{}
}

// ContributionFor finds the TiCommunityContribution for a repo, if one exists.
// TiCommunityContribution configuration can be listed for a repository
// or an organization.
func (c *Configuration) ContributionFor(org, repo string) *TiCommunityContribution {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, contribution := range c.TiCommunityContribution {
		if !sets.NewString(contribution.Repos...).Has(fullName) {
			continue
		}
		return &contribution
	}
	// If you don't find anything, loop again looking for an org config
	for _, contribution := range c.TiCommunityContribution {
		if !sets.NewString(contribution.Repos...).Has(org) {
			continue
		}
		return &contribution
	}
	return &TiCommunityContribution{}
}

// LabelBlockerFor finds the TiCommunityLabelBlocker for a repo, if one exists.
// TiCommunityLabelBlocker configuration can be listed for a repository
// or an organization.
func (c *Configuration) LabelBlockerFor(org, repo string) *TiCommunityLabelBlocker {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, labelBlocker := range c.TiCommunityLabelBlocker {
		if !sets.NewString(labelBlocker.Repos...).Has(fullName) {
			continue
		}
		return &labelBlocker
	}
	// If you don't find anything, loop again looking for an org config
	for _, labelBlocker := range c.TiCommunityLabelBlocker {
		if !sets.NewString(labelBlocker.Repos...).Has(org) {
			continue
		}
		return &labelBlocker
	}
	return &TiCommunityLabelBlocker{}
}

// CherrypickerFor finds the TiCommunityCherrypicker for a repo, if one exists.
// TiCommunityCherrypicker configuration can be listed for a repository
// or an organization.
func (c *Configuration) CherrypickerFor(org, repo string) *TiCommunityCherrypicker {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, cherrypicker := range c.TiCommunityCherrypicker {
		if !sets.NewString(cherrypicker.Repos...).Has(fullName) {
			continue
		}
		return &cherrypicker
	}
	// If you don't find anything, loop again looking for an org config
	for _, cherrypicker := range c.TiCommunityCherrypicker {
		if !sets.NewString(cherrypicker.Repos...).Has(org) {
			continue
		}
		return &cherrypicker
	}
	return &TiCommunityCherrypicker{}
}

// MatchCheckerFor finds the TiCommunityMatchChecker for a repo, if one exists.
// TiCommunityMatchChecker configuration can be listed for a repository
// or an organization.
func (c *Configuration) MatchCheckerFor(org, repo string) *TiCommunityMatchChecker {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, matchChecker := range c.TiCommunityMatchChecker {
		if !sets.NewString(matchChecker.Repos...).Has(fullName) {
			continue
		}
		return &matchChecker
	}
	// If you don't find anything, loop again looking for an org config
	for _, matchChecker := range c.TiCommunityMatchChecker {
		if !sets.NewString(matchChecker.Repos...).Has(org) {
			continue
		}
		return &matchChecker
	}
	return &TiCommunityMatchChecker{}
}

// setDefaults will set the default value for the configuration of all plugins.
func (c *Configuration) setDefaults() {
	for i := range c.TiCommunityBlunderbuss {
		c.TiCommunityBlunderbuss[i].setDefaults()
	}

	for i := range c.TiCommunityCherrypicker {
		c.TiCommunityCherrypicker[i].setDefaults()
	}

	for i := range c.TiCommunityTars {
		c.TiCommunityTars[i].setDefaults()
	}

	if len(c.LogLevel) == 0 {
		c.LogLevel = defaultLogLevel.String()
	}
}

// Validate will return an error if there are any invalid external plugin config.
func (c *Configuration) Validate() error {
	// Defaulting should run before validation.
	c.setDefaults()

	// Validate tichi web URL.
	if _, err := url.ParseRequestURI(c.TichiWebURL); err != nil {
		return err
	}

	// Validate pr process link.
	if _, err := url.ParseRequestURI(c.PRProcessLink); err != nil {
		return err
	}

	// Validate command help link.
	if _, err := url.ParseRequestURI(c.CommandHelpLink); err != nil {
		return err
	}

	if err := validateLogLevel(c.LogLevel); err != nil {
		return err
	}

	if err := validateLgtm(c.TiCommunityLgtm); err != nil {
		return err
	}

	if err := validateMerge(c.TiCommunityMerge); err != nil {
		return err
	}

	if err := validateOwners(c.TiCommunityOwners); err != nil {
		return err
	}

	if err := validateAutoresponder(c.TiCommunityAutoresponder); err != nil {
		return err
	}

	if err := validateBlunderbuss(c.TiCommunityBlunderbuss); err != nil {
		return err
	}

	if err := validateLabelBlocker(c.TiCommunityLabelBlocker); err != nil {
		return err
	}

	if err := validateMatchBlocker(c.TiCommunityMatchChecker); err != nil {
		return err
	}

	return validateTars(c.TiCommunityTars)
}

// validateLogLevel will return an error if the value of the log level is invalid.
func validateLogLevel(logLevel string) error {
	_, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}

	return nil
}

// validateLgtm will return an error if the URL configured by lgtm is invalid.
func validateLgtm(lgtms []TiCommunityLgtm) error {
	for _, lgtm := range lgtms {
		_, err := url.ParseRequestURI(lgtm.PullOwnersEndpoint)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateMerge will return an error if the URL configured by merge is invalid.
func validateMerge(merges []TiCommunityMerge) error {
	for _, merge := range merges {
		_, err := url.ParseRequestURI(merge.PullOwnersEndpoint)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateOwners will return an error if the endpoint configured by merge is invalid.
func validateOwners(owners []TiCommunityOwners) error {
	for _, merge := range owners {
		_, err := url.ParseRequestURI(merge.SigEndpoint)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateAutoresponder will return an error if the regex cannot compile.
func validateAutoresponder(autoresponders []TiCommunityAutoresponder) error {
	for _, autoresponder := range autoresponders {
		for _, respond := range autoresponder.AutoResponds {
			_, err := regexp.Compile(respond.Regex)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// validateBlunderbuss will return an error if the endpoint configured by blunderbuss is invalid.
func validateBlunderbuss(blunderbusses []TiCommunityBlunderbuss) error {
	for _, blunderbuss := range blunderbusses {
		_, err := url.ParseRequestURI(blunderbuss.PullOwnersEndpoint)
		if err != nil {
			return err
		}
		if blunderbuss.MaxReviewerCount <= 0 {
			return errors.New("max reviewer count must more than 0")
		}
		if blunderbuss.GracePeriodDuration < 0 {
			return errors.New("grace period duration must not less than 0")
		}
		if len(blunderbuss.IncludeReviewers) != 0 && len(blunderbuss.ExcludeReviewers) != 0 {
			return errors.New("cannot set both include_reviewers and exclude_reviewers configurations")
		}
	}

	return nil
}

// validateLabelBlocker will return an error if the regex cannot compile or actions is illegal.
func validateLabelBlocker(labelBlockers []TiCommunityLabelBlocker) error {
	for _, labelBlocker := range labelBlockers {
		for _, blockLabel := range labelBlocker.BlockLabels {
			_, err := regexp.Compile(blockLabel.Regex)
			if err != nil {
				return err
			}

			err = validateLabelBlockerAction(blockLabel.Actions)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// validateLabelBlockerAction used to check whether all actions filled in are allowed values.
func validateLabelBlockerAction(actions []string) error {
	if len(actions) == 0 {
		return errors.New("there must be at least one action")
	}

	allowActionSet := sets.NewString(LabeledAction, UnlabeledAction)

	for _, action := range actions {
		if allowActionSet.Has(action) {
			continue
		} else {
			return fmt.Errorf("actions contain illegal value %s", action)
		}
	}

	return nil
}

// validateTars will return an error if tars is set for org.
// If set directly to org will query the query to a large number of pull requests,
// which will create a dos attack to the CI system.
func validateTars(tars []TiCommunityTars) error {
	for _, tar := range tars {
		for _, repo := range tar.Repos {
			slashSplit := strings.Split(repo, "/")
			if n := len(slashSplit); n != 2 {
				return fmt.Errorf("found repo %s that was not in org/repo format", repo)
			}
		}
	}

	return nil
}

// validateMatchBlocker will return an error if the regex cannot compile or actions is illegal.
func validateMatchBlocker(matchBlockers []TiCommunityMatchChecker) error {
	for _, matchBlocker := range matchBlockers {
		for _, rule := range matchBlocker.RequiredMatchRules {
			_, err := regexp.Compile(rule.Regexp)
			if err != nil {
				return fmt.Errorf("the regex of matching rule is broken: %v", err)
			}
			if !rule.PullRequest && !rule.Issue {
				return fmt.Errorf("issue or pull request need to be specified to verify")
			}
			if !rule.Title && !rule.Body && !rule.CommitMessage {
				return fmt.Errorf("at least one of scopes(title, body, commit message) need to be specified")
			}
		}
	}
	return nil
}
