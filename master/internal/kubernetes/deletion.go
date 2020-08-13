package kubernetes

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// The kubernetesDeletionDealer and kubernetesDeletionWorker actors are responsible for processing
// the deletion of Kubernetes resources. The single kubernetesDeletionDealer actor receives
// deletion requests and forwards it to one of the kubernetesDeletionWorker in a round-robin
// manner.
//
// The reason that requests are processed by a pool of workers rather than processed
// asynchronously by the pod actors (the pods actor may also request to delete left-over resources
// at startup) is to avoid saturating the Kubernetes API server with deletion requests, which
// may make it less responsive to other requests (e.g., resource creation).
//
// The reason that a worker pool is used rather than a token system (the way creation is done),
// is that the pod actors often request to delete resources after stopping (in actor.PostStop),
// which makes it impossible to receive additional messages which would be necessary to request
// a token. We considered rewriting the pod actor to always delete resources before stopping,
// however we found that that would significantly increase the complexity of the pod actor.

const numKubernetesDeletionWorkers = 5

// message types received by the dealer and forwarded to the workers.
type (
	deleteKubernetesResources struct {
		handler       *actor.Ref
		podName       string
		configMapName string
	}
)

// message types sent by workers.
type (
	deletedKubernetesResources struct {
		err error
	}
)

// kubernetesDeletionDealer is responsible for distributed deletion requests
// amongst the workers in a round-robin manner.
type kubernetesDeletionDealer struct {
	podInterface       typedV1.PodInterface
	configMapInterface typedV1.ConfigMapInterface

	workers         []*actor.Ref
	nextWorkerIndex int
}

func newKubernetesDeletionDealer(
	podInterface typedV1.PodInterface,
	configMapInterface typedV1.ConfigMapInterface,
) *kubernetesDeletionDealer {
	return &kubernetesDeletionDealer{
		podInterface:       podInterface,
		configMapInterface: configMapInterface,
		workers:            make([]*actor.Ref, 0, numKubernetesDeletionWorkers),
	}
}

// Receive implements the actor interface.
func (k *kubernetesDeletionDealer) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		for i := 0; i < numKubernetesDeletionWorkers; i++ {
			newWorker, ok := ctx.ActorOf(
				fmt.Sprintf("kubernetes-deletion-worker-%d", i),
				&kubernetesDeletionWorker{
					podInterface:       k.podInterface,
					configMapInterface: k.configMapInterface,
				},
			)
			if !ok {
				return errors.Errorf("%s already exists", newWorker.Address())
			}
			k.workers = append(k.workers, newWorker)
		}

	case deleteKubernetesResources:
		ctx.Tell(k.workers[k.nextWorkerIndex], msg)
		k.nextWorkerIndex++
		k.nextWorkerIndex %= numKubernetesDeletionWorkers

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

// kubernetesClientWorker is responsible for deleting kubernetes resources.
type kubernetesDeletionWorker struct {
	podInterface       typedV1.PodInterface
	configMapInterface typedV1.ConfigMapInterface
}

// Receive implements the actor interface.
func (k *kubernetesDeletionWorker) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:

	case deleteKubernetesResources:
		k.receiveDeleteKubernetesResources(ctx, msg)

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (k *kubernetesDeletionWorker) receiveDeleteKubernetesResources(
	ctx *actor.Context,
	msg deleteKubernetesResources,
) {
	var gracePeriod int64 = 15
	var err error

	if len(msg.podName) > 0 {
		err = k.podInterface.Delete(msg.podName, &metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
		if err != nil {
			ctx.Log().WithField("handler", msg.handler.Address()).Errorf(
				"failed to delete pod %s: %s", msg.podName, err.Error())
		}
		ctx.Log().WithField("handler", msg.handler.Address()).Infof("deleted pod %s", msg.podName)
	}

	if len(msg.configMapName) > 0 {
		errDeletingConfigMap := k.configMapInterface.Delete(msg.configMapName, &metaV1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod})
		if errDeletingConfigMap != nil {
			ctx.Log().WithField("handler", msg.handler.Address()).Errorf(
				"failed to delete configMap %s: %s", msg.configMapName, errDeletingConfigMap.Error())
			err = errDeletingConfigMap
		}
		ctx.Log().WithField("handler", msg.handler.Address()).Infof(
			"deleted configMap %s", msg.configMapName)
	}

	// It is possible that the actor that sent the message is no longer around (if sent from
	// actor.PostStop). However this should have no impact on correctness.
	if err != nil {
		ctx.Tell(msg.handler, deletedKubernetesResources{err: err})
	}
}
