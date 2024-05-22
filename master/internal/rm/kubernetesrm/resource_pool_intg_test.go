package kubernetesrm

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
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
	// TODO RM-301
	t.Skip("skipping test until flake fixed")
	rp := testResourcePool(t, defaultSlots)
	_, allocID := testAddAllocation(t, rp, defaultState)

	require.True(t, rp.reschedule)
	require.False(t, rp.reqList.IsScheduled(allocID))

	rp.Schedule()

	require.False(t, rp.reschedule)
	require.True(t, rp.reqList.IsScheduled(allocID))
}

func testResourcePool(t *testing.T, slots int) *kubernetesResourcePool {
	return newResourcePool(slots, &config.ResourcePoolConfig{}, newTestJobsService(), db.SingleDB())
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
