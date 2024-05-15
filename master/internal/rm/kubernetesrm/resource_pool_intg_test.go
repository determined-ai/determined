package kubernetesrm

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	k8sClient "k8s.io/client-go/kubernetes"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

var (
	defaultState = sproto.SchedulingStateQueued
	defaultSlots = 3
)

func TestAllocateAndRelease(t *testing.T) {
	rp := testResourcePool(t, defaultSlots)
	allocID := model.AllocationID(uuid.NewString())

	// AllocateRequest
	allocReq := sproto.AllocateRequest{
		AllocationID: allocID,
		JobID:        model.NewJobID(),
		Name:         uuid.NewString(),
		BlockedNodes: []string{uuid.NewString(), uuid.NewString()},
	}

	rp.AllocateRequest(allocReq)
	req, ok := rp.reqList.TaskByID(allocID)

	require.True(t, ok)
	require.Equal(t, allocID, req.AllocationID)
	require.Equal(t, allocReq.JobID, req.JobID)
	require.Equal(t, allocReq.BlockedNodes, req.BlockedNodes)
	require.Equal(t, allocReq.Name, req.Name)

	// ResourcesReleased
	rp.ResourcesReleased(sproto.ResourcesReleased{
		AllocationID: allocID,
		ResourcePool: rp.poolConfig.PoolName,
	})
	req, ok = rp.reqList.TaskByID(allocID)
	require.False(t, ok)
	require.Nil(t, req)
}

func TestUpdatePodStatus(t *testing.T) {
	rp := testResourcePool(t, defaultSlots)

	cases := []struct {
		name       string
		state      sproto.SchedulingState
		runningPod int
	}{
		{"scheduled", sproto.SchedulingStateScheduled, 1},
		{"backfilled", sproto.SchedulingStateScheduledBackfilled, 1},
		{"queued", sproto.SchedulingStateQueued, 0},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, allocID := testAddAllocation(t, rp, tt.state)
			rp.Schedule()

			// Check that the allocation is queued first.
			require.Equal(t, int(sproto.SchedulingStateQueued), rp.allocationIDToRunningPods[allocID])
			require.Zero(t, rp.allocationIDToRunningPods[allocID])

			containerID := rp.allocationIDToContainerID[allocID]
			require.NotNil(t, containerID)

			req := sproto.UpdatePodStatus{
				// TODO (bradley, pods2jobs): new implementation won't require container ID
				ContainerID: string(containerID),
				State:       tt.state,
			}

			rp.UpdatePodStatus(req)

			// If scheduled (backfilled or scheduled), check that running pods++
			require.Equal(t, tt.runningPod, rp.allocationIDToRunningPods[allocID])
		})
	}
}

func TestPendingPreemption(t *testing.T) {
	rp := testResourcePool(t, defaultSlots)
	err := rp.PendingPreemption(sproto.PendingPreemption{
		AllocationID: *model.NewAllocationID(ptrs.Ptr(uuid.NewString())),
	})
	require.Equal(t, rmerrors.ErrNotSupported, err)
}

func TestSetGroupWeight(t *testing.T) {
	rp := testResourcePool(t, defaultSlots)
	err := rp.SetGroupWeight(sproto.SetGroupWeight{})
	require.Equal(t, rmerrors.UnsupportedError("set group weight is unsupported in k8s"), err)
}

func TestSetGroupPriority(t *testing.T) {
	rp := testResourcePool(t, defaultSlots)

	cases := []struct {
		name        string
		newPriority int
		preemptible bool
	}{
		{"not-preemptible", 0, false},
		{"no change", int(config.KubernetesDefaultPriority), true},
		{"increase", 100, true},
		{"decrease", 1, true},
		{"negative", -10, true}, // doesn't make sense, but it is allowed
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			jobID := model.NewJobID()

			rp.AllocateRequest(sproto.AllocateRequest{
				JobID:       jobID,
				Preemptible: tt.preemptible,
			})

			err := rp.SetGroupPriority(sproto.SetGroupPriority{
				Priority:     tt.newPriority,
				ResourcePool: rp.poolConfig.PoolName,
				JobID:        jobID,
			})

			if tt.preemptible {
				require.NoError(t, err)
				// TODO (bradley): check that the priority change is reflected in rm events
				// require.Equal(t, tt.newPriority, *rp.getOrCreateGroup(jobID).Priority)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestValidateResources(t *testing.T) {
	rp := testResourcePool(t, defaultSlots)

	cases := []struct {
		name        string
		slots       int
		fulfillable bool
	}{
		{"valid", 1, true},
		{"invalid", 100, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := rp.ValidateResources(sproto.ValidateResourcesRequest{
				Slots: tt.slots,
			})
			require.Equal(t, tt.fulfillable, res.Fulfillable)
		})
	}
}

func TestSchedule(t *testing.T) {
	rp := testResourcePool(t, defaultSlots)
	_, allocID := testAddAllocation(t, rp, defaultState)

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocID))

	rp.Schedule()

	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocID))
}

func testResourcePool(t *testing.T, slots int) *kubernetesResourcePool {
	return newResourcePool(slots, &config.ResourcePoolConfig{}, testPodsService(t), db.SingleDB())
}

func testAddAllocation(
	t *testing.T, rp *kubernetesResourcePool, state sproto.SchedulingState,
) (model.JobID, model.AllocationID) {
	jobID := model.NewJobID()
	allocID := uuid.NewString()

	allocReq := sproto.AllocateRequest{
		AllocationID: *model.NewAllocationID(&allocID),
		JobID:        jobID,
		Preemptible:  true,
		State:        state,
	}

	rp.AllocateRequest(allocReq)

	req, ok := rp.reqList.TaskByID(allocReq.AllocationID)
	require.True(t, ok)

	return jobID, req.AllocationID
}

func testPodsService(t *testing.T) *pods {
	config, err := readClientConfig("~/.kube/config")
	require.NoError(t, err)

	clientSet, err := k8sClient.NewForConfig(config)
	require.NoError(t, err)

	return &pods{
		wg:                           waitgroupx.WithContext(context.Background()),
		namespace:                    namespace,
		masterServiceName:            "master",
		clientSet:                    clientSet,
		podNameToPodHandler:          make(map[string]*pod),
		podNameToResourcePool:        make(map[string]string),
		containerIDToPodName:         make(map[string]string),
		containerIDToSchedulingState: make(map[string]sproto.SchedulingState),
		podNameToContainerID:         make(map[string]string),
		podHandlerToMetadata:         make(map[*pod]podMetadata),
		resourceRequestQueue: &requestQueue{
			failures:                 make(chan resourcesRequestFailure, 16),
			workerChan:               make(chan interface{}),
			queue:                    make([]*queuedResourceRequest, 0),
			creationInProgress:       make(set.Set[requestID]),
			pendingResourceCreations: make(map[requestID]*queuedResourceRequest),
			blockedResourceDeletions: make(map[requestID]*queuedResourceRequest),
			syslog:                   logrus.New().WithField("component", "kubernetesrm-queue"),
		},
	}
}
