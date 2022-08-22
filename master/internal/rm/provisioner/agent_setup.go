package provisioner

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
)

type agentSetupScriptConfig struct {
	MasterHost                   string
	MasterPort                   string
	MasterCertName               string
	StartupScriptBase64          string
	ContainerStartupScriptBase64 string
	MasterCertBase64             string
	SlotType                     device.Type
	AgentDockerRuntime           string
	AgentNetwork                 string
	AgentDockerImage             string
	AgentFluentImage             string
	AgentID                      string
	ResourcePool                 string
	LogOptions                   string
	AgentReconnectAttempts       int
	AgentReconnectBackoff        int
}

func mustMakeAgentSetupScript(config agentSetupScriptConfig) []byte {
	templateStr := string(etc.MustStaticFile(etc.AgentSetupScriptTemplateResource))
	tpl := template.Must(template.New("AgentSetupScript").Parse(templateStr))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, config); err != nil {
		panic(fmt.Sprint("cannot generate agent setup script", err))
	}
	return buf.Bytes()
}
