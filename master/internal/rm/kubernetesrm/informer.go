package kubernetesrm

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

type informerCallback func(watch.Event)

type informer struct {
	cb         informerCallback
	name       string
	syslog     *logrus.Entry
	resultChan <-chan watch.Event
}

func newPodInformer(
	ctx context.Context,
	label string,
	name string,
	namespace string,
	podInterface typedV1.PodInterface,
	cb informerCallback,
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
		"component": fmt.Sprintf("%s-informer", name),
		"namespace": namespace,
	})
	for i := range pods.Items {
		syslog.Debugf("initial inform added %s: %s", name, pods.Items[i].Name)
		cb(watch.Event{Object: &pods.Items[i]})
	}

	return &informer{
		cb:         cb,
		name:       name,
		syslog:     syslog,
		resultChan: rw.ResultChan(),
	}, nil
}

func newEventInformer(
	ctx context.Context,
	eventInterface typedV1.EventInterface,
	namespace string,
	cb informerCallback,
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
		"component": "event-informer",
		"namespace": namespace,
	})
	for i := range events.Items {
		syslog.Debugf("informer added event: %s", events.Items[i].Name)
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
	cb informerCallback,
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

func (i *informer) run(ctx context.Context) {
	i.syslog.Debugf("%s informer is starting", i.name)
	for {
		select {
		case event := <-i.resultChan:
			if event.Type == watch.Error {
				i.syslog.Warnf("%s informer emitted error %+v", i.name, event)
				continue
			}
			i.cb(event)
		case <-ctx.Done():
			i.syslog.Debugf("%s informer stopped unexpectedly: %s", i.name, ctx.Err())
			panic(fmt.Errorf("informer stopped unexpectedly: %w", ctx.Err()))
		}
	}
}
