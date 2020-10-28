package owners

// SigResponse specifies the response to the request to get owners.
type SigResponse struct {
	Data    SigInfo `json:"data,omitempty"`
	Message string  `json:"message,omitempty"`
}

// ContributorInfo specifies the contributor's github info.
type ContributorInfo struct {
	GithubName string `json:"githubName"`
	// Level specifies the level of contributor at this sig.
	Level string `json:"level,omitempty"`
	Email string `json:"email,omitempty"`
}

// ContributorInfo specifies the sig's membership.
type SigMembership struct {
	TechLeaders        []ContributorInfo `json:"techLeaders,omitempty"`
	CoLeaders          []ContributorInfo `json:"coLeaders,omitempty"`
	Committers         []ContributorInfo `json:"committers,omitempty"`
	Reviewers          []ContributorInfo `json:"reviewers,omitempty"`
	ActiveContributors []ContributorInfo `json:"activeContributors,omitempty"`
}

// ContributorInfo specifies the sig info.
type SigInfo struct {
	// Name specifies the name of sig.
	Name string `json:"name,omitempty"`
	// Membership specifies the membership of sig.
	Membership SigMembership `json:"membership,omitempty"`
	// Membership specifies the default required lgtm number of sig.
	NeedsLgtm int `json:"needsLGTM,omitempty"`
}
