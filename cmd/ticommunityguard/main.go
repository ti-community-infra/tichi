package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/interrupts"
	"k8s.io/test-infra/prow/pjutil"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/blunderbuss"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/guard"
)

func main() {
	var log = logrus.StandardLogger().WithField("plugin", guard.PluginName)
	o, err := gatherOptions()
	if err != nil {
		log.Fatalf("Invalid options: %v", err)
	}

	if err := o.validate(); err != nil {
		log.Fatalf("Invalid options: %v", err)
	}

	epa := &tiexternalplugins.ConfigAgent{}
	if err := epa.Start(o.externalPluginsConfig, false); err != nil {
		log.WithError(err).
			Fatalf("Error loading external plugin config from %q.", o.externalPluginsConfig)
	}

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.github.TokenPath, o.webhookSecretFile}); err != nil {
		log.WithError(err).Fatal("Error starting secrets agent.")
	}

	githubClient, err := o.github.GitHubClient(secretAgent, o.dryRun)
	if err != nil {
		log.WithError(err).Fatal("Error getting GitHub client.")
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

	helpProvider := blunderbuss.HelpProvider(epa)
	externalplugins.ServeExternalPluginHelp(mux, log, helpProvider)
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: mux}

	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)
}
