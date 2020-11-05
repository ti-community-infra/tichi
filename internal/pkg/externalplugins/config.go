package externalplugins

import (
	"fmt"
	"net/url"

	"k8s.io/apimachinery/pkg/util/sets"
)

// Configuration is the top-level serialization target for external plugin Configuration.
type Configuration struct {
	TiCommunityLgtm   []TiCommunityLgtm   `json:"ti-community-lgtm,omitempty"`
	TiCommunityMerge  []TiCommunityMerge  `json:"ti-community-merge,omitempty"`
	TiCommunityOwners []TiCommunityOwners `json:"ti-community-owners,omitempty"`
	TiCommunityLabel  []TiCommunityLabel  `json:"ti-community-label,omitempty"`
}

// TiCommunityLgtm specifies a configuration for a single ti community lgtm.
// The configuration for the ti community lgtm plugin is defined as a list of these structures.
type TiCommunityLgtm struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// ReviewActsAsLgtm indicates that a GitHub review of "merge" or "request changes"
	// acts as adding or removing the lgtm label.
	ReviewActsAsLgtm bool `json:"review_acts_as_lgtm,omitempty"`
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

// TiCommunityMerge specifies a configuration for a single owners.
//
// The configuration for the owners plugin is defined as a list of these structures.
type TiCommunityOwners struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// SigEndpoint specifies the URL of the sig info.
	SigEndpoint string `json:"sig_endpoint,omitempty"`
	// DefaultSigName specifies the default sig name of this repo's PR.
	DefaultSigName string `json:"default_sig_name,omitempty"`
	// WARNING: This disables the security mechanism that prevents a malicious member (or
	// compromised GitHub account) from merging arbitrary code. Use with caution.
	//
	// OwnersTrustTeam specifies the GitHub team whose members are trusted.
	OwnersTrustTeam string `json:"trusted_team_for_owners,omitempty"`
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

// Validate will return an error if there are any invalid external plugin config.
func (c *Configuration) Validate() error {
	if err := validateLgtm(c.TiCommunityLgtm); err != nil {
		return err
	}

	if err := validateMerge(c.TiCommunityMerge); err != nil {
		return err
	}

	if err := validateOwners(c.TiCommunityOwners); err != nil {
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
func validateOwners(merges []TiCommunityOwners) error {
	for _, merge := range merges {
		_, err := url.ParseRequestURI(merge.SigEndpoint)
		if err != nil {
			return err
		}
	}

	return nil
}
