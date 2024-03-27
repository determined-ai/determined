package rm

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

// NodesAPIServer implements node management gRPC endpoints.
type NodesAPIServer struct {
	rm ResourceManager
}

// NewNodesAPIServer creates a new NodesAPIServer.
func NewNodesAPIServer(rm ResourceManager) NodesAPIServer {
	return NodesAPIServer{rm: rm}
}

// GetAgents returns all nodes know about by the master's resource manager.
func (a *NodesAPIServer) GetAgents(
	ctx context.Context, req *apiv1.GetAgentsRequest,
) (*apiv1.GetAgentsResponse, error) {
	resp, err := a.rm.GetAgents()
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

// GetAgent returns information about a particular node known to the master.
func (a *NodesAPIServer) GetAgent(
	ctx context.Context, req *apiv1.GetAgentRequest,
) (*apiv1.GetAgentResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := a.rm.GetAgent(req)
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

// GetSlots returns all slots (accelerators) attached to a particular node known to the master.
func (a *NodesAPIServer) GetSlots(
	ctx context.Context, req *apiv1.GetSlotsRequest,
) (*apiv1.GetSlotsResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := a.rm.GetSlots(req)
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

// GetSlot returns Information about a particular slot (accelerator) on a node known to the master.
func (a *NodesAPIServer) GetSlot(
	ctx context.Context, req *apiv1.GetSlotRequest,
) (*apiv1.GetSlotResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := a.rm.GetSlot(req)
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

func (a *NodesAPIServer) canUpdateAgents(ctx context.Context) error {
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

// EnableAgent enables a node, allowing jobs to be scheduled on it.
func (a *NodesAPIServer) EnableAgent(
	ctx context.Context, req *apiv1.EnableAgentRequest,
) (resp *apiv1.EnableAgentResponse, err error) {
	if err := a.canUpdateAgents(ctx); err != nil {
		return nil, err
	}
	return a.rm.EnableAgent(req)
}

// DisableAgent disables a node, forbidding jobs from being scheduled on it. If drain is true then existing jobs
// scheduled on the node will be allowed to finish, else they are terminated immediately.
func (a *NodesAPIServer) DisableAgent(
	ctx context.Context, req *apiv1.DisableAgentRequest,
) (resp *apiv1.DisableAgentResponse, err error) {
	if err := a.canUpdateAgents(ctx); err != nil {
		return nil, err
	}
	return a.rm.DisableAgent(req)
}

// EnableSlot enables a slot, allowing job to be scheduled on it.
func (a *NodesAPIServer) EnableSlot(
	ctx context.Context, req *apiv1.EnableSlotRequest,
) (resp *apiv1.EnableSlotResponse, err error) {
	if err := a.canUpdateAgents(ctx); err != nil {
		return nil, err
	}

	resp, err = a.rm.EnableSlot(req)
	switch {
	case errors.Is(err, rmerrors.ErrNotSupported):
		return resp, status.Error(codes.Unimplemented, err.Error())
	case err != nil:
		return nil, err
	default:
		return resp, nil
	}
}

// DisableSlot disables a slot, forbidding jobs from being scheduled on it. If drain is true then existing jobs
// scheduled on the slot will be allowed to finish, else they are terminated immediately.
func (a *NodesAPIServer) DisableSlot(
	ctx context.Context, req *apiv1.DisableSlotRequest,
) (resp *apiv1.DisableSlotResponse, err error) {
	if err := a.canUpdateAgents(ctx); err != nil {
		return nil, err
	}

	resp, err = a.rm.DisableSlot(req)
	switch {
	case errors.Is(err, rmerrors.ErrNotSupported):
		return resp, status.Error(codes.Unimplemented, err.Error())
	case err != nil:
		return nil, err
	default:
		return resp, nil
	}
}
