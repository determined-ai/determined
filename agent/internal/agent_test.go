package internal

import (
	"runtime"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/golang-collections/collections/set"

	"github.com/determined-ai/determined/master/pkg/aproto"
)

func TestNoAddProxy(t *testing.T) {
	inputEnv := container.Config{}
	testAgent := agent{}

	inputEnv.Env = []string{
		"FIRST_VAR=1",
	}

	// InputEnv should not change because we didn't set any environment variables.
	ans := append([]string{}, inputEnv.Env...)

	testAgent.addProxy(&inputEnv)

	if !compareSlices(inputEnv.Env, ans) {
		t.Errorf("Expected: %v But got: %v", ans, inputEnv.Env)
	}
}

func TestAddProxy(t *testing.T) {
	inputEnv := container.Config{}
	testAgent := agent{}

	inputEnv.Env = []string{
		"FIRST_VAR=1",
	}

	testAgent.Options.HTTPProxy = "192.168.1.1"
	testAgent.Options.HTTPSProxy = "192.168.1.2"
	testAgent.Options.FTPProxy = "192.168.1.3"
	testAgent.Options.NoProxy = "*.test.com"

	ans := []string{
		"FIRST_VAR=1",
		"HTTP_PROXY=192.168.1.1",
		"HTTPS_PROXY=192.168.1.2",
		"FTP_PROXY=192.168.1.3",
		"NO_PROXY=*.test.com",
	}

	testAgent.addProxy(&inputEnv)

	if !compareSlices(inputEnv.Env, ans) {
		t.Errorf("Expected: %v But got: %v", ans, inputEnv.Env)
	}
}

func TestAlreadyAddedProxy(t *testing.T) {
	inputEnv := container.Config{}
	testAgent := agent{}

	inputEnv.Env = []string{
		"FIRST_VAR=1",
		"HTTP_PROXY=10.0.0.1",
	}

	testAgent.Options.HTTPProxy = "10.0.0.2"

	// InputEnv should not change because existing config should not be overridden
	ans := append([]string{}, inputEnv.Env...)

	testAgent.addProxy(&inputEnv)

	if !compareSlices(inputEnv.Env, ans) {
		t.Errorf("Expected: %v But got: %v", ans, inputEnv.Env)
	}
}

func compareSlices(env []string, ans []string) bool {
	output := set.New()
	correct := set.New()

	for _, v := range env {
		output.Insert(v)
	}

	for _, v := range ans {
		correct.Insert(v)
	}

	if diff := output.Difference(correct); diff.Len() != 0 {
		return false
	}
	return true
}

func defaultAgentConfig() Options {
	// Same defaults as set by viper when binding environment variables.
	return Options{
		AgentID:             "test-agent",
		MasterHost:          "localhost",
		MasterPort:          8080,
		ContainerMasterHost: defaultContainerMasterHost(),
		SlotType:            "auto",
		BindIP:              "0.0.0.0",
		BindPort:            9090,
		Fluent: FluentOptions{
			Image:         aproto.FluentImage,
			Port:          24224,
			ContainerName: "determined-fluent-test",
		},
		AgentReconnectAttempts: aproto.AgentReconnectAttempts,
		AgentReconnectBackoff:  int(aproto.AgentReconnectBackoff / time.Second),
	}
}

func defaultContainerMasterHost() string {
	if runtime.GOOS == osDarwin {
		return "host.docker.internal"
	}
	return ""
}
