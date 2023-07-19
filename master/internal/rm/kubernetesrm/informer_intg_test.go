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
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

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

func TestPodInformer(t *testing.T) {
	cases := []struct {
		name     string
		podNames []string
		output   []string
		expected bool
	}{
		{"zero pods", []string{}, []string{}, true},
		{
			"informer success",
			[]string{"abc"},
			[]string{"abc"},
			true,
		},
		{
			"informer success & event ordering success",
			[]string{"A", "B", "C", "D", "E"},
			[]string{"A", "B", "C", "D", "E"},
			true,
		},
		{
			"informer success & event ordering failure",
			[]string{"A", "B", "C", "D", "E"},
			[]string{"A", "A", "C", "E", "D"},
			false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(len(tt.podNames))

			ctx := context.TODO()
			eventChan := make(chan watch.Event)
			ordering := make([]string, 0)

			mockOptsList, mockOptsWatch := initializeMockOptsPod()
			mockPod := &mocks.PodInterface{}
			mockPod.On("List", ctx, mockOptsList).Return(
				&k8sV1.PodList{
					ListMeta: metaV1.ListMeta{
						ResourceVersion: "1",
					},
				},
				nil)
			mockPod.On("Watch", ctx, mockOptsWatch).Return(&mockWatcher{c: eventChan}, nil)
			mockPods := &pods{
				namespace:     namespace,
				podInterfaces: map[string]typedV1.PodInterface{namespace: mockPod},
			}
			mockPodHandler := func(pod *k8sV1.Pod) {
				t.Logf("received pod %v", pod.Name)
				ordering = append(ordering, pod.Name)
				wg.Done()
			}

			// Test creating newInformer.
			i, err := newInformer(context.TODO(),
				namespace,
				mockPods.podInterfaces[namespace],
				mockPodHandler)
			assert.NotNil(t, i)
			assert.Nil(t, err)

			// Test startInformer.
			go i.run()
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
			equality := reflect.DeepEqual(tt.output, ordering)
			assert.Equal(t, tt.expected, equality)
		})
	}
}

func TestNodeInformer(t *testing.T) {
	cases := []struct {
		name       string
		operations []operations
		output     map[string]bool
		expected   bool
	}{
		{"zero nodes", []operations{}, map[string]bool{}, true},
		{
			"informer success",
			[]operations{{"abc", watch.Added}},
			map[string]bool{"abc": true},
			true,
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
			true,
		},
		{
			"informer success & event ordering failure",
			[]operations{
				{"A", watch.Added},
				{"B", watch.Added},
				{"C", watch.Added},
				{"A", watch.Deleted},
				{"B", watch.Modified},
			},
			map[string]bool{"A": true, "C": true},
			false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(len(tt.operations))

			ctx := context.TODO()
			eventChan := make(chan watch.Event)
			currNodes := make(map[string]bool, 0)

			mockOptsList, mockOptsWatch := initializeMockOptsNode()
			mockNode := &mocks.NodeInterface{}
			mockNode.On("List", ctx, mockOptsList).Return(
				&k8sV1.NodeList{
					ListMeta: metaV1.ListMeta{
						ResourceVersion: "1",
					},
				},
				nil)
			mockNode.On("Watch", ctx, mockOptsWatch).Return(&mockWatcher{c: eventChan}, nil)
			mockNodeHandler := func(node *k8sV1.Node, action watch.EventType) {
				if node.Name != "" {
					t.Logf("received %v", node.Name)
					switch action {
					case watch.Added:
						currNodes[node.Name] = true
					case watch.Modified:
						currNodes[node.Name] = false
					case watch.Deleted:
						delete(currNodes, node.Name)
					default:
						t.Logf("Node did not expect watch.EventType %v", action)
					}
				}
				wg.Done()
			}

			// Test newNodeInformer is created.
			n, err := newNodeInformer(context.TODO(), mockNode, mockNodeHandler)
			assert.NotNil(t, n)
			assert.Nil(t, err)

			// Test startNodeInformer & iterate through/apply a set of operations
			// (podName, action) to the informer.
			go n.run()
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

			// Assert equality between expected vs actual status
			// of the nodes.
			equality := reflect.DeepEqual(currNodes, tt.output)
			assert.Equal(t, tt.expected, equality)
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

func initializeMockOptsPod() (metaV1.ListOptions, metaV1.ListOptions) {
	mockOptsList := metaV1.ListOptions{LabelSelector: determinedLabel}
	mockOptsWatch := metaV1.ListOptions{
		LabelSelector:       determinedLabel,
		ResourceVersion:     "1",
		AllowWatchBookmarks: true,
	}
	return mockOptsList, mockOptsWatch
}

func initializeMockOptsNode() (metaV1.ListOptions, metaV1.ListOptions) {
	mockOptsList := metaV1.ListOptions{}
	mockOptsWatch := metaV1.ListOptions{
		ResourceVersion:     "1",
		AllowWatchBookmarks: true,
	}
	return mockOptsList, mockOptsWatch
}
