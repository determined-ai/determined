//go:build integration
// +build integration

package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

func TestTaskAuthZ(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)

	trial := createTestTrial(t, api, curUser)
	taskID := string(trial.TaskID)

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
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), apiPkg.NotFoundErrs("task", taskID, true))

		// Experiment view error is returned unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(expectedErr).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), expectedErr)

		// Action func error returns err in forbidden.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(nil).Once()
		authZExp.On(curCase.DenyFuncName, mock.Anything, curUser, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.ErrorIs(t, curCase.IDToReqCall(taskID), expectedErr)
	}
}
