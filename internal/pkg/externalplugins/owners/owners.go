package owners

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/ownersclient"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/github"
)

const (
	// SigEndpointFmt specifies a format for sigs URL.
	SigEndpointFmt = "/sigs/%s"
	// MembersEndpoint specifies a members endpoint.
	MembersEndpoint = "/members/"
)

// Member's levels.
const (
	activeContributorLevel = "active-contributor"
	reviewerLevel          = "reviewer"
	committerLevel         = "committer"
	coLeaderLevel          = "co-leader"
	leaderLevel            = "leader"
)

const (
	// listOwnersSuccessMessage returns on success.
	listOwnersSuccessMessage = "List all owners success."
	// defaultRequireLgtmNum specifies default lgtm number.
	defaultRequireLgtmNum = 2
)

type githubClient interface {
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	ListCollaborators(org, repo string) ([]github.User, error)
	ListTeams(org string) ([]github.Team, error)
	ListTeamMembers(org string, id int, role string) ([]github.TeamMember, error)
}

type Server struct {
	// Client for get sig info.
	Client *http.Client

	TokenGenerator func() []byte
	Gc             githubClient
	ConfigAgent    *tiexternalplugins.ConfigAgent
	Log            *logrus.Entry
}

func (s *Server) listOwnersForNonSig(opts *tiexternalplugins.TiCommunityOwners,
	trustTeamMembers []string, requireLgtm int) (*ownersclient.OwnersResponse, error) {
	var committers []string
	var reviewers []string

	// Members URL.
	url := opts.SigEndpoint + MembersEndpoint

	res, err := s.Client.Get(url)
	if err != nil {
		s.Log.WithField("url", url).WithError(err).Error("Failed to get members.")
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != 200 {
		s.Log.WithField("url", url).WithError(err).Error("Failed to get members.")
		return nil, errors.New("could not get the members")
	}

	// Unmarshal members from body.
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var membersRes MembersResponse
	if err := json.Unmarshal(body, &membersRes); err != nil {
		s.Log.WithField("body", body).WithError(err).Error("Failed to unmarshal body.")
		return nil, err
	}

	members := membersRes.Data.Members
	for _, member := range members {
		// Except for activeContributor and reviewer, which are both committers.
		if member.Level != activeContributorLevel && member.Level != reviewerLevel {
			committers = append(committers, member.GithubName)
		}
		// Except for activeContributor, which are both reviewers.
		if member.Level != activeContributorLevel {
			reviewers = append(reviewers, member.GithubName)
		}
	}

	// If require lgtm no setting, use default require lgtm.
	if requireLgtm == 0 {
		requireLgtm = defaultRequireLgtmNum
	}

	return &ownersclient.OwnersResponse{
		Data: ownersclient.Owners{
			Committers: sets.NewString(committers...).Insert(trustTeamMembers...).List(),
			Reviewers:  sets.NewString(reviewers...).Insert(trustTeamMembers...).List(),
			NeedsLgtm:  requireLgtm,
		},
		Message: listOwnersSuccessMessage,
	}, nil
}

func (s *Server) listOwnersForSigs(sigNames []string,
	opts *tiexternalplugins.TiCommunityOwners, trustTeamMembers []string,
	requireLgtm int) (*ownersclient.OwnersResponse, error) {
	var committers []string
	var reviewers []string
	var maxNeedsLgtm int

	for _, sigName := range sigNames {
		url := opts.SigEndpoint + fmt.Sprintf(SigEndpointFmt, sigName)
		// Get sigName info.
		res, err := s.Client.Get(url)
		if err != nil {
			s.Log.WithField("url", url).WithError(err).Error("Failed to get sigName info.")
			return nil, err
		}

		if res.StatusCode != 200 {
			s.Log.WithField("url", url).WithError(err).Error("Failed to get sigName info.")
			return nil, fmt.Errorf("could not get the sig: %s", sigName)
		}

		// Unmarshal sigName members from body.
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		var sigRes SigResponse
		if err := json.Unmarshal(body, &sigRes); err != nil {
			s.Log.WithField("body", body).WithError(err).Error("Failed to unmarshal body.")
			return nil, err
		}

		sig := sigRes.Data

		for _, leader := range sig.Membership.TechLeaders {
			committers = append(committers, leader.GithubName)
			reviewers = append(reviewers, leader.GithubName)
		}

		for _, coLeader := range sig.Membership.CoLeaders {
			committers = append(committers, coLeader.GithubName)
			reviewers = append(reviewers, coLeader.GithubName)
		}

		for _, committer := range sig.Membership.Committers {
			committers = append(committers, committer.GithubName)
			reviewers = append(reviewers, committer.GithubName)
		}

		for _, reviewer := range sig.Membership.Reviewers {
			reviewers = append(reviewers, reviewer.GithubName)
		}

		if sig.NeedsLgtm > maxNeedsLgtm {
			maxNeedsLgtm = sig.NeedsLgtm
		}

		_ = res.Body.Close()
	}

	// If the number of lgtm is not specified, the maximum of sigName's needsLgtm is used.
	if requireLgtm == 0 {
		requireLgtm = maxNeedsLgtm
	}

	return &ownersclient.OwnersResponse{
		Data: ownersclient.Owners{
			Committers: sets.NewString(committers...).Insert(trustTeamMembers...).List(),
			Reviewers:  sets.NewString(reviewers...).Insert(trustTeamMembers...).List(),
			NeedsLgtm:  requireLgtm,
		},
		Message: listOwnersSuccessMessage,
	}, nil
}

func (s *Server) listOwnersByUseGitHubPermission(org string, repo string,
	trustTeamMembers []string, requireLgtm int) (*ownersclient.OwnersResponse, error) {
	collaborators, err := s.Gc.ListCollaborators(org, repo)
	if err != nil {
		s.Log.WithField("org", org).WithField("repo", repo).WithError(err).Error("Failed to get collaborators.")
		return nil, err
	}

	var collaboratorsLogin []string
	for _, collaborator := range collaborators {
		// Only write and admin permission can lgtm and merge PR.
		if collaborator.Permissions.Push || collaborator.Permissions.Admin {
			collaboratorsLogin = append(collaboratorsLogin, collaborator.Login)
		}
	}
	committers := sets.NewString(collaboratorsLogin...).Insert(trustTeamMembers...).List()

	if requireLgtm == 0 {
		requireLgtm = defaultRequireLgtmNum
	}

	return &ownersclient.OwnersResponse{
		Data: ownersclient.Owners{
			Committers: committers,
			Reviewers:  committers,
			NeedsLgtm:  requireLgtm,
		},
		Message: listOwnersSuccessMessage,
	}, nil
}

// ListOwners returns owners of tidb community PR.
func (s *Server) ListOwners(org string, repo string, number int,
	config *tiexternalplugins.Configuration) (*ownersclient.OwnersResponse, error) {
	// Get pull request.
	pull, err := s.Gc.GetPullRequest(org, repo, number)
	if err != nil {
		s.Log.WithField("pullNumber", number).WithError(err).Error("Failed to get pull request.")
		return nil, err
	}

	// Get the configuration.
	opts := config.OwnersFor(org, repo)

	// Get the configuration according to the name of the branch which the current PR belongs to.
	branchName := pull.Base.Ref
	branchConfig, hasBranchConfig := opts.Branches[branchName]

	// Get the require lgtm number from PR's label.
	requireLgtm, err := getRequireLgtmByLabel(pull.Labels, opts.RequireLgtmLabelPrefix)
	if err != nil {
		s.Log.WithField("pullNumber", number).WithError(err).Error("Failed to parse require lgtm.")
		return nil, err
	}

	// When we cannot find the require label from the PR, try to use the default require lgtm.
	if requireLgtm == 0 {
		if hasBranchConfig && branchConfig.DefaultRequireLgtm != 0 {
			requireLgtm = branchConfig.DefaultRequireLgtm
		} else {
			requireLgtm = opts.DefaultRequireLgtm
		}
	}

	// Notice: If the branch of the PR has extra trust team config, it will override the repository config.
	var trustTeams []string

	// Notice: If the configuration of the trust team gives an empty slice (not nil slice), the plugin
	// will consider that the branch does not trust any team.
	if hasBranchConfig && branchConfig.TrustTeams != nil {
		trustTeams = branchConfig.TrustTeams
	} else {
		trustTeams = opts.TrustTeams
	}

	// Avoid duplication when one user exists in multiple trusted teams.
	trustTeamMembers := sets.String{}

	for _, trustTeam := range trustTeams {
		members := getTrustTeamMembers(s.Log, s.Gc, org, trustTeam)
		trustTeamMembers.Insert(members...)
	}

	// Find sig names by labels.
	sigNames := getSigNamesByLabels(pull.Labels)

	// Use default sig name if cannot find.
	if len(sigNames) == 0 && len(opts.DefaultSigName) != 0 {
		sigNames = append(sigNames, opts.DefaultSigName)
	}

	useGitHubPermission := false
	if hasBranchConfig {
		useGitHubPermission = branchConfig.UseGitHubPermission
	} else {
		useGitHubPermission = opts.UseGitHubPermission
	}

	if useGitHubPermission {
		return s.listOwnersByUseGitHubPermission(org, repo, trustTeamMembers.List(), requireLgtm)
	}

	// When we cannot find a sig label for PR and there is no default sig name, we will use a collaborators.
	if len(sigNames) == 0 {
		return s.listOwnersForNonSig(opts, trustTeamMembers.List(), requireLgtm)
	}

	return s.listOwnersForSigs(sigNames, opts, trustTeamMembers.List(), requireLgtm)
}

// getSigNamesByLabels returns the names of sig when the label prefix matches.
func getSigNamesByLabels(labels []github.Label) []string {
	var sigNames []string
	for _, label := range labels {
		if strings.HasPrefix(label.Name, tiexternalplugins.SigPrefix) {
			sigName := strings.TrimPrefix(label.Name, tiexternalplugins.SigPrefix)
			sigNames = append(sigNames, sigName)
		}
	}

	return sigNames
}

// getRequireLgtmByLabel returns the number of require lgtm when the label prefix matches.
func getRequireLgtmByLabel(labels []github.Label, labelPrefix string) (int, error) {
	noRequireLgtm := 0

	if len(labelPrefix) == 0 {
		return noRequireLgtm, nil
	}

	for _, label := range labels {
		if strings.HasPrefix(label.Name, labelPrefix) {
			requireLgtm, err := strconv.Atoi(strings.TrimPrefix(label.Name, labelPrefix))
			if err != nil {
				return noRequireLgtm, err
			}
			return requireLgtm, nil
		}
	}

	return noRequireLgtm, nil
}

// getTrustTeamMembers returns the members of trust team.
func getTrustTeamMembers(log *logrus.Entry, gc githubClient, org, trustTeam string) []string {
	if len(trustTeam) > 0 {
		if teams, err := gc.ListTeams(org); err == nil {
			for _, teamInOrg := range teams {
				if strings.Compare(teamInOrg.Name, trustTeam) == 0 {
					if members, err := gc.ListTeamMembers(org, teamInOrg.ID, github.RoleAll); err == nil {
						var membersLogin []string
						for _, member := range members {
							membersLogin = append(membersLogin, member.Login)
						}
						return membersLogin
					}
					log.WithError(err).Errorf("Failed to list members in %s:%s.", org, teamInOrg.Name)
				}
			}
		} else {
			log.WithError(err).Errorf("Failed to list teams in org %s.", org)
		}
	}
	return []string{}
}
