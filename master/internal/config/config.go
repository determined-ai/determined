package config

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/config"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// These are package-level variables so that they can be set at link time.
// WARN: if you move them to a different package, you need to change the linked
// path in the make file and CI.
var (
	DefaultSegmentMasterKey = ""
	DefaultSegmentWebUIKey  = ""
)

var (
	once         sync.Once
	masterConfig *Config
)

// KubernetesDefaultPriority is the default K8 resource manager priority.
const (
	KubernetesDefaultPriority = 50
	sslModeDisable            = "disable"
)

type (
	// ExperimentConfigPatch is the updatedble fields for patching an experiment.
	ExperimentConfigPatch struct {
		Name *string `json:"name,omitempty"`
	}
)

// DefaultDBConfig returns the default configuration of the database.
func DefaultDBConfig() *DBConfig {
	return &DBConfig{
		Migrations: "file://static/migrations",
		SSLMode:    sslModeDisable,
	}
}

// CacheConfig is the configuration for file cache.
type CacheConfig struct {
	CacheDir string `json:"cache_dir"`
}

// DBConfig hosts configuration fields of the database.
type DBConfig struct {
	User        string `json:"user"`
	Password    string `json:"password"`
	Migrations  string `json:"migrations"`
	Host        string `json:"host"`
	Port        string `json:"port"`
	Name        string `json:"name"`
	SSLMode     string `json:"ssl_mode"`
	SSLRootCert string `json:"ssl_root_cert"`
}

// WebhooksConfig hosts configuration fields for webhook functionality.
type WebhooksConfig struct {
	BaseURL    string `json:"base_url"`
	SigningKey string `json:"signing_key"`
}

// DefaultConfig returns the default configuration of the master.
func DefaultConfig() *Config {
	return &Config{
		ConfigFile:            "",
		Log:                   *logger.DefaultConfig(),
		DB:                    *DefaultDBConfig(),
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
			AuthZ: *DefaultAuthZConfig(),
		},
		// If left unspecified, the port is later filled in with 8080 (no TLS) or 8443 (TLS).
		Port: 0,
		Root: "/usr/share/determined/master",
		Telemetry: config.TelemetryConfig{
			Enabled:                  true,
			OtelEnabled:              false,
			OtelExportedOtlpEndpoint: "localhost:4317",
			SegmentMasterKey:         DefaultSegmentMasterKey,
			SegmentWebUIKey:          DefaultSegmentWebUIKey,
		},
		EnableCors:  false,
		LaunchError: true,
		ClusterName: "",
		Logging: model.LoggingConfig{
			DefaultLoggingConfig: &model.DefaultLoggingConfig{},
		},
		// For developers this should be a writable directory for caching files.
		Cache: CacheConfig{
			CacheDir: "/var/cache/determined",
		},
		FeatureSwitches: []string{},
		ResourceConfig:  *DefaultResourceConfig(),
	}
}

// Config is the configuration of the master.
//
// It is populated, in the following order, by the master configuration file,
// environment variables and command line arguments.
type Config struct {
	ConfigFile            string                            `json:"config_file"`
	Log                   logger.Config                     `json:"log"`
	DB                    DBConfig                          `json:"db"`
	TensorBoardTimeout    int                               `json:"tensorboard_timeout"`
	NotebookTimeout       *int                              `json:"notebook_timeout"`
	Security              SecurityConfig                    `json:"security"`
	CheckpointStorage     expconf.CheckpointStorageConfig   `json:"checkpoint_storage"`
	TaskContainerDefaults model.TaskContainerDefaultsConfig `json:"task_container_defaults"`
	Port                  int                               `json:"port"`
	Root                  string                            `json:"root"`
	Telemetry             config.TelemetryConfig            `json:"telemetry"`
	EnableCors            bool                              `json:"enable_cors"`
	LaunchError           bool                              `json:"launch_error"`
	ClusterName           string                            `json:"cluster_name"`
	Logging               model.LoggingConfig               `json:"logging"`
	Observability         ObservabilityConfig               `json:"observability"`
	Cache                 CacheConfig                       `json:"cache"`
	Webhooks              WebhooksConfig                    `json:"webhooks"`
	FeatureSwitches       []string                          `json:"feature_switches"`
	ResourceConfig

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
	if c.DB.Password != "" {
		c.DB.Password = hiddenValue
	}
	if c.Telemetry.SegmentMasterKey != "" {
		c.Telemetry.SegmentMasterKey = hiddenValue
	}
	if c.Telemetry.SegmentWebUIKey != "" {
		c.Telemetry.SegmentWebUIKey = hiddenValue
	}
	if c.TaskContainerDefaults.RegistryAuth != nil {
		if c.TaskContainerDefaults.RegistryAuth.Password != "" {
			// RegistryAuth is a pointer, so if we need to hide the password we need to be very
			// careful to replace the pointer, not the contents behind the pointer.
			printable := *c.TaskContainerDefaults.RegistryAuth
			printable.Password = hiddenValue
			c.TaskContainerDefaults.RegistryAuth = &printable
		}
	}

	c.CheckpointStorage = c.CheckpointStorage.Printable()

	pools := make([]ResourcePoolConfig, 0, len(c.ResourcePools))
	for _, poolConfig := range c.ResourcePools {
		pools = append(pools, poolConfig.Printable())
	}
	c.ResourcePools = pools

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
		c.ResourceManager.AgentRM.Scheduler = DefaultSchedulerConfig()
	}

	if c.ResourceManager.KubernetesRM != nil {
		if c.TaskContainerDefaults.Kubernetes == nil {
			c.TaskContainerDefaults.Kubernetes = &model.KubernetesTaskContainerDefaults{}
		}

		rmMaxSlots := c.ResourceManager.KubernetesRM.MaxSlotsPerPod
		taskMaxSlots := c.TaskContainerDefaults.Kubernetes.MaxSlotsPerPod
		if (rmMaxSlots != nil) == (taskMaxSlots != nil) {
			return fmt.Errorf("must provide exactly one of " +
				"resource_manager.max_slots_per_pod and " +
				"task_container_defaults.kubernetes.max_slots_per_pod")
		}

		if rmMaxSlots != nil {
			c.TaskContainerDefaults.Kubernetes.MaxSlotsPerPod = rmMaxSlots
		}
		if taskMaxSlots != nil {
			c.ResourceManager.KubernetesRM.MaxSlotsPerPod = taskMaxSlots
		}
		if maxSlotsPerPod := *c.ResourceManager.KubernetesRM.MaxSlotsPerPod; maxSlotsPerPod < 0 {
			return fmt.Errorf("max_slots_per_pod must be >= 0 got %d", maxSlotsPerPod)
		}
	}

	if c.Webhooks.SigningKey == "" {
		b := make([]byte, 6)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		c.Webhooks.SigningKey = hex.EncodeToString(b)
	}

	if err := c.ResolveResource(); err != nil {
		return err
	}

	if err := c.Logging.Resolve(); err != nil {
		return err
	}

	if c.Security.AuthZ.StrictNTSCEnabled {
		log.Warn("_strict_ntsc_enabled option is removed and will not have any effect.")
	}

	return nil
}

// Deprecations describe fields which were recently or will soon be removed.
func (c *Config) Deprecations() (errs []error) {
	for _, rp := range c.ResourcePools {
		switch {
		case rp.AgentReattachEnabled && c.ResourceManager.KubernetesRM != nil:
			errs = append(errs, fmt.Errorf(
				"agent_reattach_enabled does not impact Kubernetes resources behavior; "+
					"reattach is always enabled for Kubernetes resource pools",
			))
		case rp.AgentReattachEnabled:
			errs = append(errs, fmt.Errorf(
				"agent_reattach_enabled is set for resource pool %s but will be ignored; "+
					"as of 0.21.0 this feature is always on", rp.PoolName,
			))
		}
	}
	return errs
}

// SecurityConfig is the security configuration for the master.
type SecurityConfig struct {
	DefaultTask model.AgentUserGroup `json:"default_task"`
	TLS         TLSConfig            `json:"tls"`
	SSH         SSHConfig            `json:"ssh"`
	AuthZ       AuthZConfig          `json:"authz"`
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

// ProxiedServerConfig is the configuration for a internal proxied server.
type ProxiedServerConfig struct {
	// Prefix is the path prefix to match for this proxy.
	PathPrefix string `json:"path_prefix"`
	// Destination is the URL to proxy to.
	Destination string `json:"destination"`
}

// InternalConfig is the configuration for internal knobs.
type InternalConfig struct {
	AuditLoggingEnabled bool                   `json:"audit_logging_enabled"`
	ExternalSessions    model.ExternalSessions `json:"external_sessions"`
	ProxiedServers      []ProxiedServerConfig  `json:"proxied_servers"`
	BugLogEveryQuery    bool                   `json:"bun_log_every_query"`
}

// Validate implements the check.Validatable interface.
func (p *ProxiedServerConfig) Validate() []error {
	var errs []error
	if p.PathPrefix == "" {
		errs = append(errs, errors.New("path_prefix must be set"))
	}
	if p.Destination == "" {
		errs = append(errs, errors.New("destination must be set"))
	}
	target, err := url.Parse(p.Destination)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "failed to parse proxied destination"))
	}
	// ensure scheme and port is set
	if target.Scheme == "" {
		target.Scheme = "http"
	}
	if target.Port() == "" {
		errs = append(errs, errors.New("proxy path must include a port"))
	}
	return errs
}

// Validate implements the check.Validatable interface.
func (i *InternalConfig) Validate() []error {
	var errs []error
	// We allow setting multiple proxied servers but leave it up to the developer
	// to ensure that they don't conflict with eachother or other det routes.
	for _, p := range i.ProxiedServers {
		errs = append(errs, p.Validate()...)
	}
	return errs
}

// ObservabilityConfig is the configuration for observability metrics.
type ObservabilityConfig struct {
	EnablePrometheus bool `json:"enable_prometheus"`
}

func readPriorityFromScheduler(conf *SchedulerConfig) *int {
	if conf == nil || conf.Priority == nil {
		return nil
	}
	return conf.Priority.DefaultPriority
}

// ReadRMPreemptionStatus resolves the preemption status for a resource manager.
// TODO(Brad): Move these to a resource pool level API.
func ReadRMPreemptionStatus(rpName string) bool {
	config := GetMasterConfig()
	return readRMPreemptionStatus(config, rpName)
}

func readRMPreemptionStatus(config *Config, rpName string) bool {
	for _, rpConfig := range config.ResourcePools {
		if rpConfig.PoolName != rpName {
			continue
		}
		if rpConfig.Scheduler != nil {
			return rpConfig.Scheduler.GetPreemption()
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
		return config.ResourceManager.KubernetesRM.GetPreemption()
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

	var schedulerConf *SchedulerConfig

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
		return KubernetesDefaultPriority
	}

	return DefaultSchedulingPriority
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

// GetCertPEM returns the PEM-encoded certificate.
func GetCertPEM(cert *tls.Certificate) []byte {
	var certBytes []byte
	if cert != nil {
		for _, c := range cert.Certificate {
			b := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: c,
			})
			certBytes = append(certBytes, b...)
		}
	}
	return certBytes
}
