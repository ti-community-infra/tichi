package ownersclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	// OwnersURLFmt specifies a format for owners URL.
	OwnersURLFmt = "%s/repos/%s/%s/pulls/%d/owners"
)

// OwnersLoader load PR's reviewers.
type OwnersLoader interface {
	LoadOwners(ownersURL string, org,
		repoName string, number int) (*Owners, error)
}

// OwnersClient for load PR's reviewers.
type OwnersClient struct {
	// Client is a HTTP client to request reviewers.
	Client *http.Client
}

// LoadOwners returns owners and needs
// lgtm from URL of pull request owners.
func (rc *OwnersClient) LoadOwners(ownersURL string,
	org, repoName string, number int) (*Owners, error) {
	url := fmt.Sprintf(OwnersURLFmt, ownersURL, org, repoName, number)
	res, err := rc.Client.Get(url)

	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != 200 {
		return nil, errors.New("could not get a owners")
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var ownersRes OwnersResponse
	if err := json.Unmarshal(body, &ownersRes); err != nil {
		return nil, err
	}
	return &ownersRes.Data, nil
}
