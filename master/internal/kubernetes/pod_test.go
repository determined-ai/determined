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
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/container"
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
	msg := sproto.StartTaskPod{
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

	newPodHandler := newPod(
		msg, cluster, clusterID, &clientSet, namespace, masterIP, masterPort,
		model.TLSClientConfig{}, model.TLSClientConfig{},
		model.LoggingConfig{DefaultLoggingConfig: &model.DefaultLoggingConfig{}},
		podInterface, configMapInterface, resourceRequestQueue, leaveKubernetesResources,
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

func createPodWithMockQueue() (
	*actor.System,
	*pod,
	*actor.Ref,
	map[string]*mockReceiver,
	map[string]*actor.Ref,
) {
	startCmd := tasks.StartCommand{
		Config: model.CommandConfig{Description: "test-config"},
	}
	task := tasks.TaskSpec{
		TaskID:         "task",
		ContainerID:    "container",
		ClusterID:      "cluster",
		AgentUserGroup: createAgentUserGroup(),
	}
	task.SetInner(&startCmd)
	system := actor.NewSystem("test-sys")
	podMap, actorMap := createReceivers(system)

	newPod := createPod(actorMap["task"], actorMap["cluster"], actorMap["resource"], task)
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
}

func cleanup(t *testing.T) {
	err := os.Remove("k8_init_container_entrypoint.sh")
	if err != nil {
		t.Logf("Failed to remove entrypoint")
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
	containerMsg, ok := message.(sproto.TaskContainerStateChanged)
	if !ok {
		t.Errorf("expected sproto.TaskContainerStateChanged but received %s", reflect.TypeOf(message))
	}
	if containerMsg.ContainerStopped == nil {
		t.Errorf("container started message not present")
	}

	assert.Equal(t, newPod.container.State, container.Terminated)
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

	assert.Equal(t, newPod.container.State, container.Assigned)
	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)

	newPod.container.State = container.Starting

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, newPod.container.State, container.Starting)
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
	assert.Equal(t, newPod.container.State, container.Starting)
	podMap["task"].Purge()
	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, newPod.container.State, container.Starting)

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
	assert.Equal(t, newPod.container.State, container.Starting)

	// Pod status running, but no Container State inside.
	t.Logf("Testing pod running with no status")
	system, newPod, ref, podMap, _ = createPodWithMockQueue()
	podMap["task"].Purge()
	status = k8sV1.PodStatus{
		Phase:             k8sV1.PodRunning,
		ContainerStatuses: []k8sV1.ContainerStatus{{Name: "determined-container"}},
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
	assert.Equal(t, newPod.container.State, container.Starting)
}

func TestMultipleContainersRunning(t *testing.T) {
	// Status update test involving two containers.
	setupEntrypoint(t)
	defer cleanup(t)

	// Testing pod with two containers and one doesn't have running state.
	t.Logf("Testing two pods and one doesn't have running state")
	system, newPod, ref, podMap, _ := createPodWithMockQueue()
	newPod.container.State = container.Starting
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
	newPod.containerNames = map[string]bool{"determined-container": false, "test-pod": false}
	statusUpdate := podStatusUpdate{updatedPod: &pod}

	system.Ask(ref, statusUpdate)
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)
	assert.Equal(t, newPod.container.State, container.Starting)

	// Multiple containers, all in running state, results in a running state.
	t.Logf("Testing two pods with running states")
	system, newPod, ref, podMap, _ = createPodWithMockQueue()
	newPod.testLogStreamer = true

	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	newPod.container.State = container.Starting
	containerStatuses[1] = k8sV1.ContainerStatus{
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

	containerMsg, ok := message.(sproto.TaskContainerStateChanged)
	fmt.Println("CONTAINER MESSAGE:", containerMsg)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	if containerMsg.ContainerStarted == nil {
		t.Errorf("container started message not present")
	}
}

func TestReceivePodEventUpdate(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _ := createPodWithMockQueue()

	msg := gpuTextReplacement
	object := k8sV1.ObjectReference{Kind: "mock", Namespace: "test", Name: "MockObject"}
	newEvent := k8sV1.Event{
		InvolvedObject: object,
		Reason:         "testing",
		Message:        msg,
	}
	newPod.gpus = 99
	podMap["task"].Purge()
	assert.Equal(t, podMap["task"].GetLength(), 0)

	system.Ask(ref, podEventUpdate{event: &newEvent})
	time.Sleep(time.Second)

	assert.Equal(t, podMap["task"].GetLength(), 1)
	message, err := podMap["task"].Pop()
	if err != nil {
		t.Errorf("Unable to pop message from task receiver queue")
	}
	correctMsg := fmt.Sprintf("Pod %s: %s", object.Name, gpuTextReplacement+"99 GPUs required.")

	containerMsg, ok := message.(sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, *containerMsg.AuxMessage, correctMsg)

	// When container is in Running state, pod actor should not forward message.
	podMap["task"].Purge()
	newPod.container.State = container.Running
	system.Ask(ref, podEventUpdate{event: &newEvent})
	time.Sleep(time.Second)
	assert.Equal(t, podMap["task"].GetLength(), 0)

	//When container is in Terminated state, pod actor should not forward message.
	newPod.container.State = container.Terminated
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

	system.Ask(ref, sproto.KillTaskPod{})
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

	system.Ask(ref, sproto.KillTaskPod{})
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

	containerMsg, ok := message.(sproto.TaskContainerStateChanged)
	if !ok {
		t.Errorf("expected sproto.TaskContainerStateChanged but received %s",
			reflect.TypeOf(message))
	}

	var correctContainerStarted *sproto.TaskContainerStarted = nil
	correctFailType := "task failed without an associated exit code"
	correctErrMsg := "agent failed while container was running"
	var correctCode *agent.ExitCode = nil

	assert.Equal(t, containerMsg.ContainerStarted, correctContainerStarted)
	assert.Equal(t, containerMsg.ContainerStopped.Failure.FailureType,
		agent.FailureType(correctFailType))
	assert.Equal(t, containerMsg.ContainerStopped.Failure.ErrMsg, correctErrMsg)
	assert.Equal(t, containerMsg.ContainerStopped.Failure.ExitCode, correctCode)
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

	containerMsg, ok := message.(sproto.TaskContainerStateChanged)
	if !ok {
		t.Errorf("expected sproto.TaskContainerStateChanged but received %s",
			reflect.TypeOf(message))
	}

	var correctContainerStarted *sproto.TaskContainerStarted = nil
	var correctCode *agent.ExitCode = nil

	assert.Equal(t, containerMsg.ContainerStarted, correctContainerStarted)
	assert.Equal(t, containerMsg.ContainerStopped.Failure.FailureType,
		agent.FailureType("task failed without an associated exit code"))
	assert.Equal(t, containerMsg.ContainerStopped.Failure.ErrMsg,
		"agent failed while container was running")
	assert.Equal(t, containerMsg.ContainerStopped.Failure.ExitCode, correctCode)
}

func TestGetPodNodeInfo(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	system, newPod, ref, podMap, _ := createPodWithMockQueue()
	newPod.gpus = 99
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
	assert.Equal(t, podInfo.numGPUs, newPod.gpus)
}
