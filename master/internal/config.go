package internal

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// These are package-level variables so that they can be set at link time.
var (
	DefaultSegmentMasterKey = ""
	DefaultSegmentWebUIKey  = ""
)

// DefaultConfig returns the default configuration of the master.
func DefaultConfig() *Config {
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
	CheckpointStorage     *expconf.CheckpointStorageConfig  `json:"checkpoint_storage"`
	TaskContainerDefaults model.TaskContainerDefaultsConfig `json:"task_container_defaults"`
	Port                  int                               `json:"port"`
	HarnessPath           string                            `json:"harness_path"`
	Root                  string                            `json:"root"`
	Telemetry             TelemetryConfig                   `json:"telemetry"`
	EnableCors            bool                              `json:"enable_cors"`
	ClusterName           string                            `json:"cluster_name"`
	Logging               model.LoggingConfig               `json:"logging"`

	Scheduler   *resourcemanagers.Config `json:"scheduler"`
	Provisioner *provisioner.Config      `json:"provisioner"`
	*resourcemanagers.ResourcePoolsConfig
	ResourceManager *resourcemanagers.ResourceManagerConfig `json:"resource_manager"`
}

// Validate implements the check.Validate interface.
func (c *Config) Validate() []error {
	if c.CheckpointStorage != nil {
		if ok, errs := schemas.IsComplete(c.CheckpointStorage); !ok {
			return errs
		}
	}
	return nil
}

// Printable returns a printable string.
func (c Config) Printable() ([]byte, error) {
	const hiddenValue = "********"
	c.DB.Password = hiddenValue
	c.Telemetry.SegmentMasterKey = hiddenValue
	c.Telemetry.SegmentWebUIKey = hiddenValue

	if c.CheckpointStorage != nil {
		// Make a quick copy of the checkpoint storage and make that printable.
		var cs expconf.CheckpointStorageConfig
		schemas.Merge(&cs, c.CheckpointStorage)
		cs.Printable()
		*c.CheckpointStorage = cs
	}

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

	c.ResourceManager, c.ResourcePoolsConfig, err = resourcemanagers.ResolveConfig(
		c.Scheduler, c.Provisioner, c.ResourceManager, c.ResourcePoolsConfig,
	)
	if err != nil {
		return err
	}
	c.Scheduler, c.Provisioner = nil, nil

	if err := c.Logging.Resolve(); err != nil {
		return err
	}

	return nil
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
