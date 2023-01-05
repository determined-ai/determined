package kubernetesrm

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const (
	// KubernetesDummyResourcePool is the name of the dummy resource pool for kubernetes.
	KubernetesDummyResourcePool = "kubernetes"
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
	ctx actor.Messenger,
	name string,
	slots int,
) (string, error) {
	// If the resource pool isn't set, fill in the default at creation time.
	if name == "" && slots == 0 {
		req := sproto.GetDefaultAuxResourcePoolRequest{}
		resp, err := k.GetDefaultAuxResourcePool(ctx, req)
		if err != nil {
			return "", fmt.Errorf("defaulting to aux pool: %w", err)
		}
		return resp.PoolName, nil
	}

	if name == "" && slots >= 0 {
		req := sproto.GetDefaultComputeResourcePoolRequest{}
		resp, err := k.GetDefaultComputeResourcePool(ctx, req)
		if err != nil {
			return "", fmt.Errorf("defaulting to compute pool: %w", err)
		}
		return resp.PoolName, nil
	}

	if err := k.ValidateResourcePool(ctx, name); err != nil {
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

// GetDefaultComputeResourcePool requests the default compute resource pool.
func (k ResourceManager) GetDefaultComputeResourcePool(
	ctx actor.Messenger,
	msg sproto.GetDefaultComputeResourcePoolRequest,
) (sproto.GetDefaultComputeResourcePoolResponse, error) {
	return sproto.GetDefaultComputeResourcePoolResponse{
		PoolName: KubernetesDummyResourcePool,
	}, nil
}

// GetDefaultAuxResourcePool requests the default aux resource pool.
func (k ResourceManager) GetDefaultAuxResourcePool(
	ctx actor.Messenger,
	msg sproto.GetDefaultAuxResourcePoolRequest,
) (sproto.GetDefaultAuxResourcePoolResponse, error) {
	return sproto.GetDefaultAuxResourcePoolResponse{
		PoolName: KubernetesDummyResourcePool,
	}, nil
}

// GetAgents gets the state of connected agents. Go around the RM and directly to the pods actor
// to avoid blocking through it.
func (k ResourceManager) GetAgents(
	ctx actor.Messenger,
	msg *apiv1.GetAgentsRequest,
) (resp *apiv1.GetAgentsResponse, err error) {
	return resp, actorrm.AskAt(k.Ref().System(), sproto.PodsAddr, msg, &resp)
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

// kubernetesResourceProvider manages the lifecycle of k8s resources.
type kubernetesResourceManager struct {
	config    *config.KubernetesResourceManagerConfig
	podsActor *actor.Ref
	pools     map[string]*actor.Ref

	echoRef         *echo.Echo
	masterTLSConfig model.TLSClientConfig
	loggingConfig   model.LoggingConfig
}

func newKubernetesResourceManager(
	config *config.KubernetesResourceManagerConfig,
	echoRef *echo.Echo,
	masterTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
) actor.Actor {
	return &kubernetesResourceManager{
		config: config,

		pools: make(map[string]*actor.Ref),

		echoRef:         echoRef,
		masterTLSConfig: masterTLSConfig,
		loggingConfig:   loggingConfig,
	}
}

func (k *kubernetesResourceManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		k.podsActor = Initialize(
			ctx.Self().System(),
			k.echoRef,
			ctx.Self(),
			k.config.Namespace,
			k.config.MasterServiceName,
			k.masterTLSConfig,
			k.loggingConfig,
			k.config.LeaveKubernetesResources,
			k.config.DefaultScheduler,
			k.config.SlotType,
			config.PodSlotResourceRequests{CPU: k.config.SlotResourceRequests.CPU},
			k.config.Fluent,
			k.config.CredsDir,
			k.config.MasterIP,
			k.config.MasterPort,
		)

		k.pools[KubernetesDummyResourcePool] = ctx.MustActorOf(
			KubernetesDummyResourcePool, newResourcePool(k.config, k.podsActor),
		)

	case
		tasklist.GroupActorStopped,
		sproto.SetGroupMaxSlots,
		sproto.SetAllocationName,
		sproto.AllocateRequest,
		sproto.ResourcesReleased,
		sproto.UpdatePodStatus,
		sproto.PendingPreemption,
		sproto.GetJobQ,
		sproto.GetJobQStats,
		sproto.SetGroupWeight,
		sproto.SetGroupPriority,
		sproto.MoveJob,
		sproto.DeleteJob,
		sproto.RecoverJobPosition,
		*apiv1.GetJobQueueStatsRequest,
		sproto.GetAllocationHandler,
		sproto.GetAllocationSummary,
		sproto.GetAllocationSummaries:
		k.forwardToPool(ctx, KubernetesDummyResourcePool, msg)

	case *apiv1.GetResourcePoolsRequest:
		summaryMap := ctx.AskAll(msg, ctx.Children()...).GetAll()
		summaries := make([]*resourcepoolv1.ResourcePool, 0, len(k.pools))
		for _, v := range summaryMap {
			summaries = append(summaries, v.(*resourcepoolv1.ResourcePool))
		}
		ctx.Respond(&apiv1.GetResourcePoolsResponse{ResourcePools: summaries})

	case sproto.ValidateCommandResourcesRequest:
		fulfillable := k.config.MaxSlotsPerPod >= msg.Slots
		ctx.Respond(sproto.ValidateCommandResourcesResponse{Fulfillable: fulfillable})

	case *apiv1.GetAgentsRequest:
		resp := ctx.Ask(k.podsActor, msg)
		ctx.Respond(resp.Get())

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (k *kubernetesResourceManager) forwardToPool(
	ctx *actor.Context, resourcePool string, msg actor.Message,
) {
	if k.pools[resourcePool] == nil {
		sender := "unknown"
		if ctx.Sender() != nil {
			sender = ctx.Sender().Address().String()
		}
		err := errors.Errorf("cannot find resource pool %s for message %T from actor %s",
			resourcePool, ctx.Message(), sender)
		ctx.Log().WithError(err).Error("")
		if ctx.ExpectingResponse() {
			ctx.Respond(err)
		}
		return
	}

	if ctx.ExpectingResponse() {
		response := ctx.Ask(k.pools[resourcePool], msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(k.pools[resourcePool], msg)
	}
}

// TaskContainerDefaults returns TaskContainerDefaults for the specified pool.
func (k ResourceManager) TaskContainerDefaults(
	ctx actor.Messenger,
	pool string,
	fallbackConfig model.TaskContainerDefaultsConfig,
) (model.TaskContainerDefaultsConfig, error) {
	return fallbackConfig, nil
}
