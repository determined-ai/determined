package main

import (
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/determined-ai/determined/agent/internal"
)

const defaultConfigPath = "/etc/determined/agent.yaml"

func readConfigFile(configPath string) ([]byte, error) {
	isDefault := configPath == ""
	if isDefault {
		configPath = defaultConfigPath
	}

	var err error
	if _, err = os.Stat(configPath); err != nil {
		if isDefault && os.IsNotExist(err) {
			logrus.Warnf("no configuration file at %s, skipping", configPath)
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

func newRunCmd() *cobra.Command {
	opts := internal.Options{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "run the Determined agent",
		Args:  cobra.NoArgs,
	}

	cmd.RunE = func(*cobra.Command, []string) error {
		bs, err := readConfigFile(opts.ConfigFile)
		if err != nil {
			return err
		}
		if err = yaml.Unmarshal(bs, &opts, yaml.DisallowUnknownFields); err != nil {
			return errors.Wrap(err, "cannot unmarshal configuration")
		}
		if err := bindEnv("DET_", cmd); err != nil {
			return err
		}

		if opts.AgentID == "" {
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}
			opts.AgentID = hostname
		}

		// TODO: Migrate to the check library.
		for _, err := range opts.Validate() {
			if err != nil {
				return errors.Wrap(err, "command-line arguments specify illegal configuration")
			}
		}

		if err := internal.Run(version, opts); err != nil {
			logrus.Fatal(err)
		}

		return nil
	}

	// Top-level flags.
	cmd.Flags().StringVar(&opts.ConfigFile, "config-file", "", "Config file")
	cmd.Flags().StringVar(&opts.MasterHost, "master-host", "", "Hostname of the master")
	cmd.Flags().IntVar(&opts.MasterPort, "master-port", 0, "Port of the master")
	cmd.Flags().StringVar(&opts.AgentID, "agent-id", "",
		"unique ID of this Determined agent")

	// Labels flags.
	cmd.Flags().StringVar(&opts.Label, "label", "",
		"Label attached to the agent for scheduling constraints")

	// TLS flags.
	cmd.Flags().BoolVar(&opts.TLS, "tls", false, "Use TLS for all connections")
	cmd.Flags().StringVar(&opts.CertFile, "tls-cert", "", "Path to TLS certificate file")
	cmd.Flags().StringVar(&opts.KeyFile, "tls-key", "", "Path to TLS key file")

	// Container flags.
	cmd.Flags().StringVar(&opts.ContainerMasterHost, "container-master-host", "",
		"Hostname of the master that the container connect to")
	cmd.Flags().IntVar(&opts.ContainerMasterPort, "container-master-port", 0,
		"Port of the master that the container connect to")

	// Device flags.
	cmd.Flags().StringVar(&opts.VisibleGPUs, "visible-gpus", "", "GPUs to expose as slots")

	// Debug flags.
	cmd.Flags().IntVar(&opts.ArtificialSlots, "artificial-slots", 0, "")
	cmd.Flags().Lookup("artificial-slots").Hidden = true

	// Endpoint flags.
	cmd.Flags().BoolVar(&opts.APIEnabled, "enable-api", false, "Enable agent API endpoints")
	cmd.Flags().StringVar(&opts.BindIP, "bind-ip", "0.0.0.0", "IP address to listen on")
	cmd.Flags().IntVar(&opts.BindPort, "bind-port", 9090, "port to listen on")

	return cmd
}
