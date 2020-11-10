package kubernetes

import (
	"regexp"

	"github.com/pkg/errors"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"

	"github.com/determined-ai/determined/master/pkg/actor"
)

const gpuTextReplacement = "Waiting for resources. "

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
	namespace   string
	podsHandler *actor.Ref
}

func newEventListener(
	clientSet *k8sClient.Clientset,
	namespace string,
	podsHandler *actor.Ref,
) *eventListener {
	return &eventListener{
		clientSet:   clientSet,
		namespace:   namespace,
		podsHandler: podsHandler,
	}
}

func (e *eventListener) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), startEventListener{})

	case startEventListener:
		if err := e.startEventListener(ctx); err != nil {
			return err
		}

	case actor.PostStop:

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (e *eventListener) startEventListener(ctx *actor.Context) error {
	watch, err := e.clientSet.CoreV1().Events(e.namespace).Watch(metaV1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "error initializing event watch")
	}

	ctx.Log().Info("event listener is starting")
	for event := range watch.ResultChan() {
		newEvent, ok := event.Object.(*k8sV1.Event)
		if !ok {
			ctx.Log().Warnf("error converting object type %T to *k8sV1.Event", event)
			continue
		}
		e.modMessage(newEvent)
		ctx.Tell(e.podsHandler, podEventUpdate{event: newEvent})
	}

	ctx.Log().Warn("event listener stopped unexpectedly")
	ctx.Tell(ctx.Self(), startEventListener{})

	return nil
}

func (e *eventListener) modMessage(msg *k8sV1.Event) {
	replacements := map[string]string{
		"nodes are available":        gpuTextReplacement,
		"pod triggered scale-up":     "Job requires additional resources, scaling up cluster.",
		"Successfully assigned":      "Pod resources allocated.",
		"skip schedule deleting pod": "Deleting unscheduled pod.",
	}

	for k, v := range replacements {
		matched, err := regexp.MatchString(k, msg.Message)
		if err != nil {
			break
		} else if matched {
			if v == gpuTextReplacement {
				v += string(msg.Message[0]) + " GPUs available, "
			}
			msg.Message = v
		}
	}
}
