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

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func errTrialNotFound(id int) error {
	return status.Errorf(codes.NotFound, "trial %d not found", id)
}

func createTestTrial(
	t *testing.T, api *apiServer, curUser model.User,
) *model.Trial {
	exp := createTestExpWithProjectID(t, api, curUser, 1)

	task := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, api.m.db.AddTask(task))

	trial := &model.Trial{
		StartTime:    time.Now(),
		State:        model.PausedState,
		ExperimentID: exp.ID,
		TaskID:       task.TaskID,
	}
	require.NoError(t, api.m.db.AddTrial(trial))

	// Return trial exactly the way the API will generally get it.
	outTrial, err := api.m.db.TrialByID(trial.ID)
	require.NoError(t, err)
	return outTrial
}

func TestTrialAuthZ(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t)
	trial := createTestTrial(t, api, curUser)

	cases := []struct {
		DenyFuncName   string
		IDToReqCall    func(id int) error
		SkipActionFunc bool
	}{
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.TrialLogs(&apiv1.TrialLogsRequest{
				TrialId: int32(id),
			}, mockStream[*apiv1.TrialLogsResponse]{ctx})
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.TrialLogsFields(&apiv1.TrialLogsFieldsRequest{
				TrialId: int32(id),
			}, mockStream[*apiv1.TrialLogsFieldsResponse]{ctx})
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetTrialCheckpoints(ctx, &apiv1.GetTrialCheckpointsRequest{
				Id: int32(id),
			})
			return err
		}, false},
		{"CanEditExperiment", func(id int) error {
			_, err := api.KillTrial(ctx, &apiv1.KillTrialRequest{
				Id: int32(id),
			})
			return err
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetTrial(ctx, &apiv1.GetTrialRequest{
				TrialId: int32(id),
			})
			return err
		}, true},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.SummarizeTrial(ctx, &apiv1.SummarizeTrialRequest{
				TrialId: int32(id),
			})
			return err
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.CompareTrials(ctx, &apiv1.CompareTrialsRequest{
				TrialIds: []int32{int32(id)},
			})
			return err
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetTrialWorkloads(ctx, &apiv1.GetTrialWorkloadsRequest{
				TrialId: int32(id),
			})
			return err
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.GetTrialProfilerMetrics(&apiv1.GetTrialProfilerMetricsRequest{
				Labels: &trialv1.TrialProfilerMetricLabels{TrialId: int32(id)},
			}, mockStream[*apiv1.GetTrialProfilerMetricsResponse]{ctx})
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.GetTrialProfilerAvailableSeries(
				&apiv1.GetTrialProfilerAvailableSeriesRequest{
					TrialId: int32(id),
				}, mockStream[*apiv1.GetTrialProfilerAvailableSeriesResponse]{ctx})
		}, false},
		{"CanEditExperiment", func(id int) error {
			_, err := api.PostTrialProfilerMetricsBatch(ctx,
				&apiv1.PostTrialProfilerMetricsBatchRequest{
					Batches: []*trialv1.TrialProfilerMetricsBatch{
						{
							Labels: &trialv1.TrialProfilerMetricLabels{TrialId: int32(id)},
						},
					},
				})
			return err
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.GetCurrentTrialSearcherOperation(ctx,
				&apiv1.GetCurrentTrialSearcherOperationRequest{
					TrialId: int32(id),
				})
			return err
		}, false},
		{"CanEditExperiment", func(id int) error {
			_, err := api.CompleteTrialSearcherValidation(ctx,
				&apiv1.CompleteTrialSearcherValidationRequest{
					TrialId: int32(id),
				})
			return err
		}, false},
		{"CanEditExperiment", func(id int) error {
			_, err := api.ReportTrialSearcherEarlyExit(ctx,
				&apiv1.ReportTrialSearcherEarlyExitRequest{
					TrialId: int32(id),
				})
			return err
		}, false},
		{"CanEditExperiment", func(id int) error {
			_, err := api.ReportTrialProgress(ctx,
				&apiv1.ReportTrialProgressRequest{
					TrialId: int32(id),
				})
			return err
		}, false},
		{"CanEditExperiment", func(id int) error {
			_, err := api.ReportTrialTrainingMetrics(ctx,
				&apiv1.ReportTrialTrainingMetricsRequest{
					TrainingMetrics: &trialv1.TrialMetrics{TrialId: int32(id)},
				})
			return err
		}, false},
		{"CanEditExperiment", func(id int) error {
			_, err := api.ReportTrialValidationMetrics(ctx,
				&apiv1.ReportTrialValidationMetricsRequest{
					ValidationMetrics: &trialv1.TrialMetrics{TrialId: int32(id)},
				})
			return err
		}, false},
		{"CanEditExperiment", func(id int) error {
			_, err := api.PostTrialRunnerMetadata(ctx, &apiv1.PostTrialRunnerMetadataRequest{
				TrialId: int32(id),
			})
			return err
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.ExpCompareMetricNames(&apiv1.ExpCompareMetricNamesRequest{
				TrialId: []int32{int32(id)},
			}, mockStream[*apiv1.ExpCompareMetricNamesResponse]{ctx})
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, err := api.LaunchTensorboard(ctx, &apiv1.LaunchTensorboardRequest{
				TrialIds: []int32{int32(id)},
			})
			return err
		}, false},
	}

	for _, curCase := range cases {
		require.ErrorIs(t, curCase.IDToReqCall(-999), errTrialNotFound(-999))

		// Can't view trials experiment gives same error.
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), errTrialNotFound(trial.ID))

		// Experiment view error returns error unmodified.
		expectedErr := fmt.Errorf("canGetTrialError")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).Return(false, expectedErr).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), expectedErr)

		// Action func error returns error in forbidden.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()
		authZExp.On(curCase.DenyFuncName, mock.Anything, curUser, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), expectedErr)
	}
}
