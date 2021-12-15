package config

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/hpimportance"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/internal/resourcemanagers/kubernetes"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// These are package-level variables so that they can be set at link time.
var (
	DefaultSegmentMasterKey = ""
	DefaultSegmentWebUIKey  = ""
)
var once sync.Once
var masterConfig *Config

// DefaultConfig returns the default configuration of the master.
func DefaultConfig() *Config {
	return &Config{
		ConfigFile:            "",
		Log:                   *logger.DefaultConfig(),
		DB:                    *db.DefaultConfig(),
		TaskContainerDefaults: *model.DefaultTaskContainerDefaults(),
		TensorBoardTimeout:    5 * 60,
		Security: SecurityConfig{
			DefaultTask: model.AgentUserGroup{
				UID:   0,
				GID:   0,
				User:  "root",
				Group: "root",
			},
			SSH: SSHConfig{
				RsaKeySize: 1024,
			},
		},
		// If left unspecified, the port is later filled in with 8080 (no TLS) or 8443 (TLS).
		Port:        0,
		HarnessPath: "/opt/determined",
		Root:        "/usr/share/determined/master",
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
		HPImportance: hpimportance.HPImportanceConfig{
			WorkersLimit:   2,
			QueueLimit:     16,
			CoresPerWorker: 1,
			MaxTrees:       100,
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
	CheckpointStorage     expconf.CheckpointStorageConfig   `json:"checkpoint_storage"`
	TaskContainerDefaults model.TaskContainerDefaultsConfig `json:"task_container_defaults"`
	Port                  int                               `json:"port"`
	HarnessPath           string                            `json:"harness_path"`
	Root                  string                            `json:"root"`
	Telemetry             TelemetryConfig                   `json:"telemetry"`
	EnableCors            bool                              `json:"enable_cors"`
	ClusterName           string                            `json:"cluster_name"`
	Logging               model.LoggingConfig               `json:"logging"`
	HPImportance          hpimportance.HPImportanceConfig   `json:"hyperparameter_importance"`
	Observability         ObservabilityConfig               `json:"observability"`
	*resourcemanagers.ResourceConfig

	// Internal contains "hidden" useful debugging configurations.
	InternalConfig InternalConfig `json:"__internal"`
}

// GetMasterConfig returns reference to the master config singleton.
func GetMasterConfig() *Config {
	once.Do(func() {
		masterConfig = DefaultConfig()
	})
	return masterConfig
}

// SetMasterConfig sets the master config singleton.
func SetMasterConfig(aConfig *Config) {
	if masterConfig != nil {
		panic("master config is already set")
	}
	if aConfig == nil {
		panic("passed in config is nil")
	}
	config := GetMasterConfig()
	*config = *aConfig
}

// Printable returns a printable string.
func (c Config) Printable() ([]byte, error) {
	const hiddenValue = "********"
	c.DB.Password = hiddenValue
	c.Telemetry.SegmentMasterKey = hiddenValue
	c.Telemetry.SegmentWebUIKey = hiddenValue

	c.CheckpointStorage = c.CheckpointStorage.Printable()

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

	if c.ResourceManager.AgentRM != nil && c.ResourceManager.AgentRM.Scheduler == nil {
		c.ResourceManager.AgentRM.Scheduler = resourcemanagers.DefaultSchedulerConfig()
	}

	if err := c.ResolveResource(); err != nil {
		return err
	}

	if err := c.Logging.Resolve(); err != nil {
		return err
	}

	return nil
}

// SecurityConfig is the security configuration for the master.
type SecurityConfig struct {
	DefaultTask model.AgentUserGroup `json:"default_task"`
	TLS         TLSConfig            `json:"tls"`
	SSH         SSHConfig            `json:"ssh"`
}

// SSHConfig is the configuration setting for SSH.
type SSHConfig struct {
	RsaKeySize int `json:"rsa_key_size"`
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

// Validate implements the check.Validatable interface.
func (t *SSHConfig) Validate() []error {
	var errs []error
	if t.RsaKeySize < 1 {
		errs = append(errs, errors.New("RSA Key size must be greater than 0"))
	} else if t.RsaKeySize > 16384 {
		errs = append(errs, errors.New("RSA Key size must be less than 16,384"))
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

// InternalConfig is the configuration for internal knobs.
type InternalConfig struct {
	ExternalSessions model.ExternalSessions `json:"external_sessions"`
}

// ObservabilityConfig is the configuration for observability metrics.
type ObservabilityConfig struct {
	EnablePrometheus bool `json:"enable_prometheus"`
}

func readPriorityFromScheduler(conf *resourcemanagers.SchedulerConfig) *int {
	if conf == nil || conf.Priority == nil {
		return nil
	}
	return conf.Priority.DefaultPriority
}

// ReadRMPreemptionStatus resolves the preemption status for a resource manager.
func ReadRMPreemptionStatus(rpName string) bool {
	config := GetMasterConfig()

	for _, rpConfig := range config.ResourcePools {
		if rpConfig.PoolName != rpName {
			continue
		}
		if rpConfig.Scheduler != nil {
			return rpConfig.Scheduler.GetPreemption()
		}
		if rpConfig.Provider != nil && rpConfig.Provider.GCP != nil {
			return rpConfig.Provider.GCP.InstanceType.Preemptible
		}
		break
	}

	// if not found, fall back to resource manager config
	switch {
	case config.ResourceManager.AgentRM != nil:
		if config.ResourceManager.AgentRM.Scheduler == nil {
			panic("scheduler not configured")
		}
		return config.ResourceManager.AgentRM.Scheduler.GetPreemption()
	case config.ResourceManager.KubernetesRM != nil:
		return config.ResourceManager.KubernetesRM.DefaultScheduler == kubernetes.PreemptionScheduler
	default:
		panic("unexpected resource configuration")
	}
}

// ReadPriority resolves the priority value for a job.
func ReadPriority(rpName string, jobConf interface{}) int {
	config := GetMasterConfig()
	var prio *int
	// look at the idividual job config
	switch conf := jobConf.(type) {
	case *expconf.ExperimentConfig:
		prio = conf.Resources().Priority()
	case *model.CommandConfig:
		prio = conf.Resources.Priority
	}
	if prio != nil {
		return *prio
	}

	var schedulerConf *resourcemanagers.SchedulerConfig

	// if not found, fall back to the resource pools config
	for _, rpConfig := range config.ResourcePools {
		if rpConfig.PoolName != rpName {
			continue
		}
		schedulerConf = rpConfig.Scheduler
	}
	prio = readPriorityFromScheduler(schedulerConf)
	if prio != nil {
		return *prio
	}

	// if not found, fall back to resource manager config
	if config.ResourceManager.AgentRM != nil {
		schedulerConf = config.ResourceManager.AgentRM.Scheduler
		prio = readPriorityFromScheduler(schedulerConf)
		if prio != nil {
			return *prio
		}
	}

	if config.ResourceManager.KubernetesRM != nil {
		return resourcemanagers.KubernetesDefaultPriority
	}

	return resourcemanagers.DefaultSchedulingPriority
}

// ReadWeight resolves the weight value for a job.
func ReadWeight(rpName string, jobConf interface{}) float64 {
	var weight float64
	switch conf := jobConf.(type) {
	case *expconf.ExperimentConfig:
		weight = conf.Resources().Weight()
	case *model.CommandConfig:
		weight = conf.Resources.Weight
	}
	return weight
}
