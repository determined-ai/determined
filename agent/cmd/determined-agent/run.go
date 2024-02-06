package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/determined-ai/determined/agent/internal"
	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/master/pkg/check"
)

const (
	defaultConfigPath = "/etc/determined/agent.yaml"
)

func newRunCmd() *cobra.Command {
	opts := options.DefaultOptions()

	cmd := &cobra.Command{
		Use:   "run",
		Short: "run the Determined agent",
		Args:  cobra.NoArgs,
	}

	cmd.RunE = func(*cobra.Command, []string) error {
		// Retrieve current Viper settings, which should presently be either default config values
		// or flags that overwrote them, and store config settings into opts.
		bs, err := json.Marshal(v.AllSettings())
		if err != nil {
			return errors.Wrap(err, "cannot marshal configuration map into json bytes")
		}
		if err = yaml.Unmarshal(bs, opts, yaml.DisallowUnknownFields); err != nil {
			return errors.Wrap(err, "cannot unmarshal configuration")
		}

		// Retrieve values from config file and merge them into Viper.
		bs, err = readConfigFile(opts.ConfigFile)
		if err != nil {
			return err
		}
		opts, err = mergeConfigIntoViper(bs)
		if err != nil {
			return err
		}

		err = opts.SetAgentID()
		if err != nil {
			return err
		}

		opts.Resolve()

		if err = check.Validate(*opts); err != nil {
			return errors.Wrap(err, "command-line arguments specify illegal configuration")
		}

		for _, deprecation := range opts.Deprecations() {
			log.Warn(deprecation.Error())
		}
		if err := internal.Run(context.Background(), version, *opts); err != nil {
			log.Fatal(err)
		}

		return nil
	}

	return cmd
}

func mergeConfigIntoViper(bs []byte) (*options.Options, error) {
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(bs, &configMap); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal yaml configuration file")
	}

	if err := v.MergeConfigMap(configMap); err != nil {
		return nil, errors.Wrap(err, "can't merge configuration to viper")
	}

	// Use updated Viper config that now has default, config, and flag values set for
	// agent configuration options with the following precedence:
	// flag > config > default (where > => overrides)
	// and return agent config updated with the new viper settings.
	return getAgentConfig(v.AllSettings())
}

func readConfigFile(configPath string) ([]byte, error) {
	isDefault := configPath == ""
	if isDefault {
		configPath = defaultConfigPath
	}

	var err error
	if _, err = os.Stat(configPath); err != nil {
		if isDefault && os.IsNotExist(err) {
			log.Warnf("no configuration file at %s, skipping", configPath)
			return nil, nil
		}
		return nil, errors.Wrap(err, "error finding configuration file")
	}
	bs, err := os.ReadFile(configPath) // #nosec G304
	if err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}
	return bs, nil
}

func getAgentConfig(map[string]interface{}) (*options.Options, error) {
	bs, err := json.Marshal(v.AllSettings())
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal configuration map into json bytes")
	}

	opts := &options.Options{}
	// Store updated agent config back into opts.
	if err = yaml.Unmarshal(bs, opts, yaml.DisallowUnknownFields); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal configuration")
	}
	return opts, nil
}
