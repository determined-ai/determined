package provisioner

import (
	"encoding/base64"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/etc"
)

func TestAgentSetupScript(t *testing.T) {
	err := etc.SetRootPath("../../static/srv/")
	assert.NilError(t, err)

	encodedScript := base64.StdEncoding.EncodeToString([]byte("sleep 5\n echo \"hello world\""))
	encodedContainerScript := base64.StdEncoding.EncodeToString([]byte("sleep"))
	conf := agentSetupScriptConfig{
		MasterHost:                   "test.master",
		MasterPort:                   "8080",
		StartupScriptBase64:          encodedScript,
		ContainerStartupScriptBase64: encodedContainerScript,
		AgentDockerImage:             "test_docker_image",
		AgentDockerRuntime:           "runc",
		AgentNetwork:                 "default",
		AgentID:                      "test.id",
	}

	// nolint
	expected := `#!/bin/bash

mkdir -p /usr/local/determined
echo c2xlZXAgNQogZWNobyAiaGVsbG8gd29ybGQi | base64 --decode > /usr/local/determined/startup_script
echo "#### PRINTING STARTUP SCRIPT START ####"
cat /usr/local/determined/startup_script
echo "#### PRINTING STARTUP SCRIPT END ####"
chmod +x /usr/local/determined/startup_script
/usr/local/determined/startup_script

echo c2xlZXA= | base64 --decode > /usr/local/determined/container_startup_script
echo "#### PRINTING CONTAINER STARTUP SCRIPT START ####"
cat /usr/local/determined/container_startup_script
echo "#### PRINTING CONTAINER STARTUP SCRIPT END ####"

docker run --init --name determined-agent  --restart always --network default --runtime=runc --gpus all \
    -e DET_AGENT_ID="test.id" \
    -e DET_MASTER_HOST="test.master" \
    -e DET_MASTER_PORT="8080" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /usr/local/determined/container_startup_script:/usr/local/determined/container_startup_script \
    "test_docker_image"
`

	res := string(mustMakeAgentSetupScript(conf))
	assert.Equal(t, res, expected)
}
