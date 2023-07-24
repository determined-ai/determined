package kubernetesrm

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

type callbackFunc func(watch.Event)

type informer struct {
	cb         callbackFunc
	name       string
	syslog     *logrus.Entry
	resultChan <-chan watch.Event
}

func newInformer(
	ctx context.Context,
	label string,
	name string,
	namespace string,
	podInterface typedV1.PodInterface,
	cb callbackFunc,
) (*informer, error) {
	pods, err := podInterface.List(ctx, metaV1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, err
	}

	rw, err := watchtools.NewRetryWatcher(pods.ResourceVersion, &cache.ListWatch{
		WatchFunc: func(options metaV1.ListOptions) (watch.Interface, error) {
			options.LabelSelector = label
			return podInterface.Watch(ctx, options)
		},
	})
	if err != nil {
		return nil, err
	}

	// Log when pods are first added to the informer (at start-up).
	syslog := logrus.WithFields(logrus.Fields{
		"component": fmt.Sprintf("%s-Informer", name),
		"namespace": namespace,
	})
	for i := range pods.Items {
		syslog.Debugf("informer added %s: %s", name, pods.Items[i].Name)
		cb(watch.Event{Object: &pods.Items[i]})
	}

	return &informer{
		cb:         cb,
		name:       name,
		syslog:     syslog,
		resultChan: rw.ResultChan(),
	}, nil
}

func newEventListener(
	ctx context.Context,
	eventInterface v1.EventInterface,
	namespace string,
	cb callbackFunc,
) (*informer, error) {
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
		cb(watch.Event{Object: &events.Items[i]})
	}

	return &informer{
		cb:         cb,
		name:       "event",
		syslog:     syslog,
		resultChan: rw.ResultChan(),
	}, nil
}

func newNodeInformer(
	ctx context.Context,
	nodeInterface typedV1.NodeInterface,
	cb callbackFunc,
) (*informer, error) {
	nodes, err := nodeInterface.List(ctx, metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	rw, err := watchtools.NewRetryWatcher(nodes.ResourceVersion, &cache.ListWatch{
		WatchFunc: func(options metaV1.ListOptions) (watch.Interface, error) {
			return nodeInterface.Watch(ctx, options)
		},
	})
	if err != nil {
		return nil, err
	}

	// Log when nodes are first added to the informer (at start-up).
	syslog := logrus.WithField("component", "nodeInformer")
	for i := range nodes.Items {
		syslog.Debugf("informer added node: %s", nodes.Items[i].Name)
		cb(watch.Event{Object: &nodes.Items[i], Type: watch.Added})
	}

	return &informer{
		cb:         cb,
		name:       "node",
		syslog:     syslog,
		resultChan: rw.ResultChan(),
	}, nil
}

func (i *informer) run() {
	i.syslog.Debugf("%s informer is starting", i.name)
	for event := range i.resultChan {
		if event.Type == watch.Error {
			i.syslog.Warnf("%s informer emitted error %+v", i.name, event)
			continue
		}
		i.cb(event)
	}
	panic(fmt.Sprintf("%s informer stopped unexpectedly", i.name))
}
