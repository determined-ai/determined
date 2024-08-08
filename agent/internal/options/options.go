package options

import (
	"crypto/tls"
	"encoding/json"
	"os"
	"reflect"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/config"
)

const (
	// RocrVisibleDevices define the ROCm resources allocated by Slurm.
	RocrVisibleDevices = "ROCR_VISIBLE_DEVICES"
	// CudaVisibleDevices define the CUDA resources allocated by Slurm.
	CudaVisibleDevices = "CUDA_VISIBLE_DEVICES"
)

// DefaultOptions returns the default configurable options for the Determined agent.
func DefaultOptions() *Options {
	return &Options{
		Log: config.LoggerConfig{
			Level: "trace",
			Color: true,
		},
		SlotType:               "auto",
		VisibleGPUs:            VisibleGPUsFromEnvironment(),
		BindIP:                 "0.0.0.0",
		BindPort:               9090,
		AgentReconnectAttempts: aproto.AgentReconnectAttempts,
		AgentReconnectBackoff:  int(aproto.AgentReconnectBackoff / time.Second),
		ContainerRuntime:       DockerContainerRuntime,
	}
}

// Options stores all the configurable options for the Determined agent.
type Options struct {
	ConfigFile string              `json:"config_file"`
	Log        config.LoggerConfig `json:"log"`

	MasterHost string `json:"master_host"`
	MasterPort int    `json:"master_port"`
	AgentID    string `json:"agent_id"`

	// Label has been deprecated; we now use ResourcePool to classify the agent.
	ResourcePool string `json:"resource_pool"`

	ContainerMasterHost string `json:"container_master_host"`
	ContainerMasterPort int    `json:"container_master_port"`

	SlotType    string `json:"slot_type"`
	VisibleGPUs string `json:"visible_gpus"`

	Security SecurityOptions `json:"security"`

	Debug           bool `json:"debug"`
	ArtificialSlots int  `json:"artificial_slots"`

	TLS         bool   `json:"tls"`
	TLSCertFile string `json:"tls_cert"`
	TLSKeyFile  string `json:"tls_key"`

	APIEnabled bool   `json:"api_enabled"`
	BindIP     string `json:"bind_ip"`
	BindPort   int    `json:"bind_port"`

	HTTPProxy  string `json:"http_proxy"`
	HTTPSProxy string `json:"https_proxy"`
	FTPProxy   string `json:"ftp_proxy"`
	NoProxy    string `json:"no_proxy"`

	AgentReconnectAttempts int `json:"agent_reconnect_attempts"`
	// TODO(ilia): switch this to better parsing with `model.Duration` similar to
	// master config.
	AgentReconnectBackoff int `json:"agent_reconnect_backoff"`

	ContainerRuntime string `json:"container_runtime"`

	ContainerAutoRemoveDisabled bool `json:"container_auto_remove_disabled"`

	Hooks HooksOptions `json:"hooks"`

	// The Fluent docker image to use, deprecated.
	Fluent FluentOptions `json:"fluent"`
}

// DefaultFluentOptions stores defaults for Agent FluentBit options, deprecated.
var DefaultFluentOptions = FluentOptions{
	ContainerName: "",
	Port:          0,
	Image:         "",
}

// Deprecations describe fields which were recently or will soon be removed.
func (o Options) Deprecations() (errs []error) {
	if !reflect.DeepEqual(o.Fluent, DefaultFluentOptions) {
		errs = append(errs, errors.Errorf("fluent options have been set for the agent; "+
			"support for fluent has been removed as of 0.24.1",
		))
	}
	return errs
}

// Validate validates the state of the Options struct.
func (o Options) Validate() []error {
	return []error{
		o.validateTLS(),
		check.In(o.SlotType, []string{"gpu", "cuda", "rocm", "cpu", "auto", "none"}),
		check.NotEmpty(o.MasterHost, "master host must be provided"),
	}
}

func (o Options) validateTLS() error {
	if !o.TLS || !o.APIEnabled {
		return nil
	}
	if o.TLSCertFile == "" {
		return errors.New("TLS cert file not specified")
	}
	if o.TLSKeyFile == "" {
		return errors.New("TLS key file not specified")
	}
	return nil
}

// Printable returns a printable string.
func (o Options) Printable() ([]byte, error) {
	optJSON, err := json.Marshal(o)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert config to JSON")
	}
	return optJSON, nil
}

// Resolve fully resolves the agent configuration, handling dynamic defaults.
func (o *Options) Resolve() {
	if o.MasterPort == 0 {
		if o.Security.TLS.Enabled {
			o.MasterPort = 443
		} else {
			o.MasterPort = 80
		}
	}
}

// SetAgentID resolves the name (or ID) of the agent.
func (o *Options) SetAgentID() error {
	if o.AgentID == "" {
		hostname, hErr := os.Hostname()
		if hErr != nil {
			return hErr
		}
		o.AgentID = hostname
	}
	return nil
}

// SecurityOptions stores configurable security-related options.
type SecurityOptions struct {
	TLS TLSOptions `json:"tls"`
}

// TLSOptions is the TLS connection configuration for the agent.
type TLSOptions struct {
	Enabled        bool   `json:"enabled"`
	SkipVerify     bool   `json:"skip_verify"`
	MasterCert     string `json:"master_cert"`
	MasterCertName string `json:"master_cert_name"`
	ClientCert     string `json:"client_cert"`
	ClientKey      string `json:"client_key"`
}

// Validate implements the check.Validatable interface.
func (t TLSOptions) Validate() []error {
	var errs []error
	if t.MasterCert != "" && t.SkipVerify {
		errs = append(errs, errors.New("cannot specify a master cert file with verification off"))
	}
	return errs
}

// ReadClientCertificate returns the client certificate described by this configuration (nil if it
// does not allow TLS to be enabled).
func (t TLSOptions) ReadClientCertificate() (*tls.Certificate, error) {
	if t.ClientCert == "" || t.ClientKey == "" {
		return nil, nil
	}
	cert, err := tls.LoadX509KeyPair(t.ClientCert, t.ClientKey)
	return &cert, err
}

// FluentOptions stores configurable Fluent Bit-related options, deprecated no longer in use.
type FluentOptions struct {
	Image         string `json:"image"`
	Port          int    `json:"port"`
	ContainerName string `json:"container_name"`
}

// HooksOptions contains external commands to be run when specific things happen.
type HooksOptions struct {
	OnConnectionLost []string `json:"on_connection_lost"`
}

// ContainerRuntime configures which container runtime to use.
type ContainerRuntime string

// Available container runtimes.
const (
	DockerContainerRuntime = "docker"
)

// VisibleGPUsFromEnvironment returns GPU visibility information from the environment
// if any, else "".
func VisibleGPUsFromEnvironment() (visDevices string) {
	visDevices, defined := os.LookupEnv(RocrVisibleDevices)
	if !defined {
		visDevices, _ = os.LookupEnv(CudaVisibleDevices)
	}
	return
}
