package model

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strconv"

	"github.com/determined-ai/determined/master/pkg/union"

	"github.com/docker/docker/api/types"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
)

// TaskContainerDefaultsConfig configures docker defaults for all containers.
type TaskContainerDefaultsConfig struct {
	DtrainNetworkInterface string                `json:"dtrain_network_interface,omitempty"`
	NCCLPortRange          string                `json:"nccl_port_range,omitempty"`
	GLOOPortRange          string                `json:"gloo_port_range,omitempty"`
	ShmSizeBytes           int64                 `json:"shm_size_bytes,omitempty"`
	NetworkMode            container.NetworkMode `json:"network_mode,omitempty"`
	CPUPodSpec             *k8sV1.Pod            `json:"cpu_pod_spec"`
	GPUPodSpec             *k8sV1.Pod            `json:"gpu_pod_spec"`
	Image                  *RuntimeItem          `json:"image,omitempty"`
	RegistryAuth           *types.AuthConfig     `json:"registry_auth,omitempty"`
	ForcePullImage         bool                  `json:"force_pull_image,omitempty"`
	LogDriverOptions       LogDriverOptions      `json:"log_driver"`
}

func validatePortRange(portRange string) []error {
	var errs []error

	if portRange == "" {
		return errs
	}

	re := regexp.MustCompile("^([0-9]+):([0-9]+)$")
	submatches := re.FindStringSubmatch(portRange)
	if submatches == nil {
		errs = append(
			errs, errors.Errorf("expected port range of format \"MIN:MAX\" but got %q", portRange),
		)
		return errs
	}

	var min, max uint64
	var err error
	if min, err = strconv.ParseUint(submatches[1], 10, 16); err != nil {
		errs = append(errs, errors.Wrap(err, "invalid minimum port value"))
	}
	if max, err = strconv.ParseUint(submatches[2], 10, 16); err != nil {
		errs = append(errs, errors.Wrap(err, "invalid maximum port value"))
	}

	if min > max {
		errs = append(errs, errors.Errorf("port range minimum exceeds maximum (%v > %v)", min, max))
	}

	return errs
}

// Validate implements the check.Validatable interface.
func (c TaskContainerDefaultsConfig) Validate() []error {
	errs := []error{
		check.GreaterThan(c.ShmSizeBytes, int64(0), "shm_size_bytes must be >= 0"),
		check.NotEmpty(string(c.NetworkMode), "network_mode must be set"),
	}

	if err := validatePortRange(c.NCCLPortRange); err != nil {
		errs = append(errs, err...)
	}

	if err := validatePortRange(c.GLOOPortRange); err != nil {
		errs = append(errs, err...)
	}

	errs = append(errs, validatePodSpec(c.CPUPodSpec)...)
	errs = append(errs, validatePodSpec(c.GPUPodSpec)...)

	return errs
}

// Resolve resolves the parts of the TaskContainerDefaultsConfig that must be evaluated on
// the master machine.
func (c TaskContainerDefaultsConfig) Resolve() error {
	if c.LogDriverOptions.ElasticLogDriver != nil {
		err := c.LogDriverOptions.ElasticLogDriver.Resolve()
		if err != nil {
			return err
		}
	}
	return nil
}

// LogDriverOptions configure logging for tasks (currently only trials) in Determined.
type LogDriverOptions struct {
	DefaultLogDriver *DefaultLogDriverOptions `union:"backend,default" json:"-"`
	ElasticLogDriver *ElasticLogDriverOptions `union:"backend,elastic" json:"-"`
}

// MarshalJSON serializes LogDriverOptions.
func (o LogDriverOptions) MarshalJSON() ([]byte, error) {
	return union.Marshal(o)
}

// UnmarshalJSON deserializes LogDriverOptions.
func (o *LogDriverOptions) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, o); err != nil {
		return err
	}

	type DefaultParser *LogDriverOptions
	return errors.Wrap(json.Unmarshal(data, DefaultParser(o)), "failed to parse logging options")
}

// DefaultLogDriverOptions configure logging for tasks using Fluent+HTTP to the master.
type DefaultLogDriverOptions struct{}

// ElasticLogDriverOptions configure logging for tasks using Fluent+Elastic.
type ElasticLogDriverOptions struct {
	Host     string                 `json:"host"`
	Port     int                    `json:"port"`
	Security ElasticSecurityOptions `json:"security"`
}

// Resolve resolves the parts of the ElasticLogDriverOptions that must be evaluated on the
// master machine.
func (o *ElasticLogDriverOptions) Resolve() error {
	if o.Security.TLS.CertificatePath != "" {
		certBytes, err := ioutil.ReadFile(
			o.Security.TLS.CertificatePath)
		if err != nil {
			return err
		}
		o.Security.TLS.CertBytes = certBytes
	}
	return nil
}

// ElasticSecurityOptions configure security-related options for the elastic logging backend.
type ElasticSecurityOptions struct {
	Username *string           `json:"username"`
	Password *string           `json:"password"`
	TLS      ElasticTLSOptions `json:"tls"`
}

// Validate implements the check.Validatable interface.
func (o ElasticSecurityOptions) Validate() []error {
	var errs []error
	if (o.Username != nil) != (o.Password != nil) {
		errs = append(errs, errors.New("username and password must be specified together"))
	}
	return errs
}

// ElasticTLSOptions are the TLS connection configuration for the elastic logging backend.
type ElasticTLSOptions struct {
	Enabled         bool   `json:"enabled"`
	SkipVerify      bool   `json:"skip_verify"`
	CertificatePath string `json:"certificate"`
	CertBytes       []byte
}

// Validate implements the check.Validatable interface.
func (t ElasticTLSOptions) Validate() []error {
	var errs []error
	if t.CertificatePath != "" && t.SkipVerify {
		errs = append(errs, errors.New("cannot specify a elastic cert file with verification off"))
	}
	return errs
}
