package actorrm

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// ResourceManager shims a deprecated RM actor to the ResourceManager interface.
type ResourceManager struct {
	ref *actor.Ref
}

// Wrap wraps an RM actor as an explicit interface. This is deprecated. New resource managers
// should satisfy the ResourceManager interface directly.
func Wrap(ref *actor.Ref) *ResourceManager {
	return &ResourceManager{ref: ref}
}

// Ref gets the underlying RM actor, for internal actor use only.
func (r *ResourceManager) Ref() *actor.Ref {
	return r.ref
}

// ResolveResourcePool is a default implementation to satisfy the interface, mostly for tests.
func (r *ResourceManager) ResolveResourcePool(
	name string,
	workspaceID,
	slots int,
) (string, error) {
	return name, nil
}

// ValidateResources is a default implementation to satisfy the interface, mostly for tests.
func (r *ResourceManager) ValidateResources(
	name string,
	slots int,
	command bool,
) error {
	return nil
}

// ValidateResourcePoolAvailability is a default implementation to satisfy the interface.
func (r *ResourceManager) ValidateResourcePoolAvailability(
	name string,
	slots int) (
	[]command.LaunchWarning,
	error,
) {
	return nil, nil
}

// ValidateResourcePool is a default implementation to satisfy the interface, mostly for tests.
func (r *ResourceManager) ValidateResourcePool(name string) error {
	return nil
}

// GetAllocationSummary requests a summary of the given allocation.
func (r *ResourceManager) GetAllocationSummary(
	msg sproto.GetAllocationSummary,
) (resp *sproto.AllocationSummary, err error) {
	return resp, r.Ask(msg, &resp)
}

// GetAllocationSummaries requests a summary of all current allocations.
func (r *ResourceManager) GetAllocationSummaries(
	msg sproto.GetAllocationSummaries,
) (resp map[model.AllocationID]sproto.AllocationSummary, err error) {
	return resp, r.Ask(msg, &resp)
}

// SetAllocationName sets a name for a given allocation.
func (r *ResourceManager) SetAllocationName(
	msg sproto.SetAllocationName,
) {
	r.Tell(msg)
}

// ValidateCommandResources validates a request for command resources.
func (r *ResourceManager) ValidateCommandResources(
	msg sproto.ValidateCommandResourcesRequest,
) (resp sproto.ValidateCommandResourcesResponse, err error) {
	return resp, r.Ask(msg, &resp)
}

// Allocate allocates some resources.
func (r *ResourceManager) Allocate(
	msg sproto.AllocateRequest,
) (*sproto.ResourcesSubscription, error) {
	sub := rmevents.Subscribe(msg.AllocationID)
	err := r.Ask(msg, nil)
	if err != nil {
		r.Release(sproto.ResourcesReleased{AllocationID: msg.AllocationID})
		sub.Close()
		return nil, err
	}
	return sub, nil
}

// Release releases some resources.
func (r *ResourceManager) Release(msg sproto.ResourcesReleased) {
	r.Tell(msg)
}

// GetResourcePools requests information about the available resource pools.
func (r *ResourceManager) GetResourcePools(
	msg *apiv1.GetResourcePoolsRequest,
) (resp *apiv1.GetResourcePoolsResponse, err error) {
	return resp, r.Ask(msg, &resp)
}

// GetDefaultComputeResourcePool requests the default compute resource pool.
func (r *ResourceManager) GetDefaultComputeResourcePool(
	msg sproto.GetDefaultComputeResourcePoolRequest,
) (resp sproto.GetDefaultComputeResourcePoolResponse, err error) {
	return resp, r.Ask(msg, &resp)
}

// GetDefaultAuxResourcePool requests the default aux resource pool.
func (r *ResourceManager) GetDefaultAuxResourcePool(
	msg sproto.GetDefaultAuxResourcePoolRequest,
) (resp sproto.GetDefaultAuxResourcePoolResponse, err error) {
	return resp, r.Ask(msg, &resp)
}

// GetJobQ gets the state of the job queue.
func (r *ResourceManager) GetJobQ(
	msg sproto.GetJobQ,
) (resp map[model.JobID]*sproto.RMJobInfo, err error) {
	return resp, r.Ask(msg, &resp)
}

// GetJobQueueStatsRequest requests other stats for a job queue.
func (r *ResourceManager) GetJobQueueStatsRequest(
	msg *apiv1.GetJobQueueStatsRequest,
) (resp *apiv1.GetJobQueueStatsResponse, err error) {
	return resp, r.Ask(msg, &resp)
}

// MoveJob moves a job ahead of or behind a peer.
func (r *ResourceManager) MoveJob(msg sproto.MoveJob) error {
	return r.Ask(msg, nil)
}

// RecoverJobPosition recovers the position of a job relative to the rest of its priority lane.
func (r *ResourceManager) RecoverJobPosition(
	msg sproto.RecoverJobPosition,
) {
	r.Tell(msg)
}

// SetGroupWeight sets the weight for a group.
func (r *ResourceManager) SetGroupWeight(msg sproto.SetGroupWeight) error {
	return r.Ask(msg, nil)
}

// SetGroupPriority sets the group priority.
func (r *ResourceManager) SetGroupPriority(msg sproto.SetGroupPriority) error {
	return r.Ask(msg, nil)
}

// SetGroupMaxSlots sets the max allocatable slots for a group.
func (r *ResourceManager) SetGroupMaxSlots(msg sproto.SetGroupMaxSlots) {
	r.Tell(msg)
}

// DeleteJob requests we clean up our state related to a given job.
func (r *ResourceManager) DeleteJob(
	msg sproto.DeleteJob,
) (resp sproto.DeleteJobResponse, err error) {
	return resp, r.Ask(msg, &resp)
}

// GetExternalJobs returns the details for External jobs.
func (r *ResourceManager) GetExternalJobs(
	msg sproto.GetExternalJobs,
) (resp []*jobv1.Job, err error) {
	return resp, r.Ask(msg, &resp)
}

// ExternalPreemptionPending requests we notify some allocation that it was preempted externally.
func (r *ResourceManager) ExternalPreemptionPending(
	msg sproto.PendingPreemption,
) error {
	return r.Ask(msg, nil)
}

// NotifyContainerRunning receives a notification from the container to let
// the master know that the container is running.
func (r *ResourceManager) NotifyContainerRunning(
	msg sproto.NotifyContainerRunning,
) error {
	// Actor Resource Manager does not implement a handler for the
	// NotifyContainerRunning message, as it is only used on HPC
	// (High Performance Computing).
	return errors.New(
		"the NotifyContainerRunning message is unsupported for ActorResourceManager")
}

// IsReattachableOnlyAfterStarted is a default implementation (true).
func (r *ResourceManager) IsReattachableOnlyAfterStarted() bool {
	return true
}

// Tell tells the underlying actor-based RM the req.
func (r *ResourceManager) Tell(req interface{}) {
	r.ref.System().Tell(r.ref, req)
}

// Ask asks the underlying actor-based RM the req, setting the response into v.
func (r *ResourceManager) Ask(req interface{}, v interface{}) error {
	if reflect.ValueOf(v).IsValid() && !reflect.ValueOf(v).Elem().CanSet() {
		return fmt.Errorf("ask to %s has valid but unsettable resp %T", r.ref.Address(), v)
	}
	expectingResponse := reflect.ValueOf(v).IsValid() && reflect.ValueOf(v).Elem().CanSet()
	switch resp := r.ref.System().Ask(r.ref, req); {
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
func (r *ResourceManager) AskAt(addr actor.Address, req interface{}, v interface{}) error {
	if reflect.ValueOf(v).IsValid() && !reflect.ValueOf(v).Elem().CanSet() {
		return fmt.Errorf("ask at %s has valid but unsettable resp %T", addr, v)
	}
	expectingResponse := reflect.ValueOf(v).IsValid() && reflect.ValueOf(v).Elem().CanSet()
	switch resp := r.ref.System().AskAt(addr, req); {
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
	pool string,
	fallbackConfig model.TaskContainerDefaultsConfig,
) (model.TaskContainerDefaultsConfig, error) {
	return fallbackConfig, nil
}

func agentAddr(agentID string) actor.Address {
	return sproto.AgentsAddr.Child(agentID)
}

func slotAddr(agentID, slotID string) actor.Address {
	return sproto.AgentsAddr.Child(agentID).Child("slots").Child(slotID)
}

// GetAgents gets the state of connected agents or reads similar information from the underlying RM.
func (r *ResourceManager) GetAgents(
	msg *apiv1.GetAgentsRequest,
) (resp *apiv1.GetAgentsResponse, err error) {
	return resp, r.Ask(msg, &resp)
}

// GetAgent implements rm.ResourceManager.
func (r *ResourceManager) GetAgent(
	req *apiv1.GetAgentRequest,
) (resp *apiv1.GetAgentResponse, err error) {
	return resp, r.AskAt(agentAddr(req.AgentId), req, &resp)
}

// EnableAgent implements rm.ResourceManager.
func (r *ResourceManager) EnableAgent(
	req *apiv1.EnableAgentRequest,
) (resp *apiv1.EnableAgentResponse, err error) {
	return resp, r.AskAt(agentAddr(req.AgentId), req, &resp)
}

// DisableAgent implements rm.ResourceManager.
func (r *ResourceManager) DisableAgent(
	req *apiv1.DisableAgentRequest,
) (resp *apiv1.DisableAgentResponse, err error) {
	return resp, r.AskAt(agentAddr(req.AgentId), req, &resp)
}

// GetSlots implements rm.ResourceManager.
func (r *ResourceManager) GetSlots(
	req *apiv1.GetSlotsRequest,
) (resp *apiv1.GetSlotsResponse, err error) {
	return resp, r.AskAt(agentAddr(req.AgentId), req, &resp)
}

// GetSlot implements rm.ResourceManager.
func (r *ResourceManager) GetSlot(
	req *apiv1.GetSlotRequest,
) (resp *apiv1.GetSlotResponse, err error) {
	return resp, r.AskAt(slotAddr(req.AgentId, req.SlotId), req, &resp)
}

// EnableSlot implements 'det slot enable...' functionality.
func (r ResourceManager) EnableSlot(
	req *apiv1.EnableSlotRequest,
) (resp *apiv1.EnableSlotResponse, err error) {
	return resp, r.AskAt(slotAddr(req.AgentId, req.SlotId), req, &resp)
}

// DisableSlot implements 'det slot disable...' functionality.
func (r ResourceManager) DisableSlot(
	req *apiv1.DisableSlotRequest,
) (resp *apiv1.DisableSlotResponse, err error) {
	return resp, r.AskAt(slotAddr(req.AgentId, req.SlotId), req, &resp)
}
