package internal

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (m *Master) getPrometheusTargets(c echo.Context) (interface{}, error) {
	resourceManager := sproto.GetCurrentRM(m.system)
	resp := m.system.Ask(resourceManager, &apiv1.GetAgentsRequest{})

	if resp.Error() != nil {
		return nil, resp.Error()
	}

	getAgentsResponse := resp.Get().(*apiv1.GetAgentsResponse)

	var agentTargetsConfig []prom.TargetSDConfig
	for _, agentSummary := range getAgentsResponse.Agents {
		agentTargetConfig := prom.TargetSDConfig{
			Labels: map[string]string{
				prom.DetAgentIDLabel:      agentSummary.Id,
				prom.DetResourcePoolLabel: strings.Join(agentSummary.ResourcePools, ","),
			},
		}
		for _, address := range agentSummary.Addresses {
			agentTargetConfig.Targets = []string{
				address + prom.DcgmPort,
				address + prom.CAdvisorPort,
			}
		}

		agentTargetsConfig = append(agentTargetsConfig, agentTargetConfig)
	}

	return agentTargetsConfig, nil
}
