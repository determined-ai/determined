package kubernetes

import (
	"time"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/pkg/actor"
)

const defaultInformerBackoff = 5 * time.Second

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
		i.startInformer(ctx)

	case actor.PostStop:

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (i *informer) startInformer(ctx *actor.Context) {
	pods, err := i.podInterface.List(metaV1.ListOptions{LabelSelector: determinedLabel})
	if err != nil {
		ctx.Log().WithError(err).Warnf("error retrieving internal resource version")
		actors.NotifyAfter(ctx, defaultInformerBackoff, startInformer{})
		return
	}

	rw, err := watchtools.NewRetryWatcher(pods.ResourceVersion, &cache.ListWatch{
		WatchFunc: func(options metaV1.ListOptions) (watch.Interface, error) {
			return i.podInterface.Watch(metaV1.ListOptions{LabelSelector: determinedLabel})
		},
	})
	if err != nil {
		ctx.Log().WithError(err).Warnf("error initializing pod retry watcher")
		actors.NotifyAfter(ctx, defaultInformerBackoff, startInformer{})
		return
	}

	ctx.Log().Info("pod informer is starting")
	for event := range rw.ResultChan() {
		if event.Type == watch.Error {
			ctx.Log().Warnf("pod informer emitted error %+v", event)
			continue
		}

		pod, ok := event.Object.(*k8sV1.Pod)
		if !ok {
			ctx.Log().Warnf("error converting event of type %T to *k8sV1.Pod: %+v", event, event)
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
}
