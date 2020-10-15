package externalplugins

import (
	"fmt"
	"net/url"

	"k8s.io/apimachinery/pkg/util/sets"
)

// Configuration is the top-level serialization target for external plugin Configuration.
type Configuration struct {
	TiCommunityLgtm []TiCommunityLgtm `json:"ti-community-lgtm,omitempty"`
	Approve         []Approve
}

// TiCommunityLgtm specifies a configuration for a single ti community lgtm.
// The configuration for the ti community lgtm plugin is defined as a list of these structures.
type TiCommunityLgtm struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// ReviewActsAsLgtm indicates that a GitHub review of "approve" or "request changes"
	// acts as adding or removing the lgtm label.
	ReviewActsAsLgtm bool `json:"review_acts_as_lgtm,omitempty"`
	// StoreTreeHash indicates if tree_hash should be stored inside a comment to detect
	// squashed commits before removing lgtm labels.
	StoreTreeHash bool `json:"store_tree_hash,omitempty"`
	// WARNING: This disables the security mechanism that prevents a malicious member (or
	// compromised GitHub account) from merging arbitrary code. Use with caution.
	//
	// StickyLgtmTeam specifies the GitHub team whose members are trusted with sticky LGTM,
	// which eliminates the need to re-lgtm minor fixes/updates.
	StickyLgtmTeam string `json:"trusted_team_for_sticky_lgtm,omitempty"`
	// PullOwnersURL specifies the URL of the reviewer of pull request.
	PullOwnersURL string `json:"pull_owners_url,omitempty"`
}

// Approve specifies a configuration for a single approve.
//
// The configuration for the approve plugin is defined as a list of these structures.
type Approve struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// IssueRequired indicates if an associated issue is required for approval in
	// the specified repos.
	IssueRequired bool `json:"issue_required,omitempty"`
	// RequireSelfApproval requires PR authors to explicitly approve their PRs.
	// Otherwise the plugin assumes the author of the PR approves the changes in the PR.
	RequireSelfApproval *bool `json:"require_self_approval,omitempty"`
	// LastLgtmActsAsApprove indicates that the lgtm command should be used to
	// indicate approval
	LastLgtmActsAsApprove bool `json:"last_lgtm_acts_as_approve,omitempty"`
	// IgnoreReviewState causes the approve plugin to ignore the GitHub review state. Otherwise:
	// * an APPROVE github review is equivalent to leaving an "/approve" message.
	// * A REQUEST_CHANGES github review is equivalent to leaving an /approve cancel" message.
	IgnoreReviewState *bool `json:"ignore_review_state,omitempty"`
	// CommandHelpLink is the link to the help page which shows the available commands for each repo.
	// The command help page is served by Deck and available under https://<deck-url>/command-help.
	CommandHelpLink string `json:"commandHelpLink,omitempty"`
	// PrProcessLink is the link to the help page which explains the code review process.
	PrProcessLink string `json:"pr_process_link,omitempty"`
	// PullOwnersURL specifies the URL of the reviewer of pull request.
	PullOwnersURL string `json:"pull_owners_url,omitempty"`
}

func (a Approve) HasSelfApproval() bool {
	if a.RequireSelfApproval != nil {
		return !*a.RequireSelfApproval
	}
	return true
}

func (a Approve) ConsiderReviewState() bool {
	if a.IgnoreReviewState != nil {
		return !*a.IgnoreReviewState
	}
	return true
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

// ApproveFor finds the Approve for a repo, if one exists.
// Approval configuration can be listed for a repository
// or an organization.
func (c *Configuration) ApproveFor(org, repo string) *Approve {
	fullName := fmt.Sprintf("%s/%s", org, repo)

	a := func() *Approve {
		// First search for repo config
		for _, approve := range c.Approve {
			if !sets.NewString(approve.Repos...).Has(fullName) {
				continue
			}
			return &approve
		}

		// If you don't find anything, loop again looking for an org config
		for _, approve := range c.Approve {
			if !sets.NewString(approve.Repos...).Has(org) {
				continue
			}
			return &approve
		}

		// Return an empty config, and use plugin defaults
		return &Approve{}
	}()
	if a.CommandHelpLink == "" {
		// TODO: use tidb community deck url.
		a.CommandHelpLink = "https://go.k8s.io/bot-commands"
	}
	if a.PrProcessLink == "" {
		// TODO: use tidb community url.
		a.PrProcessLink = "https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process"
	}
	return a
}

// Validate will return an error if there are any invalid external plugin config.
func (c *Configuration) Validate() error {
	if err := validateLgtm(c.TiCommunityLgtm); err != nil {
		return err
	}

	if err := validateApprove(c.Approve); err != nil {
		return err
	}

	return nil
}

// validateLgtm will return an error if the URL configured by lgtm is invalid.
func validateLgtm(lgtms []TiCommunityLgtm) error {
	for _, lgtm := range lgtms {
		_, err := url.ParseRequestURI(lgtm.PullOwnersURL)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateApprove will return an error if the URL configured by approve is invalid.
func validateApprove(approves []Approve) error {
	for _, approve := range approves {
		_, err := url.ParseRequestURI(approve.PullOwnersURL)
		if err != nil {
			return err
		}
	}

	return nil
}
