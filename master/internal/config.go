package internal

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
)

// These are package-level variables so that they can be set at link time.
var (
	DefaultSegmentMasterKey = ""
	DefaultSegmentWebUIKey  = ""
)

// DefaultConfig returns the default configuration of the master.
func DefaultConfig() *Config {
	defaultExp := model.DefaultExperimentConfig(nil)
	var c CheckpointStorageConfig
	if err := c.FromModel(&defaultExp.CheckpointStorage); err != nil {
		panic(err)
	}

	return &Config{
		ConfigFile: "",
		Log:        *logger.DefaultConfig(),
		DB:         *db.DefaultConfig(),
		TaskContainerDefaults: model.TaskContainerDefaultsConfig{
			ShmSizeBytes: 4294967296,
			NetworkMode:  "bridge",
		},
		TensorBoardTimeout: 5 * 60,
		Security: SecurityConfig{
			DefaultTask: model.AgentUserGroup{
				UID:   0,
				GID:   0,
				User:  "root",
				Group: "root",
			},
		},
		// If left unspecified, the port is later filled in with 8080 (no TLS) or 8443 (TLS).
		Port:              0,
		CheckpointStorage: c,
		HarnessPath:       "/opt/determined",
		Root:              "/usr/share/determined/master",
		Telemetry: TelemetryConfig{
			Enabled:          true,
			SegmentMasterKey: DefaultSegmentMasterKey,
			SegmentWebUIKey:  DefaultSegmentWebUIKey,
		},
		EnableCors:  false,
		ClusterName: "",
		Logging: model.LoggingConfig{
			DefaultLoggingConfig: &model.DefaultLoggingConfig{},
		},
		ResourceConfig: resourcemanagers.DefaultResourceConfig(),
	}
}

// Config is the configuration of the master.
//
// It is populated, in the following order, by the master configuration file,
// environment variables and command line arguments.
type Config struct {
	ConfigFile            string                            `json:"config_file"`
	Log                   logger.Config                     `json:"log"`
	DB                    db.Config                         `json:"db"`
	TensorBoardTimeout    int                               `json:"tensorboard_timeout"`
	Security              SecurityConfig                    `json:"security"`
	CheckpointStorage     CheckpointStorageConfig           `json:"checkpoint_storage"`
	TaskContainerDefaults model.TaskContainerDefaultsConfig `json:"task_container_defaults"`
	Port                  int                               `json:"port"`
	HarnessPath           string                            `json:"harness_path"`
	Root                  string                            `json:"root"`
	Telemetry             TelemetryConfig                   `json:"telemetry"`
	EnableCors            bool                              `json:"enable_cors"`
	ClusterName           string                            `json:"cluster_name"`
	Logging               model.LoggingConfig               `json:"logging"`

	*resourcemanagers.ResourceConfig
}

// Printable returns a printable string.
func (c Config) Printable() ([]byte, error) {
	const hiddenValue = "********"
	c.DB.Password = hiddenValue
	c.Telemetry.SegmentMasterKey = hiddenValue
	c.Telemetry.SegmentWebUIKey = hiddenValue

	cs, err := c.CheckpointStorage.printable()
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert checkpoint storage config to printable")
	}
	c.CheckpointStorage = cs

	optJSON, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert config to JSON")
	}
	return optJSON, nil
}

// Resolve resolves the values in the configuration.
func (c *Config) Resolve() error {
	if c.Port == 0 {
		if c.Security.TLS.Enabled() {
			c.Port = 8443
		} else {
			c.Port = 8080
		}
	}

	root, err := filepath.Abs(c.Root)
	if err != nil {
		return err
	}
	c.Root = root

	c.DB.Migrations = fmt.Sprintf("file://%s", filepath.Join(c.Root, "static/migrations"))

	if err := c.ResolveResource(); err != nil {
		return err
	}

	if err := c.Logging.Resolve(); err != nil {
		return err
	}

	return nil
}

// CheckpointStorageConfig defers the parsing of a
// model.CheckpointStorageConfig. The global (master) CheckpointStorageConfig is
// merged with the per-experiment config, so in general, validation cannot be
// performed until the per-experiment config is known.
type CheckpointStorageConfig []byte

// Validate implements the check.Validatable interface.
//
// The actual CheckpointStorageConfig is not known until the global (master)
// config is merged with the per-experiment config. Validate attempts to check
// as many fields as possible without depending on the per-experiment config.
func (c CheckpointStorageConfig) Validate() []error {
	m, err := c.ToModel()

	if err != nil {
		return []error{
			err,
		}
	}

	// We cannot validate a SharedFSConfig until users have a chance to set
	// host_path.
	m.SharedFSConfig = nil

	if err := check.Validate(m); err != nil {
		return []error{
			err,
		}
	}

	return nil
}

func (c *CheckpointStorageConfig) printable() ([]byte, error) {
	var hiddenValue = "********"
	switch csm, err := c.ToModel(); {
	case err != nil:
		return nil, err
	case csm.S3Config != nil:
		csm.S3Config.AccessKey = &hiddenValue
		csm.S3Config.SecretKey = &hiddenValue
		return csm.MarshalJSON()
	default:
		return csm.MarshalJSON()
	}
}

// FromModel initializes a CheckpointStorageConfig from the corresponding model.
func (c *CheckpointStorageConfig) FromModel(m *model.CheckpointStorageConfig) error {
	bs, err := json.Marshal(m)
	if err != nil {
		return err
	}

	*c = bs

	return nil
}

// ToModel returns the model.CheckpointStorageConfig for the current config.
func (c CheckpointStorageConfig) ToModel() (*model.CheckpointStorageConfig, error) {
	var m model.CheckpointStorageConfig

	if len(c) == 0 {
		return &m, nil
	}

	dec := json.NewDecoder(bytes.NewReader(c))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&m); err != nil {
		return nil, errors.WithStack(err)
	}

	return &m, nil
}

// MarshalJSON implements the json.Marshaler interface.
func (c CheckpointStorageConfig) MarshalJSON() ([]byte, error) {
	return c, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *CheckpointStorageConfig) UnmarshalJSON(data []byte) error {
	// Roundtrip through json.Unmarshal so that fields are updated elementwise,
	// which would be the behavior if CheckpointStorageConfig were a pure
	// struct. If we simply set *c = data, we would not preserve fields not
	// mentioned by data.

	m, err := c.ToModel()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := json.Unmarshal(data, &m); err != nil {
		return errors.WithStack(err)
	}

	return c.FromModel(m)
}

// SecurityConfig is the security configuration for the master.
type SecurityConfig struct {
	DefaultTask model.AgentUserGroup `json:"default_task"`
	TLS         TLSConfig            `json:"tls"`
}

// TLSConfig is the configuration for setting up serving over TLS.
type TLSConfig struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

// Validate implements the check.Validatable interface.
func (t *TLSConfig) Validate() []error {
	var errs []error
	if t.Cert == "" && t.Key != "" {
		errs = append(errs, errors.New("TLS key file provided without a cert file"))
	} else if t.Key == "" && t.Cert != "" {
		errs = append(errs, errors.New("TLS cert file provided without a key file"))
	}
	return errs
}

// Enabled returns whether this configuration makes it possible to enable TLS.
func (t *TLSConfig) Enabled() bool {
	return t.Cert != "" && t.Key != ""
}

// ReadCertificate returns the certificate described by this configuration (nil if it does not allow
// TLS to be enabled).
func (t *TLSConfig) ReadCertificate() (*tls.Certificate, error) {
	if !t.Enabled() {
		return nil, nil
	}
	cert, err := tls.LoadX509KeyPair(t.Cert, t.Key)
	return &cert, err
}

// TelemetryConfig is the configuration for telemetry.
type TelemetryConfig struct {
	Enabled          bool   `json:"enabled"`
	SegmentMasterKey string `json:"segment_master_key"`
	SegmentWebUIKey  string `json:"segment_webui_key"`
}
