package main

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/test-infra/pkg/flagutil"
	fu "k8s.io/test-infra/pkg/flagutil"
	pfu "k8s.io/test-infra/prow/flagutil"
)

const (
	defaultPort = 80
)

type options struct {
	port                  int
	dryRun                bool
	externalPluginsConfig string
	webhookSecretFile     string
	github                pfu.GitHubOptions
}

func (o *options) validate() error {
	for idx, group := range []fu.OptionGroup{&o.github} {
		if err := group.Validate(o.dryRun); err != nil {
			return fmt.Errorf("%d: %w", idx, err)
		}
	}

	return nil
}

func gatherOptions() (*options, error) {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&o.port, "port", defaultPort, "Port to listen on.")
	fs.StringVar(
		&o.externalPluginsConfig,
		"external-plugins-config",
		"/etc/external_plugins_config/external_plugins_config.yaml",
		"Path to external plugin config file.",
	)
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.StringVar(
		&o.webhookSecretFile,
		"hmac-secret-file",
		"/etc/webhook/hmac",
		"Path to the file containing the GitHub HMAC secret.",
	)

	for _, group := range []flagutil.OptionGroup{&o.github} {
		group.AddFlags(fs)
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}
	return &o, nil
}
