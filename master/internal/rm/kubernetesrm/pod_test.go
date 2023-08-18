package kubernetesrm

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/master/pkg/tasks"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func createPod(
	allocationID model.AllocationID,
	resourceHandler *requestQueue,
	task tasks.TaskSpec,
) *pod {
	msg := StartTaskPod{
		AllocationID: allocationID,
		Spec:         task,
		Slots:        1,
	}
	clusterID := "test"
	clientSet := k8sClient.Clientset{}
	namespace := "default"
	masterIP := "0.0.0.0"
	var masterPort int32 = 32
	podInterface := &mockPodInterface{}
	configMapInterface := clientSet.CoreV1().ConfigMaps(namespace)
	resourceRequestQueue := resourceHandler
	slotType := device.CUDA
	slotResourceRequests := config.PodSlotResourceRequests{}

	newPodHandler := newPod(
		msg, clusterID, &clientSet, namespace, masterIP, masterPort,
		model.TLSClientConfig{}, model.TLSClientConfig{},
		model.LoggingConfig{DefaultLoggingConfig: &model.DefaultLoggingConfig{}},
		podInterface, configMapInterface, resourceRequestQueue, leaveKubernetesResources,
		slotType, slotResourceRequests, "default-scheduler",
	)

	return newPodHandler
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

func createPodWithMockQueue(t *testing.T, k8sRequestQueue *requestQueue) (
	*pod,
	model.AllocationID,
	*sproto.ResourcesSubscription,
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

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	failures := make(chan resourcesRequestFailure, 1024)
	if k8sRequestQueue == nil {
		podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
		configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}
		k8sRequestQueue = startRequestQueue(
			map[string]typedV1.PodInterface{"default": podInterface},
			map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
			failures,
		)
	}

	aID := model.AllocationID(uuid.NewString())
	sub := rmevents.Subscribe(aID)
	newPod := createPod(
		aID,
		k8sRequestQueue,
		commandSpec.ToTaskSpec(),
	)

	go consumeResourceRequestFailures(ctx, failures, newPod)

	err := newPod.start()
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	return newPod, aID, sub
}

var taskContainerFiles = []string{
	"k8_init_container_entrypoint.sh",
	"task-logging-setup.sh",
	"task-logging-teardown.sh",
	"task-signal-handling.sh",
	"enrich_task_logs.py",
	"singularity-entrypoint-wrapper.sh",
}

func setupEntrypoint(t *testing.T) {
	err := etc.SetRootPath(".")
	if err != nil {
		t.Logf("Failed to set root directory")
	}

	for _, file := range taskContainerFiles {
		//nolint:gosec
		f, _ := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		err = f.Close()
		if err != nil {
			t.Logf("failed to close %s", file)
		}
	}
}

func cleanup(t *testing.T) {
	for _, file := range taskContainerFiles {
		err := os.Remove(file)
		if err != nil {
			t.Logf("failed to remove %s", file)
		}
	}
}

func checkReceiveTermination(
	t *testing.T,
	update podStatusUpdate,
	newPod *pod,
	sub *sproto.ResourcesSubscription,
) {
	state, err := newPod.podStatusUpdate(update.updatedPod)
	switch {
	case err != nil, state == cproto.Terminated:
		newPod.finalize()
	}
	time.Sleep(time.Second)

	assert.Equal(t, sub.Len(), 1)
	message := sub.Get()
	containerMsg, ok := message.(*sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf(
			"expected sproto.ResourcesStateChanged but received %s",
			reflect.TypeOf(message),
		)
	}
	if containerMsg.ResourcesStopped == nil {
		t.Errorf("container stopped message not present (state=%s)", containerMsg.ResourcesState)
	}

	assert.Equal(t, newPod.container.State, cproto.Terminated)
}

func TestResourceCreationFailed(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	const correctMsg = "already exists"

	ref, aID, sub := createPodWithMockQueue(t, nil)

	purge(aID, sub)
	assert.Equal(t, sub.Len(), 0)
	// Send a second start message to trigger an additional resource creation failure.
	err := ref.start()
	require.NoError(t, err)
	time.Sleep(time.Second)

	// We expect two messages in the queue because the pod actor sends itself a stop message.
	assert.Equal(t, sub.Len(), 2)
	message := sub.Get()
	containerMsg, ok := message.(*sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.ErrorContains(t, errors.New(*containerMsg.AuxMessage), correctMsg)
}

func TestReceivePodStatusUpdateTerminated(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	typeMeta := metaV1.TypeMeta{Kind: "rest test"}
	objectMeta := metaV1.ObjectMeta{
		Name:              "test meta",
		DeletionTimestamp: &metaV1.Time{Time: time.Now()},
	}

	t.Run("pod deleting, but in pending state", func(t *testing.T) {
		t.Logf("Testing PodPending status")
		ref, aID, sub := createPodWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		pod := k8sV1.Pod{
			TypeMeta:   typeMeta,
			ObjectMeta: objectMeta,
			Status:     k8sV1.PodStatus{Phase: k8sV1.PodPending},
		}
		statusUpdate := podStatusUpdate{updatedPod: &pod}

		checkReceiveTermination(t, statusUpdate, ref, sub)
	})

	t.Run("pod failed", func(t *testing.T) {
		t.Logf("Testing PodFailed status")
		ref, aID, sub := createPodWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		pod := k8sV1.Pod{
			TypeMeta:   typeMeta,
			ObjectMeta: objectMeta,
			Status:     k8sV1.PodStatus{Phase: k8sV1.PodFailed},
		}
		statusUpdate := podStatusUpdate{updatedPod: &pod}

		checkReceiveTermination(t, statusUpdate, ref, sub)
	})

	// Pod succeeded.
	t.Run("pod succeeded", func(t *testing.T) {
		ref, aID, sub := createPodWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		pod := k8sV1.Pod{
			TypeMeta:   typeMeta,
			ObjectMeta: objectMeta,
			Status:     k8sV1.PodStatus{Phase: k8sV1.PodSucceeded},
		}
		statusUpdate := podStatusUpdate{updatedPod: &pod}

		checkReceiveTermination(t, statusUpdate, ref, sub)
	})
}

func TestMultipleContainerTerminate(t *testing.T) {
	// Status update test involving two containers.
	setupEntrypoint(t)
	defer cleanup(t)

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

	t.Run("pod running with > 1 container, and one terminated", func(t *testing.T) {
		t.Logf("two pods with one in terminated state")
		ref, aID, sub := createPodWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)
		ref.containerNames = set.FromSlice([]string{"test-pod-1", "test-pod-2"})

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
		checkReceiveTermination(t, statusUpdate, ref, sub)
	})

	t.Run("multiple pods, 1 termination, no deletion timestamp", func(t *testing.T) {
		// This results in an error, which causes pod termination and the same outcome.
		t.Logf("two pods with one in terminated state and no deletion timestamp")
		ref, aID, sub := createPodWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		pod := k8sV1.Pod{
			TypeMeta: metaV1.TypeMeta{Kind: "rest test"},
			ObjectMeta: metaV1.ObjectMeta{
				Name: "test meta",
			},
			Status: k8sV1.PodStatus{
				Phase:             k8sV1.PodRunning,
				ContainerStatuses: containerStatuses,
			},
		}
		statusUpdate := podStatusUpdate{updatedPod: &pod}
		checkReceiveTermination(t, statusUpdate, ref, sub)
	})
}

func TestReceivePodStatusUpdateAssigned(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	ref, aID, sub := createPodWithMockQueue(t, nil)
	purge(aID, sub)
	assert.Equal(t, sub.Len(), 0)

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

	assert.Equal(t, ref.container.State, cproto.Assigned)
	_, err := ref.podStatusUpdate(statusUpdate.updatedPod)
	require.NoError(t, err)

	time.Sleep(time.Second)
	assert.Equal(t, sub.Len(), 0)

	ref.container.State = cproto.Starting

	_, err = ref.podStatusUpdate(statusUpdate.updatedPod)
	require.NoError(t, err)

	time.Sleep(time.Second)
	assert.Equal(t, sub.Len(), 0)
	assert.Equal(t, ref.container.State, cproto.Starting)
}

func TestReceivePodStatusUpdateStarting(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	typeMeta := metaV1.TypeMeta{Kind: "rest test"}
	objectMeta := metaV1.ObjectMeta{
		Name: "test meta",
	}

	t.Run("pod status pending, pod scheduled", func(t *testing.T) {
		t.Logf("Testing pod scheduled with pending status")

		ref, aID, sub := createPodWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

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

		_, err := ref.podStatusUpdate(statusUpdate.updatedPod)
		require.NoError(t, err)
		time.Sleep(time.Second)

		assert.Equal(t, sub.Len(), 2)
		assert.Equal(t, ref.container.State, cproto.Starting)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		_, err = ref.podStatusUpdate(statusUpdate.updatedPod)
		require.NoError(t, err)
		time.Sleep(time.Second)

		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)
		assert.Equal(t, ref.container.State, cproto.Starting)
	})

	t.Run("pod status Running, but container status waiting", func(t *testing.T) {
		t.Logf("Testing pod running with waiting status")

		ref, aID, sub := createPodWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		containerStatuses := []k8sV1.ContainerStatus{
			{
				Name:  "determined-container",
				State: k8sV1.ContainerState{Waiting: &k8sV1.ContainerStateWaiting{}},
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
		statusUpdate := podStatusUpdate{updatedPod: &pod}

		_, err := ref.podStatusUpdate(statusUpdate.updatedPod)
		require.NoError(t, err)
		time.Sleep(time.Second)

		assert.Equal(t, sub.Len(), 2)
		assert.Equal(t, ref.container.State, cproto.Starting)
	})

	t.Run("pod status running, but no container State inside", func(t *testing.T) {
		t.Logf("Testing pod running with no status")

		ref, aID, sub := createPodWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		status := k8sV1.PodStatus{
			Phase: k8sV1.PodRunning,
			ContainerStatuses: []k8sV1.ContainerStatus{
				{Name: "determined-container"},
			},
		}
		pod := k8sV1.Pod{
			TypeMeta:   typeMeta,
			ObjectMeta: objectMeta,
			Status:     status,
		}
		statusUpdate := podStatusUpdate{updatedPod: &pod}
		_, err := ref.podStatusUpdate(statusUpdate.updatedPod)
		require.NoError(t, err)
		time.Sleep(time.Second)

		assert.Equal(t, sub.Len(), 2)
		assert.Equal(t, ref.container.State, cproto.Starting)
	})
}

func TestMultipleContainersRunning(t *testing.T) {
	// Status update test involving two containers.
	setupEntrypoint(t)
	defer cleanup(t)

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

	t.Run("pod with two containers and one doesn't have running state", func(t *testing.T) {
		t.Logf("Testing two pods and one doesn't have running state")

		ref, aID, sub := createPodWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		ref.container.State = cproto.Starting
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		status := k8sV1.PodStatus{
			Phase:             k8sV1.PodRunning,
			ContainerStatuses: containerStatuses,
		}
		pod := k8sV1.Pod{
			TypeMeta:   typeMeta,
			ObjectMeta: objectMeta,
			Status:     status,
		}
		ref.containerNames = set.FromSlice([]string{
			"determined-container",
			"test-pod",
		})
		statusUpdate := podStatusUpdate{updatedPod: &pod}

		_, err := ref.podStatusUpdate(statusUpdate.updatedPod)
		require.NoError(t, err)
		time.Sleep(time.Second)
		assert.Equal(t, sub.Len(), 0)
		assert.Equal(t, ref.container.State, cproto.Starting)
	})

	// .
	t.Run("multiple containers, all in running state, results in a running state", func(t *testing.T) {
		t.Logf("Testing two pods with running states")

		ref, aID, sub := createPodWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		ref.container.State = cproto.Starting
		containerStatuses[1] = k8sV1.ContainerStatus{
			Name:  "test-pod-2",
			State: k8sV1.ContainerState{Running: &k8sV1.ContainerStateRunning{}},
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
		statusUpdate := podStatusUpdate{updatedPod: &pod}
		_, err := ref.podStatusUpdate(statusUpdate.updatedPod)
		require.NoError(t, err)
		time.Sleep(time.Second)

		assert.Equal(t, sub.Len(), 1)
		message := sub.Get()
		containerMsg, ok := message.(*sproto.ResourcesStateChanged)
		if !ok {
			t.Errorf("expected *sproto.ResourcesStateChanged but received %s", reflect.TypeOf(message))
		}
		if containerMsg.ResourcesStarted == nil {
			t.Errorf("container started message not present")
		}
	})
}

func TestReceivePodEventUpdate(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	ref, aID, sub := createPodWithMockQueue(t, nil)
	purge(aID, sub)
	assert.Equal(t, sub.Len(), 0)

	object := k8sV1.ObjectReference{Kind: "mock", Namespace: "test", Name: "MockObject"}
	newEvent := k8sV1.Event{
		InvolvedObject: object,
		Reason:         "testing",
		Message:        "0/99 nodes are available: 99 Insufficient cpu",
	}
	ref.slots = 99
	purge(aID, sub)
	assert.Equal(t, sub.Len(), 0)

	ref.podEventUpdate(&newEvent)
	time.Sleep(time.Second) // TODO(DET-9790): Remove sleeps.

	assert.Equal(t, sub.Len(), 1)
	message := sub.Get()
	correctMsg := fmt.Sprintf("Pod %s: %s", object.Name,
		"Waiting for resources. 0 GPUs are available, 99 GPUs required")

	containerMsg, ok := message.(*sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, *containerMsg.AuxMessage, correctMsg)

	// When container is in Running state, pod actor should not forward message.
	purge(aID, sub)
	ref.container.State = cproto.Running
	ref.podEventUpdate(&newEvent)
	time.Sleep(time.Second)
	assert.Equal(t, sub.Len(), 0)

	// When container is in Terminated state, pod actor should not forward message.
	ref.container.State = cproto.Terminated
	ref.podEventUpdate(&newEvent)
	time.Sleep(time.Second)
	assert.Equal(t, sub.Len(), 0)
}

func TestReceiveContainerLog(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	mockLogMessage := "mock log message"
	ref, aID, sub := createPodWithMockQueue(t, nil)
	purge(aID, sub)
	assert.Equal(t, sub.Len(), 0)

	ref.restore = true
	ref.container.State = cproto.Running
	ref.podInterface = &mockPodInterface{logMessage: &mockLogMessage}
	purge(aID, sub)
	assert.Equal(t, sub.Len(), 0)
	err := ref.start()
	require.NoError(t, err)
	time.Sleep(time.Second)

	assert.Equal(t, sub.Len(), 1)
	message := sub.Get()
	containerMsg, ok := message.(*sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, containerMsg.RunMessage.Value, mockLogMessage)

	// reset state to starting
	ref.container.State = cproto.Starting
	mockLogMessage = "new mock log message"

	typeMeta := metaV1.TypeMeta{Kind: "running log test"}
	objectMeta := metaV1.ObjectMeta{
		Name: "test meta",
	}
	containerStatuses := []k8sV1.ContainerStatus{
		{
			Name:  "sample-container",
			State: k8sV1.ContainerState{Running: &k8sV1.ContainerStateRunning{}},
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
	ref.containerNames = set.FromSlice([]string{
		"sample-container",
	})
	statusUpdate := podStatusUpdate{updatedPod: &pod}

	_, err = ref.podStatusUpdate(statusUpdate.updatedPod)
	require.NoError(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, sub.Len(), 2)
	assert.Equal(t, ref.container.State, cproto.Running)

	message = sub.Get()
	resourceMsg, ok := message.(*sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf("expected sproto.ResourcesStateChanged but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, resourceMsg.Container.State, cproto.Running)

	message = sub.Get()
	containerMsg, ok = message.(*sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.Equal(t, containerMsg.RunMessage.Value, mockLogMessage)
}

func TestKillTaskPod(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}
	failures := make(chan resourcesRequestFailure, 1024)
	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
		failures,
	)
	ref, _, _ := createPodWithMockQueue(t, k8sRequestQueue)

	// We take a quick nap immediately so we can purge the start message after it arrives.
	time.Sleep(time.Second)
	assert.Check(t, podInterface.hasPod(ref.podName))
	ref.KillTaskPod()
	time.Sleep(time.Second)
	assert.Check(t, !podInterface.hasPod(ref.podName))
	assert.Check(t, ref.resourcesDeleted.Load())
}

func TestResourceCreationCancelled(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	podInterface := &mockPodInterface{
		pods:             make(map[string]*k8sV1.Pod),
		operationalDelay: time.Minute * numKubernetesWorkers,
	}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}
	failures := make(chan resourcesRequestFailure, 1024)
	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
		failures,
	)

	for i := 0; i < numKubernetesWorkers; i++ {
		createPodWithMockQueue(t, k8sRequestQueue)
	}
	time.Sleep(time.Second)
	ref, aID, sub := createPodWithMockQueue(t, k8sRequestQueue)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumeResourceRequestFailures(ctx, failures, ref)

	purge(aID, sub)
	assert.Equal(t, sub.Len(), 0)

	ref.KillTaskPod()

	time.Sleep(time.Second)
	assert.Equal(t, sub.Len(), 1)

	message := sub.Get()
	containerMsg, ok := message.(*sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf("expected *sproto.ResourcesStateChanged but received %s",
			reflect.TypeOf(message))
	}

	var correctContainerStarted *sproto.ResourcesStarted
	correctFailType := "task failed without an associated exit code"
	correctErrMsg := "pod handler exited while pod was running"
	var correctCode *sproto.ExitCode

	assert.Equal(t, containerMsg.ResourcesStarted, correctContainerStarted)
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.FailureType,
		sproto.FailureType(correctFailType))
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ErrMsg, correctErrMsg)
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ExitCode, correctCode)
}

func TestResourceDeletionFailed(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}
	failures := make(chan resourcesRequestFailure, 1024)
	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
		failures,
	)

	ref, aID, sub := createPodWithMockQueue(t, k8sRequestQueue)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumeResourceRequestFailures(ctx, failures, ref)

	purge(aID, sub)
	assert.Equal(t, sub.Len(), 0)
	delete(podInterface.pods, ref.podName)

	ref.KillTaskPod()
	time.Sleep(time.Second)
	assert.Equal(t, sub.Len(), 1)

	message := sub.Get()
	containerMsg, ok := message.(*sproto.ResourcesStateChanged)
	if !ok {
		t.Errorf("expected *sproto.ResourcesStateChanged but received %s",
			reflect.TypeOf(message))
	}

	var correctContainerStarted *sproto.ResourcesStarted
	var correctCode *sproto.ExitCode

	assert.Equal(t, containerMsg.ResourcesStarted, correctContainerStarted)
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.FailureType,
		sproto.FailureType("task failed without an associated exit code"))
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ErrMsg,
		"pod handler exited while pod was running")
	assert.Equal(t, containerMsg.ResourcesStopped.Failure.ExitCode, correctCode)
}

func TestGetPodNodeInfo(t *testing.T) {
	setupEntrypoint(t)
	defer cleanup(t)

	ref, aID, sub := createPodWithMockQueue(t, nil)
	ref.slots = 99
	time.Sleep(time.Second)

	purge(aID, sub)
	assert.Equal(t, sub.Len(), 0)

	podInfo := ref.getPodNodeInfo()
	time.Sleep(time.Second)

	assert.Equal(t, podInfo.nodeName, ref.pod.Spec.NodeName)
	assert.Equal(t, podInfo.numSlots, ref.slots)
}

var sentinelEvent = &sproto.ContainerLog{ContainerID: "sentinel"}

func purge(aID model.AllocationID, sub *sproto.ResourcesSubscription) {
	rmevents.Publish(aID, sentinelEvent)
	for {
		event := sub.Get()
		if event == sentinelEvent {
			return
		}
	}
}
