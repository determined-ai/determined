package agentsetup

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

// SecureScheme is the scheme used for the master's HTTPS port.
const SecureScheme = "https"

// AgentSetupScriptConfig is the configuration for the agent setup script.
type AgentSetupScriptConfig struct {
	MasterHost                   string
	MasterPort                   string
	MasterCertName               string
	StartupScriptBase64          string
	ContainerStartupScriptBase64 string
	MasterCertBase64             string
	ConfigFileBase64             string
	SlotType                     device.Type
	AgentDockerRuntime           string
	AgentNetwork                 string
	AgentDockerImage             string
	// deprecated, no longer in use.
	AgentFluentImage       string
	AgentID                string
	ResourcePool           string
	LogOptions             string
	AgentReconnectAttempts int
	AgentReconnectBackoff  int
}

// Provider is the interface for interacting with the underlying instance provider.
type Provider interface {
	InstanceType() model.InstanceType
	SlotsPerInstance() int
	List() ([]*model.Instance, error)
	Launch(instanceNum int) error
	Terminate(instanceIDs []string)
}

// MustMakeAgentSetupScript generates the agent setup script.
func MustMakeAgentSetupScript(config AgentSetupScriptConfig) []byte {
	templateStr := string(etc.MustStaticFile(etc.AgentSetupScriptTemplateResource))
	tpl := template.Must(template.New("AgentSetupScript").Parse(templateStr))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, config); err != nil {
		panic(fmt.Sprint("cannot generate agent setup script", err))
	}
	return buf.Bytes()
}
