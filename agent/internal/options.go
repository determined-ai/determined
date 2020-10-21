package internal

import (
	"encoding/json"

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
}

// Validate validates the state of the Options struct.
func (o Options) Validate() []error {
	return []error{
		o.validateTLS(),
		check.In(o.SlotType, []string{"cpu", "gpu", "auto"}),
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

// SecurityOptions stores configurable security-related options.
type SecurityOptions struct {
	TLS TLSOptions `json:"tls"`
}

// TLSOptions is the TLS connection configuration for the agent.
type TLSOptions struct {
	Enabled    bool   `json:"enabled"`
	SkipVerify bool   `json:"skip_verify"`
	MasterCert string `json:"master_cert"`
}

// Validate implements the check.Validatable interface.
func (t TLSOptions) Validate() []error {
	var errs []error
	if t.MasterCert != "" && t.SkipVerify {
		errs = append(errs, errors.New("cannot specify a master cert file with verification off"))
	}
	return errs
}
