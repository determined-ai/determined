//go:build integration

package testutils

import (
	"runtime"
	"strconv"

	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/test/testutils"
)

const (
	defaultMasterPort = 5152
)

// DefaultAgentConfig returns a default agent config, for tests.
func DefaultAgentConfig(offset int) options.Options {
	// Same defaults as set by viper when binding environment variables.
	return options.Options{
		AgentID:             "test-agent" + strconv.Itoa(offset),
		MasterHost:          "localhost",
		MasterPort:          defaultMasterPort + offset,
		ContainerMasterHost: DefaultContainerMasterHost(),
		ContainerMasterPort: defaultMasterPort + offset,
		SlotType:            "auto",
		BindIP:              "0.0.0.0",
		BindPort:            9090 + offset,
	}
}

// DefaultMasterSetAgentConfig returns default MasterSetAgentOptions (logs go to master).
func DefaultMasterSetAgentConfig() aproto.MasterSetAgentOptions {
	return aproto.MasterSetAgentOptions{
		MasterInfo: aproto.MasterInfo{},
		LoggingOptions: model.LoggingConfig{
			DefaultLoggingConfig: &model.DefaultLoggingConfig{},
		},
	}
}

// ElasticMasterSetAgentConfig returns MasterSetAgentOptions for an elastic-configured cluster.
func ElasticMasterSetAgentConfig() aproto.MasterSetAgentOptions {
	return aproto.MasterSetAgentOptions{
		MasterInfo:     aproto.MasterInfo{},
		LoggingOptions: testutils.DefaultElasticConfig(),
	}
}

// DefaultContainerMasterHost returns the default container master host, depending on the system.
func DefaultContainerMasterHost() string {
	if runtime.GOOS == "darwin" {
		return "host.docker.internal"
	}
	return ""
}
