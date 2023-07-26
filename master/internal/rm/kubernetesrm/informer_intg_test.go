package kubernetesrm

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/determined-ai/determined/master/internal/mocks"
)

const namespace = "default"

// mockWatcher implements watch.Interface so it can be returned by mocks' calls to Watch.
type mockWatcher struct {
	c chan watch.Event
}

// operations is a tuple struct (name, action) for testing
// events handled by the node informer. Name refers to the
// node name & action refers to the Watch.Event.Type.
type operations struct {
	name   string
	action watch.EventType
}

// Since preemption & pod informers share the same informer.go
// code for newPodInformer() & run(), this tests serves to cover both.
func TestPodInformer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cases := []struct {
		name     string
		podNames []string
		ordering []string
	}{
		{"zero pods", []string{}, []string{}},
		{"informer success", []string{"abc"}, []string{"abc"}},
		{
			"informer success & event ordering success",
			[]string{"A", "B", "C", "D", "E"},
			[]string{"A", "B", "C", "D", "E"},
		},
		{
			"informer success & event ordering failure",
			[]string{"A", "B", "C", "D", "E"},
			[]string{"C", "A", "B", "D", "E"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(len(tt.podNames))

			eventChan := make(chan watch.Event)
			ordering := make([]string, 0)

			mockOptsList, mockOptsWatch := initializeMockOpts("pod")
			mockPodInterface := &mocks.PodInterface{}
			mockPodInterface.On("List", context.TODO(), mockOptsList).Return(
				&k8sV1.PodList{
					ListMeta: metaV1.ListMeta{
						ResourceVersion: "1",
					},
				},
				nil)
			mockPodInterface.On("Watch", context.TODO(), mockOptsWatch).Return(
				&mockWatcher{c: eventChan}, nil)
			mockPodHandler := func(event watch.Event) {
				pod, _ := event.Object.(*k8sV1.Pod)
				t.Logf("received pod %v", pod.Name)
				ordering = append(ordering, pod.Name)
				wg.Done()
			}

			// Test creating newPodInformer.
			i, err := newPodInformer(
				context.TODO(),
				determinedLabel,
				"pod",
				namespace,
				mockPodInterface,
				mockPodHandler)
			assert.NotNil(t, i)
			assert.Nil(t, err)

			// Test run().
			go i.run(ctx)
			for _, name := range tt.podNames {
				pod := &k8sV1.Pod{
					ObjectMeta: metaV1.ObjectMeta{
						ResourceVersion: "1",
						Name:            name,
					},
				}
				eventChan <- watch.Event{
					Type:   watch.Modified,
					Object: pod,
				}
			}

			// Assert correct ordering of pod-modified events
			// after all events are received and the channel is closed.
			wg.Wait()
			if reflect.DeepEqual(tt.podNames, tt.ordering) {
				assert.Equal(t, tt.podNames, ordering)
			} else {
				assert.NotEqual(t, tt.ordering, ordering)
			}
		})
	}
}

func TestNodeInformer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cases := []struct {
		name       string
		operations []operations
		output     map[string]bool
		ordering   map[string]bool
	}{
		{"zero nodes", []operations{}, map[string]bool{}, map[string]bool{}},
		{
			"informer success",
			[]operations{{"abc", watch.Added}},
			map[string]bool{"abc": true},
			map[string]bool{"abc": true},
		},
		{
			"informer success & event ordering success",
			[]operations{
				{"A", watch.Added},
				{"B", watch.Added},
				{"C", watch.Added},
				{"A", watch.Deleted},
				{"B", watch.Modified},
			},
			map[string]bool{"B": false, "C": true},
			map[string]bool{"B": false, "C": true},
		},
		{
			"informer success & event ordering success",
			[]operations{
				{"A", watch.Added},
				{"B", watch.Added},
				{"C", watch.Added},
				{"A", watch.Deleted},
				{"B", watch.Modified},
			},
			map[string]bool{"B": false, "C": true},
			map[string]bool{"A": true, "C": true},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(len(tt.operations))

			eventChan := make(chan watch.Event)
			currNodes := make(map[string]bool, 0)

			mockOptsList, mockOptsWatch := initializeMockOpts("node")
			mockNode := &mocks.NodeInterface{}
			mockNode.On("List", context.TODO(), mockOptsList).Return(
				&k8sV1.NodeList{
					ListMeta: metaV1.ListMeta{
						ResourceVersion: "1",
					},
				},
				nil)
			mockNode.On("Watch", context.TODO(), mockOptsWatch).Return(&mockWatcher{c: eventChan}, nil)
			mockNodeHandler := func(event watch.Event) {
				node, _ := event.Object.(*k8sV1.Node)
				if node.Name != "" {
					t.Logf("received %v", node.Name)
					switch event.Type {
					case watch.Added:
						currNodes[node.Name] = true
					case watch.Modified:
						currNodes[node.Name] = false
					case watch.Deleted:
						delete(currNodes, node.Name)
					default:
						t.Logf("Node did not expect watch.EventType %v", event.Type)
					}
				}
				wg.Done()
			}

			// Test newNodeInformer is created.
			n, err := newNodeInformer(context.TODO(), mockNode, mockNodeHandler)
			assert.NotNil(t, n)
			assert.Nil(t, err)

			// Test run() & iterate through/apply a set of events received by the informer.
			go n.run(ctx)
			for _, n := range tt.operations {
				node := &k8sV1.Node{
					ObjectMeta: metaV1.ObjectMeta{
						ResourceVersion: "1",
						Name:            n.name,
					},
				}
				eventChan <- watch.Event{
					Type:   n.action,
					Object: node,
				}
			}

			wg.Wait()
			if reflect.DeepEqual(tt.output, tt.ordering) {
				assert.Equal(t, tt.output, currNodes)
			} else {
				assert.NotEqual(t, tt.ordering, currNodes)
			}
		})
	}
}

func TestEventListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	cases := []struct {
		name       string
		eventNames []string
		ordering   []string
	}{
		{"zero events", []string{}, []string{}},
		{"listener success", []string{"A"}, []string{"A"}},
		{
			"listener success & event ordering success",
			[]string{"A", "B", "C", "D", "E"},
			[]string{"A", "B", "C", "D", "E"},
		},
		{
			"listener success & event ordering failure",
			[]string{"A", "B", "C", "D", "E"},
			[]string{"E", "D", "C", "A", "B"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(len(tt.eventNames))

			eventChan := make(chan watch.Event)
			ordering := make([]string, 0)

			mockOptsList, mockOptsWatch := initializeMockOpts("event")
			mockEventInterface := &mocks.EventInterface{}

			mockEventInterface.On("List", context.TODO(), mockOptsList).Return(
				&k8sV1.EventList{
					ListMeta: metaV1.ListMeta{
						ResourceVersion: "1",
					},
				},
				nil)
			mockEventInterface.On("Watch", context.TODO(), mockOptsWatch).Return(
				&mockWatcher{c: eventChan}, nil)

			mockEventHandler := func(event watch.Event) {
				newEvent, _ := event.Object.(*k8sV1.Event)
				t.Logf("received event %v", newEvent)
				ordering = append(ordering, newEvent.Name)
				wg.Done()
			}

			i, err := newEventInformer(
				context.TODO(),
				mockEventInterface,
				namespace,
				mockEventHandler)
			assert.NotNil(t, i)
			assert.Nil(t, err)

			go i.run(ctx)
			for _, name := range tt.eventNames {
				event := &k8sV1.Event{
					ObjectMeta: metaV1.ObjectMeta{
						ResourceVersion: "1",
						Name:            name,
					},
				}
				eventChan <- watch.Event{
					Type:   watch.Modified,
					Object: event,
				}
			}

			wg.Wait()
			if reflect.DeepEqual(tt.eventNames, tt.ordering) {
				assert.Equal(t, tt.eventNames, ordering)
			} else {
				assert.NotEqual(t, tt.ordering, ordering)
			}
		})
	}
}

// Methods for mockWatcher.
func (m *mockWatcher) Stop() {
	close(m.c)
}

func (m *mockWatcher) ResultChan() <-chan watch.Event {
	return m.c
}

func initializeMockOpts(label string) (metaV1.ListOptions, metaV1.ListOptions) {
	mockOptsList := metaV1.ListOptions{}
	mockOptsWatch := metaV1.ListOptions{}
	switch label {
	case "node":
		mockOptsWatch.ResourceVersion = "1"
		mockOptsWatch.AllowWatchBookmarks = true
	case "pod":
		mockOptsList.LabelSelector = determinedLabel
		mockOptsWatch.LabelSelector = determinedLabel
		mockOptsWatch.ResourceVersion = "1"
		mockOptsWatch.AllowWatchBookmarks = true
	case "preemption":
		mockOptsList.LabelSelector = determinedPreemptionLabel
		mockOptsWatch.LabelSelector = determinedPreemptionLabel
		mockOptsWatch.ResourceVersion = "1"
		mockOptsWatch.AllowWatchBookmarks = true
	case "event":
	default:
	}
	return mockOptsList, mockOptsWatch
}
