package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/logger"
)

const defaultConfigPath = "/etc/determined/master.yaml"

// logStoreSize is how many log events to keep in memory.
const logStoreSize = 25000

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "determined-master",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runRoot(); err != nil {
				log.Error(fmt.Sprintf("%+v", err))
				os.Exit(1)
			}
		},
	}
	cmd.AddCommand(newMigrateCmd())
	cmd.AddCommand(newPopulateCmd())
	return cmd
}

var rootCmd = newRootCmd()

func runRoot() error {
	logStore := logger.NewLogBuffer(logStoreSize)
	log.AddHook(logStore)

	err := initializeConfig()
	if err != nil {
		return err
	}
	config := config.GetMasterConfig()

	printableConfig, err := config.Printable()
	if err != nil {
		return err
	}
	log.Infof("master configuration: %s", printableConfig)

	err = os.MkdirAll(config.Cache.CacheDir, 0o700)
	if err != nil {
		log.WithError(err).Errorf("Failed to make cache directory (%s)", config.Cache.CacheDir)
		return err
	}

	m := internal.New(logStore, config)
	return m.Run(context.TODO())
}

// initializeConfig initializes master config with the validated configuration populated from config
// file, environment variables, and command line flags) and also initializes
// global logging state based on those options.
func initializeConfig() error {
	// Fetch an initial config to get the config file path and read its settings into Viper.
	initialConfig, err := getConfig(v.AllSettings())
	if err != nil {
		return err
	}

	bs, err := readConfigFile(initialConfig.ConfigFile)
	if err != nil {
		return err
	}
	// THIS LOWERCASES. If the master config has 1 pod-spec set, this is where its lowercased
	if err = mergeConfigBytesIntoViper(bs); err != nil {
		return err
	}

	// Now call viper.AllSettings() again to get the full config, containing all values from CLI flags,
	// environment variables, and the configuration file.
	// TODO CAROLINA: NOW, if pod-spec is set the same as master but case-different than lowercase,
	// its read as a diff value
	conf, err := getConfig(v.AllSettings())
	if err != nil {
		return err
	}

	if err := check.Validate(conf); err != nil {
		return err
	}

	config.SetMasterConfig(conf)

	for _, deprecation := range conf.Deprecations() {
		log.Warn(deprecation.Error())
	}

	return nil
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

func mergeConfigBytesIntoViper(bs []byte) error {
	// TODO CAROLINA
	// var configMap map[string]interface{}
	// if err := yaml.Unmarshal(bs, &configMap); err != nil {
	// 	return errors.Wrap(err, "error unmarshal yaml configuration file")
	// }
	if err := v.MergeConfig(bytes.NewReader(bs)); err != nil {
		return errors.Wrap(err, "error merge configuration to viper")
	}
	/*
		for key, val := range configMap {
			log.Warnf("CONFIG KEY: %s", key)
			// Viper will check for the key in the following order:
			// override, flag, env, config file, key/value store, default
			check := v.Get(key)
			if check == nil {
				// If this key is not already set, set it with the master config value
				v.Set(key, val)
			}
		}
	*/
	/*
		if err := v.MergeConfigMap(configMap); err != nil {
			return errors.Wrap(err, "error merge configuration to viper")
		}
	*/
	return nil
}

func getConfig(configMap map[string]interface{}) (*config.Config, error) {
	configMap, err := applyBackwardsCompatibility(configMap)
	if err != nil {
		return nil, errors.Wrap(err, "cannot apply backwards compatibility")
	}

	config := config.DefaultConfig()
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

func applyBackwardsCompatibility(configMap map[string]interface{}) (map[string]interface{}, error) {
	const (
		defaultVal    = "default"
		agentVal      = "agent"
		kubernetesVal = "kubernetes"
	)

	_, rmExisted := configMap["resource_manager"]
	_, rpsExisted := configMap["resource_pools"]
	vScheduler, schedulerExisted := configMap["scheduler"]
	vProvisioner, provisionerExisted := configMap["provisioner"]

	// Ensure we use either the old schema or the new one.
	if (rmExisted || rpsExisted) && (schedulerExisted || provisionerExisted) {
		return nil, errors.New(
			"cannot use the old and the new configuration schema at the same time",
		)
	}
	if rmExisted || rpsExisted {
		return configMap, nil
	}

	// If use the old schema, convert it to the new one.
	newScheduler := map[string]interface{}{
		"type":           "fair_share",
		"fitting_policy": "best",
	}
	newRM := map[string]interface{}{
		"type": agentVal,
	}
	if schedulerExisted {
		schedulerMap, ok := vScheduler.(map[string]interface{})
		if !ok {
			return nil, errors.New("wrong type for scheduler field")
		}

		if vFit, ok := schedulerMap["fit"]; ok {
			newScheduler["fitting_policy"], ok = vFit.(string)
			if !ok {
				return nil, errors.New("wrong type for scheduler.fit field")
			}
		}
		if vType, ok := schedulerMap["type"]; ok {
			newScheduler["type"], ok = vType.(string)
			if !ok {
				return nil, errors.New("wrong type for scheduler.type field")
			}
		}
		if vRP, ok := schedulerMap["resource_provider"]; ok {
			rpMap, ok := vRP.(map[string]interface{})
			if !ok {
				return nil, errors.New("wrong type for scheduler.resource_provider field")
			}

			vRPType, ok := rpMap["type"]
			if ok {
				switch vRPTypeStr, ok := vRPType.(string); {
				case !ok:
					return nil, errors.New("wrong type for scheduler.resource_provider.type field")
				case vRPTypeStr == defaultVal:
					newRM["type"] = agentVal
				case vRPTypeStr == kubernetesVal:
					newRM["type"] = kubernetesVal
				default:
					return nil, errors.New("wrong value for scheduler.resource_provider.type field")
				}
			} else {
				newRM["type"] = agentVal
			}
			if newRM["type"] == kubernetesVal {
				for key, value := range rpMap {
					if key == "type" {
						continue
					}
					newRM[key] = value
				}
			} else {
				newRM["default_cpu_resource_pool"] = defaultVal
				newRM["default_gpu_resource_pool"] = defaultVal
			}
		} else {
			newRM["default_cpu_resource_pool"] = defaultVal
			newRM["default_gpu_resource_pool"] = defaultVal
		}
	}
	newRM["scheduler"] = newScheduler
	configMap["resource_manager"] = newRM

	if newRM["type"] == agentVal {
		newRP := make(map[string]interface{})
		newRP["pool_name"] = defaultVal
		if provisionerExisted {
			provisionerMap, ok := vProvisioner.(map[string]interface{})
			if !ok {
				return nil, errors.New("wrong type for provisioner field")
			}

			newRP["provider"] = provisionerMap
			if vProvider, ok := provisionerMap["provider"]; ok {
				vProviderStr, ok := vProvider.(string)
				if !ok {
					return nil, errors.New("wrong type for provisioner.provider field")
				}

				if vProviderStr != "aws" && vProviderStr != "gcp" {
					return nil, errors.New("wrong value for provisioner.provider field")
				}

				provisionerMap["type"] = provisionerMap["provider"]
				delete(provisionerMap, "provider")
			}
		}
		configMap["resource_pools"] = []map[string]interface{}{newRP}
	}

	delete(configMap, "scheduler")
	delete(configMap, "provisioner")

	return configMap, nil
}
