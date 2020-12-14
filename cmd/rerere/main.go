package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/rerere"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/pod-utils/downwardapi"
)

const (
	repoOwnerEnv  = "REPO_OWNER"
	repoNameEnv   = "REPO_NAME"
	pullNumberEnv = "PULL_NUMBER"
)

type options struct {
	dryRun bool
	labels prowflagutil.Strings

	github    prowflagutil.GitHubOptions
	git       prowflagutil.GitOptions
	retesting rerere.RetestingOptions
}

func (o *options) Validate() error {
	for idx, group := range []flagutil.OptionGroup{&o.github, &o.git, &o.retesting} {
		if err := group.Validate(o.dryRun); err != nil {
			return fmt.Errorf("%d: %w", idx, err)
		}
	}

	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.Var(&o.labels, "labels", "Labels specifies the PR that can be tested.")
	for _, group := range []flagutil.OptionGroup{&o.github, &o.git, &o.retesting} {
		group.AddFlags(fs)
	}
	_ = fs.Parse(os.Args[1:])
	return o
}

func main() {
	o := gatherOptions()
	if err := o.Validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
	log := logrus.StandardLogger().WithField("plugin", "rerere")

	// Get job spec.
	rawJobSpec := os.Getenv(downwardapi.JobSpecEnv)
	if len(rawJobSpec) == 0 {
		logrus.Fatal("Error getting job spec.")
	}
	spec := &downwardapi.JobSpec{}
	err := json.Unmarshal([]byte(rawJobSpec), spec)
	if err != nil {
		logrus.WithError(err).Fatal("Error unmarshal job spec.")
	}

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.github.TokenPath}); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	githubClient, err := o.github.GitHubClient(secretAgent, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting GitHub client.")
	}

	gitClient, err := o.git.GitClient(githubClient, secretAgent.GetTokenGenerator(o.github.TokenPath), nil, o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting Git client.")
	}

	// Get pr info.
	owner := os.Getenv(repoOwnerEnv)
	if len(owner) == 0 {
		logrus.Fatal("Error getting repo owner.")
	}
	repo := os.Getenv(repoNameEnv)
	if len(repo) == 0 {
		logrus.Fatal("Error getting repo name.")
	}
	pullNumber := os.Getenv(pullNumberEnv)
	if len(pullNumber) == 0 {
		logrus.Fatal("Error getting pull number.")
	}
	number, err := strconv.Atoi(pullNumber)
	if err != nil {
		logrus.WithError(err).Fatal("Error convert pull number.")
	}

	pr, err := githubClient.GetPullRequest(owner, repo, number)
	if err != nil {
		logrus.WithError(err).Fatal("Error get pull request.")
	}

	var prLabels []string
	for _, label := range pr.Labels {
		prLabels = append(prLabels, label.Name)
	}
	// All the labels match.
	labels := sets.NewString(prLabels...)
	if !labels.HasAll(o.labels.Strings()...) {
		return
	}

	// Init client form current dir.
	client, err := gitClient.ClientFromDir(owner, repo, "")
	if err != nil {
		logrus.WithError(err).Fatal("Error init git client form current dir.")
	}

	err = rerere.Retesting(log, githubClient, client, &o.retesting, owner, repo, spec)
	if err != nil {
		logrus.WithError(err).Fatal("Error retesting.")
	}
}
