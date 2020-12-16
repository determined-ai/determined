package model

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/union"
)

// LoggingConfig configures logging for tasks (currently only trials) in Determined.
type LoggingConfig struct {
	DefaultLoggingConfig *DefaultLoggingConfig `union:"backend,default" json:"-"`
	ElasticLoggingConfig *ElasticLoggingConfig `union:"backend,elastic" json:"-"`
}

// Resolve resolves the parts of the TaskContainerDefaultsConfig that must be evaluated on
// the master machine.
func (c LoggingConfig) Resolve() error {
	if c.ElasticLoggingConfig != nil {
		err := c.ElasticLoggingConfig.Resolve()
		if err != nil {
			return err
		}
	}
	return nil
}

// MarshalJSON serializes LoggingConfig.
func (c LoggingConfig) MarshalJSON() ([]byte, error) {
	return union.Marshal(c)
}

// UnmarshalJSON deserializes LoggingConfig.
func (c *LoggingConfig) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, c); err != nil {
		return err
	}

	type DefaultParser *LoggingConfig
	return errors.Wrap(json.Unmarshal(data, DefaultParser(c)), "failed to parse logging options")
}

// DefaultLoggingConfig configures logging for tasks using Fluent+HTTP to the master.
type DefaultLoggingConfig struct{}

// ElasticLoggingConfig configures logging for tasks using Fluent+Elastic.
type ElasticLoggingConfig struct {
	Host     string                `json:"host"`
	Port     int                   `json:"port"`
	Security ElasticSecurityConfig `json:"security"`
}

// Resolve resolves the configuration.
func (o *ElasticLoggingConfig) Resolve() error {
	return o.Security.Resolve()
}

// ElasticSecurityConfig configures security-related options for the elastic logging backend.
type ElasticSecurityConfig struct {
	Username *string         `json:"username"`
	Password *string         `json:"password"`
	TLS      TLSClientConfig `json:"tls"`
}

// Validate implements the check.Validatable interface.
func (o ElasticSecurityConfig) Validate() []error {
	var errs []error
	if (o.Username != nil) != (o.Password != nil) {
		errs = append(errs, errors.New("username and password must be specified together"))
	}
	return errs
}

// Resolve resolves the configuration.
func (o *ElasticSecurityConfig) Resolve() error {
	return o.TLS.Resolve()
}

// TLSClientConfig configures how to make a TLS connection.
type TLSClientConfig struct {
	Enabled         bool   `json:"enabled"`
	SkipVerify      bool   `json:"skip_verify"`
	CertificatePath string `json:"certificate"`
	CertificateName string `json:"certificate_name"`
	CertBytes       []byte
}

// Validate implements the check.Validatable interface.
func (t TLSClientConfig) Validate() []error {
	var errs []error
	if t.CertificatePath != "" && t.SkipVerify {
		errs = append(errs, errors.New("cannot specify a cert file with verification off"))
	}
	return errs
}

// Resolve resolves the configuration.
func (t *TLSClientConfig) Resolve() error {
	if t.CertificatePath == "" {
		return nil
	}
	certBytes, err := ioutil.ReadFile(t.CertificatePath)
	if err != nil {
		return err
	}
	t.CertBytes = certBytes
	return nil
}
