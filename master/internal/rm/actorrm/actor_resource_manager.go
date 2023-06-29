package actorrm

import (
	"fmt"
	"reflect"

	"github.com/determined-ai/determined/master/internal/rm/rmevents"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// ResourceManager shims a RM actor to the ResourceManager interface.
type ResourceManager struct {
	ref *actor.Ref
}

// Wrap wraps an RM actor as an explicit interface.
func Wrap(ref *actor.Ref) *ResourceManager {
	return &ResourceManager{ref: ref}
}

// GetResourcePoolRef is a default implementation to satisfy the interface, mostly for tests.
func (r *ResourceManager) GetResourcePoolRef(
	ctx actor.Messenger,
	name string,
) (*actor.Ref, error) {
	return r.ref, nil
}

// ResolveResourcePool is a default implementation to satisfy the interface, mostly for tests.
func (r *ResourceManager) ResolveResourcePool(
	ctx actor.Messenger,
	name string,
	workspaceID,
	slots int,
) (string, error) {
	return name, nil
}

// ValidateResources is a default implementation to satisfy the interface, mostly for tests.
func (r *ResourceManager) ValidateResources(
	ctx actor.Messenger,
	name string,
	slots int,
	command bool,
) error {
	return nil
}

// ValidateResourcePoolAvailability is a default implementation to satisfy the interface.
func (r *ResourceManager) ValidateResourcePoolAvailability(
	ctx actor.Messenger,
	name string,
	slots int) (
	[]command.LaunchWarning,
	error,
) {
	return nil, nil
}

// ValidateResourcePool is a default implementation to satisfy the interface, mostly for tests.
func (r *ResourceManager) ValidateResourcePool(ctx actor.Messenger, name string) error {
	return nil
}

// Ref gets the underlying RM actor, for backwards compatibility. This is deprecated.
func (r *ResourceManager) Ref() *actor.Ref {
	return r.ref
}

// GetAllocationSummary requests a summary of the given allocation.
func (r *ResourceManager) GetAllocationSummary(
	ctx actor.Messenger,
	msg sproto.GetAllocationSummary,
) (resp *sproto.AllocationSummary, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// GetAllocationSummaries requests a summary of all current allocations.
func (r *ResourceManager) GetAllocationSummaries(
	ctx actor.Messenger,
	msg sproto.GetAllocationSummaries,
) (resp map[model.AllocationID]sproto.AllocationSummary, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// SetAllocationName sets a name for a given allocation.
func (r *ResourceManager) SetAllocationName(
	ctx actor.Messenger,
	msg sproto.SetAllocationName,
) {
	r.Tell(ctx, msg)
}

// ValidateCommandResources validates a request for command resources.
func (r *ResourceManager) ValidateCommandResources(
	ctx actor.Messenger,
	msg sproto.ValidateCommandResourcesRequest,
) (resp sproto.ValidateCommandResourcesResponse, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// Allocate allocates some resources.
func (r *ResourceManager) Allocate(
	ctx actor.Messenger,
	msg sproto.AllocateRequest,
) (resp *sproto.AllocationSubscription, err error) {
	// We want to subscribe before allocating, but also free
	// the subscription for the caller if we fail.
	sub := rmevents.Subscribe(msg.AllocationID)
	defer func() {
		if err != nil {
			sub.Close()
		}
	}()

	err = r.Ask(ctx, msg, nil)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// Release releases some resources.
func (r *ResourceManager) Release(ctx actor.Messenger, msg sproto.ResourcesReleased) {
	r.Tell(ctx, msg)
}

// GetResourcePools requests information about the available resource pools.
func (r *ResourceManager) GetResourcePools(
	ctx actor.Messenger,
	msg *apiv1.GetResourcePoolsRequest,
) (resp *apiv1.GetResourcePoolsResponse, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// GetDefaultComputeResourcePool requests the default compute resource pool.
func (r *ResourceManager) GetDefaultComputeResourcePool(
	ctx actor.Messenger,
	msg sproto.GetDefaultComputeResourcePoolRequest,
) (resp sproto.GetDefaultComputeResourcePoolResponse, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// GetDefaultAuxResourcePool requests the default aux resource pool.
func (r *ResourceManager) GetDefaultAuxResourcePool(
	ctx actor.Messenger,
	msg sproto.GetDefaultAuxResourcePoolRequest,
) (resp sproto.GetDefaultAuxResourcePoolResponse, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// GetAgents gets the state of connected agents or reads similar information from the underlying RM.
func (r *ResourceManager) GetAgents(
	ctx actor.Messenger,
	msg *apiv1.GetAgentsRequest,
) (resp *apiv1.GetAgentsResponse, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// GetJobQ gets the state of the job queue.
func (r *ResourceManager) GetJobQ(
	ctx actor.Messenger,
	msg sproto.GetJobQ,
) (resp map[model.JobID]*sproto.RMJobInfo, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// GetJobQStats requests stats for a job queue.
func (r *ResourceManager) GetJobQStats(
	ctx actor.Messenger,
	msg sproto.GetJobQStats,
) (resp *jobv1.QueueStats, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// GetJobQueueStatsRequest requests other stats for a job queue.
func (r *ResourceManager) GetJobQueueStatsRequest(
	ctx actor.Messenger,
	msg *apiv1.GetJobQueueStatsRequest,
) (resp *apiv1.GetJobQueueStatsResponse, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// MoveJob moves a job ahead of or behind a peer.
func (r *ResourceManager) MoveJob(ctx actor.Messenger, msg sproto.MoveJob) error {
	return r.Ask(ctx, msg, nil)
}

// RecoverJobPosition recovers the position of a job relative to the rest of its priority lane.
func (r *ResourceManager) RecoverJobPosition(
	ctx actor.Messenger,
	msg sproto.RecoverJobPosition,
) {
	r.Tell(ctx, msg)
}

// SetGroupWeight sets the weight for a group.
func (r *ResourceManager) SetGroupWeight(
	ctx actor.Messenger,
	msg sproto.SetGroupWeight,
) error {
	return r.Ask(ctx, msg, nil)
}

// SetGroupPriority sets the group priority.
func (r *ResourceManager) SetGroupPriority(
	ctx actor.Messenger,
	msg sproto.SetGroupPriority,
) error {
	return r.Ask(ctx, msg, nil)
}

// SetGroupMaxSlots sets the max allocatable slots for a group.
func (r *ResourceManager) SetGroupMaxSlots(ctx actor.Messenger, msg sproto.SetGroupMaxSlots) {
	r.Tell(ctx, msg)
}

// DeleteJob requests we clean up our state related to a given job.
func (r *ResourceManager) DeleteJob(
	ctx actor.Messenger,
	msg sproto.DeleteJob,
) (resp sproto.DeleteJobResponse, err error) {
	return resp, r.Ask(ctx, msg, &resp)
}

// ExternalPreemptionPending requests we notify some allocation that it was preempted externally.
func (r *ResourceManager) ExternalPreemptionPending(
	ctx actor.Messenger,
	msg sproto.PendingPreemption,
) error {
	return r.Ask(ctx, msg, nil)
}

// NotifyContainerRunning receives a notification from the container to let
// the master know that the container is running.
func (r *ResourceManager) NotifyContainerRunning(
	ctx actor.Messenger,
	msg sproto.NotifyContainerRunning,
) error {
	// Actor Resource Manager does not implement a handler for the
	// NotifyContainerRunning message, as it is only used on HPC
	// (High Performance Computing).
	return errors.New(
		"the NotifyContainerRunning message is unsupported for ActorResourceManager")
}

// IsReattachEnabled is a default implementation (not Reattachable).
func (r *ResourceManager) IsReattachEnabled(ctx actor.Messenger) bool {
	return true
}

// IsReattachableOnlyAfterStarted is a default implementation (true).
func (r *ResourceManager) IsReattachableOnlyAfterStarted(ctx actor.Messenger) bool {
	return true
}

// IsReattachEnabledForRP is a default implementation for an RP being reattachable (false).
func (r *ResourceManager) IsReattachEnabledForRP(ctx actor.Messenger, rpName string) bool {
	return true
}

// Tell tells the underlying actor-based RM the req.
func (r *ResourceManager) Tell(ctx actor.Messenger, req interface{}) {
	ctx.Tell(r.ref, req)
}

// Ask asks the underlying actor-based RM the req, setting the response into v.
func (r *ResourceManager) Ask(ctx actor.Messenger, req interface{}, v interface{}) error {
	if reflect.ValueOf(v).IsValid() && !reflect.ValueOf(v).Elem().CanSet() {
		return fmt.Errorf("ask to %s has valid but unsettable resp %T", r.ref.Address(), v)
	}
	expectingResponse := reflect.ValueOf(v).IsValid() && reflect.ValueOf(v).Elem().CanSet()
	switch resp := ctx.Ask(r.ref, req); {
	case resp.Source() == nil:
		return fmt.Errorf("actor %s could not be found", r.ref.Address())
	case expectingResponse && resp.Empty(), expectingResponse && resp.Get() == nil:
		return fmt.Errorf("actor %s did not response", r.ref.Address())
	case resp.Error() != nil:
		return resp.Error()
	default:
		if expectingResponse {
			if reflect.ValueOf(v).Elem().Type() != reflect.ValueOf(resp.Get()).Type() {
				return fmt.Errorf(
					"%s returned unexpected resp (%T): %v",
					r.ref.Address(),
					resp,
					resp,
				)
			}
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(resp.Get()))
		}
		return nil
	}
}

// AskAt asks an actor and sets the response in v. It returns an error if the actor doesn't
// respond, respond with an error, or v isn't settable.
// TODO(Brad): Consolidate occurrences of this code.
func AskAt(sys *actor.System, addr actor.Address, req interface{}, v interface{}) error {
	if reflect.ValueOf(v).IsValid() && !reflect.ValueOf(v).Elem().CanSet() {
		return fmt.Errorf("ask at %s has valid but unsettable resp %T", addr, v)
	}
	expectingResponse := reflect.ValueOf(v).IsValid() && reflect.ValueOf(v).Elem().CanSet()
	switch resp := sys.AskAt(addr, req); {
	case resp.Source() == nil:
		return fmt.Errorf("actor %s could not be found", addr)
	case expectingResponse && resp.Empty(), expectingResponse && resp.Get() == nil:
		return fmt.Errorf("actor %s did not response", addr)
	case resp.Error() != nil:
		return resp.Error()
	default:
		if expectingResponse {
			if reflect.ValueOf(v).Elem().Type() != reflect.ValueOf(resp.Get()).Type() {
				return fmt.Errorf("%s returned unexpected resp (%T): %v", addr, resp, resp)
			}
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(resp.Get()))
		}
		return nil
	}
}

// TaskContainerDefaults returns TaskContainerDefaults for the specified pool.
func (r ResourceManager) TaskContainerDefaults(
	ctx actor.Messenger,
	pool string,
	fallbackConfig model.TaskContainerDefaultsConfig,
) (model.TaskContainerDefaultsConfig, error) {
	return fallbackConfig, nil
}

// SlotAddr calculates and returns a slot address.
func SlotAddr(agentID, slotID string) actor.Address {
	return sproto.AgentsAddr.Child(agentID).Child("slots").Child(slotID)
}

// EnableSlot implements 'det slot enable...' functionality.
func (r ResourceManager) EnableSlot(
	m actor.Messenger,
	req *apiv1.EnableSlotRequest,
) (resp *apiv1.EnableSlotResponse, err error) {
	return resp, AskAt(r.Ref().System(), SlotAddr(req.AgentId, req.SlotId), req, &resp)
}

// DisableSlot implements 'det slot disable...' functionality.
func (r ResourceManager) DisableSlot(
	m actor.Messenger,
	req *apiv1.DisableSlotRequest,
) (resp *apiv1.DisableSlotResponse, err error) {
	return resp, AskAt(r.Ref().System(), SlotAddr(req.AgentId, req.SlotId), req, &resp)
}
