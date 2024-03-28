package model

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"

	"github.com/determined-ai/determined/master/pkg/union"
)

// LoggingConfig configures logging for tasks (currently only trials) in Determined.
type LoggingConfig struct {
	DefaultLoggingConfig *DefaultLoggingConfig `union:"type,default" json:"-"`
	ElasticLoggingConfig *ElasticLoggingConfig `union:"type,elastic" json:"-"`
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

// DefaultLoggingConfig configures logging for tasks using HTTP to the master.
type DefaultLoggingConfig struct{}

// ElasticLoggingConfig configures logging for tasks using Elastic.
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

// LogRetentionPolicy configures the default log retention policy for trials and tasks.
type LogRetentionPolicy struct {
	// Days is the default number of days to retain logs for.
	Days *int16 `json:"days"`
	// Schedule is a time duration or cron expression interval to cleanup logs.
	Schedule *string `json:"schedule"`
}

var (
	errLogRetentionDaysParse     = errors.New("log retention days must be between -1 and 32767")
	errLogRetentionScheduleParse = errors.New("log retention schedule must be a valid duration or cron expression")
)

// Validate implements the check.Validatable interface.
func (p LogRetentionPolicy) Validate() []error {
	var errs []error
	if p.Days != nil && *p.Days < -1 {
		errs = append(errs, errLogRetentionDaysParse)
	}
	if p.Schedule != nil {
		if _, err := time.ParseDuration(*p.Schedule); err != nil {
			if _, err := cron.ParseStandard(*p.Schedule); err != nil {
				errs = append(errs, errLogRetentionScheduleParse)
			}
		}
	}
	return errs
}

// TLSClientConfig configures how to make a TLS connection.
type TLSClientConfig struct {
	Enabled         bool   `json:"enabled"`
	SkipVerify      bool   `json:"skip_verify"`
	CertificatePath string `json:"certificate"`
	CertificateName string `json:"certificate_name"`
	CertBytes       []byte
}

// MakeTLSConfig constructs a TLSClientConfig to use the provided tls.Certificate.
func MakeTLSConfig(cert *tls.Certificate) (TLSClientConfig, error) {
	if cert == nil {
		return TLSClientConfig{}, nil
	}
	var content bytes.Buffer
	for _, c := range cert.Certificate {
		if err := pem.Encode(&content, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: c,
		}); err != nil {
			return TLSClientConfig{}, errors.Wrap(err, "failed to encode PEM")
		}
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return TLSClientConfig{}, errors.Wrap(err, "failed to parse certificate")
	}
	certName := ""
	if len(leaf.DNSNames) > 0 {
		certName = leaf.DNSNames[0]
	} else if len(leaf.IPAddresses) > 0 {
		certName = leaf.IPAddresses[0].String()
	}

	return TLSClientConfig{
		Enabled:         true,
		CertBytes:       content.Bytes(),
		CertificateName: certName,
	}, nil
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
	certBytes, err := os.ReadFile(t.CertificatePath)
	if err != nil {
		return err
	}
	t.CertBytes = certBytes
	return nil
}
