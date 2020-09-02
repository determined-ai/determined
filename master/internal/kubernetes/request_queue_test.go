package kubernetes

import (
	"fmt"
	"testing"
	"time"

	petName "github.com/dustinkirkland/golang-petname"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	case resourceCreationCancelled:

	case resourceCreationFailed, resourceDeletionFailed:
		ctx.Log().Errorf("should not hit these messages during testing %T", msg)
		return actor.ErrUnexpectedMessage(ctx)

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func getNumberOfActivePods(podInterface typedV1.PodInterface) int {
	podList, err := podInterface.List(metaV1.ListOptions{})
	if err != nil {
		panic(err)
	}

	return len(podList.Items)
}

func waitForPendingRequestToFinish(k8RequestQueue *requestQueue) {
	time.Sleep(time.Second)

	// Wait for queue to finish all in flight requests.
	for len(k8RequestQueue.queue) > 0 &&
		len(k8RequestQueue.blockedResourceDeletions) == 0 &&
		len(k8RequestQueue.availableWorkers) < numKubernetesWorkers {
	}
}

func TestRequestQueueCreatingManyPod(t *testing.T) {
	system := actor.NewSystem(t.Name())

	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

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

	waitForPendingRequestToFinish(k8RequestQueue)
	assert.Equal(t, getNumberOfActivePods(podInterface), numPods)
}

func TestRequestQueueCreatingAndDeletingManyPod(t *testing.T) {
	system := actor.NewSystem(t.Name())

	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

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

	waitForPendingRequestToFinish(k8RequestQueue)
	assert.Equal(t, getNumberOfActivePods(podInterface), 0)
}

func TestRequestQueueCreatingThenDeletingManyPods(t *testing.T) {
	system := actor.NewSystem(t.Name())

	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

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

	waitForPendingRequestToFinish(k8RequestQueue)
	assert.Equal(t, getNumberOfActivePods(podInterface), numPods)

	system.AskAll(deleteMockPod{}, podActors...)
	system.AskAll(mockPodActorPing{}, podActors...).GetAll()

	waitForPendingRequestToFinish(k8RequestQueue)
	assert.Equal(t, getNumberOfActivePods(podInterface), 0)
}

func TestRequestQueueCreatingAndDeletingManyPodWithDelay(t *testing.T) {
	system := actor.NewSystem(t.Name())

	podInterface := &mockPodInterface{
		pods:             make(map[string]*k8sV1.Pod),
		operationalDelay: time.Millisecond * 500,
	}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

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

	waitForPendingRequestToFinish(k8RequestQueue)
	assert.Equal(t, getNumberOfActivePods(podInterface), 0)
}
