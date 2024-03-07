package internal

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (m *Master) getPrometheusTargets(c echo.Context) (interface{}, error) {
	resp, err := m.rm.GetAgents(&apiv1.GetAgentsRequest{})
	if err != nil {
		return nil, fmt.Errorf("gather agent statuses: %w", err)
	}

	var agentTargetsConfig []prom.TargetSDConfig
	for _, agentSummary := range resp.Agents {
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
