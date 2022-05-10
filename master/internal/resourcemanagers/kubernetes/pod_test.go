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


