package options

import (
	"crypto/tls"
	"encoding/json"
	"reflect"

	"github.com/determined-ai/determined/master/pkg/check"

	"github.com/pkg/errors"
)

// Options stores all the configurable options for the Determined agent.
type Options struct {
	ConfigFile string `json:"config_file"`

	MasterHost      string `json:"master_host"`
	MasterPort      int    `json:"master_port"`
	AgentID         string `json:"agent_id"`
	ArtificialSlots int    `json:"artificial_slots"`
	SlotType        string `json:"slot_type"`

	ContainerMasterHost string `json:"container_master_host"`
	ContainerMasterPort int    `json:"container_master_port"`

	Label        string `json:"label"`
	ResourcePool string `json:"resource_pool"`

	APIEnabled bool   `json:"api_enabled"`
	BindIP     string `json:"bind_ip"`
	BindPort   int    `json:"bind_port"`

	VisibleGPUs string `json:"visible_gpus"`

	TLS      bool   `json:"tls"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`

	HTTPProxy  string `json:"http_proxy"`
	HTTPSProxy string `json:"https_proxy"`
	FTPProxy   string `json:"ftp_proxy"`
	NoProxy    string `json:"no_proxy"`

	Security SecurityOptions `json:"security"`

	// The Fluent docker image to use, deprecated.
	Fluent FluentOptions `json:"fluent"`

	ContainerAutoRemoveDisabled bool `json:"container_auto_remove_disabled"`

	AgentReconnectAttempts int `json:"agent_reconnect_attempts"`
	// TODO(ilia): switch this to better parsing with `model.Duration` similar to
	// master config.
	AgentReconnectBackoff int `json:"agent_reconnect_backoff"`

	Hooks HooksOptions `json:"hooks"`

	ContainerRuntime   string             `json:"container_runtime"`
	ImageRoot          string             `json:"image_root"`
	SingularityOptions SingularityOptions `json:"singularity_options"`
	PodmanOptions      PodmanOptions      `json:"podman_options"`

	Debug bool `json:"debug"`
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
	if o.CertFile == "" {
		return errors.New("TLS cert file not specified")
	}
	if o.KeyFile == "" {
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
	ApptainerContainerRuntime   = "apptainer"
	SingularityContainerRuntime = "singularity"
	DockerContainerRuntime      = "docker"
	PodmanContainerRuntime      = "podman"
)

// SingularityOptions configures how we interact with Singularity.
type SingularityOptions struct {
	// AllowNetworkCreation allows the agent to use `singularity run`'s `--net` option, which sets
	// up and launches containers into a new network namespace. Disabled by default since this
	// requires root or a suid installation with /etc/subuid --fakeroot.
	AllowNetworkCreation bool `json:"allow_network_creation"`
}

// PodmanOptions configures how we interact with podman.
type PodmanOptions struct {
	AllowNetworkCreation bool `json:"allow_network_creation"` // review
}
