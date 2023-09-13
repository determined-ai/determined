//go:build integration
// +build integration

package internal

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

func TestTaskAuthZ(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)

	mockUserArg := mock.MatchedBy(func(u model.User) bool {
		return u.ID == curUser.ID
	})

	_, task := createTestTrial(t, api, curUser)
	taskID := string(task.TaskID)

	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id string) error
	}{
		{"CanGetExperimentArtifacts", func(id string) error {
			_, err := api.GetTask(ctx, &apiv1.GetTaskRequest{
				TaskId: id,
			})
			return err
		}},
		{"CanEditExperiment", func(id string) error {
			_, err := api.ReportCheckpoint(ctx, &apiv1.ReportCheckpointRequest{
				Checkpoint: &checkpointv1.Checkpoint{TaskId: id},
			})
			return err
		}},
		{"CanGetExperimentArtifacts", func(id string) error {
			return api.TaskLogs(&apiv1.TaskLogsRequest{
				TaskId: id,
			}, &mockStream[*apiv1.TaskLogsResponse]{ctx: ctx})
		}},
		{"CanGetExperimentArtifacts", func(id string) error {
			return api.TaskLogsFields(&apiv1.TaskLogsFieldsRequest{
				TaskId: id,
			}, &mockStream[*apiv1.TaskLogsFieldsResponse]{ctx: ctx})
		}},
	}

	for _, curCase := range cases {
		require.ErrorIs(t, curCase.IDToReqCall("-999"), apiPkg.NotFoundErrs("task", "-999", true))

		// Can't view allocation's experiment gives same error.
		authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), apiPkg.NotFoundErrs("task", taskID, true))

		// Experiment view error is returned unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", mock.Anything, mockUserArg, mock.Anything).
			Return(expectedErr).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), expectedErr)

		// Action func error returns err in forbidden.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(nil).Once()
		authZExp.On(curCase.DenyFuncName, mock.Anything, mockUserArg, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), expectedErr)
	}
}

func TestAllocationAcceleratorData(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	tID := model.NewTaskID()
	task := &model.Task{
		TaskID:    tID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}
	require.NoError(t, api.m.db.AddTask(task), "failed to add task")

	aID1 := tID + "-1"
	a1 := &model.Allocation{
		TaskID:       tID,
		AllocationID: model.AllocationID(aID1),
		Slots:        1,
		ResourcePool: "default",
	}
	require.NoError(t, api.m.db.AddAllocation(a1), "failed to add allocation")
	accData := &model.AcceleratorData{
		ContainerID:      uuid.NewString(),
		AllocationID:     model.AllocationID(aID1),
		NodeName:         "NodeName",
		AcceleratorType:  "cpu-test",
		AcceleratorUuids: []string{"g", "h", "i"},
	}
	require.NoError(t,
		db.AddAllocationAcceleratorData(ctx, *accData), "failed to add allocation")

	// Add another allocation that does not have associated acceleration data with it
	aID2 := tID + "-2"
	a2 := &model.Allocation{
		TaskID:       tID,
		AllocationID: model.AllocationID(aID2),
		Slots:        1,
		ResourcePool: "default",
	}
	require.NoError(t, api.m.db.AddAllocation(a2), "failed to add allocation")

	resp, err := api.GetTaskAcceleratorData(ctx,
		&apiv1.GetTaskAcceleratorDataRequest{TaskId: tID.String()})
	require.NoError(t, err, "failed to get task AccelerationData")
	require.Equal(t, len(resp.AcceleratorData), 1)
	require.Equal(t, resp.AcceleratorData[0].AllocationId, aID1.String())
}
