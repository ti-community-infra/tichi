package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/owners"
	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/interrupts"
	"k8s.io/test-infra/prow/pjutil"
	"k8s.io/test-infra/prow/plugins/lgtm"
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
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.StringVar(&o.externalPluginsConfig, "external-plugins-config",
		"/etc/external_plugins_config/external_plugins_config.yaml", "Path to external plugin config file.")
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

	logrus.SetFormatter(&logrus.JSONFormatter{})
	log := logrus.StandardLogger().WithField("plugin", lgtm.PluginName)

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.github.TokenPath, o.webhookSecretFile}); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	epa := &tiexternalplugins.ConfigAgent{}
	if err := epa.Start(o.externalPluginsConfig, false); err != nil {
		log.WithError(err).Fatalf("Error loading external plugin config from %q.", o.externalPluginsConfig)
	}

	githubClient, err := o.github.GitHubClient(secretAgent, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting GitHub client.")
	}
	githubClient.Throttle(360, 360)

	// Skip https verify.
	//nolint:gosec
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	server := &owners.Server{
		Client:         client,
		TokenGenerator: secretAgent.GetTokenGenerator(o.webhookSecretFile),
		Gc:             githubClient,
		ConfigAgent:    epa,
		Log:            log,
	}

	health := pjutil.NewHealth()
	health.ServeReady()

	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ti-community-owners")
	})
	router.GET("/ti-community-owners/repos/:org/:repo/pulls/:number/owners", func(c *gin.Context) {
		owner := c.Param("org")
		repo := c.Param("repo")
		number := c.Param("number")

		pullNumber, err := strconv.Atoi(number)
		if err != nil {
			c.Status(http.StatusNotFound)
			log.WithError(err).Error("Failed convert pull number.")
			return
		}

		// Get config everytime.
		config := server.ConfigAgent.Config()
		ownersData, err := server.ListOwners(owner, repo, pullNumber, config)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			log.WithError(err).Error("Failed list owners.")
			return
		}

		c.JSON(http.StatusOK, ownersData)
	})

	httpServer := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: router}

	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)
}
