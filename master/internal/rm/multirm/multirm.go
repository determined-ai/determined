package multirm

import (
	"fmt"
	"slices"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	"golang.org/x/sync/errgroup"

	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// ErrRMConflict returns a detailed error if multiple resource managers define a resource pool
// with the same name.
func ErrRMConflict(rmNames []string, rp string) error {
	slices.Sort(rmNames)
	return fmt.Errorf("resource pool %s exists for both resource managers %v,", rp, rmNames)
}

// ErrRMNotDefined returns a detailed error if a resource manager isn't found in the MultiRMRouter map.
func ErrRMNotDefined(rm string) error {
	return fmt.Errorf("resource manager %s not defined", rm)
}

// ErrRPNotDefined returns a detailed error if a resource pool isn't found.
func ErrRPNotDefined(rp string) error {
	return fmt.Errorf("could not find resource pool %s", rp)
}

// MultiRMRouter tracks all resource managers in the system.
type MultiRMRouter struct {
	defaultRMName string
	rms           map[string]rm.ResourceManager
	syslog        *logrus.Entry
}

// New returns a new MultiRM.
func New(defaultRMName string, rms map[string]rm.ResourceManager) *MultiRMRouter {
	return &MultiRMRouter{
		defaultRMName: defaultRMName,
		rms:           rms,
		syslog:        logrus.WithField("component", "resource-router"),
	}
}

// GetAllocationSummaries returns the allocation summaries for all resource pools across all resource managers.
func (m *MultiRMRouter) GetAllocationSummaries() (
	map[model.AllocationID]sproto.AllocationSummary,
	error,
) {
	res, err := fanOutRMCall(m, func(rm rm.ResourceManager) (map[model.AllocationID]sproto.AllocationSummary, error) {
		return rm.GetAllocationSummaries()
	})
	if err != nil {
		return nil, err
	}

	all := map[model.AllocationID]sproto.AllocationSummary{}
	for _, r := range res {
		maps.Copy(all, r)
	}
	return all, nil
}

// Allocate routes an AllocateRequest to the specified RM.
func (m *MultiRMRouter) Allocate(rmName string, req sproto.AllocateRequest) (*sproto.ResourcesSubscription, error) {
	resolvedRMName, err := m.getRM(rmName, req.ResourcePool)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].Allocate(resolvedRMName, req)
}

// Release routes an allocation release request.
func (m *MultiRMRouter) Release(rmName string, req sproto.ResourcesReleased) {
	resolvedRMName, err := m.getRM(rmName, req.ResourcePool)
	if err != nil {
		m.syslog.WithError(err)
		return
	}

	m.rms[resolvedRMName].Release(resolvedRMName, req)
}

// ValidateResources routes a validation request for a specified resource manager/pool.
func (m *MultiRMRouter) ValidateResources(
	rmName string, req sproto.ValidateResourcesRequest,
) ([]command.LaunchWarning, error) {
	resolvedRMName, err := m.getRM(rmName, req.ResourcePool)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].ValidateResources(resolvedRMName, req)
}

// DeleteJob routes a DeleteJob request to the specified resource manager.
func (m *MultiRMRouter) DeleteJob(req sproto.DeleteJob) (sproto.DeleteJobResponse, error) {
	m.syslog.WithError(fmt.Errorf("DeleteJob is not implemented for agent, kubernetes, or multi-rm"))
	return sproto.EmptyDeleteJobResponse(), nil
}

// NotifyContainerRunning routes a NotifyContainerRunning request to a specified resource manager/pool.
func (m *MultiRMRouter) NotifyContainerRunning(req sproto.NotifyContainerRunning) error {
	// MultiRM is currently only implemented for Kubernetes, which doesn't support this.
	m.syslog.WithError(fmt.Errorf("NotifyContainerRunning is not implemented for agent, kubernetes, or multi-rm"))
	return rmerrors.ErrNotSupported
}

// SetGroupMaxSlots routes a SetGroupMaxSlots request to a specified resource manager/pool.
func (m *MultiRMRouter) SetGroupMaxSlots(rmName string, req sproto.SetGroupMaxSlots) {
	resolvedRMName, err := m.getRM(rmName, req.ResourcePool)
	if err != nil {
		m.syslog.WithError(err)
		return
	}

	m.rms[resolvedRMName].SetGroupMaxSlots(resolvedRMName, req)
}

// SetGroupWeight routes a SetGroupWeight request to a specified resource manager/pool.
func (m *MultiRMRouter) SetGroupWeight(rmName string, req sproto.SetGroupWeight) error {
	resolvedRMName, err := m.getRM(rmName, req.ResourcePool)
	if err != nil {
		return err
	}

	return m.rms[resolvedRMName].SetGroupWeight(resolvedRMName, req)
}

// SetGroupPriority routes a SetGroupPriority request to a specified resource manager/pool.
func (m *MultiRMRouter) SetGroupPriority(rmName string, req sproto.SetGroupPriority) error {
	resolvedRMName, err := m.getRM(rmName, req.ResourcePool)
	if err != nil {
		return err
	}

	return m.rms[resolvedRMName].SetGroupPriority(resolvedRMName, req)
}

// ExternalPreemptionPending routes an ExternalPreemptionPending request to the specified resource manager.
func (m *MultiRMRouter) ExternalPreemptionPending(allocationID model.AllocationID) error {
	// MultiRM is currently only implemented for Kubernetes, which doesn't support this.
	m.syslog.WithError(fmt.Errorf("ExternalPreemptionPending is not implemented for agent, kubernetes, or multi-rm"))
	return rmerrors.ErrNotSupported
}

// IsReattachableOnlyAfterStarted routes a IsReattachableOnlyAfterStarted call to a specified resource manager/pool.
func (m *MultiRMRouter) IsReattachableOnlyAfterStarted(rmName string) bool {
	resolvedRMName, err := m.getRM(rmName, "")
	if err != nil {
		m.syslog.WithError(err)
		return false // Not sure what else to return here.
	}

	return m.rms[resolvedRMName].IsReattachableOnlyAfterStarted(resolvedRMName)
}

// GetResourcePools returns all resource pools across all resource managers.
func (m *MultiRMRouter) GetResourcePools() (*apiv1.GetResourcePoolsResponse, error) {
	res, err := fanOutRMCall(m, func(rm rm.ResourceManager) (*apiv1.GetResourcePoolsResponse, error) {
		return rm.GetResourcePools()
	})
	if err != nil {
		return nil, err
	}

	all := &apiv1.GetResourcePoolsResponse{}
	for _, r := range res {
		all.ResourcePools = append(all.ResourcePools, r.ResourcePools...)
	}
	return all, nil
}

// GetDefaultComputeResourcePool routes a GetDefaultComputeResourcePool to the specified resource manager.
func (m *MultiRMRouter) GetDefaultComputeResourcePool(rmName string) (
	sproto.GetDefaultComputeResourcePoolResponse, error,
) {
	resolvedRMName, err := m.getRM(rmName, "")
	if err != nil {
		return sproto.GetDefaultComputeResourcePoolResponse{}, err
	}

	return m.rms[resolvedRMName].GetDefaultComputeResourcePool(resolvedRMName)
}

// GetDefaultAuxResourcePool routes a GetDefaultAuxResourcePool to the specified resource manager.
func (m *MultiRMRouter) GetDefaultAuxResourcePool(rmName string) (sproto.GetDefaultAuxResourcePoolResponse, error) {
	resolvedRMName, err := m.getRM(rmName, "")
	if err != nil {
		return sproto.GetDefaultAuxResourcePoolResponse{}, err
	}

	return m.rms[resolvedRMName].GetDefaultAuxResourcePool(resolvedRMName)
}

// ValidateResourcePool routes a ValidateResourcePool call to the specified resource manager.
func (m *MultiRMRouter) ValidateResourcePool(rmName string, rpName string) error {
	resolvedRMName, err := m.getRM(rmName, rpName)
	if err != nil {
		return err
	}

	return m.rms[resolvedRMName].ValidateResourcePool(resolvedRMName, rpName)
}

// ResolveResourcePool routes a ResolveResourcePool request for a specific resource manager/pool.
func (m *MultiRMRouter) ResolveResourcePool(rmName string, req sproto.ResolveResourcesRequest) (
	resourceManager, resourcePool string, err error,
) {
	resolvedRMName, err := m.getRM(rmName, req.ResourcePool)
	if err != nil {
		return rmName, req.ResourcePool, err
	}

	return m.rms[resolvedRMName].ResolveResourcePool(resolvedRMName, req)
}

// TaskContainerDefaults routes a TaskContainerDefaults call to a specific resource manager/pool.
func (m *MultiRMRouter) TaskContainerDefaults(
	rmName, rpName string,
	fallbackConfig model.TaskContainerDefaultsConfig,
) (model.TaskContainerDefaultsConfig, error) {
	resolvedRMName, err := m.getRM(rmName, rpName)
	if err != nil {
		return model.TaskContainerDefaultsConfig{}, err
	}

	return m.rms[resolvedRMName].TaskContainerDefaults(resolvedRMName, rpName, fallbackConfig)
}

// GetJobQ routes a GetJobQ call to a specified resource manager/pool.
func (m *MultiRMRouter) GetJobQ(rmName, rpName string) (map[model.JobID]*sproto.RMJobInfo, error) {
	resolvedRMName, err := m.getRM(rmName, rpName)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetJobQ(resolvedRMName, rpName)
}

// GetJobQueueStatsRequest routes a GetJobQueueStatsRequest to the specified resource manager.
func (m *MultiRMRouter) GetJobQueueStatsRequest(rmName string, req *apiv1.GetJobQueueStatsRequest) (
	*apiv1.GetJobQueueStatsResponse, error,
) {
	resolvedRMName, err := m.getRM(rmName, req.ResourcePools[0])
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetJobQueueStatsRequest(resolvedRMName, req)
}

// MoveJob routes a MoveJob call to a specified resource manager/pool.
func (m *MultiRMRouter) MoveJob(rmName string, req sproto.MoveJob) error {
	resolvedRMName, err := m.getRM(rmName, req.ResourcePool)
	if err != nil {
		return err
	}

	return m.rms[resolvedRMName].MoveJob(resolvedRMName, req)
}

// RecoverJobPosition routes a RecoverJobPosition call to a specified resource manager/pool.
func (m *MultiRMRouter) RecoverJobPosition(rmName string, req sproto.RecoverJobPosition) {
	resolvedRMName, err := m.getRM(rmName, req.ResourcePool)
	if err != nil {
		m.syslog.WithError(err)
		return
	}

	m.rms[resolvedRMName].RecoverJobPosition(resolvedRMName, req)
}

// GetExternalJobs routes a GetExternalJobs request to a specified resource manager.
func (m *MultiRMRouter) GetExternalJobs(rmName string, rpName string) ([]*jobv1.Job, error) {
	resolvedRMName, err := m.getRM(rmName, rpName)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetExternalJobs(resolvedRMName, rpName)
}

// GetAgents returns all agents across all resource managers.
func (m *MultiRMRouter) GetAgents() (*apiv1.GetAgentsResponse, error) {
	res, err := fanOutRMCall(m, func(rm rm.ResourceManager) (*apiv1.GetAgentsResponse, error) {
		return rm.GetAgents()
	})
	if err != nil {
		return nil, err
	}

	all := &apiv1.GetAgentsResponse{}
	for _, r := range res {
		all.Agents = append(all.Agents, r.Agents...)
	}
	return all, nil
}

// GetAgent routes a GetAgent request to the specified resource manager & agent.
func (m *MultiRMRouter) GetAgent(rmName string, req *apiv1.GetAgentRequest) (*apiv1.GetAgentResponse, error) {
	resolvedRMName, err := m.getRM(rmName, req.AgentId)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetAgent(resolvedRMName, req)
}

// EnableAgent routes an EnableAgent request to the specified resource manager & agent.
func (m *MultiRMRouter) EnableAgent(rmName string, req *apiv1.EnableAgentRequest) (*apiv1.EnableAgentResponse, error) {
	resolvedRMName, err := m.getRM(rmName, req.AgentId)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].EnableAgent(resolvedRMName, req)
}

// DisableAgent routes an DisableAgent request to the specified resource manager & agent.
func (m *MultiRMRouter) DisableAgent(rmName string, req *apiv1.DisableAgentRequest) (
	*apiv1.DisableAgentResponse, error,
) {
	resolvedRMName, err := m.getRM(rmName, req.AgentId)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].DisableAgent(resolvedRMName, req)
}

// GetSlots routes an GetSlots request to the specified resource manager & agent.
func (m *MultiRMRouter) GetSlots(rmName string, req *apiv1.GetSlotsRequest) (*apiv1.GetSlotsResponse, error) {
	resolvedRMName, err := m.getRM(rmName, req.AgentId)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetSlots(resolvedRMName, req)
}

// GetSlot routes an GetSlot request to the specified resource manager & agent.
func (m *MultiRMRouter) GetSlot(rmName string, req *apiv1.GetSlotRequest) (*apiv1.GetSlotResponse, error) {
	resolvedRMName, err := m.getRM(rmName, req.AgentId)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetSlot(resolvedRMName, req)
}

// EnableSlot routes an EnableSlot request to the specified resource manager & agent.
func (m *MultiRMRouter) EnableSlot(rmName string, req *apiv1.EnableSlotRequest) (*apiv1.EnableSlotResponse, error) {
	resolvedRMName, err := m.getRM(rmName, req.AgentId)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].EnableSlot(resolvedRMName, req)
}

// DisableSlot routes an DisableSlot request to the specified resource manager & agent.
func (m *MultiRMRouter) DisableSlot(rmName string, req *apiv1.DisableSlotRequest) (*apiv1.DisableSlotResponse, error) {
	resolvedRMName, err := m.getRM(rmName, req.AgentId)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].DisableSlot(resolvedRMName, req)
}

func (m *MultiRMRouter) getRM(rmName string, rpName string) (string, error) {
	if rmName != "" {
		// If explicitly given the RMName, check that it exists in the map.
		_, ok := m.rms[rmName]
		if !ok {
			return rmName, ErrRMNotDefined(rmName)
		}
		return rmName, nil
	}

	// If given neither the RM or RP name, route to default RM.
	if rpName == "" {
		m.syslog.Infof("RM undefined, routing to default resource manager")
		return m.defaultRMName, nil
	}

	// If just given the RP name, search through all resource managers for a single match.
	rmMatches := []string{}
	for name, r := range m.rms {
		rps, err := r.GetResourcePools()
		if err != nil {
			return name, fmt.Errorf("could not get resource pools for %s", r)
		}
		for _, p := range rps.ResourcePools {
			if p.Name == rpName {
				rmMatches = append(rmMatches, name)
			}
		}
	}

	if len(rmMatches) == 0 {
		// If the resolvedRMName isn't set, then the RP was not found.
		return rmName, ErrRPNotDefined(rpName)
	} else if len(rmMatches) > 1 {
		// If the resolvedRMName is already set, we assume there is a conflict.
		return "", ErrRMConflict(rmMatches, rpName)
	}
	return rmMatches[0], nil
}

func fanOutRMCall[TReturn any](m *MultiRMRouter, f func(rm.ResourceManager) (TReturn, error)) ([]TReturn, error) {
	res := make([]TReturn, len(m.rms))
	var eg errgroup.Group
	for i, rm := range maps.Values(m.rms) {
		i, rm := i, rm
		eg.Go(func() error {
			r, err := f(rm)
			if err != nil {
				return err
			}
			res[i] = r
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return res, nil
}
