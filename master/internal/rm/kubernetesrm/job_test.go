package kubernetesrm

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
	batchV1 "k8s.io/api/batch/v1"
	k8sV1 "k8s.io/api/core/v1"
	k8sClient "k8s.io/client-go/kubernetes"
	typedBatchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

func createPod(
	allocationID model.AllocationID,
	resourceHandler *requestQueue,
	task tasks.TaskSpec,
) *job {
	msg := startJob{
		req:          &sproto.AllocateRequest{},
		allocationID: allocationID,
		spec:         task,
		slots:        1,
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

	newJobHandler := newJob(
		configureUniqueName(msg.spec),
		msg, clusterID, &clientSet, namespace, masterIP, masterPort,
		model.TLSClientConfig{},
		podInterface, configMapInterface, resourceRequestQueue,
		slotType, slotResourceRequests, "default-scheduler",
	)

	return newJobHandler
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

func createJobWithMockQueue(t *testing.T, k8sRequestQueue *requestQueue) (
	*job,
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
		jobInterface := &mockJobInterface{jobs: make(map[string]*batchV1.Job)}
		podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
		configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}
		k8sRequestQueue = startRequestQueue(
			map[string]typedBatchV1.JobInterface{"default": jobInterface},
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

func setupEntrypoint(t *testing.T) {
	err := etc.SetRootPath("../../../static/srv")
	if err != nil {
		t.Logf("Failed to set root directory")
	}
}

func checkReceiveTermination(
	t *testing.T,
	update *batchV1.Job,
	newJob *job,
	sub *sproto.ResourcesSubscription,
) {
	state, err := newJob.jobUpdatedCallback(update)
	switch {
	case err != nil, state == cproto.Terminated:
		newJob.finalize()
	}
	time.Sleep(time.Second)

	require.True(t, waitForCondition(time.Second, func() bool {
		return sub.Len() == 1
	}), "didn't receive termination event in time")
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

	assert.Equal(t, newJob.container.State, cproto.Terminated)
}

func TestResourceCreationFailed(t *testing.T) {
	setupEntrypoint(t)

	const correctMsg = "already exists"

	ref, aID, sub := createJobWithMockQueue(t, nil)

	purge(aID, sub)
	assert.Equal(t, sub.Len(), 0)
	// Send a second start message to trigger an additional resource creation failure.
	err := ref.start()
	require.NoError(t, err)
	time.Sleep(time.Second)

	// We expect two messages in the queue because the pod actor sends itself a stop message.
	require.True(t, waitForCondition(time.Second, func() bool {
		return sub.Len() == 2
	}), "didn't receive termination event in time")
	message := sub.Get()
	containerMsg, ok := message.(*sproto.ContainerLog)
	if !ok {
		t.Errorf("expected sproto.ContainerLog but received %s", reflect.TypeOf(message))
	}
	assert.ErrorContains(t, errors.New(*containerMsg.AuxMessage), correctMsg)
}

func TestReceivePodStatusUpdateTerminated(t *testing.T) {
	setupEntrypoint(t)

	t.Run("job deleting, but in pending state", func(t *testing.T) {
		t.Logf("Testing PodPending status")
		ref, aID, sub := createJobWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		ref.jobDeletedCallback()
		ref.finalize()

		require.True(t, waitForCondition(time.Second, func() bool {
			return sub.Len() == 1
		}), "didn't receive termination event in time")
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

		assert.Equal(t, ref.container.State, cproto.Terminated)
	})

	t.Run("job failed", func(t *testing.T) {
		t.Logf("Testing PodFailed status")
		ref, aID, sub := createJobWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		job := batchV1.Job{
			Status: batchV1.JobStatus{
				Conditions: []batchV1.JobCondition{
					{
						Type:   batchV1.JobFailed,
						Status: k8sV1.ConditionTrue,
					},
				},
			},
		}
		checkReceiveTermination(t, &job, ref, sub)
	})

	t.Run("pod succeeded", func(t *testing.T) {
		ref, aID, sub := createJobWithMockQueue(t, nil)
		purge(aID, sub)
		assert.Equal(t, sub.Len(), 0)

		job := batchV1.Job{
			Status: batchV1.JobStatus{
				Conditions: []batchV1.JobCondition{
					{
						Type:   batchV1.JobComplete,
						Status: k8sV1.ConditionTrue,
					},
				},
			},
		}
		checkReceiveTermination(t, &job, ref, sub)
	})
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

var tickInterval = 10 * time.Millisecond

func waitForCondition(timeout time.Duration, condition func() bool) bool {
	for i := 0; i < int(timeout/tickInterval); i++ {
		if condition() {
			return true
		}
		time.Sleep(tickInterval)
	}
	return false
}
