package rm

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// ActorResourceManager shims a RM actor to the ResourceManager interface.
type ActorResourceManager struct {
	ref *actor.Ref
}

// WrapRMActor wraps an RM actor as an explicit interface.
func WrapRMActor(ref *actor.Ref) *ActorResourceManager {
	return &ActorResourceManager{ref: ref}
}

// GetResourcePoolRef is a default implementation to satisfy the interface, mostly for tests.
func (r *ActorResourceManager) GetResourcePoolRef(
	ctx actor.Messenger,
	name string,
) (*actor.Ref, error) {
	return r.ref, nil
}

// ResolveResourcePool is a default implementation to satisfy the interface, mostly for tests.
func (r *ActorResourceManager) ResolveResourcePool(
	ctx actor.Messenger,
	name string,
	slots int,
	command bool,
) (string, error) {
	return name, nil
}

// ValidateResourcePool is a default implementation to satisfy the interface, mostly for tests.
func (r *ActorResourceManager) ValidateResourcePool(ctx actor.Messenger, name string) error {
	return nil
}

// Ref gets the underlying RM actor, for backwards compatibility. This is deprecated.
func (r *ActorResourceManager) Ref() *actor.Ref {
	return r.ref
}

// GetAllocationHandler requests the allocation actor for the given allocation.
func (r *ActorResourceManager) GetAllocationHandler(
	ctx actor.Messenger,
	msg sproto.GetAllocationHandler,
) (resp *actor.Ref, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// GetAllocationSummary requests a summary of the given allocation.
func (r *ActorResourceManager) GetAllocationSummary(
	ctx actor.Messenger,
	msg sproto.GetAllocationSummary,
) (resp *sproto.AllocationSummary, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// GetAllocationSummaries requests a summary of all current allocations.
func (r *ActorResourceManager) GetAllocationSummaries(
	ctx actor.Messenger,
	msg sproto.GetAllocationSummaries,
) (resp map[model.AllocationID]sproto.AllocationSummary, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// SetAllocationName sets a name for a given allocation.
func (r *ActorResourceManager) SetAllocationName(
	ctx actor.Messenger,
	msg sproto.SetAllocationName,
) {
	r.tell(ctx, msg)
}

// ValidateCommandResources validates a request for command resources.
func (r *ActorResourceManager) ValidateCommandResources(
	ctx actor.Messenger,
	msg sproto.ValidateCommandResourcesRequest,
) (resp sproto.ValidateCommandResourcesResponse, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// Allocate allocates some resources.
func (r *ActorResourceManager) Allocate(ctx actor.Messenger, msg sproto.AllocateRequest) error {
	return r.ask(ctx, msg, nil)
}

// Release releases some resources.
func (r *ActorResourceManager) Release(ctx actor.Messenger, msg sproto.ResourcesReleased) {
	r.tell(ctx, msg)
}

// GetResourcePools requests information about the available resource pools.
func (r *ActorResourceManager) GetResourcePools(
	ctx actor.Messenger,
	msg *apiv1.GetResourcePoolsRequest,
) (resp *apiv1.GetResourcePoolsResponse, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// GetDefaultComputeResourcePool requests the default compute resource pool.
func (r *ActorResourceManager) GetDefaultComputeResourcePool(
	ctx actor.Messenger,
	msg sproto.GetDefaultComputeResourcePoolRequest,
) (resp sproto.GetDefaultComputeResourcePoolResponse, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// GetDefaultAuxResourcePool requests the default aux resource pool.
func (r *ActorResourceManager) GetDefaultAuxResourcePool(
	ctx actor.Messenger,
	msg sproto.GetDefaultAuxResourcePoolRequest,
) (resp sproto.GetDefaultAuxResourcePoolResponse, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// GetAgents gets the state of connected agents or reads similar information from the underlying RM.
func (r *ActorResourceManager) GetAgents(
	ctx actor.Messenger,
	msg *apiv1.GetAgentsRequest,
) (resp *apiv1.GetAgentsResponse, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// GetJobQ gets the state of the job queue.
func (r *ActorResourceManager) GetJobQ(
	ctx actor.Messenger,
	msg sproto.GetJobQ,
) (resp map[model.JobID]*sproto.RMJobInfo, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// GetJobQStats requests stats for a job queue.
func (r *ActorResourceManager) GetJobQStats(
	ctx actor.Messenger,
	msg sproto.GetJobQStats,
) (resp *jobv1.QueueStats, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// GetJobQueueStatsRequest requests other stats for a job queue.
func (r *ActorResourceManager) GetJobQueueStatsRequest(
	ctx actor.Messenger,
	msg *apiv1.GetJobQueueStatsRequest,
) (resp *apiv1.GetJobQueueStatsResponse, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// MoveJob moves a job ahead of or behind a peer.
func (r *ActorResourceManager) MoveJob(ctx actor.Messenger, msg sproto.MoveJob) error {
	return r.ask(ctx, msg, nil)
}

// RecoverJobPosition recovers the position of a job relative to the rest of its priority lane.
func (r *ActorResourceManager) RecoverJobPosition(
	ctx actor.Messenger,
	msg sproto.RecoverJobPosition,
) {
	r.tell(ctx, msg)
}

// SetGroupWeight sets the weight for a group.
func (r *ActorResourceManager) SetGroupWeight(
	ctx actor.Messenger,
	msg sproto.SetGroupWeight,
) error {
	return r.ask(ctx, msg, nil)
}

// SetGroupPriority sets the group priority.
func (r *ActorResourceManager) SetGroupPriority(
	ctx actor.Messenger,
	msg sproto.SetGroupPriority,
) error {
	return r.ask(ctx, msg, nil)
}

// SetGroupMaxSlots sets the max allocatable slots for a group.
func (r *ActorResourceManager) SetGroupMaxSlots(ctx actor.Messenger, msg sproto.SetGroupMaxSlots) {
	r.tell(ctx, msg)
}

// DeleteJob requests we clean up our state related to a given job.
func (r *ActorResourceManager) DeleteJob(
	ctx actor.Messenger,
	msg sproto.DeleteJob,
) (resp sproto.DeleteJobResponse, err error) {
	return resp, r.ask(ctx, msg, &resp)
}

// ExternalPreemptionPending requests we notify some allocation that it was preempted externally.
func (r *ActorResourceManager) ExternalPreemptionPending(
	ctx actor.Messenger,
	msg sproto.PendingPreemption,
) error {
	return r.ask(ctx, msg, nil)
}

// NotifyContainerRunning receives a notification from the container to let
// the master know that the container is running.
func (r *ActorResourceManager) NotifyContainerRunning(
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
func (r *ActorResourceManager) IsReattachEnabled(ctx actor.Messenger) bool {
	return false
}

// IsReattachableOnlyAfterStarted is a default implementation (true).
func (r *ActorResourceManager) IsReattachableOnlyAfterStarted(ctx actor.Messenger) bool {
	return true
}

// IsReattachEnabledForRP is a default implementation for an RP being reattachable (false).
func (r *ActorResourceManager) IsReattachEnabledForRP(ctx actor.Messenger, rpName string) bool {
	return false
}

func (r *ActorResourceManager) tell(ctx actor.Messenger, req interface{}) {
	ctx.Tell(r.ref, req)
}

func (r *ActorResourceManager) ask(ctx actor.Messenger, req interface{}, v interface{}) error {
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
				return fmt.Errorf("%s returned unexpected resp (%T): %v", r.ref.Address(), resp, resp)
			}
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(resp.Get()))
		}
		return nil
	}
}

func askAt(sys *actor.System, addr actor.Address, req interface{}, v interface{}) error {
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
