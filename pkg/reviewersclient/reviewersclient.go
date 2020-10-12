package reviewersclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/tidb-community-bots/ti-community-prow/pkg/externalplugins"
)

const (
	// ReviewersURLFmt specifies a format for reviewer's URL.
	ReviewersURLFmt = "%s/repos/%s/%s/pulls/%d/reviewers"
)

// ReviewersAndNeedsLgtm contains reviewers and the number of lgtm required by PR.
type ReviewersAndNeedsLgtm struct {
	Reviewers []string `json:"reviewers"`
	NeedsLgtm int      `json:"needsLGTM"`
}

// ReviewersResponse specifies the response to the request to get reviewers.
type ReviewersResponse struct {
	Data    ReviewersAndNeedsLgtm `json:"data"`
	Message string                `json:"message"`
}

// ReviewersLoader load PR's reviewers.
type ReviewersLoader interface {
	LoadReviewersAndNeedsLgtm(opts *externalplugins.TiCommunityLgtm, org,
		repoName string, number int) (*ReviewersAndNeedsLgtm, error)
}

// ReviewersLoader for load PR's reviewers.
type ReviewersClient struct {
	// Client is a HTTP client to request reviewers.
	Client *http.Client
}

// LoadReviewersAndNeedsLgtm returns all reviewers and needs
// lgtm from pull reviewers URL.
func (rc *ReviewersClient) LoadReviewersAndNeedsLgtm(opts *externalplugins.TiCommunityLgtm,
	org, repoName string, number int) (*ReviewersAndNeedsLgtm, error) {
	url := fmt.Sprintf(ReviewersURLFmt, opts.PullReviewersURL, org, repoName, number)
	res, err := rc.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode != 200 {
		return nil, errors.New("could not get a reviewers")
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var reviewersRes ReviewersResponse
	if err := json.Unmarshal(body, &reviewersRes); err != nil {
		return nil, err
	}
	return &reviewersRes.Data, nil
}
