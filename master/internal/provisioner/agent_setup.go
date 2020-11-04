package provisioner

import (
	"bytes"
	"text/template"

	"github.com/determined-ai/determined/master/pkg/etc"
)

type agentSetupScriptConfig struct {
	MasterHost                   string
	MasterPort                   string
	StartupScriptBase64          string
	ContainerStartupScriptBase64 string
	MasterCertBase64             string
	AgentUseGPUs                 bool
	AgentDockerRuntime           string
	AgentNetwork                 string
	AgentDockerImage             string
	AgentFluentImage             string
	AgentID                      string
	ResourcePool                 string
	LogOptions                   string
}

func mustMakeAgentSetupScript(config agentSetupScriptConfig) []byte {
	templateStr := string(etc.MustStaticFile(etc.AgentSetupScriptTemplateResource))
	tpl := template.Must(template.New("AgentSetupScript").Parse(templateStr))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, config); err != nil {
		panic("cannot generate agent setup script")
	}
	return buf.Bytes()
}
