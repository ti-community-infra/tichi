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
	log := s.Log.WithFields(logrus.Fields{"org": org, "repo": repo, "prNum": prNum})

	// judge if the commenter is memmber of the ORG's member.
	if ok, err := s.GitHubClient.IsMember(org, commentUser); err != nil {
		log.WithError(err).Error("judge org member failed")
		return err
	} else if !ok {
		comment := fmt.Sprintf("@%s you're not a member of org `%s`", commentUser, org)
		if err := s.GitHubClient.CreateComment(org, repo, prNum, comment); err != nil {
			s.Log.WithError(err).Error("create comment failed")
			return err
		}
	}

	forkToOwner := s.BotUser.Login
	forkToRepo := repo
	forkToFullRepo := fmt.Sprintf("%s/%s", forkToOwner, forkToRepo)

	// already collaborator in forked repo.
	if ok, err := s.GitHubClient.IsCollaborator(forkToOwner, forkToRepo, commentUser); err != nil {
		log.Error(err)
		return err
	} else if ok {
		comment := fmt.Sprintf("@%s you're already a collaborator in repo `%s`",
			commentUser, forkToFullRepo)
		if err := s.GitHubClient.CreateComment(org, repo, prNum, comment); err != nil {
			s.Log.WithError(err).Error("create comment failed")
			return err
		}
	}

	err := s.GitHubClient.AddCollaborator(forkToOwner, forkToRepo, commentUser, collaboratorPermission)
	if err != nil {
		comment := fmt.Sprintf("@%s failed when inviting you as a collaborator in repo `%s`.",
			commentUser, forkToFullRepo)
		if err := s.GitHubClient.CreateComment(org, repo, prNum, comment); err != nil {
			s.Log.WithError(err).Error("create comment failed")
			return err
		}
		return errors.Wrap(err, "invite failed")
	}

	// notice user
	invitationURL := fmt.Sprintf("%s/%s/invitations", s.GitHubURL, forkToFullRepo)
	comment := fmt.Sprintf(cherryPickInviteNotifyMsgTpl, commentUser, cherryPickInviteExample, invitationURL)
	if err := s.GitHubClient.CreateComment(org, repo, prNum, comment); err != nil {
		s.Log.WithError(err).Error("create comment failed")
		return err
	}

	return nil
}
