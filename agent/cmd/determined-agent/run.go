package main

import (
	"context"
	"os"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/determined-ai/determined/agent/internal"
	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
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

func newRunCmd() *cobra.Command {
	opts := options.Options{}

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
		if err = bindEnv("DET_", cmd); err != nil {
			return err
		}

		if opts.AgentID == "" {
			hostname, hErr := os.Hostname()
			if hErr != nil {
				return hErr
			}
			opts.AgentID = hostname
		}

		if err = check.Validate(opts); err != nil {
			return errors.Wrap(err, "command-line arguments specify illegal configuration")
		}

		opts.Resolve()

		if err := internal.Run(context.Background(), version, opts); err != nil {
			log.Fatal(err)
		}

		return nil
	}

	// Top-level flags.
	cmd.Flags().StringVar(&opts.ConfigFile, "config-file", "", "Path to agent configuration file")
	cmd.Flags().StringVar(&opts.MasterHost, "master-host", "", "Hostname of the master")
	cmd.Flags().IntVar(&opts.MasterPort, "master-port", 0, "Port of the master")
	cmd.Flags().StringVar(&opts.AgentID, "agent-id", "", "Unique ID of this Determined agent")

	// Labels flags.
	cmd.Flags().StringVar(&opts.Label, "label", "",
		"This field has been deprecated and will be ignored, use ``resource_pool`` instead.")

	// ResourcePool flags.
	cmd.Flags().StringVar(&opts.ResourcePool, "resource-pool", "",
		"Resource Pool the agent belongs to")

	// Container flags.
	cmd.Flags().StringVar(&opts.ContainerMasterHost, "container-master-host", "",
		"Master hostname that containers started by this agent will connect to")
	cmd.Flags().IntVar(&opts.ContainerMasterPort, "container-master-port", 0,
		"Master port that containers started by this agent will connect to")

	// Device flags.
	cmd.Flags().StringVar(&opts.SlotType, "slot-type", "auto", "slot type to expose")
	cmd.Flags().StringVar(&opts.VisibleGPUs, "visible-gpus", "", "GPUs to expose as slots")

	// Security flags.
	cmd.Flags().BoolVar(
		&opts.Security.TLS.Enabled, "security-tls-enabled", false,
		"Whether to use TLS to connect to the master",
	)
	cmd.Flags().BoolVar(
		&opts.Security.TLS.SkipVerify, "security-tls-skip-verify", false,
		"Whether to skip verifying the master certificate when TLS is on (insecure!)",
	)
	cmd.Flags().StringVar(
		&opts.Security.TLS.MasterCert, "security-tls-master-cert", "", "CA cert file for the master",
	)
	cmd.Flags().StringVar(
		&opts.Security.TLS.MasterCertName, "security-tls-master-cert-name", "",
		"expected address in the master TLS certificate (if different than the one used for connecting)",
	)

	// Debug flags.
	cmd.Flags().BoolVar(&opts.Debug, "debug", false, "Enable verbose script output")
	cmd.Flags().IntVar(&opts.ArtificialSlots, "artificial-slots", 0, "")
	cmd.Flags().Lookup("artificial-slots").Hidden = true
	cmd.Flags().StringVar(&opts.ImageRoot, "image-root", "", "Path to local container image cache")

	// Endpoint TLS flags.
	cmd.Flags().BoolVar(&opts.TLS, "tls", false, "Use TLS for the API server")
	cmd.Flags().StringVar(&opts.CertFile, "tls-cert", "", "Path to TLS certificate file")
	cmd.Flags().StringVar(&opts.KeyFile, "tls-key", "", "Path to TLS key file")

	// Endpoint flags.
	cmd.Flags().BoolVar(&opts.APIEnabled, "enable-api", false, "Enable agent API endpoints")
	cmd.Flags().StringVar(&opts.BindIP, "bind-ip", "0.0.0.0",
		"IP address to listen on for API requests")
	cmd.Flags().IntVar(&opts.BindPort, "bind-port", 9090, "Port to listen on for API requests")

	// Proxy flags.
	cmd.Flags().StringVar(&opts.HTTPProxy, "http-proxy", "",
		"The HTTP proxy address for the agent's containers")
	cmd.Flags().StringVar(&opts.HTTPSProxy, "https-proxy", "",
		"The HTTPS proxy address for the agent's containers")
	cmd.Flags().StringVar(&opts.FTPProxy, "ftp-proxy", "",
		"The FTP proxy address for the agent's containers")
	cmd.Flags().StringVar(&opts.NoProxy, "no-proxy", "",
		"Addresses that the agent's containers should not proxy")

	// Logging flags.
	cmd.Flags().StringVar(&opts.Fluent.Image, "fluent-image", aproto.FluentImage,
		"Docker image to use for the managed Fluent Bit daemon")
	cmd.Flags().IntVar(&opts.Fluent.Port, "fluent-port", 24224,
		"TCP port for the Fluent Bit daemon to listen on")
	cmd.Flags().StringVar(&opts.Fluent.ContainerName, "fluent-container-name", "determined-fluent",
		"Name for the Fluent Bit container")

	// Fault-tolerance flags.
	cmd.Flags().IntVar(&opts.AgentReconnectAttempts, "agent-reconnect-attempts",
		aproto.AgentReconnectAttempts, "Max attempts agent has to reconnect")
	cmd.Flags().IntVar(&opts.AgentReconnectBackoff, "agent-reconnect-backoff",
		int(aproto.AgentReconnectBackoff/time.Second), "Time between agent reconnect attempts")

	cmd.Flags().StringVar(&opts.ContainerRuntime, "container-runtime",
		options.DockerContainerRuntime, "The container runtime to use")

	return cmd
}
