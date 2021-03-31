package main

import (
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/version"
)

var v *viper.Viper

// viperKeyDelimiter marks nested values in the configuration. For example, with a key delimiter
// of ".", viper will expect `{ db { host = "something" } }` to be stored and supplied as
// `db.host : "something"`. This also implies that if there is a key like `my.key: "ok"`, viper
// becomes unable to disambiguate the key from an object key delimited using ".". Because of this,
// viper will tell us the config map looks like `{ my { key = "ok" } }`, not `{ my.key = "ok"}`.
// The key delimiter is chosen as ".." because users would like to allow "." in keys without them
// being considered an object by viper and ".." causes a proper subset of previously unhandled
// configurations to be handled correctly; in otherwise, this doesn't break anything that was not
// already broken and fixes some parts of what was broken that people care about.
const viperKeyDelimiter = ".."

//nolint:gochecknoinit
func init() {
	// The version of rootCmd is set in init() rather than when `rootCmd` is initialized,
	// because link-time variable assignments are not applied when package-scoped variables
	// are initialized.
	rootCmd.Version = version.Version
	registerConfig()
}

type configKey []string

func (c configKey) EnvName() string {
	return "DET_" + strings.ReplaceAll(strings.ToUpper(c.FlagName()), "-", "_")
}

func (c configKey) AccessPath() string {
	return strings.ReplaceAll(strings.Join(c, viperKeyDelimiter), "-", "_")
}

func (c configKey) FlagName() string {
	return strings.Join(c, "-")
}

func registerString(flags *pflag.FlagSet, name configKey, value string, usage string) {
	flags.String(name.FlagName(), value, usage)
	_ = v.BindEnv(name.AccessPath(), name.EnvName())
	_ = v.BindPFlag(name.AccessPath(), flags.Lookup(name.FlagName()))
	v.SetDefault(name.AccessPath(), value)
}

func registerBool(flags *pflag.FlagSet, name configKey, value bool, usage string) {
	flags.Bool(name.FlagName(), value, usage)
	_ = v.BindEnv(name.AccessPath(), name.EnvName())
	_ = v.BindPFlag(name.AccessPath(), flags.Lookup(name.FlagName()))
	v.SetDefault(name.AccessPath(), value)
}

func registerInt(flags *pflag.FlagSet, name configKey, value int, usage string) {
	flags.Int(name.FlagName(), value, usage)
	_ = v.BindEnv(name.AccessPath(), name.EnvName())
	_ = v.BindPFlag(name.AccessPath(), flags.Lookup(name.FlagName()))
	v.SetDefault(name.AccessPath(), value)
}

func registerConfig() {
	// Relies on https://github.com/spf13/viper/pull/794. Once the points in the commentary
	// are addressed, specifically adding the option `v.AllowDelimiterInKey`, it may be better
	// to switch to that.
	v = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDelimiter))
	v.SetTypeByDefaultValue(true)

	defaults := internal.DefaultConfig()

	// Register flags and environment variables, and set default values for the flags.
	flags := rootCmd.Flags()
	name := func(components ...string) configKey { return components }

	registerString(flags, name("config-file"),
		defaults.ConfigFile, "location of config file")

	registerString(flags, name("log", "level"),
		defaults.Log.Level, "choose logging level from [trace, debug, info, warn, error, fatal]")
	registerBool(flags, name("log", "color"),
		defaults.Log.Color, "output logs in color")

	registerString(flags, name("db", "user"),
		defaults.DB.User, "database username")
	registerString(flags, name("db", "password"),
		defaults.DB.Password, "database password")
	registerString(flags, name("db", "host"),
		defaults.DB.Host, "database host")
	registerString(flags, name("db", "port"),
		defaults.DB.Port, "database port")
	registerString(flags, name("db", "name"),
		defaults.DB.Name, "database name")
	registerString(flags, name("db", "ssl-mode"),
		defaults.DB.SSLMode, "database ssl mode (disable, verify-ca, ...)")
	registerString(flags, name("db", "ssl-root-cert"),
		defaults.DB.SSLRootCert, "database ssl root cert path")

	registerInt(flags, name("security", "default-task", "uid"),
		defaults.Security.DefaultTask.UID, "security default task UID")
	registerInt(flags, name("security", "default-task", "gid"),
		defaults.Security.DefaultTask.GID, "security default task GID")
	registerString(flags, name("security", "default-task", "user"),
		defaults.Security.DefaultTask.User, "security default task username")
	registerString(flags, name("security", "default-task", "group"),
		defaults.Security.DefaultTask.Group, "security default task group name")

	registerString(flags, name("security", "tls", "cert"),
		defaults.Security.TLS.Cert, "TLS cert file")
	registerString(flags, name("security", "tls", "key"),
		defaults.Security.TLS.Key, "TLS key file")

	registerInt(flags, name("port"),
		defaults.Port, "server port")

	registerString(flags, name("root"),
		defaults.Root, "static file root directory")

	registerBool(flags, name("telemetry", "enabled"),
		defaults.Telemetry.Enabled, "enable telemetry")
	registerString(flags, name("telemetry", "segment-master-key"),
		defaults.Telemetry.SegmentMasterKey, "the Segment write key for the master")
	registerString(flags, name("telemetry", "segment-webui-key"),
		defaults.Telemetry.SegmentWebUIKey, "the Segment write key for the WebUI")

	registerString(flags, name("checkpoint-storage", "type"),
		model.DefaultCheckpointStorageType, "checkpoint storage type")
	registerString(flags, name("checkpoint-storage", "host-path"),
		model.DefaultSharedFSHostPath, "checkpoint storage host path")
}
