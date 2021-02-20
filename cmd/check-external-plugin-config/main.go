package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"sigs.k8s.io/yaml"
)

// options specifies command line parameters.
type options struct {
	externalPluginConfigPath string
}

func (o *options) DefaultAndValidate() error {
	if o.externalPluginConfigPath == "" {
		return errors.New("required flag --external-plugin-config-path was unset")
	}
	return nil
}

// parseOptions is used to parse command line parameters.
func parseOptions() (options, error) {
	o := options{}

	if err := o.gatherOptions(flag.CommandLine, os.Args[1:]); err != nil {
		return options{}, err
	}

	return o, nil
}

func (o *options) gatherOptions(flag *flag.FlagSet, args []string) error {
	flag.StringVar(&o.externalPluginConfigPath, "external-plugin-config-path", "",
		"Path to external_plugin_config.yaml.")

	if err := flag.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %v", err)
	}
	if err := o.DefaultAndValidate(); err != nil {
		return fmt.Errorf("invalid options: %v", err)
	}

	return nil
}

func main() {
	o, err := parseOptions()
	if err != nil {
		logrus.Fatalf("Error parsing options - %v", err)
	}

	if err := validate(o); err != nil {
		logrus.WithError(err).Fatal("Validation failed.")
	} else {
		logrus.Info("checkpluginconfig passes without any error!")
	}
}

func validate(o options) error {
	bytes, err := ioutil.ReadFile(o.externalPluginConfigPath)
	if err != nil {
		return err
	}

	config := &externalplugins.Configuration{}
	if err := yaml.Unmarshal(bytes, config); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	return nil
}
