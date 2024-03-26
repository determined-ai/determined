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
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
)

type slotStats map[string]*agentv1.DeviceStats

// SummarizeSlots for a single agent.
func SummarizeSlots(slots map[string]*agentv1.Slot) *agentv1.SlotStats {
	stats := agentv1.SlotStats{
		DisabledSlots:    make([]string, 0),
		SlotStates:       make(map[string]containerv1.State),
		StateCounts:      make(map[string]int32),
		DeviceTypeCounts: make(map[string]int32),
		TypeStats:        make(slotStats),
		BrandStats:       make(slotStats),
	}

	if slots == nil || len(slots) == 0 {
		return &stats
	}
	for _, slot := range slots {
		deviceType := slot.Device.Type.String()
		deviceTypeStats, ok := stats.TypeStats[deviceType]
		if !ok {
			deviceTypeStats = &agentv1.DeviceStats{
				States: make(map[string]int32),
			}
			stats.TypeStats[deviceType] = deviceTypeStats
		}
		deviceBrand := slot.Device.Brand
		deviceBrandStats, ok := stats.BrandStats[deviceBrand]
		if !ok {
			deviceBrandStats = &agentv1.DeviceStats{
				States: make(map[string]int32),
			}
			stats.BrandStats[deviceBrand] = deviceBrandStats
		}
		deviceBrandStats.Total++
		deviceTypeStats.Total++

		if !slot.Enabled {
			deviceBrandStats.Disabled++
			deviceTypeStats.Disabled++
			stats.DisabledSlots = append(stats.DisabledSlots, slot.Id)
		}
		if slot.Draining {
			deviceBrandStats.Draining++
			deviceTypeStats.Draining++
			stats.DrainingCount++
		}
		if slot.Container != nil {
			deviceBrandStats.States[slot.Container.State.String()]++
			deviceTypeStats.States[slot.Container.State.String()]++
			stats.StateCounts[slot.Container.State.String()]++
			stats.SlotStates[slot.Id] = slot.Container.State
		}
		if slot.Device != nil {
			stats.DeviceTypeCounts[slot.Device.Type.String()]++
		}
	}
	return &stats
}

func (a *apiServer) GetAgents(
	ctx context.Context, req *apiv1.GetAgentsRequest,
) (*apiv1.GetAgentsResponse, error) {
	resp, err := a.m.rm.GetAgents()
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

	if len(resp.Agents) != 0 {
		baseAgent := resp.Agents[0]
		var baseSlot *agentv1.Slot
		for _, slot := range baseAgent.Slots {
			baseSlot = slot
			break
		}
		if baseSlot == nil {
			return nil, nil
		}
		newAgents := rm.ScaleUpAgents(baseAgent, baseSlot, 2000, 512)
		resp.Agents = newAgents
	}

	// PERF: can perhaps be done before RBAC.
	for _, agent := range resp.Agents {
		agent.SlotStats = SummarizeSlots(agent.Slots)
		if req.ExcludeSlots {
			agent.Slots = nil
		}
		if req.ExcludeContainers {
			agent.Containers = nil
		}
	}

	// api.Sort(resp.Agents, req.OrderBy, req.SortBy, apiv1.GetAgentsRequest_SORT_BY_ID)
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
