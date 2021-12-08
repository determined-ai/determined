package internal

import (
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (m *Master) getPrometheusTargets(c echo.Context) (interface{}, error) {
	var resp actor.Response

	switch {
	case sproto.UseAgentRM(m.system):
		resp = m.system.AskAt(sproto.AgentsAddr, &apiv1.GetAgentsRequest{})
	case sproto.UseK8sRM(m.system):
		resp = m.system.AskAt(sproto.PodsAddr, &apiv1.GetAgentsRequest{})
	default:
		return nil, status.Error(codes.NotFound, "agents or pods actor not found")
	}

	if resp.Error() != nil {
		return nil, resp.Error()
	}

	getAgentsResponse := resp.Get().(*apiv1.GetAgentsResponse)

	var agentTargetsConfig []prom.TargetSDConfig
	for _, agentSummary := range getAgentsResponse.Agents {
		agentTargetConfig := prom.TargetSDConfig{
			Labels: map[string]string{
				prom.DetAgentIDLabel:      agentSummary.Id,
				prom.DetResourcePoolLabel: agentSummary.ResourcePool,
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
