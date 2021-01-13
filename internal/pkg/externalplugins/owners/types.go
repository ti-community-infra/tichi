package owners

// SigResponse specifies the response to the request to get owners.
type SigResponse struct {
	Data    SigInfo `json:"data,omitempty"`
	Message string  `json:"message,omitempty"`
}

// SigInfo specifies the sig info.
type SigInfo struct {
	// Name specifies the name of sig.
	Name string `json:"name,omitempty"`
	// Membership specifies the membership of sig.
	Membership SigMembership `json:"membership,omitempty"`
	// Membership specifies the default required lgtm number of sig.
	NeedsLgtm int `json:"needsLGTM,omitempty"`
}

// MemberInfo specifies the contributor's github info.
type MemberInfo struct {
	GithubName string `json:"githubName"`
	// Level specifies the level of contributor at this sig.
	Level string `json:"level,omitempty"`
}

// MemberInfo specifies the sig's membership.
type SigMembership struct {
	TechLeaders []MemberInfo `json:"techLeaders,omitempty"`
	CoLeaders   []MemberInfo `json:"coLeaders,omitempty"`
	Committers  []MemberInfo `json:"committers,omitempty"`
	Reviewers   []MemberInfo `json:"reviewers,omitempty"`
}

// MembersResponse specifies the response to the request to get members.
type MembersResponse struct {
	Data    MembersInfo `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// MembersInfo specifies the members info.
type MembersInfo struct {
	Members []MemberInfo `json:"members,omitempty"`
	Total   int          `json:"total,omitempty"`
}
