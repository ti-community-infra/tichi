package externalplugins

import "fmt"

const (
	// CanMergeLabel is the name of the merge label applied by the merge plugin.
	CanMergeLabel = "status/can-merge"
)

const (
	// LgtmLabelPrefix is the name of the lgtm label applied by the lgtm plugin.
	LgtmLabelPrefix = "status/LGT"

	// SigPrefix is a default sig label prefix.
	SigPrefix = "sig/"
)

const (
	// ContributionLabel is the name of the contribution label applied by the contribution plugin.
	ContributionLabel = "contribution"
	// FirstTimeContributorLabel is the name of the first-time-contributor label applied by the contribution plugin.
	FirstTimeContributorLabel = "first-time-contributor"
)

func FormatTestLabels(labels ...string) []string {
	var r []string
	for _, l := range labels {
		r = append(r, fmt.Sprintf("%s/%s#%d:%s", "org", "repo", 1, l))
	}
	if len(r) == 0 {
		return nil
	}
	return r
}
