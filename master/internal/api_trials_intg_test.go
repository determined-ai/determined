//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func newProtoStruct(t *testing.T, in map[string]any) *structpb.Struct {
	s, err := structpb.NewStruct(in)
	require.NoError(t, err)
	return s
}

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

func TestGetTrialWorkloads(t *testing.T) {
	api, curUser, ctx := setupAPITest(t)
	trial := createTestTrial(t, api, curUser)
	alloc := &model.Allocation{
		AllocationID: model.AllocationID(trial.TaskID + ".1"),
		TaskID:       trial.TaskID,
	}
	require.NoError(t, api.m.db.AddAllocation(alloc))

	// Test data.
	var expected []*apiv1.WorkloadContainer
	for i := 0; i < 10; i++ {
		m := &commonv1.Metrics{
			AvgMetrics: newProtoStruct(t, map[string]any{"loss": float64(100 - len(expected))}),
		}
		_, err := api.ReportTrialTrainingMetrics(ctx, &apiv1.ReportTrialTrainingMetricsRequest{
			TrainingMetrics: &trialv1.TrialMetrics{
				TrialId:        int32(trial.ID),
				TrialRunId:     0,
				StepsCompleted: int32(i),
				Metrics:        m,
			},
		})
		require.NoError(t, err)
		expected = append(expected, &apiv1.WorkloadContainer{
			Workload: &apiv1.WorkloadContainer_Training{
				&trialv1.MetricsWorkload{
					Metrics:      m,
					TotalBatches: int32(i),
					State:        experimentv1.State_STATE_COMPLETED,
				},
			},
		})

		if i == 5 || i == 9 {
			v := &commonv1.Metrics{
				AvgMetrics: newProtoStruct(t, map[string]any{"loss": float64(100 - len(expected))}),
			}
			_, err := api.ReportTrialValidationMetrics(ctx,
				&apiv1.ReportTrialValidationMetricsRequest{
					ValidationMetrics: &trialv1.TrialMetrics{
						TrialId:        int32(trial.ID),
						TrialRunId:     0,
						StepsCompleted: int32(i),
						Metrics:        v,
					},
				})
			require.NoError(t, err)
			expected = append(expected, &apiv1.WorkloadContainer{
				Workload: &apiv1.WorkloadContainer_Validation{
					&trialv1.MetricsWorkload{
						Metrics:      v,
						TotalBatches: int32(i),
						State:        experimentv1.State_STATE_COMPLETED,
					},
				},
			})
		}

		if i == 9 {
			checkpointID := uuid.New().String()
			_, err := api.ReportCheckpoint(ctx, &apiv1.ReportCheckpointRequest{
				Checkpoint: &checkpointv1.Checkpoint{
					TaskId:       string(trial.TaskID),
					AllocationId: string(alloc.AllocationID),
					Uuid:         checkpointID,
					Metadata:     newProtoStruct(t, map[string]any{"steps_completed": i}),
				},
			})
			require.NoError(t, err)

			expected = append(expected, &apiv1.WorkloadContainer{
				Workload: &apiv1.WorkloadContainer_Checkpoint{
					&trialv1.CheckpointWorkload{
						Uuid:         checkpointID,
						TotalBatches: int32(i),
						State:        checkpointv1.State_STATE_COMPLETED,
						Metadata:     newProtoStruct(t, map[string]any{"steps_completed": i}),
					},
				},
			})
		}
	}

	trialWorkloadsTestCase(ctx, t, api, &apiv1.GetTrialWorkloadsRequest{
		TrialId: int32(trial.ID),
	}, expected)
	trialWorkloadsTestCase(ctx, t, api, &apiv1.GetTrialWorkloadsRequest{
		TrialId: int32(trial.ID),
		OrderBy: apiv1.OrderBy_ORDER_BY_ASC,
	}, expected)

	var reversed []*apiv1.WorkloadContainer
	for i := len(expected) - 1; i >= 0; i-- {
		reversed = append(reversed, expected[i])
	}
	trialWorkloadsTestCase(ctx, t, api, &apiv1.GetTrialWorkloadsRequest{
		TrialId: int32(trial.ID),
		OrderBy: apiv1.OrderBy_ORDER_BY_DESC,
	}, reversed)

	// Checkpoint in sortByLast is last.
	var sortByLoss []*apiv1.WorkloadContainer
	for i := 1; i < len(reversed); i++ {
		sortByLoss = append(sortByLoss, reversed[i])
	}
	sortByLoss = append(sortByLoss, reversed[0])

	trialWorkloadsTestCase(ctx, t, api, &apiv1.GetTrialWorkloadsRequest{
		TrialId: int32(trial.ID),
		SortKey: "loss",
	}, sortByLoss)
	trialWorkloadsTestCase(ctx, t, api, &apiv1.GetTrialWorkloadsRequest{
		TrialId: int32(trial.ID),
		OrderBy: apiv1.OrderBy_ORDER_BY_DESC,
		SortKey: "loss",
	}, expected)

	var justCheckpoints, justValidations, validationOrCheckpoints []*apiv1.WorkloadContainer
	for _, e := range expected {
		w := e
		switch w.Workload.(type) {
		// case *apiv1.WorkloadContainer_Training:
		case *apiv1.WorkloadContainer_Validation:
			justValidations = append(justValidations, w)
			validationOrCheckpoints = append(validationOrCheckpoints, w)
		case *apiv1.WorkloadContainer_Checkpoint:
			justCheckpoints = append(justCheckpoints, w)
			validationOrCheckpoints = append(validationOrCheckpoints, w)
		}
	}

	trialWorkloadsTestCase(ctx, t, api, &apiv1.GetTrialWorkloadsRequest{
		TrialId: int32(trial.ID),
		Filter:  apiv1.GetTrialWorkloadsRequest_FILTER_OPTION_CHECKPOINT,
	}, justCheckpoints)
	trialWorkloadsTestCase(ctx, t, api, &apiv1.GetTrialWorkloadsRequest{
		TrialId: int32(trial.ID),
		Filter:  apiv1.GetTrialWorkloadsRequest_FILTER_OPTION_VALIDATION,
	}, justValidations)
	trialWorkloadsTestCase(ctx, t, api, &apiv1.GetTrialWorkloadsRequest{
		TrialId: int32(trial.ID),
		Filter:  apiv1.GetTrialWorkloadsRequest_FILTER_OPTION_CHECKPOINT_OR_VALIDATION,
	}, validationOrCheckpoints)

	for offset := 0; offset < len(expected); offset++ {
		for limit := 1; limit < len(expected); limit++ {
			end := offset + limit
			if end > len(expected) {
				end = len(expected)
			}
			trialWorkloadsTestCase(ctx, t, api, &apiv1.GetTrialWorkloadsRequest{
				TrialId: int32(trial.ID),
				Offset:  int32(offset),
				Limit:   int32(limit),
			}, expected[offset:end])
		}
	}
}

func trialWorkloadsTestCase(ctx context.Context, t *testing.T,
	api *apiServer, req *apiv1.GetTrialWorkloadsRequest, expected []*apiv1.WorkloadContainer,
) {
	resp, err := api.GetTrialWorkloads(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Workloads, len(expected))

	for i := 0; i < len(expected); i++ {
		// Clear timestamp since that is hard to test.
		switch w := resp.Workloads[i].Workload.(type) {
		case *apiv1.WorkloadContainer_Training:
			w.Training.EndTime = nil
		case *apiv1.WorkloadContainer_Validation:
			w.Validation.EndTime = nil
		case *apiv1.WorkloadContainer_Checkpoint:
			w.Checkpoint.EndTime = nil
		}

		proto.Equal(expected[i], resp.Workloads[i])
		require.Equal(t, expected[i], resp.Workloads[i])
	}
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
		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(false, nil).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), errTrialNotFound(trial.ID))

		// Experiment view error returns error unmodified.
		expectedErr := fmt.Errorf("canGetTrialError")
		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(false, expectedErr).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), expectedErr)

		// Action func error returns error in forbidden.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", curUser, mock.Anything).Return(true, nil).Once()
		authZExp.On(curCase.DenyFuncName, curUser, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), expectedErr)
	}
}
