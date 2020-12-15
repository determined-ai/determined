package internal

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetResourcePools(
	_ context.Context, req *apiv1.GetResourcePoolsRequest,
) (resp *apiv1.GetResourcePoolsResponse, err error) {
	switch {
	case sproto.UseAgentRM(a.m.system):
		resourcePoolSummaries := a.m.system.AskAt(sproto.AgentRMAddr, req).Get()
		if resourcePoolSummaries == nil {
			// TODO: Handle this
		}
	case sproto.UseK8sRM(a.m.system):
		resourcePoolSummaries := a.m.system.AskAt(sproto.K8sRMAddr, req).Get()
		if resourcePoolSummaries == nil {
			// TODO: Handle this
		}
	default:
		err = status.Error(codes.NotFound, "cannot find appropriate resource manager")
	}

	//
	//if err != nil {
	//	return nil, err
	//}
	//a.filter(&resp.Agents, func(i int) bool {
	//	v := resp.Agents[i]
	//	return req.Label == "" || v.Label == req.Label
	//})
	//a.sort(resp.Agents, req.OrderBy, req.SortBy, apiv1.GetAgentsRequest_SORT_BY_ID)
	//return resp, a.paginate(&resp.Pagination, &resp.Agents, req.Offset, req.Limit)
}



func (a *apiServer) GetResourcePool(
	_ context.Context, req *apiv1.GetResourcePoolRequest) (resp *apiv1.GetResourcePoolResponse, err error) {

	switch {
	case sproto.UseAgentRM(a.m.system):
		resourcePoolSummary := a.m.system.AskAt(sproto.AgentRMAddr, req).Get()
		if resourcePoolSummary == nil {
			// TODO: Handle this
		}
	case sproto.UseK8sRM(a.m.system):
		resourcePoolSummary := a.m.system.AskAt(sproto.K8sRMAddr, req).Get()
		if resourcePoolSummary == nil {
			// TODO: Handle this
		}
	default:
		err = status.Error(codes.NotFound, "cannot find appropriate resource manager")
	}


}
