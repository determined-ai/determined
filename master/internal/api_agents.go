package internal

import (
	"context"

	proto "github.com/determined-ai/determined/master/pkg/proto/apiv1"
)

func (a *apiServer) GetAgents(
	_ context.Context, req *proto.GetAgentsRequest,
) (resp *proto.GetAgentsResponse, err error) {
	err = a.actorRequest("/agents", req, &resp)
	return resp, err
}
