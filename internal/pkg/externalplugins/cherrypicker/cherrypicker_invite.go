package cherrypicker

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (s *Server) inviteCollaborator(org, repo, username string, prNum int) error {
	log := s.Log.WithFields(logrus.Fields{"org": org, "repo": repo, "prNum": prNum})

	// already collaborator
	if ok, err := s.GitHubClient.IsCollaborator(org, repo, username); err != nil {
		log.Error(err)
		return err
	} else if ok {
		comment := fmt.Sprintf("@%s you're already a collaborator in bot's repo.", username)
		if err := s.GitHubClient.CreateComment(org, repo, prNum, comment); err != nil {
			s.Log.WithError(err).Error("create comment failed")
			return err
		}
	}
	err := s.GitHubClient.AddCollaborator(org, repo, username, collaboratorPermission)
	if err != nil {
		comment := fmt.Sprintf("@%s failed when inviting you as a collaborator in bot's repo.", username)
		if err := s.GitHubClient.CreateComment(org, repo, prNum, comment); err != nil {
			s.Log.WithError(err).Error("create comment failed")
			return err
		}
		return errors.Wrap(err, "invite failed")
	}

	// notice user
	invitationURL := fmt.Sprintf("%s/%s/%s/invitations", s.GitHubURL, org, repo)
	comment := fmt.Sprintf(cherryPickInviteNotifyMsgTpl, username, cherryPickInviteExample, invitationURL)
	if err := s.GitHubClient.CreateComment(org, repo, prNum, comment); err != nil {
		s.Log.WithError(err).Error("create comment failed")
		return err
	}

	return nil
}
