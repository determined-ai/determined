package kubernetes

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"
)

type mockReceiver struct {
	name      string
	responses []actor.Message
}

func newMockReceiver(name string) *mockReceiver {
	return &mockReceiver{name: name, responses: []actor.Message{}}
}

func (m *mockReceiver) Receive(ctx *actor.Context) error {
	m.responses = append(m.responses, ctx.Message())
	return nil
}

func (m *mockReceiver) GetLength() actor.Message {
	return len(m.responses)
}

func (m *mockReceiver) Purge() {
	m.responses = []actor.Message{}
}

func (m *mockReceiver) Pop() (actor.Message, error) {
	if len(m.responses) > 0 {
		output := m.responses[0]
		m.responses = m.responses[1:]
		return output, nil
	}
	return actor.PreStart{}, fmt.Errorf("nothing left in responses")
}

func createPod(
	taskHandler *actor.Ref,
	clusterHandler *actor.Ref,
	resourceHandler *actor.Ref,
	task tasks.TaskSpec,
) *pod {
	msg := StartTaskPod{
		TaskActor: taskHandler,
		Spec:      task,
		Slots:     1,
	}
	cluster := clusterHandler
	clusterID := "test"
	clientSet := k8sClient.Clientset{}
	namespace := "test_namespace"
	masterIP := "0.0.0.0"
	var masterPort int32 = 32
	podInterface := clientSet.CoreV1().Pods(namespace)
	configMapInterface := clientSet.CoreV1().ConfigMaps(namespace)
	resourceRequestQueue := resourceHandler
	leaveKubernetesResources := false
	slotType := device.CUDA
	slotResourceRequests := PodSlotResourceRequests{}

	newPodHandler := newPod(
		msg, cluster, clusterID, &clientSet, namespace, masterIP, masterPort,
		model.TLSClientConfig{}, model.TLSClientConfig{},
		model.LoggingConfig{DefaultLoggingConfig: &model.DefaultLoggingConfig{}},
		podInterface, configMapInterface, resourceRequestQueue, leaveKubernetesResources,
		slotType, slotResourceRequests, "default-scheduler", DefaultFluentConfig,
	)

	return newPodHandler
}

func createReceivers(system *actor.System) (map[string]*mockReceiver, map[string]*actor.Ref) {
	podMap := make(map[string]*mockReceiver)
	actorMap := make(map[string]*actor.Ref)

	podMap["task"] = newMockReceiver("task-receiver")
	actorMap["task"], _ = system.ActorOf(
		actor.Addr("task-pod"),
		podMap["task"],
	)

	podMap["cluster"] = newMockReceiver("cluster-receiver")
	actorMap["cluster"], _ = system.ActorOf(
		actor.Addr("cluster-pod"),
		podMap["cluster"],
	)

	podMap["resource"] = newMockReceiver("resource-receiver")
	actorMap["resource"], _ = system.ActorOf(
		actor.Addr("resource-pod"),
		podMap["resource"],
	)

	return podMap, actorMap
}

func createAgentUserGroup() *model.AgentUserGroup {
	return &model.AgentUserGroup{
		ID:     1,
		UserID: 1,
		User:   "determined",
		UID:    1,
		Group:  "test-group",
		GID:    1,
	}
}

func createUser() *model.User {
	return &model.User{
		ID:       1,
		Username: "determined",
		Active:   true,
		Admin:    false,
	}
}

func createPodWithMockQueue() (
	*actor.System,
	*pod,
	*actor.Ref,
	map[string]*mockReceiver,
	map[string]*actor.Ref,
) {
	commandSpec := tasks.GenericCommandSpec{
		Base: tasks.TaskSpec{
			AllocationID:     "task",
			ContainerID:      "container",
			ClusterID:        "cluster",
			AgentUserGroup:   createAgentUserGroup(),
			Owner:            createUser(),
			UserSessionToken: "bogus",
		},
		Config: model.CommandConfig{Description: "test-config"},
	}
	system := actor.NewSystem("test-sys")
	podMap, actorMap := createReceivers(system)

	newPod := createPod(
		actorMap["task"],
		actorMap["cluster"],
		actorMap["resource"],
		commandSpec.ToTaskSpec(nil),
	)
	ref, _ := system.ActorOf(
		actor.Addr("pod-actor-test"),
		newPod,
	)
	time.Sleep(time.Millisecond * 500)

	return system, newPod, ref, podMap, actorMap
}

func setupEntrypoint(t *testing.T) {
	err := etc.SetRootPath(".")
	if err != nil {
		t.Logf("Failed to set root directory")
	}

	f, _ := os.OpenFile("k8_init_container_entrypoint.sh", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	err = f.Close()
	if err != nil {
		t.Logf("Failed to close entrypoint")
	}

	f, _ = os.OpenFile("task-logging-setup.sh", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	err = f.Close()
	if err != nil {
		t.Logf("Failed to close task-logging-setup.sh")
	}

	f, _ = os.OpenFile("task-logging-teardown.sh", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	err = f.Close()
	if err != nil {
		t.Logf("Failed to close task-logging-teardown.sh")
	}

	f, _ = os.OpenFile("task-signal-handling.sh", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	err = f.Close()
	if err != nil {
		t.Logf("Failed to close task-signal-handling.sh")
	}
}

func cleanup(t *testing.T) {
	err := os.Remove("k8_init_container_entrypoint.sh")
	if err != nil {
		t.Logf("Failed to remove entrypoint")
	}

	err = os.Remove("task-logging-setup.sh")
	if err != nil {
		t.Logf("Failed to remove task-logging-setup.sh")
	}

	err = os.Remove("task-logging-teardown.sh")
	if err != nil {
		t.Logf("Failed to remove task-logging-teardown.sh")
	}

	err = os.Remove("task-signal-handling.sh")
	if err != nil {
		t.Logf("Failed to remove task-signal-handling.sh")
	}
}

func checkReceiveTermination(
	t *testing.T,
	update podStatusUpdate,
	system *actor.System,
	ref *actor.Ref,
	newPod *pod,
	podMap map[string]*mockReceiver,
) {
	system.Ask(ref, update)
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 1)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}
	containerMsg, ok := message.(sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf("expected sproto.TaskContainerStateChanged but received %s", reflect.TypeOf(message))
	}
	if containerMsg.ResourcesStopped == nil {
		t.Errorf("container started message not present")
	}

	assert.Equal(t, newPod.container.State, cproto.Terminated)
}

func TestResourceCreationFailed(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	const correctMsg = "mock error"

	system, _, ref, podMap, _ := createPodWithMockQueue()

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	system.Ask(ref, resourceCreationFailed{err: fmt.Errorf(correctMsg)})
	time.Sleep(time.Second)

	// We expect two messages in the queue because the pod actor sends itself a stop message.
	assert.Equal(t, podMap["task"].GetLength(), 2)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}

	containerMsg, ok := message.(sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, *containerMsg.AuxMessage, correctMsg)
}

func TestReceivePodStatusUpdateTerminated(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	// Pod deleting, but in pending state.
	t.Logf("Testing PodPending status")
	system, newPod, ref, podMap, _ := createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	typeMeta := metaV1.TypeMeta{Kind: "rest test"}
	objectMeta := metaV1.ObjectMeta{
		Name:              "test meta",
		DeletionTimestamp: &metaV1.Time{Time: time.Now()},
	}
	pod := k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     k8sV1.PodStatus{Phase: k8sV1.PodPending},
	}

	statusUpdate := podStatusUpdate{updatedPod: &pod}

	checkReceiveTermination(t, statusUpdate, system, ref, newPod, podMap)

	// Pod failed.
	t.Logf("Testing PodFailed status")
	system, newPod, ref, podMap, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)
	pod = k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     k8sV1.PodStatus{Phase: k8sV1.PodFailed},
	}
	statusUpdate = podStatusUpdate{updatedPod: &pod}

	checkReceiveTermination(t, statusUpdate, system, ref, newPod, podMap)

	// Pod succeeded.
	system, newPod, ref, podMap, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)
	pod = k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     k8sV1.PodStatus{Phase: k8sV1.PodSucceeded},
	}
	statusUpdate = podStatusUpdate{updatedPod: &pod}

	checkReceiveTermination(t, statusUpdate, system, ref, newPod, podMap)

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
}

func TestMultipleContainerTerminate(t *testing.T) {
	// Status update test involving two containers.
	setupEntrypoint(t)
	defer cleanup(t)

	// Pod running with > 1 container, and one terminated.
	t.Logf("Testing two pods with one in terminated state")
	system, newPod, ref, podMap, _ := createPodWithMockQueue()
	containerStatuses := []k8sV1.ContainerStatus{
		{
			Name: "test-pod-1",
			State: k8sV1.ContainerState{
				Running: &k8sV1.ContainerStateRunning{},
			},
		},
		{
			Name: "test-pod-2",
			State: k8sV1.ContainerState{
				Terminated: &k8sV1.ContainerStateTerminated{},
			},
		},
	}
	newPod.containerNames = map[string]bool{"test-pod-1": false, "test-pod-2": false}
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	pod := k8sV1.Pod{
		TypeMeta: metaV1.TypeMeta{Kind: "rest test"},
		ObjectMeta: metaV1.ObjectMeta{
			Name:              "test meta",
			DeletionTimestamp: &metaV1.Time{Time: time.Now()},
		},
		Status: k8sV1.PodStatus{
			Phase:             k8sV1.PodRunning,
			ContainerStatuses: containerStatuses,
		},
	}

	statusUpdate := podStatusUpdate{updatedPod: &pod}
	checkReceiveTermination(t, statusUpdate, system, ref, newPod, podMap)

	// Multiple pods, 1 termination, no deletion timestamp.
	// This results in an error, which causes pod termination and the same outcome.
	t.Logf("Testing two pods with one in terminated state and no deletion timestamp")
	system, newPod, ref, podMap, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	pod = k8sV1.Pod{
		TypeMeta: metaV1.TypeMeta{Kind: "rest test"},
		ObjectMeta: metaV1.ObjectMeta{
			Name: "test meta",
		},
		Status: k8sV1.PodStatus{
			Phase:             k8sV1.PodRunning,
			ContainerStatuses: containerStatuses,
		},
	}

	statusUpdate = podStatusUpdate{updatedPod: &pod}
	checkReceiveTermination(t, statusUpdate, system, ref, newPod, podMap)
}

func TestReceivePodStatusUpdateAssigned(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _ := createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	typeMeta := metaV1.TypeMeta{Kind: "rest test"}
	objectMeta := metaV1.ObjectMeta{
		Name: "test meta",
	}
	pod := k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     k8sV1.PodStatus{Phase: k8sV1.PodPending},
	}

	statusUpdate := podStatusUpdate{updatedPod: &pod}

	assert.Equal(t, newPod.container.State, cproto.Assigned)
	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)

	newPod.container.State = cproto.Starting

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, newPod.container.State, cproto.Starting)
}

func TestReceivePodStatusUpdateStarting(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _ := createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	// Pod status Pending, Pod Scheduled.
	t.Logf("Testing pod scheduled with pending status")
	typeMeta := metaV1.TypeMeta{Kind: "rest test"}
	objectMeta := metaV1.ObjectMeta{
		Name: "test meta",
	}
	condition := k8sV1.PodCondition{
		Type:    k8sV1.PodScheduled,
		Status:  k8sV1.ConditionTrue,
		Message: "This doesn't matter :)",
	}
	status := k8sV1.PodStatus{
		Phase:      k8sV1.PodPending,
		Conditions: []k8sV1.PodCondition{condition},
	}
	pod := k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	statusUpdate := podStatusUpdate{updatedPod: &pod}

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 2)
	assert.Equal(t, newPod.container.State, cproto.Starting)
	podMap["task"].Purge()
	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, newPod.container.State, cproto.Starting)

	// Pod status Running, but container status Waiting.
	t.Logf("Testing pod running with waiting status")
	system, newPod, ref, podMap, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	containerStatuses := []k8sV1.ContainerStatus{
		{
			Name:  "determined-container",
			State: k8sV1.ContainerState{Waiting: &k8sV1.ContainerStateWaiting{}},
		},
		{
			Name:  "determined-fluent-container",
			State: k8sV1.ContainerState{Waiting: &k8sV1.ContainerStateWaiting{}},
		},
	}
	status = k8sV1.PodStatus{
		Phase:             k8sV1.PodRunning,
		ContainerStatuses: containerStatuses,
	}
	pod = k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	statusUpdate = podStatusUpdate{updatedPod: &pod}

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 2)
	assert.Equal(t, newPod.container.State, cproto.Starting)

	// Pod status running, but no Container State inside.
	t.Logf("Testing pod running with no status")
	system, newPod, ref, podMap, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	status = k8sV1.PodStatus{
		Phase: k8sV1.PodRunning,
		ContainerStatuses: []k8sV1.ContainerStatus{
			{Name: "determined-container"},
			{Name: "determined-fluent-container"},
		},
	}
	pod = k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	statusUpdate = podStatusUpdate{updatedPod: &pod}
	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 2)
	assert.Equal(t, newPod.container.State, cproto.Starting)
}

func TestMultipleContainersRunning(t *testing.T) {
	// Status update test involving two containers.
	setupEntrypoint(t)
	defer cleanup(t)

	// Testing pod with two containers and one doesn't have running state.
	t.Logf("Testing two pods and one doesn't have running state")
	system, newPod, ref, podMap, _ := createPodWithMockQueue()
	newPod.container.State = cproto.Starting
	newPod.testLogStreamer = true

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	typeMeta := metaV1.TypeMeta{Kind: "rest test"}
	objectMeta := metaV1.ObjectMeta{
		Name: "test meta",
	}
	containerStatuses := []k8sV1.ContainerStatus{
		{
			Name:  "determined-container",
			State: k8sV1.ContainerState{Running: &k8sV1.ContainerStateRunning{}},
		},
		{
			Name:  "determined-fluent-container",
			State: k8sV1.ContainerState{Running: &k8sV1.ContainerStateRunning{}},
		},
		{
			Name: "test-pod",
		},
	}
	status := k8sV1.PodStatus{
		Phase:             k8sV1.PodRunning,
		ContainerStatuses: containerStatuses,
	}
	pod := k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	newPod.containerNames = map[string]bool{
		"determined-container":        false,
		"determined-fluent-container": false,
		"test-pod":                    false,
	}
	statusUpdate := podStatusUpdate{updatedPod: &pod}

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, newPod.container.State, cproto.Starting)

	// Multiple containers, all in running state, results in a running state.
	t.Logf("Testing two pods with running states")
	system, newPod, ref, podMap, _ = createPodWithMockQueue()
	newPod.testLogStreamer = true

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	newPod.container.State = cproto.Starting
	containerStatuses[2] = k8sV1.ContainerStatus{
		Name:  "test-pod-2",
		State: k8sV1.ContainerState{Running: &k8sV1.ContainerStateRunning{}},
	}
	status = k8sV1.PodStatus{
		Phase:             k8sV1.PodRunning,
		ContainerStatuses: containerStatuses,
	}
	pod = k8sV1.Pod{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Status:     status,
	}
	statusUpdate = podStatusUpdate{updatedPod: &pod}
	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 1)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}

	containerMsg, ok := message.(sproto.ResourcesStateChanged)
	fmt.Println("CONTAINER MESSAGE:", containerMsg)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	if containerMsg.ResourcesStarted == nil {
		t.Errorf("container started message not present")
	}
}

func TestReceivePodEventUpdate(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _ := createPodWithMockQueue()

	object := k8sV1.ObjectReference{Kind: "mock", Namespace: "test", Name: "MockObject"}
	newEvent := k8sV1.Event{
		InvolvedObject: object,
		Reason:         "testing",
		Message:        "0/99 nodes are available: 99 Insufficient cpu",
	}
	newPod.slots = 99
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	system.Ask(ref, podEventUpdate{event: &newEvent})
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 1)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}
	correctMsg := fmt.Sprintf("Pod %s: %s", object.Name,
		"Waiting for resources. 0 GPUs are available, 99 GPUs required")

	containerMsg, ok := message.(sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, *containerMsg.AuxMessage, correctMsg)

	// When container is in Running state, pod actor should not forward message.
	podMap["task"].Purge()
	newPod.container.State = cproto.Running
	system.Ask(ref, podEventUpdate{event: &newEvent})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)

	// When container is in Terminated state, pod actor should not forward message.
	newPod.container.State = cproto.Terminated
	system.Ask(ref, podEventUpdate{event: &newEvent})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
}

func TestReceiveContainerLog(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, _, ref, podMap, _ := createPodWithMockQueue()
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	rightNow := time.Now()
	correctMsg := "This is a mock message."

	newEvent := sproto.ContainerLog{
		Timestamp:  rightNow,
		AuxMessage: &correctMsg,
	}

	system.Ask(ref, newEvent)
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 1)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}

	containerMsg, ok := message.(sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, *containerMsg.AuxMessage, correctMsg)
}

func TestKillTaskPod(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _ := createPodWithMockQueue()
	// We take a quick nap immediately so we can purge the start message after it arrives.
	time.Sleep(time.Second)

	podMap["resource"].Purge()
	assert.Equal(t, podMap["resource"].GetLength(), 0)

	system.Ask(ref, KillTaskPod{})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["resource"].GetLength(), 1)

	message, err := podMap["resource"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from resources receiver queue")
	}
	assert.Equal(t, message, deleteKubernetesResources{
		handler:       ref,
		podName:       newPod.podName,
		configMapName: newPod.configMapName,
	},
	)

	system.Ask(ref, KillTaskPod{})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["resource"].GetLength(), 0)
}

func TestResourceCreationCancelled(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, _, ref, podMap, _ := createPodWithMockQueue()

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	system.Ask(ref, resourceCreationCancelled{})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 1)

	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}

	containerMsg, ok := message.(sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf("expected sproto.TaskContainerStateChanged but received %s",
			reflect.TypeOf(message))
	}

	var correctContainerStarted *sproto.ResourcesStarted = nil
	correctFailType := "task failed without an associated exit code"
	correctErrMsg := "pod actor exited while pod was running"
	var correctCode *sproto.ExitCode = nil

	assert.Equal(t, containerMsg.ResourcesStarted, correctContainerStarted)
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.FailureType,
		sproto.FailureType(correctFailType))
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ErrMsg, correctErrMsg)
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ExitCode, correctCode)
}

func TestResourceDeletionFailed(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, _, ref, podMap, _ := createPodWithMockQueue()

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	errMsg := "mock error"
	system.Ask(ref, resourceDeletionFailed{err: fmt.Errorf(errMsg)})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 1)

	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}

	containerMsg, ok := message.(sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf("expected sproto.TaskContainerStateChanged but received %s",
			reflect.TypeOf(message))
	}

	var correctContainerStarted *sproto.ResourcesStarted = nil
	var correctCode *sproto.ExitCode = nil

	assert.Equal(t, containerMsg.ResourcesStarted, correctContainerStarted)
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.FailureType,
		sproto.FailureType("task failed without an associated exit code"))
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ErrMsg,
		"pod actor exited while pod was running")
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ExitCode, correctCode)
}

func TestGetPodNodeInfo(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _ := createPodWithMockQueue()
	newPod.slots = 99
	time.Sleep(time.Second)

	podMap["task"].Purge()
	podMap["cluster"].Purge()
	podMap["resource"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, podMap["cluster"].GetLength(), 0)
	assert.Equal(t, podMap["resource"].GetLength(), 0)

	data := system.Ask(ref, getPodNodeInfo{})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, podMap["cluster"].GetLength(), 0)
	assert.Equal(t, podMap["resource"].GetLength(), 0)

	podInfo, ok := data.Get().(podNodeInfo)
	if !ok {
		t.Errorf("expected podNodeInfo but received %s", reflect.TypeOf(data))
	}

	assert.Equal(t, podInfo.nodeName, newPod.pod.Spec.NodeName)
	assert.Equal(t, podInfo.numSlots, newPod.slots)
}
