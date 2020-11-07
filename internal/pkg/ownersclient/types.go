package ownersclient

// OwnersResponse specifies the response to the request to get owners.
type OwnersResponse struct {
	Data    Owners `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// Owners contains owners and the number of lgtm required by PR.
type Owners struct {
	Committers []string `json:"committers,omitempty"`
	Reviewers  []string `json:"reviewers,omitempty"`
	NeedsLgtm  int      `json:"needsLGTM,omitempty"`
}
