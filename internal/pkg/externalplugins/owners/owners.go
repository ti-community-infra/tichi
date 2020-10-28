package owners

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/ownersclient"
	"k8s.io/test-infra/prow/github"
)

// TODO: we should use a sig info api.
var sigInfoURL = "https://github.com"

const (
	// sigPrefix is a default sig label prefix.
	sigPrefix = "sig/"
	// listOwnersSuccessMessage returns on success.
	listOwnersSuccessMessage = "List all owners success."
	// FIXME : This fmt should be a sig info restful URL.
	defaultSifInfoFileURLFmt = "/%s/community/blob/master/sig/%s/membership.json"
)

const (
	// FIXME: should use sig info's needs lgtm number.
	lgtmTwo = 2
)

type githubClient interface {
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	ListCollaborators(org, repo string) ([]github.User, error)
}

type Server struct {
	// Client for get sig info.
	Client *http.Client

	TokenGenerator func() []byte
	Gc             githubClient
	Log            *logrus.Entry
}

// ListOwners returns owners of tidb community PR.
func (s *Server) ListOwners(org string, repo string, number int) (*ownersclient.OwnersResponse, error) {
	// Get pull request.
	pull, err := s.Gc.GetPullRequest(org, repo, number)
	if err != nil {
		s.Log.WithField("pullNumber", number).WithError(err).Error("Failed get pull request.")
		return nil, err
	}

	// Find sig label.
	sigName := GetSigNameByLabel(pull.Labels)

	// When we cannot find a sig label for PR, we will use a partner.
	if sigName == "" {
		collaborators, err := s.Gc.ListCollaborators(org, repo)
		if err != nil {
			s.Log.WithField("org", org).WithField("repo", repo).WithError(err).Error("Failed get collaborators.")
			return nil, err
		}

		var collaboratorsLogin []string
		for _, collaborator := range collaborators {
			collaboratorsLogin = append(collaboratorsLogin, collaborator.Login)
		}

		return &ownersclient.OwnersResponse{
			Data: ownersclient.Owners{
				Approvers: collaboratorsLogin,
				Reviewers: collaboratorsLogin,
				NeedsLgtm: lgtmTwo,
			},
			Message: listOwnersSuccessMessage,
		}, nil
	}

	url := sigInfoURL + fmt.Sprintf(defaultSifInfoFileURLFmt, org, sigName)
	// Get sig info.
	res, err := s.Client.Get(url)
	if err != nil {
		s.Log.WithField("url", url).WithError(err).Error("Failed get sig info.")
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	// Unmarshal sig members from body.
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var sigInfo SigMembersInfo
	if err := json.Unmarshal(body, &sigInfo); err != nil {
		s.Log.WithField("body", body).WithError(err).Error("Failed unmarshal body.")
		return nil, err
	}

	var approvers []string
	var reviewers []string

	for _, leader := range sigInfo.TechLeaders {
		approvers = append(approvers, leader.GithubName)
		reviewers = append(reviewers, leader.GithubName)
	}

	for _, coLeader := range sigInfo.CoLeaders {
		approvers = append(approvers, coLeader.GithubName)
		reviewers = append(reviewers, coLeader.GithubName)
	}

	for _, committer := range sigInfo.Committers {
		approvers = append(approvers, committer.GithubName)
		reviewers = append(reviewers, committer.GithubName)
	}

	for _, reviewer := range sigInfo.Reviewers {
		reviewers = append(reviewers, reviewer.GithubName)
	}

	return &ownersclient.OwnersResponse{
		Data: ownersclient.Owners{
			Approvers: approvers,
			Reviewers: reviewers,
			NeedsLgtm: lgtmTwo,
		},
		Message: listOwnersSuccessMessage,
	}, nil
}

// GetSigNameByLabel returns the name of sig when the label prefix matches.
func GetSigNameByLabel(labels []github.Label) string {
	var sigName string
	for _, label := range labels {
		if strings.HasPrefix(label.Name, sigPrefix) {
			sigName = strings.TrimPrefix(label.Name, sigPrefix)
			return sigName
		}
	}

	return ""
}
