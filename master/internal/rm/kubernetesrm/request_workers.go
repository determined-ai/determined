package kubernetesrm

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/actor"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type requestProcessingWorker struct {
	podInterfaces       map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface
}

func (r *requestProcessingWorker) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self().Parent(), workerAvailable{})
	case actor.PostStop:
		// This should not happen since the request worker actors would not stop during
		// the master is running.

	case createKubernetesResources:
		r.receiveCreateKubernetesResources(ctx, msg)
		ctx.Tell(ctx.Self().Parent(), workerAvailable{msg.handler})

	case deleteKubernetesResources:
		r.receiveDeleteKubernetesResources(ctx, msg)
		ctx.Tell(ctx.Self().Parent(), workerAvailable{})

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (r *requestProcessingWorker) receiveCreateKubernetesResources(
	ctx *actor.Context,
	msg createKubernetesResources,
) {
	configMap, err := r.configMapInterfaces[msg.podSpec.Namespace].Create(
		context.TODO(), msg.configMapSpec, metaV1.CreateOptions{})
	if err != nil {
		ctx.Log().WithField("handler", msg.handler.Address()).WithError(err).Errorf(
			"error creating configMap %s", msg.configMapSpec.Name)
		ctx.Tell(msg.handler, resourceCreationFailed{err: err})
		return
	}
	ctx.Log().WithField("handler", msg.handler.Address()).Infof(
		"created configMap %s", configMap.Name)

	ctx.Log().Debugf("launching pod with spec %v", msg.podSpec)
	pod, err := r.podInterfaces[msg.podSpec.Namespace].Create(
		context.TODO(), msg.podSpec, metaV1.CreateOptions{},
	)
	if err != nil {
		ctx.Log().WithField("handler", msg.handler.Address()).WithError(err).Errorf(
			"error creating pod %s", msg.podSpec.Name)
		ctx.Tell(msg.handler, resourceCreationFailed{err: err})
		return
	}
	ctx.Log().WithField("handler", msg.handler.Address()).Infof("created pod %s", pod.Name)
}

func (r *requestProcessingWorker) receiveDeleteKubernetesResources(
	ctx *actor.Context,
	msg deleteKubernetesResources,
) {
	var gracePeriod int64 = deletionGracePeriod
	var err error

	// If resource creation failed, we will still try to delete those resources which
	// will also result in a failure.
	if len(msg.podName) > 0 {
		err = r.podInterfaces[msg.namespace].Delete(
			context.TODO(), msg.podName, metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
		if err != nil {
			ctx.Log().WithField("handler", msg.handler.Address()).WithError(err).Errorf(
				"failed to delete pod %s", msg.podName)
		} else {
			ctx.Log().WithField("handler", msg.handler.Address()).Infof(
				"deleted pod %s", msg.podName)
		}
	}

	if len(msg.configMapName) > 0 {
		errDeletingConfigMap := r.configMapInterfaces[msg.namespace].Delete(
			context.TODO(), msg.configMapName,
			metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
		if errDeletingConfigMap != nil {
			ctx.Log().WithField("handler", msg.handler.Address()).WithError(err).Errorf(
				"failed to delete configMap %s", msg.configMapName)
			err = errDeletingConfigMap
		} else {
			ctx.Log().WithField("handler", msg.handler.Address()).Infof(
				"deleted configMap %s", msg.configMapName)
		}
	}

	// It is possible that the actor that sent the message is no longer around (if sent from
	// actor.PostStop). However this should have no impact on correctness.
	if err != nil {
		ctx.Tell(msg.handler, resourceDeletionFailed{err: err})
	}
}
