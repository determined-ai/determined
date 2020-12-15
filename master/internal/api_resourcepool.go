package internal

import (
	"context"
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
		err = a.actorRequest(sproto.AgentRMAddr.String(), req, &resp)
	case sproto.UseK8sRM(a.m.system):
		err = a.actorRequest(sproto.K8sRMAddr.String(), req, &resp)

	default:
		err = status.Error(codes.NotFound, "cannot find appropriate resource manager")
	}
	if err != nil {
		return nil, err
	}

	return resp, a.paginate(&resp.Pagination, &resp.ResourcePools, req.Offset, req.Limit)
}


