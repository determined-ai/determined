package kubernetesrm

import (
	"context"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/set"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"
)

// Messages that are sent to the preemption listener.
type startPreemptionListener struct{}

type preemptionListener struct {
	clientSet   *k8sClient.Clientset
	podsHandler *actor.Ref
	namespaces  set.Set[string]
}

func newPreemptionListener(
	clientSet *k8sClient.Clientset,
	podsHandler *actor.Ref,
	namespaces set.Set[string],
) *preemptionListener {
	return &preemptionListener{
		clientSet:   clientSet,
		podsHandler: podsHandler,
		namespaces:  namespaces,
	}
}

func (p *preemptionListener) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), startPreemptionListener{})

	case startPreemptionListener:
		p.startPreemptionListener(ctx)

	case actor.PostStop:

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (p *preemptionListener) startPreemptionListener(ctx *actor.Context) {
	// check if there are pods to preempt on startup
	pods, err := p.clientSet.CoreV1().Pods(metaV1.NamespaceAll).List(
		context.TODO(), metaV1.ListOptions{LabelSelector: determinedPreemptionLabel})
	if err != nil {
		ctx.Log().WithError(err).Warnf(
			"error in initializing preemption listener: checking for pods to preempt",
		)
		actors.NotifyAfter(ctx, defaultInformerBackoff, startPreemptionListener{})
		return
	}

	for _, pod := range pods.Items {
		if !p.namespaces.Contains(pod.Namespace) {
			continue
		}

		ctx.Tell(p.podsHandler, PreemptTaskPod{PodName: pod.Name})
	}

	rw, err := watchtools.NewRetryWatcher(pods.ResourceVersion, &cache.ListWatch{
		WatchFunc: func(options metaV1.ListOptions) (watch.Interface, error) {
			return p.clientSet.CoreV1().Pods(metaV1.NamespaceAll).Watch(
				context.TODO(), metaV1.ListOptions{LabelSelector: determinedPreemptionLabel})
		},
	})
	if err != nil {
		ctx.Log().WithError(err).Warnf("error initializing preemption watch")
		actors.NotifyAfter(ctx, defaultInformerBackoff, startPreemptionListener{})
		return
	}

	ctx.Log().Info("preemption listener is starting")
	for e := range rw.ResultChan() {
		if e.Type == watch.Error {
			ctx.Log().WithField("error", e.Object).Warnf("preemption listener encountered error")
			continue
		}

		pod, ok := e.Object.(*k8sV1.Pod)
		if !ok {
			ctx.Log().Warnf("error converting object type %T to *k8sV1.Pod: %+v", e, e)
			continue
		}
		if !p.namespaces.Contains(pod.Namespace) {
			continue
		}

		ctx.Tell(p.podsHandler, PreemptTaskPod{PodName: pod.Name})
	}

	ctx.Log().Warn("preemption listener stopped unexpectedly")
	ctx.Tell(ctx.Self(), startPreemptionListener{})
}
