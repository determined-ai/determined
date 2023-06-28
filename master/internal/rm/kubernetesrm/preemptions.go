package kubernetesrm

import (
	"context"

	"github.com/sirupsen/logrus"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

type preemptCallbackFunc func(string)

type preemptionListener struct {
	cb         preemptCallbackFunc
	syslog     *logrus.Entry
	resultChan <-chan watch.Event
}

func newPreemptionListener(
	ctx context.Context,
	namespace string,
	podInterface typedV1.PodInterface,
	cb preemptCallbackFunc,
) (*preemptionListener, error) {
	// Check if there are pods to preempt on startup.
	pods, err := podInterface.List(ctx,
		metaV1.ListOptions{LabelSelector: determinedPreemptionLabel})
	if err != nil {
		return nil, err
	}

	rw, err := watchtools.NewRetryWatcher(pods.ResourceVersion, &cache.ListWatch{
		WatchFunc: func(options metaV1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = determinedPreemptionLabel
			return podInterface.Watch(ctx, options)
		},
	})
	if err != nil {
		return nil, err
	}

	// Log when pods are first added to the informer (at start-up).
	syslog := logrus.WithFields(logrus.Fields{
		"component": "preemptionListener",
		"namespace": namespace,
	})
	for i := range pods.Items {
		syslog.Debugf("preemption listener added: %s", pods.Items[i].Name)
		cb(pods.Items[i].Name)
	}

	return &preemptionListener{
		cb:         cb,
		syslog:     syslog,
		resultChan: rw.ResultChan(),
	}, nil
}

func (p *preemptionListener) run() {
	p.syslog.Info("preemption listener is starting")
	for e := range p.resultChan {
		if e.Type == watch.Error {
			p.syslog.WithField("error", e.Object).Warnf("preemption listener encountered error")
			continue
		}

		pod, ok := e.Object.(*k8sV1.Pod)
		if !ok {
			p.syslog.Warnf("error converting object type %T to *k8sV1.Pod: %+v", e, e)
			continue
		}

		p.cb(pod.Name)
	}
	p.syslog.Warn("preemption listener stopped unexpectedly")
}
