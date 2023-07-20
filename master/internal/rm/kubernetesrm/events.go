package kubernetesrm

import (
	"context"

	"github.com/sirupsen/logrus"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

type eventCallbackFunc func(*k8sV1.Event)

type eventListener struct {
	cb         eventCallbackFunc
	syslog     *logrus.Entry
	resultChan <-chan watch.Event
}

func newEventListener(
	ctx context.Context,
	eventInterface v1.EventInterface,
	namespace string,
	cb eventCallbackFunc,
) (*eventListener, error) {
	events, err := eventInterface.List(ctx, metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	rw, err := watchtools.NewRetryWatcher(events.ResourceVersion, &cache.ListWatch{
		WatchFunc: func(options metaV1.ListOptions) (watch.Interface, error) {
			return eventInterface.Watch(ctx, metaV1.ListOptions{})
		},
	})
	if err != nil {
		return nil, err
	}

	// Log when pods are first added to the informer (at start-up).
	syslog := logrus.WithFields(logrus.Fields{
		"component": "eventListener",
		"namespace": namespace,
	})
	for i := range events.Items {
		syslog.Debugf("listener added event: %s", events.Items[i].Name)
		cb(&events.Items[i])
	}

	return &eventListener{
		cb:         cb,
		syslog:     syslog,
		resultChan: rw.ResultChan(),
	}, nil
}

func (e *eventListener) run() {
	e.syslog.Info("event listener is starting")
	for event := range e.resultChan {
		if event.Type == watch.Error {
			e.syslog.WithField("error", event.Object).Warnf("event listener encountered error")
			continue
		}

		newEvent, ok := event.Object.(*k8sV1.Event)
		if !ok {
			e.syslog.Warnf("error converting object type %T to *k8sV1.Event: %+v", event, event)
			continue
		}

		e.syslog.Debugf("listener got new event: %s", newEvent.Message)
		e.cb(newEvent)
	}
	panic("event listener stopped unexpectedly")
}
