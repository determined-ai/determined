package main

import (
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/determined-ai/determined/agent/internal/options"
)

const viperKeyDelimiter = ".."

var v *viper.Viper

//nolint:gochecknoinits
func init() {
	registerAgentConfig()
}

type optionsKey []string

func (c optionsKey) EnvName() string {
	return "DET_" + strings.ReplaceAll(strings.ToUpper(c.FlagName()), "-", "_")
}

func (c optionsKey) AccessPath() string {
	return strings.ReplaceAll(strings.Join(c, viperKeyDelimiter), "-", "_")
}

func (c optionsKey) FlagName() string {
	return strings.Join(c, "-")
}

func registerString(flags *pflag.FlagSet, name optionsKey, value string, usage string) {
	flags.String(name.FlagName(), value, usage)
	_ = v.BindPFlag(name.AccessPath(), flags.Lookup(name.FlagName()))
	_ = v.BindEnv(name.AccessPath(), name.EnvName())
	v.SetDefault(name.AccessPath(), value)
}

func registerBool(flags *pflag.FlagSet, name optionsKey, value bool, usage string) {
	flags.Bool(name.FlagName(), value, usage)
	_ = v.BindEnv(name.AccessPath(), name.EnvName())
	_ = v.BindPFlag(name.AccessPath(), flags.Lookup(name.FlagName()))
	v.SetDefault(name.AccessPath(), value)
}

func registerInt(flags *pflag.FlagSet, name optionsKey, value int, usage string) {
	flags.Int(name.FlagName(), value, usage)
	_ = v.BindEnv(name.AccessPath(), name.EnvName())
	_ = v.BindPFlag(name.AccessPath(), flags.Lookup(name.FlagName()))
	v.SetDefault(name.AccessPath(), value)
}

func registerAgentConfig() {
	v = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDelimiter))
	v.SetTypeByDefaultValue(true)

	defaults := options.DefaultOptions()
	name := func(components ...string) optionsKey { return components }

	// Register flags and environment variables, and set default values for viper settings.

	// TODO(DET-8884): Configure log level through agent config file.
	rootCmd.PersistentFlags().StringP("log-level", "l", "info",
		"set the logging level (can be one of: debug, info, warn, error, or fatal)")
	rootCmd.PersistentFlags().Bool("log-color", true, "disable colored output")

	flags := runCmd.Flags()
	iFlags := runCmd.InheritedFlags()

	// Logging flags.
	logLevelName := name("log", "level")
	_ = v.BindEnv(logLevelName.AccessPath(), logLevelName.EnvName())
	_ = v.BindPFlag(logLevelName.AccessPath(), iFlags.Lookup(logLevelName.FlagName()))
	v.SetDefault(logLevelName.AccessPath(), defaults.Log.Level)

	logColorName := name("log", "color")
	_ = v.BindEnv(logColorName.AccessPath(), logColorName.EnvName())
	_ = v.BindPFlag(logColorName.AccessPath(), iFlags.Lookup(logColorName.FlagName()))
	v.SetDefault(logColorName.AccessPath(), true)

	// Top-level flags.
	registerString(flags, name("config-file"), defaults.ConfigFile,
		"Path to agent configuration file")
	registerString(flags, name("master-host"), defaults.MasterHost, "Hostname of the master")
	registerInt(flags, name("master-port"), defaults.MasterPort, "Port of the master")
	registerString(flags, name("agent-id"), defaults.AgentID, "Unique ID of this Determined agent")

	// ResourcePool flags.
	registerString(flags, name("resource-pool"), defaults.ResourcePool,
		"Resource Pool the agent belongs to")

	// Container flags.
	registerString(flags, name("container-master-host"), defaults.ContainerMasterHost,
		"Master hostname that containers started by this agent will connect to")
	registerInt(flags, name("container-master-port"), defaults.ContainerMasterPort,
		"Master port that containers started by this agent will connect to")

	// Device flags.
	registerString(flags, name("slot-type"), defaults.SlotType, "slot type to expose")
	registerString(flags, name("visible-gpus"), defaults.VisibleGPUs, "GPUs to expose as slots")

	// Security flags.
	registerBool(flags, name("security", "tls", "enabled"), defaults.Security.TLS.Enabled,
		"Whether to use TLS to connect to the master")
	registerBool(flags, name("security", "tls", "skip-verify"), defaults.Security.TLS.SkipVerify,
		"Whether to skip verifying the master certificate when TLS is on (insecure!)")
	registerString(flags, name("security", "tls", "master-cert"), defaults.Security.TLS.MasterCert,
		"CA cert file for the master")
	registerString(flags, name("security", "tls", "master-cert-name"),
		defaults.Security.TLS.MasterCertName,
		"expected address in the master TLS certificate (if different than the one used for connecting)",
	)

	// Debug flags.
	registerBool(flags, name("debug"), defaults.Debug, "Enable verbose script output")
	registerInt(flags, name("artificial-slots"), defaults.ArtificialSlots, "")
	flags.Lookup("artificial-slots").Hidden = true

	// Endpoint TLS flags.
	registerBool(flags, name("tls"), defaults.TLS, "Use TLS for the API server")
	registerString(flags, name("tls-cert"), defaults.TLSCertFile, "Path to TLS certification file")
	registerString(flags, name("tls-key"), defaults.TLSKeyFile, "Path to TLS key file")

	// Endpoint flags.
	registerBool(flags, name("api-enabled"), defaults.APIEnabled, "Enable agent API endpoints")
	registerString(flags, name("bind-ip"), defaults.BindIP,
		"IP address to listen on for API requests")
	registerInt(flags, name("bind-port"), defaults.BindPort, "Port to listen on for API requests")

	// Proxy flags.
	registerString(flags, name("http-proxy"), defaults.HTTPProxy,
		"The HTTP proxy address for the agent's containers")
	registerString(flags, name("https-proxy"), defaults.HTTPSProxy,
		"The HTTPS proxy address for the agent's containers")
	registerString(flags, name("ftp-proxy"), defaults.FTPProxy,
		"THe FTP proxy address for the agent's containers")
	registerString(flags, name("no-proxy"), defaults.NoProxy,
		"Addresses that the agent's containers should not proxy")

	// Fault-tolerance flags.
	registerInt(flags, name("agent-reconnect-attempts"), defaults.AgentReconnectAttempts,
		"Max attempts agent has to reconnect")
	registerInt(flags, name("agent-reconnect-backoff"), defaults.AgentReconnectBackoff,
		"Time between agent reconnect attempts")

	registerString(flags, name("container-runtime"), defaults.ContainerRuntime,
		"The container runtime to use")
}
