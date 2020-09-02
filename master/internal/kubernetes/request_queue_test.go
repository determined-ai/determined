package kubernetes

import (
	"fmt"
	"testing"
	"time"

	"gotest.tools/assert"

	petName "github.com/dustinkirkland/golang-petname"

	"github.com/determined-ai/determined/master/pkg/actor"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeK8 "k8s.io/client-go/kubernetes/fake"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type (
	mockPodActorPing struct{}
	deleteMockPod    struct{}
)

type mockPodActor struct {
	requestQueue *actor.Ref
	name         string
}

func newMockPodActor(requestQueue *actor.Ref) *mockPodActor {
	return &mockPodActor{
		requestQueue: requestQueue,
		name:         petName.Generate(3, "-"),
	}
}

func (m *mockPodActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		podSpec := k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: m.name}}
		cmSpec := k8sV1.ConfigMap{ObjectMeta: metaV1.ObjectMeta{Name: m.name}}

		ctx.Tell(m.requestQueue, createKubernetesResources{
			handler:       ctx.Self(),
			podSpec:       &podSpec,
			configMapSpec: &cmSpec,
		})

	case mockPodActorPing:
		ctx.Respond(mockPodActorPing{})

	case deleteMockPod:
		ctx.Ask(m.requestQueue, deleteKubernetesResources{
			handler:       ctx.Self(),
			podName:       m.name,
			configMapName: m.name,
		})

	case resourceCreationCancelled, resourceDeletionFailed:

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func getNumberOfActivePods(podInterface typedV1.PodInterface) int {
	// Because the k8s fake client doesn't not always process requests
	// as soon at is says it does, we sleep here for a second.
	time.Sleep(time.Second)
	podList, err := podInterface.List(metaV1.ListOptions{})
	if err != nil {
		panic(err)
	}

	return len(podList.Items)
}

func TestRequestQueueCreatingManyPod(t *testing.T) {
	system := actor.NewSystem(t.Name())

	clientSet := fakeK8.NewSimpleClientset()
	podInterface := clientSet.CoreV1().Pods("")
	configMapInterface := clientSet.CoreV1().ConfigMaps("")

	k8RequestQueue := newRequestQueue(podInterface, configMapInterface)
	requestQueueActor, _ := system.ActorOf(
		actor.Addr("request-queue"),
		k8RequestQueue,
	)

	numPods := 15
	podActors := make([]*actor.Ref, 0)
	for i := 0; i < numPods; i++ {
		newMockPodActor, _ := system.ActorOf(
			actor.Addr(fmt.Sprintf("mock-pod-%d", i)),
			newMockPodActor(requestQueueActor),
		)

		podActors = append(podActors, newMockPodActor)
	}
	system.AskAll(mockPodActorPing{}, podActors...).GetAll()

	// Wait for queue to finish all in flight requests.
	for len(k8RequestQueue.queue) > 0 {
	}
	for len(k8RequestQueue.availableWorkers) < numKubernetesWorkers {
	}

	podList, err := podInterface.List(metaV1.ListOptions{})
	if err != nil {
		panic(err)
	}
	assert.Equal(t, len(podList.Items), numPods)
}

func TestRequestQueueCreatingAndDeletingManyPod(t *testing.T) {
	system := actor.NewSystem(t.Name())

	clientSet := fakeK8.NewSimpleClientset()
	podInterface := clientSet.CoreV1().Pods("")
	configMapInterface := clientSet.CoreV1().ConfigMaps("")

	k8RequestQueue := newRequestQueue(podInterface, configMapInterface)
	requestQueueActor, _ := system.ActorOf(
		actor.Addr("request-queue"),
		k8RequestQueue,
	)

	numPods := 15
	podActors := make([]*actor.Ref, 0)
	for i := 0; i < numPods; i++ {
		newMockPodActor, _ := system.ActorOf(
			actor.Addr(fmt.Sprintf("mock-pod-%d", i)),
			newMockPodActor(requestQueueActor),
		)

		podActors = append(podActors, newMockPodActor)
	}
	system.AskAll(deleteMockPod{}, podActors...)
	system.AskAll(mockPodActorPing{}, podActors...).GetAll()

	// Wait for queue to finish all in flight requests.
	for len(k8RequestQueue.queue) > 0 &&
		len(k8RequestQueue.availableWorkers) < numKubernetesWorkers {
	}
	assert.Equal(t, getNumberOfActivePods(podInterface), 0)
}

func TestRequestQueueCreatingThenDeletingManyPods(t *testing.T) {
	system := actor.NewSystem(t.Name())

	clientSet := fakeK8.NewSimpleClientset()
	podInterface := clientSet.CoreV1().Pods("")
	configMapInterface := clientSet.CoreV1().ConfigMaps("")

	k8RequestQueue := newRequestQueue(podInterface, configMapInterface)
	requestQueueActor, _ := system.ActorOf(
		actor.Addr("request-queue"),
		k8RequestQueue,
	)

	numPods := 15
	podActors := make([]*actor.Ref, 0)
	for i := 0; i < numPods; i++ {
		newMockPodActor, _ := system.ActorOf(
			actor.Addr(fmt.Sprintf("mock-pod-%d", i)),
			newMockPodActor(requestQueueActor),
		)

		podActors = append(podActors, newMockPodActor)
	}
	system.AskAll(mockPodActorPing{}, podActors...).GetAll()

	// Wait for queue to finish all in flight requests.
	for len(k8RequestQueue.queue) > 0 &&
		len(k8RequestQueue.availableWorkers) < numKubernetesWorkers {
	}
	assert.Equal(t, getNumberOfActivePods(podInterface), numPods)

	system.AskAll(deleteMockPod{}, podActors...)
	system.AskAll(mockPodActorPing{}, podActors...).GetAll()

	// Wait for queue to finish all in flight requests.
	for len(k8RequestQueue.queue) > 0 &&
		len(k8RequestQueue.availableWorkers) < numKubernetesWorkers {
	}
	assert.Equal(t, getNumberOfActivePods(podInterface), 0)
}
