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
	encodedMasterCert := base64.StdEncoding.EncodeToString([]byte("==== cert ===="))
	conf := agentSetupScriptConfig{
		MasterHost:                   "test.master",
		MasterPort:                   "8080",
		StartupScriptBase64:          encodedScript,
		ContainerStartupScriptBase64: encodedContainerScript,
		MasterCertBase64:             encodedMasterCert,
		AgentUseGPUs:                 true,
		AgentDockerImage:             "test_docker_image",
		AgentDockerRuntime:           "runc",
		AgentNetwork:                 "default",
		AgentID:                      "test.id",
		ResourcePool:                 "test-pool",
	}

	// nolint
	expected := `#!/bin/bash

docker_args=()

mkdir -p /usr/local/determined
echo c2xlZXAgNQogZWNobyAiaGVsbG8gd29ybGQi | base64 --decode > /usr/local/determined/startup_script
echo "#### PRINTING STARTUP SCRIPT START ####"
cat /usr/local/determined/startup_script
echo "#### PRINTING STARTUP SCRIPT END ####"
chmod +x /usr/local/determined/startup_script
/usr/local/determined/startup_script

use_gpus=true
if $use_gpus
then
    echo "#### Starting agent with GPUs"
    docker_args+=(--gpus all)
else
    echo "#### Starting agent with only CPUs"
fi

cert_b64=PT09PSBjZXJ0ID09PT0=
if [ -n "$cert_b64" ]; then
    echo "$cert_b64" | base64 --decode > /usr/local/determined/master.crt
    echo "#### PRINTING MASTER CERT START ####"
    cat /usr/local/determined/master.crt
    echo "#### PRINTING MASTER CERT END ####"
    docker_args+=(-v /usr/local/determined/master.crt:/usr/local/determined/master.crt)
    docker_args+=(-e DET_SECURITY_TLS_ENABLED=true)
    docker_args+=(-e DET_SECURITY_TLS_MASTER_CERT=/usr/local/determined/master.crt)
fi

echo c2xlZXA= | base64 --decode > /usr/local/determined/container_startup_script
echo "#### PRINTING CONTAINER STARTUP SCRIPT START ####"
cat /usr/local/determined/container_startup_script
echo "#### PRINTING CONTAINER STARTUP SCRIPT END ####"

docker run --init --name determined-agent  \
    --restart always \
    --network default \
    --runtime=runc \
    -e DET_AGENT_ID="test.id" \
    -e DET_MASTER_HOST="test.master" \
    -e DET_MASTER_PORT="8080" \
    -e DET_RESOURCE_POOL="test-pool" \
    -e DET_SLOT_TYPE=gpu \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /usr/local/determined/container_startup_script:/usr/local/determined/container_startup_script \
    "${docker_args[@]}" \
    "test_docker_image"
`

	res := string(mustMakeAgentSetupScript(conf))
	assert.Equal(t, res, expected)
}
