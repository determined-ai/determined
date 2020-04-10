package provisioner

import (
	"bytes"
	"text/template"

	"github.com/determined-ai/determined/master/pkg/etc"
)

type agentSetupScriptConfig struct {
	MasterHost          string
	MasterPort          string
	StartupScriptBase64 string
	AgentDockerRuntime  string
	AgentNetwork        string
	AgentDockerImage    string
	AgentID             string
	LogOptions          string
}

func mustMakeAgentSetupScript(config agentSetupScriptConfig) []byte {
	tpl := template.Must(template.New("AgentSetupScript").
		Parse(string(etc.MustStaticFile(etc.AgentSetupScriptTemplateResource))))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, config); err != nil {
		panic("cannot generate agent setup script")
	}
	return buf.Bytes()
}
