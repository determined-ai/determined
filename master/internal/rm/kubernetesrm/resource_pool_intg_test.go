package kubernetesrm

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	k8sClient "k8s.io/client-go/kubernetes"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

var defaultSlots = 3

func TestSetGroupMaxSlots(t *testing.T) {
	cases := []struct {
		name          string
		originalSlots int
		newSlots      int
	}{
		{"no change", 5, 5},
		{"increase", 3, 6},
		{"decrease", 4, 2},
		{"negative slots", -2, 0}, // This is allowed, although it doesn't make sense.
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rp, jobID := testResourcePoolWithJob(t, tt.originalSlots)
			group := rp.getOrCreateGroup(jobID)
			require.Equal(t, tt.originalSlots, *group.MaxSlots)

			rp.SetGroupMaxSlots(sproto.SetGroupMaxSlots{
				MaxSlots:     &tt.newSlots,
				ResourcePool: rp.poolConfig.PoolName,
				JobID:        jobID,
			})

			slots := rp.getOrCreateGroup(jobID).MaxSlots
			require.Equal(t, tt.newSlots, *slots)
		})
	}
}

func TestAllocateAndRelease(t *testing.T) {
	cases := []struct {
		name    string
		allocID []model.AllocationID
	}{
		{"simple", []model.AllocationID{model.AllocationID(uuid.NewString())}},
		{"multiple", []model.AllocationID{
			model.AllocationID(uuid.NewString()),
			model.AllocationID(uuid.NewString()),
		}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for _, n := range tt.allocID {
				rp, jobID := testResourcePoolWithJob(t, defaultSlots)

				// AllocateRequest
				allocReq := sproto.AllocateRequest{
					AllocationID: n,
					JobID:        jobID,
					Name:         uuid.NewString(),
					BlockedNodes: []string{uuid.NewString(), uuid.NewString()},
				}
				rp.AllocateRequest(allocReq)
				req, ok := rp.reqList.TaskByID(n)
				require.True(t, ok)
				require.Equal(t, n, req.AllocationID)
				require.Equal(t, allocReq.JobID, req.JobID)
				require.Equal(t, allocReq.Name, req.Name)
				require.Equal(t, allocReq.BlockedNodes, req.BlockedNodes)

				// ResourcesReleased
				rp.ResourcesReleased(sproto.ResourcesReleased{
					AllocationID: n,
					ResourcePool: rp.poolConfig.PoolName,
				})
				req, ok = rp.reqList.TaskByID(n)
				require.False(t, ok)
				require.Nil(t, req)
				require.Empty(t, rp.allocationIDToContainerID[n])
				require.Empty(t, rp.allocationIDToJobID[n])
			}
		})
	}
}

func TestUpdatePodStatus(t *testing.T) {
	cases := []struct {
		name           string
		state          sproto.SchedulingState
		containerMatch bool
		runningPod     int
	}{
		{"scheduled", sproto.SchedulingStateScheduled, true, 1},
		{"backfilled", sproto.SchedulingStateScheduledBackfilled, true, 1},
		{"queued", sproto.SchedulingStateQueued, true, 0},
		{"container error", sproto.SchedulingStateScheduled, false, 0},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rp, jobID := testResourcePoolWithJob(t, defaultSlots)
			newAllocID := newTestPod(t, rp, jobID)

			// Check that the allocation is queued first.
			require.Equal(t, int(sproto.SchedulingStateQueued), rp.allocationIDToRunningPods[newAllocID])
			require.Zero(t, rp.allocationIDToRunningPods[newAllocID])

			req := sproto.UpdatePodStatus{
				ContainerID: "bogus",
				State:       tt.state,
			}
			if tt.containerMatch {
				req.ContainerID = string(rp.allocationIDToContainerID[newAllocID])
			}
			rp.UpdatePodStatus(req)

			// If scheduled (backfilled or scheduled), check that running pods++
			require.Equal(t, tt.runningPod, rp.allocationIDToRunningPods[newAllocID])
		})
	}
}

func TestPendingPreemption(t *testing.T) {
	rp, jobID := testResourcePoolWithJob(t, defaultSlots)
	allocID := newTestPod(t, rp, jobID)
	err := rp.PendingPreemption(sproto.PendingPreemption{
		AllocationID: allocID,
	})
	require.Equal(t, rmerrors.ErrNotSupported, err)
}

func TestSetGroupWeight(t *testing.T) {
	rp, _ := testResourcePoolWithJob(t, defaultSlots)
	err := rp.SetGroupWeight(sproto.SetGroupWeight{})
	require.Equal(t, rmerrors.UnsupportedError("set group weight is unsupported in k8s"), err)
}

func TestSetGroupPriority(t *testing.T) {
	cases := []struct {
		name        string
		newPriority int
	}{
		{"no change", int(config.KubernetesDefaultPriority)},
		{"increase", 100},
		{"decrease", 1},
		{"negative", -10}, // doesn't make sense, but it is allowed
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rp, jobID := testResourcePoolWithJob(t, defaultSlots)
			group := rp.getOrCreateGroup(jobID)
			require.Equal(t, int(config.KubernetesDefaultPriority), *group.Priority)

			err := rp.SetGroupPriority(sproto.SetGroupPriority{
				Priority:     tt.newPriority,
				ResourcePool: rp.poolConfig.PoolName,
				JobID:        jobID,
			})

			require.NoError(t, err)
			require.Equal(t, tt.newPriority, *rp.getOrCreateGroup(jobID).Priority)
		})
	}
}

func TestRecoverJobPosition(t *testing.T) {
	cases := []struct {
		name     string
		position decimal.Decimal
	}{
		{"simple", decimal.New(200, 10)},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rp, jobID := testResourcePoolWithJob(t, defaultSlots)

			rp.RecoverJobPosition(sproto.RecoverJobPosition{
				ResourcePool: rp.poolConfig.PoolName,
				JobID:        jobID,
				JobPosition:  tt.position,
			})
			require.Equal(t, tt.position, rp.queuePositions[jobID])
		})
	}
}

func TestValidateResources(t *testing.T) {
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
			rp, _ := testResourcePoolWithJob(t, defaultSlots)
			res := rp.ValidateResources(sproto.ValidateResourcesRequest{
				Slots: tt.slots,
			})
			require.Equal(t, tt.fulfillable, res.Fulfillable)
		})
	}
}

func TestSchedule(t *testing.T) {
	rp, jobID := testResourcePoolWithJob(t, defaultSlots)

	allocID := newTestPod(t, rp, jobID)
	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocID))

	rp.Schedule()

	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocID))
}

func newTestPod(t *testing.T, rp *kubernetesResourcePool, jobID model.JobID) model.AllocationID {
	allocID := uuid.NewString()
	containerID := uuid.NewString()

	allocReq := sproto.AllocateRequest{
		AllocationID: *model.NewAllocationID(&allocID),
		JobID:        jobID,
		TaskID:       model.NewTaskID(),
		Name:         uuid.NewString(),
		BlockedNodes: []string{uuid.NewString(), uuid.NewString()},
		Preemptible:  true,
		State:        sproto.SchedulingStateQueued,
		SlotsNeeded:  2,
	}

	rp.allocationIDToContainerID[allocReq.AllocationID] = cproto.ID(containerID)
	rp.containerIDtoAllocationID[containerID] = allocReq.AllocationID

	rp.AllocateRequest(allocReq)

	req, ok := rp.reqList.TaskByID(allocReq.AllocationID)
	require.True(t, ok)

	return req.AllocationID
}

func testResourcePoolWithJob(t *testing.T, slots int) (*kubernetesResourcePool, model.JobID) {
	rp := newResourcePool(slots, &config.ResourcePoolConfig{}, testPodsService(t), db.SingleDB())
	priority := int(config.KubernetesDefaultPriority)
	jobID := model.NewJobID()
	rp.groups[jobID] = &tasklist.Group{
		JobID:    jobID,
		MaxSlots: &slots,
		Priority: &priority,
	}
	return rp, jobID
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
