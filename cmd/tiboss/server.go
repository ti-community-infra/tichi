package main

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"

	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/boss"
	"github.com/ti-community-infra/tichi/internal/pkg/ownersclient"
)

var _ http.Handler = (*server)(nil)

// server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type server struct {
	tokenGenerator func() []byte
	gc             github.Client
	ol             ownersclient.OwnersLoader
	configAgent    *externalplugins.ConfigAgent
	log            logrus.FieldLogger
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.tokenGenerator)
	if !ok {
		return
	}

	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

// handleEvent distributed events and handles them.
func (s *server) handleEvent(eventType, eventGUID string, payload []byte) error {
	// Get external plugins config.
	config := s.configAgent.Config()

	l := s.log.WithFields(logrus.Fields{"event-type": eventType, github.EventGUID: eventGUID})
	switch eventType {
	case externalplugins.IssueCommentEvent:
		return s.handleIssueCommentEvent(config, l, payload)
	case externalplugins.PullRequestEvent:
		return s.handlePullRequestEvent(config, l, payload)
	default:
		s.log.Debugf("received an event of type %q but didn't ask for it", eventType)
		return nil
	}
}

func (s *server) handleIssueCommentEvent(
	config *externalplugins.Configuration,
	logger *logrus.Entry,
	payload []byte,
) error {
	var ice github.IssueCommentEvent
	if err := json.Unmarshal(payload, &ice); err != nil {
		return err
	}

	go func() {
		if err := boss.HandleIssueCommentEvent(s.gc, &ice, config, s.ol, logger); err != nil {
			logger.WithError(err).Info("Error handling event.")
		}
	}()

	return nil
}

func (s *server) handlePullRequestEvent(
	config *externalplugins.Configuration,
	logger *logrus.Entry,
	payload []byte,
) error {
	var pe github.PullRequestEvent
	if err := json.Unmarshal(payload, &pe); err != nil {
		return err
	}

	go func() {
		if err := boss.HandlePullRequestEvent(s.gc, &pe, config, s.ol, logger); err != nil {
			logger.WithError(err).Info("Error handling event.")
		}
	}()

	return nil
}
