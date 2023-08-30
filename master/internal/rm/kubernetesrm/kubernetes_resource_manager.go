package kubernetesrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/rmutils"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
	"github.com/determined-ai/determined/master/pkg/syncx/mapx"
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
	ref, _ := system.ActorOf(
		sproto.K8sRMAddr,
		newKubernetesResourceManager(
			config.ResourceManager.KubernetesRM,
			config.ResourcePools,
			taskContainerDefaults,
			echo,
			tlsConfig,
			opts.LoggingOptions,
		),
	)
	system.Ask(ref, actor.Ping{}).Get()
	return ResourceManager{ResourceManager: actorrm.Wrap(ref)}
}

// GetResourcePoolRef gets an actor ref to a resource pool by name.
func (k ResourceManager) GetResourcePoolRef(
	ctx actor.Messenger,
	name string,
) (*actor.Ref, error) {
	rp := k.Ref().Child(name)
	if rp == nil {
		return nil, fmt.Errorf("cannot find resource pool: %s", name)
	}
	return rp, nil
}

// ResolveResourcePool resolves the resource pool completely.
func (k ResourceManager) ResolveResourcePool(
	actorCtx actor.Messenger,
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
			resp, err := k.GetDefaultAuxResourcePool(actorCtx, req)
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
			resp, err := k.GetDefaultComputeResourcePool(actorCtx, req)
			if err != nil {
				return "", fmt.Errorf("defaulting to compute pool: %w", err)
			}
			return resp.PoolName, nil
		}
		name = defaultComputePool
	}

	resp, err := k.GetResourcePools(actorCtx, &apiv1.GetResourcePoolsRequest{})
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

	if err := k.ValidateResourcePool(actorCtx, name); err != nil {
		return "", fmt.Errorf("validating pool: %w", err)
	}
	return name, nil
}

// ValidateResources ensures enough resources are available in the resource pool.
// This is a no-op for k8s.
func (k ResourceManager) ValidateResources(
	ctx actor.Messenger,
	name string,
	slots int,
	command bool,
) error {
	return nil
}

// ValidateResourcePool validates that the named resource pool exists.
func (k ResourceManager) ValidateResourcePool(ctx actor.Messenger, name string) error {
	_, err := k.GetResourcePoolRef(ctx, name)
	return err
}

// ValidateResourcePoolAvailability checks the available resources for a given pool.
// This is a no-op for k8s.
func (k ResourceManager) ValidateResourcePoolAvailability(
	ctx actor.Messenger,
	name string,
	slots int,
) ([]command.LaunchWarning, error) {
	if _, err := k.GetResourcePoolRef(ctx, name); err != nil {
		return nil, fmt.Errorf("%s is an invalid resource pool", name)
	}

	return nil, nil
}

// NotifyContainerRunning receives a notification from the container to let
// the master know that the container is running.
func (k ResourceManager) NotifyContainerRunning(
	ctx actor.Messenger,
	msg sproto.NotifyContainerRunning,
) error {
	// Kubernetes Resource Manager does not implement a handler for the
	// NotifyContainerRunning message, as it is only used on HPC
	// (High Performance Computing).
	return errors.New(
		"the NotifyContainerRunning message is unsupported for KubernetesResourceManager")
}

// IsReattachableOnlyAfterStarted always returns false for the k8s resource manager.
func (k ResourceManager) IsReattachableOnlyAfterStarted(ctx actor.Messenger) bool {
	return false
}

// IsReattachEnabled is true if any RP is configured to support it.
func (k ResourceManager) IsReattachEnabled(ctx actor.Messenger) bool {
	config := config.GetMasterConfig()

	for _, rpConfig := range config.ResourcePools {
		if rpConfig.AgentReattachEnabled {
			return true
		}
	}
	return false
}

// IsReattachEnabledForRP returns true, if the specified RP has AgentReattachEnabled.
func (k ResourceManager) IsReattachEnabledForRP(ctx actor.Messenger, rp string) bool {
	return isReattachEnabledForRP(rp)
}

func isReattachEnabledForRP(rp string) bool {
	config := config.GetMasterConfig()

	for _, rpConfig := range config.ResourcePools {
		if rpConfig.PoolName == rp && rpConfig.AgentReattachEnabled {
			return true
		}
	}
	return false
}

// kubernetesResourceProvider manages the lifecycle of k8s resources.
type kubernetesResourceManager struct {
	config                *config.KubernetesResourceManagerConfig
	poolsConfig           []config.ResourcePoolConfig
	taskContainerDefaults *model.TaskContainerDefaultsConfig

	eg errgroupx.Group

	pods  *pods
	pools map[string]*kubernetesResourcePool

	echoRef         *echo.Echo
	masterTLSConfig model.TLSClientConfig
	loggingConfig   model.LoggingConfig
}

func newKubernetesResourceManager(
	config *config.KubernetesResourceManagerConfig,
	poolsConfig []config.ResourcePoolConfig,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
	echoRef *echo.Echo,
	masterTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
) actor.Actor {
	return &kubernetesResourceManager{
		config:                config,
		poolsConfig:           poolsConfig,
		taskContainerDefaults: taskContainerDefaults,

		eg: errgroupx.WithContext(context.Background()),

		pools: make(map[string]*kubernetesResourcePool),

		echoRef:         echoRef,
		masterTLSConfig: masterTLSConfig,
		loggingConfig:   loggingConfig,
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

		podStatusUpdates := make(chan sproto.UpdatePodStatus)
		pods, err := newPodsService(
			k.config.Namespace,
			poolNamespaces,
			k.config.MasterServiceName,
			k.masterTLSConfig,
			k.loggingConfig,
			k.config.LeaveKubernetesResources,
			k.config.DefaultScheduler,
			k.config.SlotType,
			config.PodSlotResourceRequests{CPU: k.config.SlotResourceRequests.CPU},
			k.config.Fluent,
			k.poolsConfig,
			k.taskContainerDefaults,
			k.config.CredsDir,
			k.config.MasterIP,
			k.config.MasterPort,
			podStatusUpdates,
		)
		if err != nil {
			return fmt.Errorf("initializing pod service: %w", err)
		}
		k.pods = pods
		go func() {
			for update := range podStatusUpdates {
				ctx.Tell(ctx.Self(), update)
			}
		}()

		for _, poolConfig := range k.poolsConfig {
			poolConfig := poolConfig
			pool := newResourcePool(k.config, &poolConfig, k.pods)
			k.pools[poolConfig.PoolName] = pool
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
		k.forPool(msg.ResourcePool, func(pool *kubernetesResourcePool) error {
			pool.AllocateRequest(msg)
			return nil
		})

	case sproto.ResourcesReleased:
		k.forEachPool(func(name string, pool *kubernetesResourcePool) error {
			pool.ResourcesReleased(msg)
			return nil
		})
	case sproto.SetGroupMaxSlots:
		k.forEachPool(func(name string, pool *kubernetesResourcePool) error {
			pool.SetGroupMaxSlots(msg)
			return nil
		})
	case sproto.SetGroupWeight:
		if ctx.ExpectingResponse() {
			ctx.Respond(rmerrors.ErrUnsupported("set group weight is unsupported in k8s"))
		}
	case sproto.SetGroupPriority:
		ctx.Respond(k.forEachPool(func(name string, pool *kubernetesResourcePool) error {
			return pool.SetGroupPriority(msg)
		}))
	case sproto.MoveJob:
		ctx.Respond(k.forEachPool(func(name string, pool *kubernetesResourcePool) error {
			return pool.MoveJob(msg)
		}))

	case sproto.PendingPreemption:
		ctx.Respond(actor.ErrUnexpectedMessage(ctx))
		return nil

	case sproto.DeleteJob:
		// For now, there is nothing to clean up in k8s.
		ctx.Respond(sproto.EmptyDeleteJobResponse())

	case sproto.RecoverJobPosition:
		_ = k.forEachPool(func(name string, pool *kubernetesResourcePool) error {
			pool.RecoverJobPosition(msg)
			return nil
		})

	case sproto.GetAllocationSummary:
		var resp sproto.AllocationSummary
		err := k.forEachPool(func(name string, pool *kubernetesResourcePool) error {
			summary := pool.AllocationSummary(msg)
			if summary != nil {
				resp = *summary
			}
			return nil
		})
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(resp)

	case sproto.GetAllocationSummaries:
		summaries := mapx.New[model.AllocationID, sproto.AllocationSummary]()
		err := k.forEachPool(func(name string, pool *kubernetesResourcePool) error {
			resp := pool.AllocationSummaries(msg)
			for id, summary := range resp {
				summaries.Store(id, summary)
			}
			return nil
		})
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		var resp map[model.AllocationID]sproto.AllocationSummary
		summaries.WithLock(func(m map[model.AllocationID]sproto.AllocationSummary) {
			resp = m
		})
		ctx.Respond(resp)

	case sproto.SetAllocationName:
		_ = k.forEachPool(func(name string, pool *kubernetesResourcePool) error {
			pool.SetAllocationName(msg)
			return nil
		})

	case sproto.GetDefaultComputeResourcePoolRequest:
		ctx.Respond(sproto.GetDefaultComputeResourcePoolResponse{
			PoolName: k.config.DefaultComputeResourcePool,
		})

	case sproto.GetDefaultAuxResourcePoolRequest:
		ctx.Respond(sproto.GetDefaultAuxResourcePoolResponse{PoolName: k.config.DefaultAuxResourcePool})

	case sproto.ValidateCommandResourcesRequest:
		var resp sproto.ValidateCommandResourcesResponse
		k.forPool(msg.ResourcePool, func(pool *kubernetesResourcePool) error {
			resp = pool.ValidateCommandResources(msg)
			return nil
		})
		ctx.Respond(resp)

	case *apiv1.GetResourcePoolsRequest:
		summaries := make([]*resourcepoolv1.ResourcePool, 0, len(k.poolsConfig))
		for _, pool := range k.poolsConfig {
			summary, err := k.createResourcePoolSummary(ctx, pool.PoolName)
			if err != nil {
				// Should only raise an error if the resource pool doesn't exist and that can't happen.
				// But best to handle it anyway in case the implementation changes in the future.
				ctx.Log().WithError(err).Error("")
				ctx.Respond(err)
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

		rpRef := ctx.Child(msg.ResourcePool)
		if rpRef == nil {
			ctx.Respond(errors.Errorf("resource pool %s not found", msg.ResourcePool))
			return nil
		}
		resp := ctx.Ask(rpRef, msg).Get()
		ctx.Respond(resp)

	case *apiv1.GetJobQueueStatsRequest:
		resp := &apiv1.GetJobQueueStatsResponse{
			Results: make([]*apiv1.RPQueueStat, 0),
		}
		rpRefs := make([]*actor.Ref, 0)
		if len(msg.ResourcePools) == 0 {
			rpRefs = append(rpRefs, ctx.Children()...)
		} else {
			for _, rp := range msg.ResourcePools {
				rpRefs = append(rpRefs, ctx.Child(rp))
			}
		}

		actorResps := ctx.AskAll(sproto.GetJobQStats{}, rpRefs...).GetAll()
		for _, rpRef := range rpRefs {
			poolName := rpRef.Address().Local()
			qStats := apiv1.RPQueueStat{ResourcePool: poolName}
			aResp := actorResps[rpRef]
			switch aMsg := aResp.(type) {
			case error:
				ctx.Log().WithError(aMsg).Error("")
				ctx.Respond(aMsg)
				return nil
			case *jobv1.QueueStats:
				qStats.Stats = aMsg
				aggregates, err := k.fetchAvgQueuedTime(poolName)
				if err != nil {
					return fmt.Errorf("fetch average queued time: %s", err)
				}
				qStats.Aggregates = aggregates
				resp.Results = append(resp.Results, &qStats)
			default:
				return fmt.Errorf("unexpected response type: %T", aMsg)
			}
		}
		ctx.Respond(resp)
		return nil

	case sproto.GetJobQStats:
		resp := ctx.Ask(ctx.Child(msg.ResourcePool), msg).Get()
		ctx.Respond(resp)

	case taskContainerDefaults:
		ctx.Respond(k.getTaskContainerDefaults(msg))

	case tasklist.GroupActorStopped:
		_ = k.forEachPool(func(name string, pool *kubernetesResourcePool) error {
			pool.GroupActorStopped(msg.Ref)
			return nil
		})

	case sproto.UpdatePodStatus:
		_ = k.forEachPool(func(name string, pool *kubernetesResourcePool) error {
			pool.UpdatePodStatus(msg)
			return nil
		})

	case *apiv1.GetAgentsRequest:
		ctx.Respond(k.pods.GetAgents())

	case sproto.GetExternalJobs:
		ctx.Respond(rmerrors.ErrNotSupported)

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (k *kubernetesResourceManager) forPool(
	name string,
	f func(pool *kubernetesResourcePool) error,
) error {
	pool := k.pools[name]
	if pool == nil {
		return fmt.Errorf("no such pool: %s", name)
	}
	return f(pool)
}

// TODO(!!!): A lot of usages of `forEachPool` are a huge hack, some are straight up wrong.
func (k *kubernetesResourceManager) forEachPool(
	f func(name string, pool *kubernetesResourcePool) error,
) error {
	eg := errgroupx.WithContext(context.Background())
	for name, pool := range k.pools {
		eg.Go(func(ctx context.Context) error {
			return f(name, pool)
		})
	}
	return eg.Wait()
}

type taskContainerDefaults struct {
	fallbackDefault model.TaskContainerDefaultsConfig
	resourcePool    string
}

// TaskContainerDefaults returns TaskContainerDefaults for the specified pool.
func (k ResourceManager) TaskContainerDefaults(
	ctx actor.Messenger,
	pool string,
	fallbackConfig model.TaskContainerDefaultsConfig,
) (result model.TaskContainerDefaultsConfig, err error) {
	req := taskContainerDefaults{fallbackDefault: fallbackConfig, resourcePool: pool}
	return result, k.Ask(ctx, req, &result)
}

func (k *kubernetesResourceManager) aggregateTaskSummaries(
	resps map[*actor.Ref]actor.Message,
) map[model.AllocationID]sproto.AllocationSummary {
	summaries := make(map[model.AllocationID]sproto.AllocationSummary)
	for _, resp := range resps {
		if resp != nil {
			typed := resp.(map[model.AllocationID]sproto.AllocationSummary)
			for id, summary := range typed {
				summaries[id] = summary
			}
		}
	}
	return summaries
}

func (k *kubernetesResourceManager) createResourcePoolSummary(
	ctx *actor.Context,
	poolName string,
) (*resourcepoolv1.ResourcePool, error) {
	pool, err := k.getResourcePoolConfig(poolName)
	if err != nil {
		return &resourcepoolv1.ResourcePool{}, err
	}

	const na = "n/a"

	poolType := resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_K8S
	preemptible := k.config.GetPreemption()
	location := na
	imageID := ""
	instanceType := na
	slotsPerAgent := k.config.MaxSlotsPerPod

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

	var resourceSummary resourceSummary
	err = k.forPool(poolName, func(pool *kubernetesResourcePool) error {
		resp, err := pool.getResourceSummary()
		if err != nil {
			return err
		}
		resourceSummary = resp
		return nil
	})
	if err != nil {
		return nil, err
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

func (k *kubernetesResourceManager) aggregateTaskSummary(
	resps map[*actor.Ref]actor.Message,
) *sproto.AllocationSummary {
	for _, resp := range resps {
		if resp != nil {
			typed := resp.(sproto.AllocationSummary)
			return &typed
		}
	}
	return nil
}

func (k *kubernetesResourceManager) getPoolJobStats(
	ctx *actor.Context, pool config.ResourcePoolConfig,
) (*jobv1.QueueStats, error) {
	var resp *jobv1.QueueStats
	_ = k.forPool(pool.PoolName, func(pool *kubernetesResourcePool) error {
		resp = pool.JobQStats()
		return nil
	})
	return resp, nil
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

// EnableSlot implements 'det slot enable...' functionality.
func (k ResourceManager) EnableSlot(
	m actor.Messenger,
	req *apiv1.EnableSlotRequest,
) (resp *apiv1.EnableSlotResponse, err error) {
	return nil, rmerrors.ErrNotSupported
}

// DisableSlot implements 'det slot disable...' functionality.
func (k ResourceManager) DisableSlot(
	m actor.Messenger,
	req *apiv1.DisableSlotRequest,
) (resp *apiv1.DisableSlotResponse, err error) {
	return nil, rmerrors.ErrNotSupported
}
