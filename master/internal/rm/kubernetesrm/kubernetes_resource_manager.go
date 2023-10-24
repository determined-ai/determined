package kubernetesrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/rmutils"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
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
	// ActionCoolDown is the rate limit for job submission.
	ActionCoolDown = 500 * time.Millisecond
)

// SchedulerTick notifies the Resource Manager to submit pending jobs.
type SchedulerTick struct{}

// ResourceManager is a resource manager that manages k8s resources.
type ResourceManager struct {
	*actorrm.ResourceManager
}

// New returns a new ResourceManager, which communicates with
// and submits work to a Kubernetes apiserver.
func New(
	system *actor.System,
	db *db.PgDB,
	echo *echo.Echo,
	config *config.ResourceConfig,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) ResourceManager {
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

	ref, _ := system.ActorOf(
		sproto.K8sRMAddr,
		newKubernetesResourceManager(
			config.ResourceManager.KubernetesRM,
			config.ResourcePools,
			taskContainerDefaults,
			echo,
			tlsConfig,
			opts.LoggingOptions,
			db,
		),
	)
	system.Ask(ref, actor.Ping{}).Get()
	return ResourceManager{ResourceManager: actorrm.Wrap(ref)}
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
	name string,
	slots int,
) ([]command.LaunchWarning, error) {
	return nil, k.resourcePoolExists(name)
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
	req := taskContainerDefaults{fallbackDefault: fallbackConfig, resourcePool: pool}
	return result, k.Ask(req, &result)
}

// kubernetesResourceProvider manages the lifecycle of k8s resources.
type kubernetesResourceManager struct {
	config                *config.KubernetesResourceManagerConfig
	poolsConfig           []config.ResourcePoolConfig
	taskContainerDefaults *model.TaskContainerDefaultsConfig

	podsService *pods
	pools       map[string]*kubernetesResourcePool

	echoRef         *echo.Echo
	masterTLSConfig model.TLSClientConfig
	loggingConfig   model.LoggingConfig

	db *db.PgDB
}

func newKubernetesResourceManager(
	config *config.KubernetesResourceManagerConfig,
	poolsConfig []config.ResourcePoolConfig,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
	echoRef *echo.Echo,
	masterTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
	db *db.PgDB,
) *kubernetesResourceManager {
	return &kubernetesResourceManager{
		config:                config,
		poolsConfig:           poolsConfig,
		taskContainerDefaults: taskContainerDefaults,

		pools: make(map[string]*kubernetesResourcePool),

		echoRef:         echoRef,
		masterTLSConfig: masterTLSConfig,
		loggingConfig:   loggingConfig,

		db: db,
	}
}

func (k *kubernetesResourceManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		poolNamespaces := make(map[string]string)
		for i := range k.poolsConfig {
			if k.poolsConfig[i].KubernetesNamespace == "" {
				k.poolsConfig[i].KubernetesNamespace = k.config.Namespace
			}

			poolNamespaces[k.poolsConfig[i].KubernetesNamespace] = k.poolsConfig[i].PoolName
		}

		k.podsService = newPodsService(
			ctx.Self().System(),
			k.echoRef,
			ctx.Self(),
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
				t := time.NewTicker(ActionCoolDown)
				defer t.Stop()
				for range t.C {
					rp.Schedule(ctx.Self().System())
				}
			}()
			k.pools[poolConfig.PoolName] = rp
		}

	case sproto.AllocateRequest:
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
			ctx.Respond(err)
			return nil
		}
		rp.AllocateRequest(msg)

	case sproto.ResourcesReleased:
		for _, rp := range k.pools {
			rp.ResourcesReleased(msg)
		}

	case sproto.SetGroupMaxSlots:
		for _, rp := range k.pools {
			rp.SetGroupMaxSlots(msg)
		}
	case sproto.SetGroupWeight:
		for _, rp := range k.pools {
			err := rp.SetGroupWeight(msg)
			if err != nil {
				ctx.Respond(err)
				return nil
			}
		}
	case sproto.SetGroupPriority:
		for _, rp := range k.pools {
			err := rp.SetGroupPriority(msg)
			if err != nil {
				ctx.Respond(err)
				return nil
			}
		}
	case sproto.MoveJob:
		for _, rp := range k.pools {
			err := rp.MoveJob(msg)
			if err != nil {
				ctx.Respond(err)
				return nil
			}
		}

	case sproto.PendingPreemption:
		ctx.Respond(actor.ErrUnexpectedMessage(ctx))
		return nil

	case sproto.DeleteJob:
		// For now, there is nothing to clean up in k8s.
		ctx.Respond(sproto.EmptyDeleteJobResponse())

	case sproto.RecoverJobPosition:
		rp, err := k.poolByName(msg.ResourcePool)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		rp.RecoverJobPosition(msg)

	case sproto.GetAllocationSummary:
		for _, rp := range k.pools {
			resp := rp.GetAllocationSummary(msg)
			if resp != nil {
				ctx.Respond(resp)
				return nil
			}
		}

	case sproto.GetAllocationSummaries:
		summaries := make(map[model.AllocationID]sproto.AllocationSummary)
		for _, rp := range k.pools {
			rpSummaries := rp.GetAllocationSummaries(msg)
			maps.Copy(summaries, rpSummaries)
		}
		ctx.Respond(summaries)

	case sproto.SetAllocationName:
		for _, rp := range k.pools {
			rp.SetAllocationName(msg)
		}

	case sproto.GetDefaultComputeResourcePoolRequest:
		if k.config.DefaultComputeResourcePool == "" {
			ctx.Respond(rmerrors.ErrNoDefaultResourcePool)
		} else {
			ctx.Respond(sproto.GetDefaultComputeResourcePoolResponse{
				PoolName: k.config.DefaultComputeResourcePool,
			})
		}

	case sproto.GetDefaultAuxResourcePoolRequest:
		if k.config.DefaultComputeResourcePool == "" {
			ctx.Respond(rmerrors.ErrNoDefaultResourcePool)
		} else {
			ctx.Respond(sproto.GetDefaultAuxResourcePoolResponse{PoolName: k.config.DefaultAuxResourcePool})
		}

	case sproto.ValidateCommandResourcesRequest:
		rp, err := k.poolByName(msg.ResourcePool)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(rp.ValidateCommandResources(msg))

	case *apiv1.GetResourcePoolsRequest:
		summaries := make([]*resourcepoolv1.ResourcePool, 0, len(k.poolsConfig))
		for _, pool := range k.poolsConfig {
			summary, err := k.createResourcePoolSummary(ctx, pool.PoolName)
			if err != nil {
				// Should only raise an error if the resource pool doesn't exist and that can't happen.
				// But best to handle it anyway in case the implementation changes in the future.
				ctx.Log().WithError(err).Error("")
				ctx.Respond(err)
				return nil
			}

			jobStats, err := k.getPoolJobStats(ctx, pool)
			if err != nil {
				ctx.Respond(err)
			}

			summary.Stats = jobStats
			summaries = append(summaries, summary)
		}
		resp := &apiv1.GetResourcePoolsResponse{ResourcePools: summaries}
		ctx.Respond(resp)

	case sproto.GetJobQ:
		if msg.ResourcePool == "" {
			msg.ResourcePool = k.config.DefaultComputeResourcePool
		}

		rp, err := k.poolByName(msg.ResourcePool)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(rp.GetJobQ(msg))

	case *apiv1.GetJobQueueStatsRequest:
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
				ctx.Respond(fmt.Errorf("fetch average queued time: %s", err))
				return nil
			}
			qStats.Aggregates = aggregates

			resp.Results = append(resp.Results, &qStats)
		}

		ctx.Respond(resp)
		return nil

	case taskContainerDefaults:
		ctx.Respond(k.getTaskContainerDefaults(msg))

	case sproto.UpdatePodStatus:
		for _, rp := range k.pools {
			rp.UpdatePodStatus(msg)
		}

	case *apiv1.GetAgentsRequest:
		ctx.Respond(k.podsService.GetAgents(msg))

	case *apiv1.EnableAgentRequest:
		ctx.RespondCheckError(k.podsService.EnableAgent(msg))

	case *apiv1.DisableAgentRequest:
		ctx.RespondCheckError(k.podsService.DisableAgent(msg))

	case sproto.GetExternalJobs:
		ctx.Respond(rmerrors.ErrNotSupported)

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (k *kubernetesResourceManager) poolByName(resourcePool string) (*kubernetesResourcePool, error) {
	rp, ok := k.pools[resourcePool]
	if !ok {
		return nil, fmt.Errorf("cannot find resource pool %s", resourcePool)
	}
	return rp, nil
}

type taskContainerDefaults struct {
	fallbackDefault model.TaskContainerDefaultsConfig
	resourcePool    string
}

func (k *kubernetesResourceManager) createResourcePoolSummary(
	ctx *actor.Context,
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

func (k *kubernetesResourceManager) fetchAvgQueuedTime(pool string) (
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

func (k *kubernetesResourceManager) getPoolJobStats(
	ctx *actor.Context, pool config.ResourcePoolConfig,
) (*jobv1.QueueStats, error) {
	rp, err := k.poolByName(pool.PoolName)
	if err != nil {
		return nil, err
	}

	jobStats := rp.GetJobQStats(sproto.GetJobQStats{})
	return jobStats, nil
}

func (k *kubernetesResourceManager) getResourcePoolConfig(poolName string) (
	config.ResourcePoolConfig, error,
) {
	for i := range k.poolsConfig {
		if k.poolsConfig[i].PoolName == poolName {
			return k.poolsConfig[i], nil
		}
	}
	return config.ResourcePoolConfig{}, errors.Errorf("cannot find resource pool %s", poolName)
}

func (k *kubernetesResourceManager) getTaskContainerDefaults(
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
func (k ResourceManager) EnableAgent(
	req *apiv1.EnableAgentRequest,
) (resp *apiv1.EnableAgentResponse, err error) {
	return resp, k.Ask(req, &resp)
}

// DisableAgent prevents scheduling on a node and has the option to kill running jobs.
func (k ResourceManager) DisableAgent(
	req *apiv1.DisableAgentRequest,
) (resp *apiv1.DisableAgentResponse, err error) {
	return resp, k.Ask(req, &resp)
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
