package multirm

import (
	"fmt"

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

// ErrRPNotDefined returns a detailed error if a resource pool isn't found.
func ErrRPNotDefined(rp rm.ResourcePoolName) error {
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
func (m *MultiRMRouter) Allocate(req sproto.AllocateRequest) (*sproto.ResourcesSubscription, error) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.ResourcePool))
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].Allocate(req)
}

// Release routes an allocation release request.
func (m *MultiRMRouter) Release(req sproto.ResourcesReleased) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.ResourcePool))
	if err != nil {
		m.syslog.WithError(err)
		return
	}

	m.rms[resolvedRMName].Release(req)
}

// ValidateResources routes a validation request for a specified resource manager/pool.
func (m *MultiRMRouter) ValidateResources(req sproto.ValidateResourcesRequest) ([]command.LaunchWarning, error) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.ResourcePool))
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].ValidateResources(req)
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
func (m *MultiRMRouter) SetGroupMaxSlots(req sproto.SetGroupMaxSlots) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.ResourcePool))
	if err != nil {
		m.syslog.WithError(err)
		return
	}

	m.rms[resolvedRMName].SetGroupMaxSlots(req)
}

// SetGroupWeight routes a SetGroupWeight request to a specified resource manager/pool.
func (m *MultiRMRouter) SetGroupWeight(req sproto.SetGroupWeight) error {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.ResourcePool))
	if err != nil {
		return err
	}

	return m.rms[resolvedRMName].SetGroupWeight(req)
}

// SetGroupPriority routes a SetGroupPriority request to a specified resource manager/pool.
func (m *MultiRMRouter) SetGroupPriority(req sproto.SetGroupPriority) error {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.ResourcePool))
	if err != nil {
		return err
	}

	return m.rms[resolvedRMName].SetGroupPriority(req)
}

// ExternalPreemptionPending routes an ExternalPreemptionPending request to the specified resource manager.
func (m *MultiRMRouter) ExternalPreemptionPending(sproto.PendingPreemption) error {
	// MultiRM is currently only implemented for Kubernetes, which doesn't support this.
	m.syslog.WithError(fmt.Errorf("ExternalPreemptionPending is not implemented for agent, kubernetes, or multi-rm"))
	return rmerrors.ErrNotSupported
}

// IsReattachableOnlyAfterStarted routes a IsReattachableOnlyAfterStarted call to a specified resource manager/pool.
func (m *MultiRMRouter) IsReattachableOnlyAfterStarted() bool {
	resolvedRMName, err := m.getRMName("")
	if err != nil {
		m.syslog.WithError(err)
		return false // Not sure what else to return here.
	}

	return m.rms[resolvedRMName].IsReattachableOnlyAfterStarted()
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
func (m *MultiRMRouter) GetDefaultComputeResourcePool() (rm.ResourcePoolName, error) {
	resolvedRMName, err := m.getRMName("")
	if err != nil {
		return "", err
	}

	return m.rms[resolvedRMName].GetDefaultComputeResourcePool()
}

// GetDefaultAuxResourcePool routes a GetDefaultAuxResourcePool to the specified resource manager.
func (m *MultiRMRouter) GetDefaultAuxResourcePool() (rm.ResourcePoolName, error) {
	resolvedRMName, err := m.getRMName("")
	if err != nil {
		return "", err
	}

	return m.rms[resolvedRMName].GetDefaultAuxResourcePool()
}

// ValidateResourcePool routes a ValidateResourcePool call to the specified resource manager.
func (m *MultiRMRouter) ValidateResourcePool(rpName rm.ResourcePoolName) error {
	resolvedRMName, err := m.getRMName(rpName)
	if err != nil {
		return err
	}

	return m.rms[resolvedRMName].ValidateResourcePool(rpName)
}

// ResolveResourcePool routes a ResolveResourcePool request for a specific resource manager/pool.
func (m *MultiRMRouter) ResolveResourcePool(rpName rm.ResourcePoolName, workspace, slots int) (
	rm.ResourcePoolName, error,
) {
	resolvedRMName, err := m.getRMName(rpName)
	if err != nil {
		return rpName, err
	}

	return m.rms[resolvedRMName].ResolveResourcePool(rpName, workspace, slots)
}

// TaskContainerDefaults routes a TaskContainerDefaults call to a specific resource manager/pool.
func (m *MultiRMRouter) TaskContainerDefaults(
	rpName rm.ResourcePoolName,
	fallbackConfig model.TaskContainerDefaultsConfig,
) (model.TaskContainerDefaultsConfig, error) {
	resolvedRMName, err := m.getRMName(rpName)
	if err != nil {
		return model.TaskContainerDefaultsConfig{}, err
	}

	return m.rms[resolvedRMName].TaskContainerDefaults(rpName, fallbackConfig)
}

// GetJobQ routes a GetJobQ call to a specified resource manager/pool.
func (m *MultiRMRouter) GetJobQ(rpName rm.ResourcePoolName) (map[model.JobID]*sproto.RMJobInfo, error) {
	resolvedRMName, err := m.getRMName(rpName)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetJobQ(rpName)
}

// GetJobQueueStatsRequest routes a GetJobQueueStatsRequest to the specified resource manager.
func (m *MultiRMRouter) GetJobQueueStatsRequest(req *apiv1.GetJobQueueStatsRequest) (
	*apiv1.GetJobQueueStatsResponse, error,
) {
	res, err := fanOutRMCall(m, func(rm rm.ResourceManager) (*apiv1.GetJobQueueStatsResponse, error) {
		return rm.GetJobQueueStatsRequest(req)
	})
	if err != nil {
		return nil, err
	}

	all := &apiv1.GetJobQueueStatsResponse{}
	for _, r := range res {
		all.Results = append(all.Results, r.Results...)
	}
	return all, nil
}

// MoveJob routes a MoveJob call to a specified resource manager/pool.
func (m *MultiRMRouter) MoveJob(req sproto.MoveJob) error {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.ResourcePool))
	if err != nil {
		return err
	}

	return m.rms[resolvedRMName].MoveJob(req)
}

// RecoverJobPosition routes a RecoverJobPosition call to a specified resource manager/pool.
func (m *MultiRMRouter) RecoverJobPosition(req sproto.RecoverJobPosition) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.ResourcePool))
	if err != nil {
		m.syslog.WithError(err)
		return
	}

	m.rms[resolvedRMName].RecoverJobPosition(req)
}

// GetExternalJobs routes a GetExternalJobs request to a specified resource manager.
func (m *MultiRMRouter) GetExternalJobs(rpName rm.ResourcePoolName) ([]*jobv1.Job, error) {
	resolvedRMName, err := m.getRMName(rpName)
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetExternalJobs(rpName)
}

// HealthCheck calls HealthCheck on all the resource managers.
func (m *MultiRMRouter) HealthCheck() []model.ResourceManagerHealth {
	res, _ := fanOutRMCall(m, func(rm rm.ResourceManager) ([]model.ResourceManagerHealth, error) {
		return rm.HealthCheck(), nil
	})

	var flattened []model.ResourceManagerHealth
	for _, r := range res {
		flattened = append(flattened, r...)
	}

	return flattened
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
func (m *MultiRMRouter) GetAgent(req *apiv1.GetAgentRequest) (*apiv1.GetAgentResponse, error) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.AgentId))
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetAgent(req)
}

// EnableAgent routes an EnableAgent request to the specified resource manager & agent.
func (m *MultiRMRouter) EnableAgent(req *apiv1.EnableAgentRequest) (*apiv1.EnableAgentResponse, error) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.AgentId))
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].EnableAgent(req)
}

// DisableAgent routes an DisableAgent request to the specified resource manager & agent.
func (m *MultiRMRouter) DisableAgent(req *apiv1.DisableAgentRequest) (
	*apiv1.DisableAgentResponse, error,
) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.AgentId))
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].DisableAgent(req)
}

// GetSlots routes an GetSlots request to the specified resource manager & agent.
func (m *MultiRMRouter) GetSlots(req *apiv1.GetSlotsRequest) (*apiv1.GetSlotsResponse, error) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.AgentId))
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetSlots(req)
}

// GetSlot routes an GetSlot request to the specified resource manager & agent.
func (m *MultiRMRouter) GetSlot(req *apiv1.GetSlotRequest) (*apiv1.GetSlotResponse, error) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.AgentId))
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].GetSlot(req)
}

// EnableSlot routes an EnableSlot request to the specified resource manager & agent.
func (m *MultiRMRouter) EnableSlot(req *apiv1.EnableSlotRequest) (*apiv1.EnableSlotResponse, error) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.AgentId))
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].EnableSlot(req)
}

// DisableSlot routes an DisableSlot request to the specified resource manager & agent.
func (m *MultiRMRouter) DisableSlot(req *apiv1.DisableSlotRequest) (*apiv1.DisableSlotResponse, error) {
	resolvedRMName, err := m.getRMName(rm.ResourcePoolName(req.AgentId))
	if err != nil {
		return nil, err
	}

	return m.rms[resolvedRMName].DisableSlot(req)
}

func (m *MultiRMRouter) CreateNamespace(autoCreateNamespace bool, namespaceName string,
	clusterName string) error {
	rm, err := m.getRM(clusterName)
	if err != nil {
		return fmt.Errorf("Error getting resource manager for cluster %s: %w", clusterName, err)
	}
	err = rm.CreateNamespace(autoCreateNamespace, namespaceName, clusterName)
	if err != nil {
		return fmt.Errorf("Error creating namespace %s: %w", namespaceName, err)
	}

	return nil
}

func (m *MultiRMRouter) getRMName(rpName rm.ResourcePoolName) (string, error) {
	// If not given RP name, route to default RM.
	if rpName == "" {
		m.syslog.Tracef("RM undefined, routing to default resource manager")
		return m.defaultRMName, nil
	}

	for name, r := range m.rms {
		rps, err := r.GetResourcePools()
		if err != nil {
			return name, fmt.Errorf("could not get resource pools for %s", r)
		}
		for _, p := range rps.ResourcePools {
			if p.Name == rpName.String() {
				m.syslog.Tracef("RM defined as %s, %s", name, p.Name)
				return name, nil
			}
		}
	}
	return "", ErrRPNotDefined(rpName)
}

func (m *MultiRMRouter) getRM(name string) (rm.ResourceManager, error) {
	// If not given RM name, route to default RM.
	for rmName, r := range m.rms {
		if rmName == name {
			return r, nil

		} else if rmName == m.defaultRMName && name == "" {
			m.syslog.Tracef("no name, routing to default resource manager")
			return r, nil

		}
	}
	return nil, fmt.Errorf("No resource manager of the specified name")
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
