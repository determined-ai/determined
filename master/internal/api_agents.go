package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/cluster"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetAgents(
	ctx context.Context, req *apiv1.GetAgentsRequest,
) (*apiv1.GetAgentsResponse, error) {
	resp, err := a.m.rm.GetAgents(req)
	if err != nil {
		return nil, err
	}

	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	permErr, err := cluster.AuthZProvider.Get().CanGetSensitiveAgentInfo(ctx, user)
	switch {
	case err != nil:
		return nil, err
	case permErr != nil:
		for _, agent := range resp.Agents {
			if err := authz.ObfuscateAgent(agent); err != nil {
				return nil, err
			}
		}
	}

	api.Sort(resp.Agents, req.OrderBy, req.SortBy, apiv1.GetAgentsRequest_SORT_BY_ID)
	return resp, api.Paginate(&resp.Pagination, &resp.Agents, req.Offset, req.Limit)
}

func (a *apiServer) GetAgent(
	ctx context.Context, req *apiv1.GetAgentRequest,
) (*apiv1.GetAgentResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := a.m.rm.GetAgent(req)
	if err != nil {
		return nil, err
	}

	permErr, err := cluster.AuthZProvider.Get().CanGetSensitiveAgentInfo(ctx, user)
	switch {
	case err != nil:
		return nil, err
	case permErr != nil:
		if err := authz.ObfuscateAgent(resp.Agent); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (a *apiServer) GetSlots(
	ctx context.Context, req *apiv1.GetSlotsRequest,
) (*apiv1.GetSlotsResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := a.m.rm.GetSlots(req)
	if err != nil {
		return nil, err
	}

	permErr, err := cluster.AuthZProvider.Get().CanGetSensitiveAgentInfo(ctx, user)
	switch {
	case err != nil:
		return nil, err
	case permErr != nil:
		for _, slot := range resp.Slots {
			if err := authz.ObfuscateSlot(slot); err != nil {
				return nil, err
			}
		}
	}

	return resp, nil
}

func (a *apiServer) GetSlot(
	ctx context.Context, req *apiv1.GetSlotRequest,
) (*apiv1.GetSlotResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := a.m.rm.GetSlot(req)
	if err != nil {
		return nil, err
	}

	permErr, err := cluster.AuthZProvider.Get().CanGetSensitiveAgentInfo(ctx, user)
	switch {
	case err != nil:
		return nil, err
	case permErr != nil:
		if err := authz.ObfuscateSlot(resp.Slot); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

func (a *apiServer) canUpdateAgents(ctx context.Context) error {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}
	permErr, err := cluster.AuthZProvider.Get().CanUpdateAgents(ctx, user)
	switch {
	case err != nil:
		return err
	case permErr != nil:
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
	return a.m.rm.EnableAgent(req)
}

func (a *apiServer) DisableAgent(
	ctx context.Context, req *apiv1.DisableAgentRequest,
) (resp *apiv1.DisableAgentResponse, err error) {
	if err := a.canUpdateAgents(ctx); err != nil {
		return nil, err
	}
	return a.m.rm.DisableAgent(req)
}

func (a *apiServer) EnableSlot(
	ctx context.Context, req *apiv1.EnableSlotRequest,
) (resp *apiv1.EnableSlotResponse, err error) {
	if err := a.canUpdateAgents(ctx); err != nil {
		return nil, err
	}

	resp, err = a.m.rm.EnableSlot(req)
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

	resp, err = a.m.rm.DisableSlot(req)
	switch {
	case errors.Is(err, rmerrors.ErrNotSupported):
		return resp, status.Error(codes.Unimplemented, err.Error())
	case err != nil:
		return nil, err
	default:
		return resp, nil
	}
}
