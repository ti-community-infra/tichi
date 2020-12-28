package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/ti-community-prow/internal/pkg/rerere"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/pod-utils/downwardapi"
)

const (
	// repoOwnerEnv specifies the repo's owner from the environment variable.
	repoOwnerEnv = "REPO_OWNER"
	// repoOwnerEnv specifies the repo's name from the environment variable.
	repoNameEnv = "REPO_NAME"
	// repoOwnerEnv specifies the pull's number from the environment variable.
	pullNumberEnv = "PULL_NUMBER"
	// pullBaseRefEnv specifies the pull's base hash from the environment variable.
	pullBaseRefEnv = "PULL_BASE_REF"
	// defaultWaitingPeriod specifies the default merge waiting period for Tide.
	defaultWaitingPeriod = time.Minute * 3
)

type options struct {
	dryRun bool
	labels prowflagutil.Strings

	github    prowflagutil.GitHubOptions
	git       prowflagutil.GitOptions
	retesting rerere.RetestingOptions
}

// validate validates options.
func (o *options) validate() error {
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
	if err := o.validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
	log := logrus.StandardLogger().WithField("component", "rerere")

	// Get job spec.
	rawJobSpec := os.Getenv(downwardapi.JobSpecEnv)
	if len(rawJobSpec) == 0 {
		log.Fatal("Error getting job spec.")
	}
	spec := &downwardapi.JobSpec{}
	err := json.Unmarshal([]byte(rawJobSpec), spec)
	if err != nil {
		log.WithError(err).Fatal("Error unmarshal job spec.")
	}

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.github.TokenPath}); err != nil {
		log.WithError(err).Fatal("Error starting secrets agent.")
	}

	githubClient, err := o.github.GitHubClient(secretAgent, o.dryRun)
	if err != nil {
		log.WithError(err).Fatal("Error getting GitHub client.")
	}

	gitClient, err := o.git.GitClient(githubClient, secretAgent.GetTokenGenerator(o.github.TokenPath), nil, o.dryRun)
	if err != nil {
		log.WithError(err).Fatal("Error getting Git client.")
	}

	// Get pr info.
	owner := os.Getenv(repoOwnerEnv)
	if len(owner) == 0 {
		log.Fatal("Error getting repo owner.")
	}
	repo := os.Getenv(repoNameEnv)
	if len(repo) == 0 {
		log.Fatal("Error getting repo name.")
	}
	// If not the batch prow job, we have to check the labels.
	pullNumber := os.Getenv(pullNumberEnv)
	if len(pullNumber) != 0 {
		number, err := strconv.Atoi(pullNumber)
		if err != nil {
			log.WithError(err).Fatal("Error convert pull number.")
		}
		pr, err := githubClient.GetPullRequest(owner, repo, number)
		if err != nil {
			log.WithError(err).Fatal("Error get pull request.")
		}

		var prLabels []string
		for _, label := range pr.Labels {
			prLabels = append(prLabels, label.Name)
		}
		// All the labels match.
		labels := sets.NewString(prLabels...)
		if !labels.HasAll(o.labels.Strings()...) {
			log.Infof("Skip this retesting, labels missing: %v.", o.labels.StringSet().Difference(labels).List())
			return
		}
	}

	log.Info("Waiting previous PRs merged.")
	// Before we start retesting, we have to give the Tide some time to merge the previous PRs.
	time.Sleep(defaultWaitingPeriod)

	pullBaseRef := os.Getenv(pullBaseRefEnv)
	if len(pullBaseRef) == 0 {
		log.Fatal("Error pull request base ref.")
	}

	// If we found the base commit not latest, we need to skip this retesting.
	// Tide will retesting it before merge.
	latestBaseCommit, err := githubClient.GetSingleCommit(owner, repo, pullBaseRef)
	if err != nil {
		log.WithError(err).Fatal("Error get latest base commit.")
	}
	if latestBaseCommit.SHA != spec.Refs.BaseSHA {
		log.Infof("Skip this retesting, base sha mismatch: expect %s, got %s.", latestBaseCommit.SHA, spec.Refs.BaseSHA)
		return
	}

	// Init client form current dir.
	client, err := gitClient.ClientFromDir(owner, repo, "")
	if err != nil {
		log.WithError(err).Fatal("Error init git client form current dir.")
	}

	// Retesting it.
	err = rerere.Retesting(log, githubClient, client, &o.retesting, owner, repo, spec)
	if err != nil {
		log.WithError(err).Fatal("Error retesting.")
	}
}
