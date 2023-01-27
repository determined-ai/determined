package kubernetesrm

import (
	"context"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8sClient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/set"
)

// Messages that are sent to the event listener.
type (
	startEventListener struct{}
)

// Messages that are sent by the event listener.
type (
	podEventUpdate struct {
		event *k8sV1.Event
	}
)

type eventListener struct {
	clientSet   *k8sClient.Clientset
	podsHandler *actor.Ref
	namespaces  set.Set[string]
}

func newEventListener(
	clientSet *k8sClient.Clientset,
	podsHandler *actor.Ref,
	namespaces set.Set[string],
) *eventListener {
	return &eventListener{
		clientSet:   clientSet,
		podsHandler: podsHandler,
		namespaces:  namespaces,
	}
}

func (e *eventListener) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), startEventListener{})

	case startEventListener:
		e.startEventListener(ctx)

	case actor.PostStop:

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (e *eventListener) startEventListener(ctx *actor.Context) {
	events, err := e.clientSet.CoreV1().Events(metaV1.NamespaceAll).List(
		context.TODO(), metaV1.ListOptions{})
	if err != nil {
		ctx.Log().WithError(err).Warnf("error retrieving internal resource version")
		actors.NotifyAfter(ctx, defaultInformerBackoff, startEventListener{})
		return
	}

	rw, err := watchtools.NewRetryWatcher(events.ResourceVersion, &cache.ListWatch{
		WatchFunc: func(options metaV1.ListOptions) (watch.Interface, error) {
			return e.clientSet.CoreV1().Events(metaV1.NamespaceAll).Watch(
				context.TODO(), metaV1.ListOptions{})
		},
	})
	if err != nil {
		ctx.Log().WithError(err).Warnf("error initializing event retry watcher")
		actors.NotifyAfter(ctx, defaultInformerBackoff, startEventListener{})
		return
	}

	ctx.Log().Info("event listener is starting")
	for event := range rw.ResultChan() {
		if event.Type == watch.Error {
			ctx.Log().WithField("error", event.Object).Warnf("event listener encountered error")
			continue
		}

		newEvent, ok := event.Object.(*k8sV1.Event)
		if !ok {
			ctx.Log().Warnf("error converting object type %T to *k8sV1.Event: %+v", event, event)
			continue
		}

		if !e.namespaces.Contains(newEvent.Namespace) {
			continue
		}

		ctx.Tell(e.podsHandler, podEventUpdate{event: newEvent})
	}

	ctx.Log().Warn("event listener stopped unexpectedly")
	ctx.Tell(ctx.Self(), startEventListener{})
}
