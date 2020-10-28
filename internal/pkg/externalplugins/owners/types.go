package owners

// SigResponse specifies the response to the request to get owners.
type SigResponse struct {
	Data    SigInfo `json:"data"`
	Message string  `json:"message"`
}

// ContributorInfo specifies the contributor's github info.
type ContributorInfo struct {
	GithubName string `json:"githubName"`
	// Level specifies the level of contributor at this sig.
	Level string `json:"level"`
	Email string `json:"email"`
}

// ContributorInfo specifies the sig's membership.
type SigMembership struct {
	TechLeaders        []ContributorInfo `json:"techLeaders"`
	CoLeaders          []ContributorInfo `json:"coLeaders"`
	Committers         []ContributorInfo `json:"committers"`
	Reviewers          []ContributorInfo `json:"reviewers"`
	ActiveContributors []ContributorInfo `json:"activeContributors"`
}

// ContributorInfo specifies the sig info.
type SigInfo struct {
	// Name specifies the name of sig.
	Name string `json:"name"`
	// Membership specifies the membership of sig.
	Membership SigMembership `json:"membership"`
	// Membership specifies the default required lgtm number of sig.
	NeedsLgtm int `json:"needsLGTM"`
}
