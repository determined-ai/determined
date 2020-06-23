package scheduler

import (
	"github.com/determined-ai/determined/master/internal/kubernetes"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

// kubernetesResourceProvider manages the lifecycle of k8 resources.
type kubernetesResourceProvider struct {
	clusterID             string
	namespace             string
	slotsPerNode          int
	outOfCluster          bool
	kubeConfigPath        string
	proxy                 *actor.Ref
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig
}

// NewKubernetesResourceProvider initializes a new kubernetesResourceProvider.
func NewKubernetesResourceProvider(
	clusterID string,
	namespace string,
	slotsPerNode int,
	outOfCluster bool,
	kubeConfigPath string,
	proxy *actor.Ref,
	harnessPath string,
	taskContainerDefaults model.TaskContainerDefaultsConfig,
) actor.Actor {
	return &kubernetesResourceProvider{
		clusterID:             clusterID,
		namespace:             namespace,
		slotsPerNode:          slotsPerNode,
		outOfCluster:          outOfCluster,
		kubeConfigPath:        kubeConfigPath,
		proxy:                 proxy,
		harnessPath:           harnessPath,
		taskContainerDefaults: taskContainerDefaults,
	}
}

func (k *kubernetesResourceProvider) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:

	case sproto.ConfigureEndpoints:
		ctx.Log().Infof("initializing endpoints for pods")
		kubernetes.Initialize(
			msg.System,
			msg.Echo,
			ctx.Self(),
			k.namespace,
			k.outOfCluster,
			k.kubeConfigPath,
		)

	default:
		ctx.Log().Error("Unexpected message", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}
