package owners

type ContributorInfo struct {
	GithubName string `json:"githubName"`
}

type SigMembersInfo struct {
	TechLeaders        []ContributorInfo `json:"techLeaders"`
	CoLeaders          []ContributorInfo `json:"coLeaders"`
	Committers         []ContributorInfo `json:"committers"`
	Reviewers          []ContributorInfo `json:"reviewers"`
	ActiveContributors []ContributorInfo `json:"active_contributors"`
}
