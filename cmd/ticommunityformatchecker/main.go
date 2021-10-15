package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/formatchecker"
	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/interrupts"
	"k8s.io/test-infra/prow/pjutil"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
)

type options struct {
	port int

	dryRun bool
	github prowflagutil.GitHubOptions

	externalPluginsConfig string

	webhookSecretFile string
}

// validate validates github options.
func (o *options) validate() error {
	for idx, group := range []flagutil.OptionGroup{&o.github} {
		if err := group.Validate(o.dryRun); err != nil {
			return fmt.Errorf("%d: %w", idx, err)
		}
	}

	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&o.port, "port", 80, "Port to listen on.")
	fs.StringVar(&o.externalPluginsConfig, "external-plugins-config",
		"/etc/external_plugins_config/external_plugins_config.yaml", "Path to external plugin config file.")
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.StringVar(&o.webhookSecretFile, "hmac-secret-file",
		"/etc/webhook/hmac", "Path to the file containing the GitHub HMAC secret.")

	for _, group := range []flagutil.OptionGroup{&o.github} {
		group.AddFlags(fs)
	}
	_ = fs.Parse(os.Args[1:])
	return o
}

func main() {
	o := gatherOptions()
	if err := o.validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	log := logrus.StandardLogger().WithField("plugin", formatchecker.PluginName)

	epa := &tiexternalplugins.ConfigAgent{}
	if err := epa.Start(o.externalPluginsConfig, false); err != nil {
		log.WithError(err).Fatalf("Error loading external plugin config from %q.", o.externalPluginsConfig)
	}

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.github.TokenPath, o.webhookSecretFile}); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	githubClient, err := o.github.GitHubClient(secretAgent, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting GitHub client.")
	}
	// NOTICE: This error is only possible when using the GitHub APP,
	// but if we use the APP auth later we will have to handle the err.
	_ = githubClient.Throttle(360, 360)

	server := &server{
		tokenGenerator: secretAgent.GetTokenGenerator(o.webhookSecretFile),
		gc:             githubClient,
		configAgent:    epa,
		log:            log,
	}

	health := pjutil.NewHealth()
	health.ServeReady()

	mux := http.NewServeMux()
	mux.Handle("/", server)

	helpProvider := formatchecker.HelpProvider(epa)
	externalplugins.ServeExternalPluginHelp(mux, log, helpProvider)
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: mux}

	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)
}

// server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type server struct {
	tokenGenerator func() []byte
	gc             github.Client

	configAgent *tiexternalplugins.ConfigAgent
	log         *logrus.Entry
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
	l := s.log.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)
	// Get external plugins config.
	config := s.configAgent.Config()
	switch eventType {
	case tiexternalplugins.PullRequestEvent:
		var pe github.PullRequestEvent
		if err := json.Unmarshal(payload, &pe); err != nil {
			return err
		}
		go func() {
			if err := formatchecker.HandlePullRequestEvent(s.gc, &pe, config, l); err != nil {
				l.WithField("event-type", eventType).WithError(err).Info("Error handling event.")
			}
		}()
	case tiexternalplugins.IssuesEvent:
		var ie github.IssueEvent
		if err := json.Unmarshal(payload, &ie); err != nil {
			return err
		}
		go func() {
			if err := formatchecker.HandleIssueEvent(s.gc, &ie, config, l); err != nil {
				l.WithField("event-type", eventType).WithError(err).Info("Error handling event.")
			}
		}()
	default:
		s.log.Debugf("received an event of type %q but didn't ask for it", eventType)
	}
	return nil
}
