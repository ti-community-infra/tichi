package owners

// SigResponse specifies the response to the request to get owners.
type SigResponse struct {
	Data    SigInfo `json:"data"`
	Message string  `json:"message"`
}

type ContributorInfo struct {
	GithubName string `json:"githubName"`
	Level      string `json:"level"`
	Email      string `json:"email"`
}

type SigMembership struct {
	TechLeaders        []ContributorInfo `json:"techLeaders"`
	CoLeaders          []ContributorInfo `json:"coLeaders"`
	Committers         []ContributorInfo `json:"committers"`
	Reviewers          []ContributorInfo `json:"reviewers"`
	ActiveContributors []ContributorInfo `json:"activeContributors"`
}

type SigInfo struct {
	Name       string        `json:"name"`
	Membership SigMembership `json:"membership"`
	NeedsLgtm  int           `json:"needsLGTM"`
}
