package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/cherrypicker"
	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
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

	log := logrus.StandardLogger().WithField("plugin", cherrypicker.PluginName)

	epa := &tiexternalplugins.ConfigAgent{}
	if err := epa.Start(o.externalPluginsConfig, false); err != nil {
		log.WithError(err).Fatalf("Error loading external plugin config from %q.", o.externalPluginsConfig)
	}

	if err := secret.Add(o.webhookSecretFile); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	githubClient, err := o.github.GitHubClient(o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting GitHub client.")
	}
	// NOTICE: This error is only possible when using the GitHub APP,
	// but if we use the APP auth later we will have to handle the err.
	_ = githubClient.Throttle(360, 360)

	gitClient, err := o.github.GitClientFactory("", nil, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting Git client.")
	}
	interrupts.OnInterrupt(func() {
		if err := gitClient.Clean(); err != nil {
			logrus.WithError(err).Error("Could not clean up git client cache.")
		}
	})

	email, err := githubClient.Email()
	if err != nil {
		log.WithError(err).Fatal("Error getting bot e-mail.")
	}

	botUser, err := githubClient.BotUser()
	if err != nil {
		logrus.WithError(err).Fatal("Error getting bot name.")
	}

	repos, err := githubClient.GetRepos(botUser.Login, true)
	if err != nil {
		log.WithError(err).Fatal("Error listing bot repositories.")
	}

	githubTokenGenerator := secret.GetTokenGenerator(o.github.TokenPath)
	server := &cherrypicker.Server{
		WebhookSecretGenerator: secret.GetTokenGenerator(o.webhookSecretFile),
		GitHubTokenGenerator:   githubTokenGenerator,
		BotUser:                botUser,
		Email:                  email,
		ConfigAgent:            epa,

		GitClient:    gitClient,
		GitHubClient: newExtGithubClient(githubClient, githubTokenGenerator),
		Log:          log,

		Bare:      &http.Client{},
		PatchURL:  "https://patch-diff.githubusercontent.com",
		GitHubURL: "https://github.com",

		Repos: repos,
	}

	health := pjutil.NewHealth()
	health.ServeReady()

	mux := http.NewServeMux()
	mux.Handle("/", server)

	helpProvider := cherrypicker.HelpProvider(epa)
	externalplugins.ServeExternalPluginHelp(mux, log, helpProvider)
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)
}
