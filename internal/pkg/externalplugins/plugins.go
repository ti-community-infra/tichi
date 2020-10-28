package externalplugins

import (
	"io/ioutil"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

var (
	// pullDuration is a duration for pull config form file.
	pullDuration = 1 * time.Minute
)

// ConfigAgent contains the agent mutex and the agent configuration.
type ConfigAgent struct {
	mut           sync.Mutex
	configuration *Configuration
}

// Load attempts to load config from the path. It returns an error if either
// the file can't be read or the configuration is invalid.
// If checkUnknownPlugins is true, unrecognized plugin names will make config
// loading fail.
func (pa *ConfigAgent) Load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	np := &Configuration{}
	if err := yaml.Unmarshal(b, np); err != nil {
		return err
	}

	if err := np.Validate(); err != nil {
		return err
	}

	pa.Set(np)
	return nil
}

// Set attempts to set the plugins config.
func (pa *ConfigAgent) Set(pc *Configuration) {
	pa.mut.Lock()
	defer pa.mut.Unlock()
	pa.configuration = pc
}

// Start starts polling path for plugin config. If the first attempt fails,
// then start returns the error. Future errors will halt updates but not stop.
// If checkUnknownPlugins is true, unrecognized plugin names will make config
// loading fail.
func (pa *ConfigAgent) Start(path string, checkUnknownPlugins bool) error {
	if err := pa.Load(path); err != nil {
		return err
	}
	// nolint:staticcheck
	ticker := time.Tick(pullDuration)
	go func() {
		for range ticker {
			if err := pa.Load(path); err != nil {
				logrus.WithField("path", path).WithError(err).Error("Error loading plugin config.")
			}
		}
	}()
	return nil
}

// Config returns the agent current Configuration.
func (pa *ConfigAgent) Config() *Configuration {
	pa.mut.Lock()
	defer pa.mut.Unlock()
	return pa.configuration
}
