package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/version"
)

const defaultConfigPath = "/etc/determined/master.yaml"

// logStoreSize is how many log events to keep in memory.
const logStoreSize = 25000

var rootCmd = &cobra.Command{
	Use: "determined-master",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runRoot(); err != nil {
			log.Error(fmt.Sprintf("%+v", err))
			os.Exit(1)
		}
	},
}

func runRoot() error {
	logStore := logger.NewLogBuffer(logStoreSize)
	log.AddHook(logStore)

	config, err := initializeConfig()
	if err != nil {
		return err
	}

	printableConfig, err := config.Printable()
	if err != nil {
		return err
	}
	log.Infof("master configuration: %s", printableConfig)

	m := internal.New(version.Version, logStore, config)
	return m.Run(context.TODO())
}

// initializeConfig returns the validated configuration populated from config
// file, environment variables, and command line flags) and also initializes
// global logging state based on those options.
func initializeConfig() (*internal.Config, error) {
	// Fetch an initial config to get the config file path and read its settings into Viper.
	initialConfig, err := getConfig(viper.AllSettings())
	if err != nil {
		return nil, err
	}

	bs, err := readConfigFile(initialConfig.ConfigFile)
	if err != nil {
		return nil, err
	}
	if err = mergeConfigBytesIntoViper(bs); err != nil {
		return nil, err
	}

	// Now call viper.AllSettings() again to get the full config, containing all values from CLI flags,
	// environment variables, and the configuration file.
	config, err := getConfig(viper.AllSettings())
	if err != nil {
		return nil, err
	}

	if err := check.Validate(config); err != nil {
		return nil, err
	}

	return config, nil
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
	bs, err := ioutil.ReadFile(configPath) // #nosec G304
	if err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}
	return bs, nil
}

func mergeConfigBytesIntoViper(bs []byte) error {
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(bs, &configMap); err != nil {
		return errors.Wrap(err, "error unmarshal yaml configuration file")
	}
	if err := viper.MergeConfigMap(configMap); err != nil {
		return errors.Wrap(err, "error merge configuration to viper")
	}
	return nil
}

func getConfig(configMap map[string]interface{}) (*internal.Config, error) {
	config := internal.DefaultConfig()
	bs, err := json.Marshal(configMap)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal configuration map into json bytes")
	}
	if err = yaml.Unmarshal(bs, &config, yaml.DisallowUnknownFields); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal configuration")
	}

	if err := config.Resolve(); err != nil {
		return nil, err
	}
	return config, nil
}
