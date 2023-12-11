package kubernetesrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"maps"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/rm/rmutils"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const (
	// KubernetesScheduler is the "name" of the kubernetes scheduler, for informational reasons.
	kubernetesScheduler = "kubernetes"
	// podSubmissionInterval is the rate limit for job submission.
	podSubmissionInterval = 500 * time.Millisecond
)

// ResourceManager is a resource manager that manages k8s resources.
type ResourceManager struct {
	syslog *logrus.Entry

	config                *config.KubernetesResourceManagerConfig
	poolsConfig           []config.ResourcePoolConfig
	taskContainerDefaults *model.TaskContainerDefaultsConfig

	podsService *pods
	pools       map[string]*kubernetesResourcePool // immutable after initialization in new.

	masterTLSConfig model.TLSClientConfig
	loggingConfig   model.LoggingConfig

	db *db.PgDB
}

// New returns a new ResourceManager, which communicates with
// and submits work to a Kubernetes apiserver.
func New(
	db *db.PgDB,
	rmConfigs *config.ResourceConfig,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) *ResourceManager {
	tlsConfig, err := model.MakeTLSConfig(cert)
	if err != nil {
		panic(errors.Wrap(err, "failed to set up TLS config"))
	}

	// TODO(DET-9833) clusterID should just be a `internal/config` package singleton.
	clusterID, err := db.GetOrCreateClusterID("")
	if err != nil {
		panic(fmt.Errorf("getting clusterID: %w", err))
	}
	setClusterID(clusterID)

	k := &ResourceManager{
		syslog: logrus.WithField("component", "k8srm"),

		config:                rmConfigs.ResourceManager.KubernetesRM,
		poolsConfig:           rmConfigs.ResourcePools,
		taskContainerDefaults: taskContainerDefaults,

		pools: make(map[string]*kubernetesResourcePool),

		masterTLSConfig: tlsConfig,
		loggingConfig:   opts.LoggingOptions,

		db: db,
	}

	poolNamespaces := make(map[string]string)
	for i := range k.poolsConfig {
		if k.poolsConfig[i].KubernetesNamespace == "" {
			k.poolsConfig[i].KubernetesNamespace = k.config.Namespace
		}

		poolNamespaces[k.poolsConfig[i].KubernetesNamespace] = k.poolsConfig[i].PoolName
	}

	k.podsService = newPodsService(
		k.config.Namespace,
		poolNamespaces,
		k.config.MasterServiceName,
		k.masterTLSConfig,
		k.loggingConfig,
		k.config.DefaultScheduler,
		k.config.SlotType,
		config.PodSlotResourceRequests{CPU: k.config.SlotResourceRequests.CPU},
		k.poolsConfig,
		k.taskContainerDefaults,
		k.config.CredsDir,
		k.config.MasterIP,
		k.config.MasterPort,
		k.podStatusUpdateCallback,
	)

	for _, poolConfig := range k.poolsConfig {
		maxSlotsPerPod := 0
		if k.taskContainerDefaults.Kubernetes.MaxSlotsPerPod != nil {
			maxSlotsPerPod = *k.taskContainerDefaults.Kubernetes.MaxSlotsPerPod
		}
		if poolConfig.TaskContainerDefaults != nil &&
			poolConfig.TaskContainerDefaults.Kubernetes != nil &&
			poolConfig.TaskContainerDefaults.Kubernetes.MaxSlotsPerPod != nil {
			maxSlotsPerPod = *poolConfig.TaskContainerDefaults.Kubernetes.MaxSlotsPerPod
		}

		poolConfig := poolConfig
		rp := newResourcePool(maxSlotsPerPod, &poolConfig, k.podsService, k.db)
		go func() {
			t := time.NewTicker(podSubmissionInterval)
			defer t.Stop()
			for range t.C {
				rp.Schedule()
			}
		}()
		k.pools[poolConfig.PoolName] = rp
	}
	return k
}

// Allocate implements rm.ResourceManager.
func (k *ResourceManager) Allocate(msg sproto.AllocateRequest) (*sproto.ResourcesSubscription, error) {
	// This code exists to handle the case where an experiment does not have
	// an explicit resource pool specified in the config. This should never happen
	// for newly created/forked experiments as the default pool is filled in to the
	// config at creation time. However, old experiments which were created prior to
	// the introduction of resource pools could have no resource pool associated with
	// them and so we need to handle that case gracefully.
	if len(msg.ResourcePool) == 0 {
		if msg.SlotsNeeded == 0 {
			msg.ResourcePool = k.config.DefaultAuxResourcePool
		} else {
			msg.ResourcePool = k.config.DefaultComputeResourcePool
		}
	}

	rp, err := k.poolByName(msg.ResourcePool)
	if err != nil {
		return nil, err
	}
	sub := rmevents.Subscribe(msg.AllocationID)
	rp.AllocateRequest(msg)
	return sub, nil
}

// DeleteJob implements rm.ResourceManager.
func (ResourceManager) DeleteJob(sproto.DeleteJob) (sproto.DeleteJobResponse, error) {
	// For now, there is nothing to clean up in k8s.
	return sproto.EmptyDeleteJobResponse(), nil
}

// ExternalPreemptionPending implements rm.ResourceManager.
func (ResourceManager) ExternalPreemptionPending(sproto.PendingPreemption) error {
	return rmerrors.ErrNotSupported
}

// GetAgent implements rm.ResourceManager.
func (ResourceManager) GetAgent(*apiv1.GetAgentRequest) (*apiv1.GetAgentResponse, error) {
	// TODO(DET-9921): Ticket or add this, was missing previously.
	return nil, rmerrors.ErrNotSupported
}

// GetAgents implements rm.ResourceManager.
func (k *ResourceManager) GetAgents(msg *apiv1.GetAgentsRequest) (*apiv1.GetAgentsResponse, error) {
	return k.podsService.GetAgents(msg), nil
}

// GetAllocationSummaries implements rm.ResourceManager.
func (k *ResourceManager) GetAllocationSummaries(
	msg sproto.GetAllocationSummaries,
) (map[model.AllocationID]sproto.AllocationSummary, error) {
	summaries := make(map[model.AllocationID]sproto.AllocationSummary)
	for _, rp := range k.pools {
		rpSummaries := rp.GetAllocationSummaries(msg)
		maps.Copy(summaries, rpSummaries)
	}
	return summaries, nil
}

// GetAllocationSummary implements rm.ResourceManager.
func (k *ResourceManager) GetAllocationSummary(msg sproto.GetAllocationSummary) (*sproto.AllocationSummary, error) {
	for _, rp := range k.pools {
		resp := rp.GetAllocationSummary(msg)
		if resp != nil {
			return resp, nil
		}
	}
	return nil, fmt.Errorf("allocation not found: %s", msg.ID)
}

// GetDefaultAuxResourcePool implements rm.ResourceManager.
func (k *ResourceManager) GetDefaultAuxResourcePool(
	sproto.GetDefaultAuxResourcePoolRequest,
) (sproto.GetDefaultAuxResourcePoolResponse, error) {
	if k.config.DefaultComputeResourcePool == "" {
		return sproto.GetDefaultAuxResourcePoolResponse{}, rmerrors.ErrNoDefaultResourcePool
	}
	return sproto.GetDefaultAuxResourcePoolResponse{PoolName: k.config.DefaultAuxResourcePool}, nil
}

// GetDefaultComputeResourcePool implements rm.ResourceManager.
func (k *ResourceManager) GetDefaultComputeResourcePool(
	sproto.GetDefaultComputeResourcePoolRequest,
) (sproto.GetDefaultComputeResourcePoolResponse, error) {
	if k.config.DefaultComputeResourcePool == "" {
		return sproto.GetDefaultComputeResourcePoolResponse{}, rmerrors.ErrNoDefaultResourcePool
	}
	return sproto.GetDefaultComputeResourcePoolResponse{PoolName: k.config.DefaultComputeResourcePool}, nil
}

// GetExternalJobs implements rm.ResourceManager.
func (ResourceManager) GetExternalJobs(sproto.GetExternalJobs) ([]*jobv1.Job, error) {
	return nil, rmerrors.ErrNotSupported
}

// GetJobQ implements rm.ResourceManager.
func (k *ResourceManager) GetJobQ(msg sproto.GetJobQ) (map[model.JobID]*sproto.RMJobInfo, error) {
	if msg.ResourcePool == "" {
		msg.ResourcePool = k.config.DefaultComputeResourcePool
	}

	rp, err := k.poolByName(msg.ResourcePool)
	if err != nil {
		return nil, err
	}
	resp := rp.GetJobQ(msg)
	return resp, nil
}

// GetJobQueueStatsRequest implements rm.ResourceManager.
func (k *ResourceManager) GetJobQueueStatsRequest(
	*apiv1.GetJobQueueStatsRequest,
) (*apiv1.GetJobQueueStatsResponse, error) {
	resp := &apiv1.GetJobQueueStatsResponse{
		Results: make([]*apiv1.RPQueueStat, 0),
	}

	for poolName, rp := range k.pools {
		qStats := apiv1.RPQueueStat{
			ResourcePool: poolName,
			Stats:        rp.GetJobQStats(sproto.GetJobQStats{}),
		}

		aggregates, err := k.fetchAvgQueuedTime(poolName)
		if err != nil {
			return nil, fmt.Errorf("fetch average queued time: %s", err)
		}
		qStats.Aggregates = aggregates

		resp.Results = append(resp.Results, &qStats)
	}

	return resp, nil
}

// GetResourcePools implements rm.ResourceManager.
func (k *ResourceManager) GetResourcePools(*apiv1.GetResourcePoolsRequest) (*apiv1.GetResourcePoolsResponse, error) {
	summaries := make([]*resourcepoolv1.ResourcePool, 0, len(k.poolsConfig))
	for _, pool := range k.poolsConfig {
		summary, err := k.createResourcePoolSummary(pool.PoolName)
		if err != nil {
			// Should only raise an error if the resource pool doesn't exist and that can't happen.
			// But best to handle it anyway in case the implementation changes in the future.
			return nil, err
		}

		jobStats, err := k.getPoolJobStats(pool)
		if err != nil {
			return nil, err
		}

		summary.Stats = jobStats
		summaries = append(summaries, summary)
	}
	return &apiv1.GetResourcePoolsResponse{ResourcePools: summaries}, nil
}

// GetSlot implements rm.ResourceManager.
// TODO(DET-9919): Implement GetSlot for Kubernetes RM.
func (ResourceManager) GetSlot(*apiv1.GetSlotRequest) (*apiv1.GetSlotResponse, error) {
	return nil, rmerrors.ErrNotSupported
}

// GetSlots implements rm.ResourceManager.
// TODO(DET-9919): Implement GetSlots for Kubernetes RM.
func (ResourceManager) GetSlots(*apiv1.GetSlotsRequest) (*apiv1.GetSlotsResponse, error) {
	return nil, rmerrors.ErrNotSupported
}

// MoveJob implements rm.ResourceManager.
func (k *ResourceManager) MoveJob(msg sproto.MoveJob) error {
	rp, err := k.poolByName(msg.ResourcePool)
	if err != nil {
		return fmt.Errorf("move job found no resource pool with name %s: %w", msg.ResourcePool, err)
	}
	return rp.MoveJob(msg)
}

// RecoverJobPosition implements rm.ResourceManager.
func (k *ResourceManager) RecoverJobPosition(msg sproto.RecoverJobPosition) {
	rp, err := k.poolByName(msg.ResourcePool)
	if err != nil {
		k.syslog.WithError(err).Warnf("recover job position found no resource pool with name %s", msg.ResourcePool)
		return
	}
	rp.RecoverJobPosition(msg)
}

// Release implements rm.ResourceManager.
func (k *ResourceManager) Release(msg sproto.ResourcesReleased) {
	rp, err := k.poolByName(msg.ResourcePool)
	if err != nil {
		k.syslog.WithError(err).Warnf("release found no resource pool with name %s",
			msg.ResourcePool)
		return
	}
	rp.ResourcesReleased(msg)
}

// SetAllocationName implements rm.ResourceManager.
func (k *ResourceManager) SetAllocationName(msg sproto.SetAllocationName) {
	rp, err := k.poolByName(msg.ResourcePool)
	if err != nil {
		k.syslog.WithError(err).Warnf("set allocation name found no resource pool with name %s",
			msg.ResourcePool)
		return
	}
	rp.SetAllocationName(msg)
}

// SetGroupMaxSlots implements rm.ResourceManager.
func (k *ResourceManager) SetGroupMaxSlots(msg sproto.SetGroupMaxSlots) {
	rp, err := k.poolByName(msg.ResourcePool)
	if err != nil {
		k.syslog.WithError(err).Warnf("set group max slots found no resource pool with name %s",
			msg.ResourcePool)
		return
	}
	rp.SetGroupMaxSlots(msg)
}

// SetGroupPriority implements rm.ResourceManager.
func (k *ResourceManager) SetGroupPriority(msg sproto.SetGroupPriority) error {
	rp, err := k.poolByName(msg.ResourcePool)
	if err != nil {
		return fmt.Errorf("set group priority found no resource pool with name %s: %w",
			msg.ResourcePool, err)
	}
	return rp.SetGroupPriority(msg)
}

// SetGroupWeight implements rm.ResourceManager.
func (k *ResourceManager) SetGroupWeight(msg sproto.SetGroupWeight) error {
	rp, err := k.poolByName(msg.ResourcePool)
	if err != nil {
		return fmt.Errorf("set group weight found no resource pool with name %s: %w",
			msg.ResourcePool, err)
	}
	return rp.SetGroupWeight(msg)
}

// ValidateCommandResources implements rm.ResourceManager.
func (k *ResourceManager) ValidateCommandResources(
	msg sproto.ValidateCommandResourcesRequest,
) (sproto.ValidateCommandResourcesResponse, error) {
	rp, err := k.poolByName(msg.ResourcePool)
	if err != nil {
		return sproto.ValidateCommandResourcesResponse{}, err
	}
	return rp.ValidateCommandResources(msg), nil
}

// getResourcePoolRef gets an actor ref to a resource pool by name.
func (k ResourceManager) resourcePoolExists(
	name string,
) error {
	resp, err := k.GetResourcePools(&apiv1.GetResourcePoolsRequest{})
	if err != nil {
		return err
	}

	for _, rp := range resp.ResourcePools {
		if rp.Name == name {
			return nil
		}
	}
	return fmt.Errorf("cannot find resource pool: %s", name)
}

// ResolveResourcePool resolves the resource pool completely.
func (k ResourceManager) ResolveResourcePool(
	name string,
	workspaceID int,
	slots int,
) (string, error) {
	ctx := context.TODO()
	defaultComputePool, defaultAuxPool, err := db.GetDefaultPoolsForWorkspace(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	// If the resource pool isn't set, fill in the default at creation time.
	if name == "" && slots == 0 {
		if defaultAuxPool == "" {
			req := sproto.GetDefaultAuxResourcePoolRequest{}
			resp, err := k.GetDefaultAuxResourcePool(req)
			if err != nil {
				return "", fmt.Errorf("defaulting to aux pool: %w", err)
			}
			return resp.PoolName, nil
		}
		name = defaultAuxPool
	}

	if name == "" && slots >= 0 {
		if defaultComputePool == "" {
			req := sproto.GetDefaultComputeResourcePoolRequest{}
			resp, err := k.GetDefaultComputeResourcePool(req)
			if err != nil {
				return "", fmt.Errorf("defaulting to compute pool: %w", err)
			}
			return resp.PoolName, nil
		}
		name = defaultComputePool
	}

	resp, err := k.GetResourcePools(&apiv1.GetResourcePoolsRequest{})
	if err != nil {
		return "", err
	}

	poolNames, _, err := db.ReadRPsAvailableToWorkspace(
		ctx, int32(workspaceID), 0, -1, rmutils.ResourcePoolsToConfig(resp.ResourcePools))
	if err != nil {
		return "", err
	}
	found := false
	for _, poolName := range poolNames {
		if name == poolName {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf(
			"resource pool %s does not exist or is not available to workspace ID %d",
			name, workspaceID)
	}

	if err := k.ValidateResourcePool(name); err != nil {
		return "", fmt.Errorf("validating pool: %w", err)
	}
	return name, nil
}

// ValidateResources ensures enough resources are available in the resource pool.
// This is a no-op for k8s.
func (k ResourceManager) ValidateResources(
	name string,
	slots int,
	command bool,
) error {
	return nil
}

// ValidateResourcePool validates that the named resource pool exists.
func (k ResourceManager) ValidateResourcePool(name string) error {
	return k.resourcePoolExists(name)
}

// ValidateResourcePoolAvailability checks the available resources for a given pool.
// This is a no-op for k8s.
func (k ResourceManager) ValidateResourcePoolAvailability(
	v *sproto.ValidateResourcePoolAvailabilityRequest,
) ([]command.LaunchWarning, error) {
	if err := k.resourcePoolExists(v.Name); err != nil {
		return nil, fmt.Errorf("%s is an invalid resource pool", v.Name)
	}

	return nil, nil
}

// NotifyContainerRunning receives a notification from the container to let
// the master know that the container is running.
func (k ResourceManager) NotifyContainerRunning(
	msg sproto.NotifyContainerRunning,
) error {
	// Kubernetes Resource Manager does not implement a handler for the
	// NotifyContainerRunning message, as it is only used on HPC
	// (High Performance Computing).
	return errors.New(
		"the NotifyContainerRunning message is unsupported for KubernetesResourceManager")
}

// IsReattachableOnlyAfterStarted always returns false for the k8s resource manager.
func (k ResourceManager) IsReattachableOnlyAfterStarted() bool {
	return false
}

// TaskContainerDefaults returns TaskContainerDefaults for the specified pool.
func (k ResourceManager) TaskContainerDefaults(
	pool string,
	fallbackConfig model.TaskContainerDefaultsConfig,
) (result model.TaskContainerDefaultsConfig, err error) {
	return k.getTaskContainerDefaults(taskContainerDefaults{fallbackDefault: fallbackConfig, resourcePool: pool}), nil
}

func (k *ResourceManager) podStatusUpdateCallback(msg sproto.UpdatePodStatus) {
	for _, rp := range k.pools {
		rp.UpdatePodStatus(msg)
	}
}

func (k *ResourceManager) poolByName(resourcePool string) (*kubernetesResourcePool, error) {
	if resourcePool == "" {
		return nil, errors.New("invalid call: cannot get a resource pool with no name")
	}
	rp, ok := k.pools[resourcePool]
	if !ok {
		return nil, fmt.Errorf("cannot find resource pool %s", resourcePool)
	}
	return rp, nil
}

func (k *ResourceManager) createResourcePoolSummary(
	poolName string,
) (*resourcepoolv1.ResourcePool, error) {
	pool, err := k.getResourcePoolConfig(poolName)
	if err != nil {
		return &resourcepoolv1.ResourcePool{}, err
	}

	// TODO actor refactor, this is just getting resourcePool[poolName].maxSlotsPerPod
	slotsPerAgent := 0
	if k.taskContainerDefaults.Kubernetes.MaxSlotsPerPod != nil {
		slotsPerAgent = *k.taskContainerDefaults.Kubernetes.MaxSlotsPerPod
	}
	if pool.TaskContainerDefaults != nil &&
		pool.TaskContainerDefaults.Kubernetes != nil &&
		pool.TaskContainerDefaults.Kubernetes.MaxSlotsPerPod != nil {
		slotsPerAgent = *pool.TaskContainerDefaults.Kubernetes.MaxSlotsPerPod
	}

	const na = "n/a"

	poolType := resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_K8S
	preemptible := k.config.GetPreemption()
	location := na
	imageID := ""
	instanceType := na

	accelerator := ""
	schedulerType := resourcepoolv1.SchedulerType_SCHEDULER_TYPE_KUBERNETES

	resp := &resourcepoolv1.ResourcePool{
		Name:                         pool.PoolName,
		Description:                  pool.Description,
		Type:                         poolType,
		SlotType:                     k.config.SlotType.Proto(),
		DefaultAuxPool:               k.config.DefaultAuxResourcePool == poolName,
		DefaultComputePool:           k.config.DefaultComputeResourcePool == poolName,
		Preemptible:                  preemptible,
		SlotsPerAgent:                int32(slotsPerAgent),
		AuxContainerCapacityPerAgent: int32(pool.MaxAuxContainersPerAgent),
		SchedulerType:                schedulerType,
		SchedulerFittingPolicy:       resourcepoolv1.FittingPolicy_FITTING_POLICY_KUBERNETES,
		Location:                     location,
		ImageId:                      imageID,
		InstanceType:                 instanceType,
		Details:                      &resourcepoolv1.ResourcePoolDetail{},
		Accelerator:                  accelerator,
	}

	rp, err := k.poolByName(poolName)
	if err != nil {
		return &resourcepoolv1.ResourcePool{}, err
	}

	resourceSummary, err := rp.getResourceSummary(getResourceSummary{})
	if err != nil {
		return &resourcepoolv1.ResourcePool{}, err
	}

	resp.NumAgents = int32(resourceSummary.numAgents)
	resp.SlotsAvailable = int32(resourceSummary.numTotalSlots)
	resp.SlotsUsed = int32(resourceSummary.numActiveSlots)
	resp.AuxContainerCapacity = int32(resourceSummary.maxNumAuxContainers)
	resp.AuxContainersRunning = int32(resourceSummary.numActiveAuxContainers)

	return resp, nil
}

func (k *ResourceManager) fetchAvgQueuedTime(pool string) (
	[]*jobv1.AggregateQueueStats, error,
) {
	aggregates := []model.ResourceAggregates{}
	err := db.Bun().NewSelect().Model(&aggregates).
		Where("aggregation_type = ?", "queued").
		Where("aggregation_key = ?", pool).
		Where("date >= CURRENT_TIMESTAMP - interval '30 days'").
		Order("date ASC").Scan(context.TODO())
	if err != nil {
		return nil, err
	}
	res := make([]*jobv1.AggregateQueueStats, 0)
	for _, record := range aggregates {
		res = append(res, &jobv1.AggregateQueueStats{
			PeriodStart: record.Date.Format("2006-01-02"),
			Seconds:     record.Seconds,
		})
	}
	today := float32(0)
	subq := db.Bun().NewSelect().TableExpr("allocations").Column("allocation_id").
		Where("resource_pool = ?", pool).
		Where("start_time >= CURRENT_DATE")
	err = db.Bun().NewSelect().TableExpr("task_stats").ColumnExpr(
		"avg(extract(epoch FROM end_time - start_time))",
	).Where("event_type = ?", "QUEUED").
		Where("end_time >= CURRENT_DATE AND allocation_id IN (?) ", subq).
		Scan(context.TODO(), &today)
	if err != nil {
		return nil, err
	}
	res = append(res, &jobv1.AggregateQueueStats{
		PeriodStart: time.Now().Format("2006-01-02"),
		Seconds:     today,
	})
	return res, nil
}

func (k *ResourceManager) getPoolJobStats(
	pool config.ResourcePoolConfig,
) (*jobv1.QueueStats, error) {
	rp, err := k.poolByName(pool.PoolName)
	if err != nil {
		return nil, err
	}

	jobStats := rp.GetJobQStats(sproto.GetJobQStats{})
	return jobStats, nil
}

func (k *ResourceManager) getResourcePoolConfig(poolName string) (
	config.ResourcePoolConfig, error,
) {
	for i := range k.poolsConfig {
		if k.poolsConfig[i].PoolName == poolName {
			return k.poolsConfig[i], nil
		}
	}
	return config.ResourcePoolConfig{}, errors.Errorf("cannot find resource pool %s", poolName)
}

type taskContainerDefaults struct {
	fallbackDefault model.TaskContainerDefaultsConfig
	resourcePool    string
}

func (k *ResourceManager) getTaskContainerDefaults(
	msg taskContainerDefaults,
) model.TaskContainerDefaultsConfig {
	result := msg.fallbackDefault
	// Iterate through configured pools looking for a TaskContainerDefaults setting.
	for _, pool := range k.poolsConfig {
		if msg.resourcePool == pool.PoolName {
			if pool.TaskContainerDefaults == nil {
				break
			}
			result = *pool.TaskContainerDefaults
		}
	}
	return result
}

// EnableAgent allows scheduling on a node that has been disabled.
func (k *ResourceManager) EnableAgent(
	req *apiv1.EnableAgentRequest,
) (resp *apiv1.EnableAgentResponse, err error) {
	return k.podsService.EnableAgent(req)
}

// DisableAgent prevents scheduling on a node and has the option to kill running jobs.
func (k *ResourceManager) DisableAgent(
	req *apiv1.DisableAgentRequest,
) (resp *apiv1.DisableAgentResponse, err error) {
	return k.podsService.DisableAgent(req)
}

// EnableSlot implements 'det slot enable...' functionality.
func (k ResourceManager) EnableSlot(
	req *apiv1.EnableSlotRequest,
) (resp *apiv1.EnableSlotResponse, err error) {
	return nil, rmerrors.ErrNotSupported
}

// DisableSlot implements 'det slot disable...' functionality.
func (k ResourceManager) DisableSlot(
	req *apiv1.DisableSlotRequest,
) (resp *apiv1.DisableSlotResponse, err error) {
	return nil, rmerrors.ErrNotSupported
}
