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
		err = a.actorRequest(sproto.AgentsAddr.String(), req, &resp)
	case sproto.UseK8sRM(a.m.system):
		// TODO: What should we do in k8s land?
		panic("Cannot call resource pools with k8s")
		//err = a.actorRequest(sproto.PodsAddr.String(), req, &resp)
	default:
		err = status.Error(codes.NotFound, "cannot find resource actor")
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

	// Send the request to the resourceManager for the general information
	// 		Return err if the resource pool doesn't exist
	// 		Otherwise return a resource pool config
	// Transform config into correct shape for API response

	// Send a get to all agents
	// Group slot information by resource pool

	err = a.actorRequest(fmt.Sprintf("/agents/%s", req.AgentId), req, &resp)
	return resp, err
}
