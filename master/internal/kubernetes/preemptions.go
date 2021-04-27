package kubernetes

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"
)

// Messages that are sent to the preemption listener.
type startPreemptionListener struct{}

// Messages that are sent by the preemption listener.
type podPreemption struct {
	podName string
}

type preemptionListener struct {
	clientSet   *k8sClient.Clientset
	namespace   string
	podsHandler *actor.Ref
}

func newPreemptionListener(
	clientSet *k8sClient.Clientset,
	namespace string,
	podsHandler *actor.Ref,
) *preemptionListener {
	return &preemptionListener{
		clientSet:   clientSet,
		namespace:   namespace,
		podsHandler: podsHandler,
	}
}

func (p *preemptionListener) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), startPreemptionListener{})

	case startPreemptionListener:
		if err := p.startPreemptionListener(ctx); err != nil {
			return err
		}

	case actor.PostStop:

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (p *preemptionListener) startPreemptionListener(ctx *actor.Context) error {
	watch, err := p.clientSet.CoreV1().Pods(p.namespace).Watch(
		metaV1.ListOptions{LabelSelector: "determined-preemption"})
	if err != nil {
		return errors.Wrap(err, "error initializing preemption watch")
	}

	ctx.Log().Info("preemption listener is starting")
	for pod := range watch.ResultChan() {
		newPod, ok := pod.Object.(*k8sV1.Pod)
		if !ok {
			ctx.Log().Warnf("error converting object type %T to *k8sV1.Pod: %+v", pod, pod)
			continue
		}
		ctx.Tell(p.podsHandler, podPreemption{podName: newPod.Name})
	}

	ctx.Log().Warn("preemption listener stopped unexpectedly")
	ctx.Tell(ctx.Self(), startPreemptionListener{})

	return nil
}
