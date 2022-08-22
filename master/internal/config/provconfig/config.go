package provconfig

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/union"
	"github.com/determined-ai/determined/master/version"
)

const defaultMasterPort = "8080"

// Config describes config for provisioner.
type Config struct {
	MasterURL              string            `json:"master_url"`
	MasterCertName         string            `json:"master_cert_name"`
	StartupScript          string            `json:"startup_script"`
	ContainerStartupScript string            `json:"container_startup_script"`
	AgentDockerNetwork     string            `json:"agent_docker_network"`
	AgentDockerRuntime     string            `json:"agent_docker_runtime"`
	AgentDockerImage       string            `json:"agent_docker_image"`
	AgentFluentImage       string            `json:"agent_fluent_image"`
	AgentReconnectAttempts int               `json:"agent_reconnect_attempts"`
	AgentReconnectBackoff  int               `json:"agent_reconnect_backoff"`
	AWS                    *AWSClusterConfig `union:"type,aws" json:"-"`
	GCP                    *GCPClusterConfig `union:"type,gcp" json:"-"`
	MaxIdleAgentPeriod     model.Duration    `json:"max_idle_agent_period"`
	MaxAgentStartingPeriod model.Duration    `json:"max_agent_starting_period"`
	MinInstances           int               `json:"min_instances"`
	MaxInstances           int               `json:"max_instances"`
}

// DefaultConfig returns the default configuration of the provisioner.
func DefaultConfig() *Config {
	return &Config{
		AgentDockerRuntime:     "runc",
		AgentDockerNetwork:     "default",
		AgentDockerImage:       fmt.Sprintf("determinedai/determined-agent:%s", version.Version),
		AgentFluentImage:       aproto.FluentImage,
		MaxIdleAgentPeriod:     model.Duration(20 * time.Minute),
		MaxAgentStartingPeriod: model.Duration(20 * time.Minute),
		MinInstances:           0,
		MaxInstances:           5,
		AgentReconnectAttempts: aproto.AgentReconnectAttempts,
		AgentReconnectBackoff:  aproto.AgentReconnectBackoffValue,
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *Config) UnmarshalJSON(data []byte) error {
	*c = *DefaultConfig()
	if err := union.Unmarshal(data, c); err != nil {
		return err
	}
	type DefaultParser *Config
	return json.Unmarshal(data, DefaultParser(c))
}

// MarshalJSON implements the json.Marshaler interface.
func (c Config) MarshalJSON() ([]byte, error) {
	return union.Marshal(c)
}

// Validate implements the check.Validatable interface.
func (c Config) Validate() []error {
	var errs []error
	masterURL, err := url.Parse(c.MasterURL)
	var masterURLErr error
	switch {
	case err != nil:
		errs = append(errs, errors.Wrap(err, "cannot parse master url"))
	case len(c.MasterURL) != 0:
		errs = append(errs, check.True(len(masterURL.Path) == 0,
			"invalid master url (expecting scheme://host:port)"))
		errs = append(errs, check.In(masterURL.Scheme, []string{"http", "https"},
			"master url scheme must be within [http, https]"))
	}
	errs = append(errs, []error{
		masterURLErr,
		check.NotEmpty(c.AgentDockerImage, "must configure an agent docker image"),
		check.False(c.AWS != nil && c.GCP != nil, "must configure only one cluster"),
		check.False(c.AWS == nil && c.GCP == nil, "must configure aws or gcp cluster"),
		check.GreaterThan(
			int64(c.MaxIdleAgentPeriod), int64(0), "max idle agent period must be greater than 0"),
		check.GreaterThan(
			int64(c.MaxAgentStartingPeriod), int64(0), "max agent starting period must be greater than 0"),
		check.GreaterThanOrEqualTo(int64(c.MinInstances), int64(0),
			"min instance must be greater than or equal to 0"),
		check.GreaterThan(int64(c.MaxInstances), int64(0), "max instance must be greater than 0"),
		check.GreaterThanOrEqualTo(int64(c.MaxInstances), int64(c.MinInstances),
			"max instance must be greater than or equal to min instance"),
	}...)
	return errs
}

func (c Config) mustParseMasterURL() url.URL {
	masterURL, err := url.Parse(c.MasterURL)
	if err != nil {
		panic("invalid master url")
	}
	return *masterURL
}

// InitMasterAddress init master address.
func (c *Config) InitMasterAddress() error {
	masterURL := c.mustParseMasterURL()
	scheme, host, port := masterURL.Scheme, masterURL.Hostname(), masterURL.Port()

	if scheme == "" {
		scheme = "http"
	}

	var err error
	switch {
	case (host == "internal-ip" || host == "") && metadata.OnGCE():
		host, err = metadata.InternalIP()
	case host == "external-ip" && metadata.OnGCE():
		host, err = metadata.ExternalIP()
	case (host == "local-ipv4" || host == "") && onEC2():
		host, err = getEC2Metadata("local-ipv4")
	case host == "public-ipv4" && onEC2():
		host, err = getEC2Metadata("public-ipv4")
	case host == "local-hostname" && onEC2():
		host, err = getEC2Metadata("local-hostname")
	case host == "public-hostname" && onEC2():
		host, err = getEC2Metadata("public-hostname")
	}
	if err != nil {
		return errors.Wrap(err, "cannot get metadata")
	}

	if len(port) == 0 {
		port = defaultMasterPort
	}
	c.MasterURL = (&url.URL{Scheme: scheme, Host: fmt.Sprintf("%s:%s", host, port)}).String()
	return nil
}
