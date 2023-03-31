package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/cluster"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetAgents(
	_ context.Context, req *apiv1.GetAgentsRequest,
) (*apiv1.GetAgentsResponse, error) {
	resp, err := a.m.rm.GetAgents(a.m.system, req)
	if err != nil {
		return nil, err
	}
	a.sort(resp.Agents, req.OrderBy, req.SortBy, apiv1.GetAgentsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Agents, req.Offset, req.Limit)
}

func agentAddr(agentID string) actor.Address {
	return sproto.AgentsAddr.Child(agentID)
}

func slotAddr(agentID, slotID string) actor.Address {
	return actorrm.SlotAddr(agentID, slotID)
}

func (a *apiServer) GetAgent(
	_ context.Context, req *apiv1.GetAgentRequest,
) (resp *apiv1.GetAgentResponse, err error) {
	return resp, a.ask(agentAddr(req.AgentId), req, &resp)
}

func (a *apiServer) GetSlots(
	_ context.Context, req *apiv1.GetSlotsRequest,
) (resp *apiv1.GetSlotsResponse, err error) {
	return resp, a.ask(agentAddr(req.AgentId), req, &resp)
}

func (a *apiServer) GetSlot(
	_ context.Context, req *apiv1.GetSlotRequest,
) (resp *apiv1.GetSlotResponse, err error) {
	return resp, a.ask(slotAddr(req.AgentId, req.SlotId), req, &resp)
}

func (a *apiServer) canUpdateAgents(ctx context.Context) error {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}
	permErr, err := cluster.AuthZProvider.Get().CanUpdateAgents(ctx, user)
	if err != nil {
		return err
	}
	if permErr != nil {
		return status.Error(codes.PermissionDenied, permErr.Error())
	}
	return nil
}

func (a *apiServer) EnableAgent(
	ctx context.Context, req *apiv1.EnableAgentRequest,
) (resp *apiv1.EnableAgentResponse, err error) {
	if err := a.canUpdateAgents(ctx); err != nil {
		return nil, err
	}

	return resp, a.ask(agentAddr(req.AgentId), req, &resp)
}

func (a *apiServer) DisableAgent(
	ctx context.Context, req *apiv1.DisableAgentRequest,
) (resp *apiv1.DisableAgentResponse, err error) {
	if err := a.canUpdateAgents(ctx); err != nil {
		return nil, err
	}
	return resp, a.ask(agentAddr(req.AgentId), req, &resp)
}

func (a *apiServer) EnableSlot(
	ctx context.Context, req *apiv1.EnableSlotRequest,
) (resp *apiv1.EnableSlotResponse, err error) {
	if err := a.canUpdateAgents(ctx); err != nil {
		return nil, err
	}
	resp, err = a.m.rm.EnableSlot(a.m.system, req)
	switch {
	case errors.Is(err, rmerrors.ErrNotSupported):
		return resp, status.Error(codes.Unimplemented, err.Error())
	case err != nil:
		return nil, err
	default:
		return resp, nil
	}
}

func (a *apiServer) DisableSlot(
	ctx context.Context, req *apiv1.DisableSlotRequest,
) (resp *apiv1.DisableSlotResponse, err error) {
	if err := a.canUpdateAgents(ctx); err != nil {
		return nil, err
	}
	resp, err = a.m.rm.DisableSlot(a.m.system, req)
	switch {
	case errors.Is(err, rmerrors.ErrNotSupported):
		return resp, status.Error(codes.Unimplemented, err.Error())
	case err != nil:
		return nil, err
	default:
		return resp, nil
	}
}
