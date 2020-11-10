package kubernetes

import (
	"github.com/pkg/errors"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// messages that are sent to the informer.
type (
	startInformer struct{}
)

// messages that are sent by the informer.
type (
	podStatusUpdate struct {
		updatedPod *k8sV1.Pod
	}
)

type informer struct {
	podInterface typedV1.PodInterface
	namespace    string
	podsHandler  *actor.Ref
}

func newInformer(
	podInterface typedV1.PodInterface,
	namespace string,
	podsHandler *actor.Ref,
) *informer {
	return &informer{
		podInterface: podInterface,
		namespace:    namespace,
		podsHandler:  podsHandler,
	}
}

// Receive implements the actor interface.
func (i *informer) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), startInformer{})

	case startInformer:
		if err := i.startInformer(ctx); err != nil {
			return err
		}

	case actor.PostStop:

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (i *informer) startInformer(ctx *actor.Context) error {
	watch, err := i.podInterface.Watch(metaV1.ListOptions{LabelSelector: determinedLabel})
	if err != nil {
		return errors.Wrap(err, "error initializing pod watch")
	}

	ctx.Log().Info("pod informer is starting")
	for event := range watch.ResultChan() {
		pod, ok := event.Object.(*k8sV1.Pod)
		if !ok {
			ctx.Log().Errorf("error converting event of type %T to *k8sV1.Pod: %+v", event, event)
			continue
		}

		if pod.Namespace != i.namespace {
			continue
		}

		ctx.Log().Debugf("informer got new pod event for pod: %s %s", pod.Name, pod.Status.Phase)
		ctx.Tell(i.podsHandler, podStatusUpdate{updatedPod: pod})
	}

	ctx.Log().Warn("pod informer stopped unexpectedly")
	ctx.Tell(ctx.Self(), startInformer{})

	return nil
}
