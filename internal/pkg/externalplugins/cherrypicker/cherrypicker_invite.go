package cherrypicker

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
)

func (s *Server) inviteCollaborator(ic *github.IssueCommentEvent) error {
	org := ic.Repo.Owner.Login
	repo := ic.Repo.Name
	prNum := ic.Issue.Number
	commentUser := ic.Comment.User.Login
	forkToOwner := s.BotUser.Login
	forkToRepo := repo
	forkToFullRepo := fmt.Sprintf("%s/%s", forkToOwner, forkToRepo)
	log := s.Log.WithFields(logrus.Fields{"org": org, "repo": repo, "prNum": prNum})

	// already collaborator in forked repo.
	if ok, err := s.GitHubClient.IsCollaborator(forkToOwner, forkToRepo, commentUser); err != nil {
		log.Error(err)
		return err
	} else if ok {
		comment := fmt.Sprintf("@%s you're already a collaborator in repo `%s`", commentUser, forkToFullRepo)
		return s.dealGithubAPICallErr(prNum, nil, org, repo, comment)
	}

	// judge if he was already in inviting list.
	openedInvitation, err := s.GitHubClient.ListRepoInvitations(forkToOwner, forkToRepo)
	if err != nil {
		s.Log.WithError(err).Error("get repo invitation failed")
		return err
	}

	for _, i := range openedInvitation {
		if i.Invitee.GetLogin() == commentUser {
			s.Log.Infof("user was invited already in invitation: %s", i.GetHTMLURL())
			return nil
		}
	}

	// judge if the commenter is the ORG's member.
	if ok, err := s.GitHubClient.IsMember(org, commentUser); err != nil {
		log.WithError(err).Error("judge org member failed")
		return err
	} else if !ok {
		comment := fmt.Sprintf("@%s you're not a member of org `%s`", commentUser, org)
		return s.dealGithubAPICallErr(prNum, nil, org, repo, comment)
	}

	if err := s.GitHubClient.AddCollaborator(forkToOwner, forkToRepo,
		commentUser, collaboratorPermission); err != nil {
		comment := fmt.Sprintf(
			"@%s failed when inviting you as a collaborator in repo `%s`.",
			commentUser, forkToFullRepo,
		)
		return s.dealGithubAPICallErr(prNum, errors.Wrap(err, "invite failed"), org, repo, comment)
	}

	// notice user
	invitationURL := fmt.Sprintf("%s/%s/invitations", s.GitHubURL, forkToFullRepo)
	comment := fmt.Sprintf(cherryPickInviteNotifyMsgTpl, commentUser, cherryPickInviteExample, invitationURL)
	return s.dealGithubAPICallErr(prNum, nil, org, repo, comment)
}

// commentErrorReason comment the logic error to pull request or issue.
func (s *Server) dealGithubAPICallErr(prNum int, apiErr error, org, repo, comment string) error {
	s.Log.Error(apiErr)

	// send comment into PR.
	if err := s.GitHubClient.CreateComment(org, repo, prNum, comment); err != nil {
		// only log the error.
		s.Log.WithError(err).Error("create comment failed")
		if apiErr == nil {
			return err
		}
	}

	return apiErr
}
