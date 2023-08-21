//go:build integration
// +build integration

package internal

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func createTestTrial(
	t *testing.T, api *apiServer, curUser model.User,
) (*model.Trial, *model.Task) {
	exp := createTestExpWithProjectID(t, api, curUser, 1)

	task := &model.Task{
		TaskType:   model.TaskTypeTrial,
		LogVersion: model.TaskLogVersion1,
		StartTime:  time.Now(),
		TaskID:     trialTaskID(exp.ID, model.NewRequestID(rand.Reader)),
	}
	require.NoError(t, api.m.db.AddTask(task))

	trial := &model.Trial{
		StartTime:    time.Now(),
		State:        model.PausedState,
		ExperimentID: exp.ID,
	}
	require.NoError(t, db.AddTrial(context.TODO(), trial, task.TaskID))

	// Return trial exactly the way the API will generally get it.
	outTrial, err := db.TrialByID(context.TODO(), trial.ID)
	require.NoError(t, err)
	return outTrial, task
}

func createTestTrialWithMetrics(
	ctx context.Context, t *testing.T, api *apiServer, curUser model.User, includeBatchMetrics bool,
) (*model.Trial, map[model.MetricGroup][]*commonv1.Metrics) {
	var trainingMetrics, validationMetrics []*commonv1.Metrics
	trial, _ := createTestTrial(t, api, curUser)
	metrics := make(map[model.MetricGroup][]*commonv1.Metrics)

	for i := 0; i < 10; i++ {
		trainMetrics := &commonv1.Metrics{
			AvgMetrics: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"epoch": {
						Kind: &structpb.Value_NumberValue{
							NumberValue: float64(i),
						},
					},
					"loss": {
						Kind: &structpb.Value_NumberValue{
							NumberValue: float64(i),
						},
					},

					"zgroup_b/me.t r%i]\\c_1": {
						Kind: &structpb.Value_NumberValue{
							NumberValue: float64(i),
						},
					},
					"textMetric": {
						Kind: &structpb.Value_StringValue{
							StringValue: "random_text",
						},
					},
				},
			},
		}

		group := model.MetricGroup("mygroup")
		_, err := api.ReportTrialMetrics(ctx,
			&apiv1.ReportTrialMetricsRequest{
				Metrics: &trialv1.TrialMetrics{
					TrialId:        int32(trial.ID),
					TrialRunId:     0,
					StepsCompleted: int32(i),
					Metrics:        trainMetrics,
				},
				Group: group.ToString(),
			})
		require.NoError(t, err)
		metrics[group] = append(metrics[group], trainMetrics)

		if includeBatchMetrics {
			trainMetrics.BatchMetrics = []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"batch_loss": {
							Kind: &structpb.Value_NumberValue{
								NumberValue: float64(i),
							},
						},
					},
				},
			}
		}

		_, err = api.ReportTrialTrainingMetrics(ctx,
			&apiv1.ReportTrialTrainingMetricsRequest{
				TrainingMetrics: &trialv1.TrialMetrics{
					TrialId:        int32(trial.ID),
					TrialRunId:     0,
					StepsCompleted: int32(i),
					Metrics:        trainMetrics,
				},
			})
		require.NoError(t, err)
		trainingMetrics = append(trainingMetrics, trainMetrics)

		valMetrics := &commonv1.Metrics{
			AvgMetrics: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"epoch": {
						Kind: &structpb.Value_NumberValue{
							NumberValue: float64(i),
						},
					},
					"loss": {
						Kind: &structpb.Value_NumberValue{
							NumberValue: float64(i),
						},
					},

					"val_loss2": {
						Kind: &structpb.Value_NumberValue{
							NumberValue: float64(i),
						},
					},
					"textMetric": {
						Kind: &structpb.Value_StringValue{
							StringValue: "random_text",
						},
					},
				},
			},
		}
		_, err = api.ReportTrialValidationMetrics(ctx,
			&apiv1.ReportTrialValidationMetricsRequest{
				ValidationMetrics: &trialv1.TrialMetrics{
					TrialId:        int32(trial.ID),
					TrialRunId:     0,
					StepsCompleted: int32(i),
					Metrics:        valMetrics,
				},
			})
		require.NoError(t, err)
		validationMetrics = append(validationMetrics, valMetrics)
	}

	metrics[model.TrainingMetricGroup] = trainingMetrics
	metrics[model.ValidationMetricGroup] = validationMetrics

	return trial, metrics
}

func compareMetrics(
	t *testing.T, trialIDs []int,
	resp []*trialv1.MetricsReport, expected []*commonv1.Metrics, isValidation bool,
) {
	require.NotNil(t, resp)

	trialIndex := 0
	totalBatches := 0
	for i, actual := range resp {
		if i != 0 && i%(len(expected)/len(trialIDs)) == 0 {
			trialIndex++
			totalBatches = 0
		}

		metrics := map[string]any{
			"avg_metrics":   expected[i].AvgMetrics.AsMap(),
			"batch_metrics": nil,
		}
		if expected[i].BatchMetrics != nil {
			var batchMetrics []any
			for _, b := range expected[i].BatchMetrics {
				batchMetrics = append(batchMetrics, b.AsMap())
			}
			metrics["batch_metrics"] = batchMetrics
		}
		if isValidation {
			metrics = map[string]any{
				"validation_metrics": expected[i].AvgMetrics.AsMap(),
			}
		}
		protoStruct, err := structpb.NewStruct(metrics)
		require.NoError(t, err)

		expectedRow := &trialv1.MetricsReport{
			TrialId:      int32(trialIDs[trialIndex]),
			EndTime:      actual.EndTime,
			Metrics:      protoStruct,
			TotalBatches: int32(totalBatches),
			TrialRunId:   int32(0),
			Id:           actual.Id,
		}
		proto.Equal(actual, expectedRow)
		require.Equal(t, expectedRow.Metrics.AsMap(), actual.Metrics.AsMap())

		totalBatches++
	}
}

func isMultiTrialSampleCorrect(expectedMetrics []*commonv1.Metrics,
	actualMetrics *apiv1.DownsampledMetrics,
) bool {
	// Checking if metric names and their values are equal.
	for i := 0; i < len(actualMetrics.Data); i++ {
		allActualAvgMetrics := actualMetrics.Data
		epoch := int(*allActualAvgMetrics[i].Epoch)
		// use epoch to match because in downsampling returned values are randomized.
		expectedAvgMetrics := expectedMetrics[epoch].AvgMetrics.AsMap()
		for metricName := range expectedAvgMetrics {
			actualAvgMetrics := allActualAvgMetrics[i].Values.AsMap()
			switch expectedAvgMetrics[metricName].(type) { //nolint:gocritic
			case float64:
				expectedVal := expectedAvgMetrics[metricName].(float64)
				if metricName == "epoch" {
					if expectedVal != float64(*allActualAvgMetrics[i].Epoch) {
						return false
					}
					continue
				}
				if actualAvgMetrics[metricName] == nil {
					return false
				}
				actualVal := actualAvgMetrics[metricName].(float64)
				if expectedVal != actualVal {
					return false
				}
			case string:
				if actual, ok := actualAvgMetrics[metricName].(string); !ok {
					return false
				} else if actual != expectedAvgMetrics[metricName].(string) {
					return false
				}
			default:
				panic("unexpected metric type in multi trial sample")
			}
		}
	}
	return true
}

func TestMultiTrialSampleSpecialMetrics(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	trial, _ := createTestTrialWithMetrics(
		ctx, t, api, curUser, false)

	maxDataPoints := 7

	actualMetrics, err := api.multiTrialSample(int32(trial.ID), []string{},
		"", maxDataPoints, 0, 10, nil, []string{
			"mygroup.zgroup_b/me.t r%i]\\c_1",
		})
	require.Equal(t, 1, len(actualMetrics))
	require.NoError(t, err)
	mygroup := actualMetrics[0]
	require.Equal(t, maxDataPoints, len(mygroup.Data))
	require.Equal(t, 1, len(mygroup.Data[0].Values.AsMap()))
}

func TestMultiTrialSampleMetrics(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	trial, expectedMetrics := createTestTrialWithMetrics(
		ctx, t, api, curUser, false)

	expectedTrainMetrics := expectedMetrics[model.TrainingMetricGroup]
	expectedValMetrics := expectedMetrics[model.ValidationMetricGroup]
	maxDataPoints := 7

	var trainMetricNames []string
	var metricIds []string
	for metricName := range expectedTrainMetrics[0].AvgMetrics.AsMap() {
		trainMetricNames = append(trainMetricNames, metricName)
		metricIds = append(metricIds, "training."+metricName)
	}
	actualTrainingMetrics, err := api.multiTrialSample(int32(trial.ID), trainMetricNames,
		model.TrainingMetricGroup, maxDataPoints, 0, 10, nil, []string{})
	require.NoError(t, err)
	require.Equal(t, 1, len(actualTrainingMetrics))

	var validationMetricNames []string
	for metricName := range expectedValMetrics[0].AvgMetrics.AsMap() {
		validationMetricNames = append(validationMetricNames, metricName)
		metricIds = append(metricIds, "validation."+metricName)
	}
	actualValidationTrainingMetrics, err := api.multiTrialSample(int32(trial.ID),
		validationMetricNames, model.ValidationMetricGroup, maxDataPoints,
		0, 10, nil, []string{})
	require.Equal(t, 1, len(actualValidationTrainingMetrics))
	require.NoError(t, err)

	var genericMetricNames []string
	for metricName := range expectedValMetrics[0].AvgMetrics.AsMap() {
		genericMetricNames = append(genericMetricNames, metricName)
		metricIds = append(metricIds, "mygroup."+metricName)
	}
	actualGenericTrainingMetrics, err := api.multiTrialSample(int32(trial.ID),
		genericMetricNames, model.MetricGroup("mygroup"), maxDataPoints,
		0, 10, nil, []string{})
	require.Equal(t, 1, len(actualGenericTrainingMetrics))
	require.NoError(t, err)

	require.True(t, isMultiTrialSampleCorrect(expectedTrainMetrics, actualTrainingMetrics[0]))
	require.True(t, isMultiTrialSampleCorrect(expectedValMetrics, actualValidationTrainingMetrics[0]))

	actualAllMetrics, err := api.multiTrialSample(int32(trial.ID), []string{},
		"", maxDataPoints, 0, 10, nil, metricIds)
	require.Equal(t, 3, len(actualAllMetrics))
	require.NoError(t, err)
	require.Equal(t, maxDataPoints, len(actualAllMetrics[1].Data)) // max datapoints check
	require.Equal(t, maxDataPoints, len(actualAllMetrics[2].Data)) // max datapoints check
	require.True(t, isMultiTrialSampleCorrect(expectedTrainMetrics, actualAllMetrics[1]))
	require.True(t, isMultiTrialSampleCorrect(expectedValMetrics, actualAllMetrics[2]))
}

func TestStreamTrainingMetrics(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	var trials []*model.Trial
	var trainingMetrics, validationMetrics [][]*commonv1.Metrics
	for _, haveBatchMetrics := range []bool{false, true} {
		trial, metrics := createTestTrialWithMetrics(
			ctx, t, api, curUser, haveBatchMetrics)
		trials = append(trials, trial)
		trainMetrics := metrics[model.TrainingMetricGroup]
		valMetrics := metrics[model.ValidationMetricGroup]
		trainingMetrics = append(trainingMetrics, trainMetrics)
		validationMetrics = append(validationMetrics, valMetrics)
	}

	cases := []struct {
		requestFunc  func(trialIDs []int32) ([]*trialv1.MetricsReport, error)
		metrics      [][]*commonv1.Metrics
		isValidation bool
	}{
		{
			func(trialIDs []int32) ([]*trialv1.MetricsReport, error) {
				res := &mockStream[*apiv1.GetTrainingMetricsResponse]{ctx: ctx}
				err := api.GetTrainingMetrics(&apiv1.GetTrainingMetricsRequest{
					TrialIds: trialIDs,
				}, res)
				if err != nil {
					return nil, err
				}
				var out []*trialv1.MetricsReport
				for _, d := range res.data {
					out = append(out, d.Metrics...)
				}
				return out, nil
			}, trainingMetrics, false,
		},
		{
			func(trialIDs []int32) ([]*trialv1.MetricsReport, error) {
				res := &mockStream[*apiv1.GetValidationMetricsResponse]{ctx: ctx}
				err := api.GetValidationMetrics(&apiv1.GetValidationMetricsRequest{
					TrialIds: trialIDs,
				}, res)
				if err != nil {
					return nil, err
				}
				var out []*trialv1.MetricsReport
				for _, d := range res.data {
					out = append(out, d.Metrics...)
				}
				return out, nil
			}, validationMetrics, true,
		},
	}
	for _, curCase := range cases {
		// No trial IDs.
		_, err := curCase.requestFunc([]int32{})
		require.Error(t, err)
		require.Equal(t, status.Code(err), codes.InvalidArgument)

		// Trial IDs not found.
		_, err = curCase.requestFunc([]int32{-1})
		require.Equal(t, status.Code(err), codes.NotFound)

		// One trial.
		resp, err := curCase.requestFunc([]int32{int32(trials[0].ID)})
		require.NoError(t, err)
		compareMetrics(t, []int{trials[0].ID}, resp, curCase.metrics[0], curCase.isValidation)

		// Other trial.
		resp, err = curCase.requestFunc([]int32{int32(trials[1].ID)})
		require.NoError(t, err)
		compareMetrics(t, []int{trials[1].ID}, resp, curCase.metrics[1], curCase.isValidation)

		// Both trials.
		resp, err = curCase.requestFunc([]int32{int32(trials[1].ID), int32(trials[0].ID)})
		require.NoError(t, err)
		compareMetrics(t, []int{trials[0].ID, trials[1].ID}, resp,
			append(curCase.metrics[0], curCase.metrics[1]...), curCase.isValidation)
	}
}

func TestNonNumericEpochMetric(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	expectedMetricsMap := map[string]any{
		"numeric_met": 1.5,
		"epoch":       "x",
	}
	expectedMetrics, err := structpb.NewStruct(expectedMetricsMap)
	require.NoError(t, err)

	trial, _ := createTestTrial(t, api, curUser)
	_, err = api.ReportTrialValidationMetrics(ctx, &apiv1.ReportTrialValidationMetricsRequest{
		ValidationMetrics: &trialv1.TrialMetrics{
			TrialId:        int32(trial.ID),
			TrialRunId:     0,
			StepsCompleted: 1,
			Metrics: &commonv1.Metrics{
				AvgMetrics: expectedMetrics,
			},
		},
	})
	require.Equal(t, fmt.Errorf("cannot add metric with non numeric 'epoch' value got x"), err)
}

func TestTrialsNonNumericMetrics(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	expectedMetricsMap := map[string]any{
		"string_met":  "abc",
		"numeric_met": 1.5,
		"date_met":    "2021-03-15T13:32:18.91626111111Z",
		"bool_met":    false,
		"null_met":    nil,
	}
	expectedMetrics, err := structpb.NewStruct(expectedMetricsMap)
	require.NoError(t, err)

	trial, _ := createTestTrial(t, api, curUser)
	_, err = api.ReportTrialMetrics(ctx, &apiv1.ReportTrialMetricsRequest{
		Metrics: &trialv1.TrialMetrics{
			TrialId:        int32(trial.ID),
			TrialRunId:     0,
			StepsCompleted: 1,
			Metrics: &commonv1.Metrics{
				AvgMetrics: expectedMetrics,
			},
		},
		Group: model.ValidationMetricGroup.ToString(),
	})
	require.NoError(t, err)

	t.Run("CompareTrialsNonNumeric", func(t *testing.T) {
		resp, err := api.CompareTrials(ctx, &apiv1.CompareTrialsRequest{
			TrialIds:    []int32{int32(trial.ID)},
			MetricNames: maps.Keys(expectedMetricsMap),
		})
		require.NoError(t, err)

		require.Len(t, resp.Trials, 1)
		require.Len(t, resp.Trials[0].Metrics, 1)
		require.Len(t, resp.Trials[0].Metrics[0].Data, 1)
		require.Equal(t, expectedMetricsMap, resp.Trials[0].Metrics[0].Data[0].Values.AsMap())
	})

	t.Run("TrialsSample", func(t *testing.T) {
		_, err := db.Bun().NewUpdate().Table("experiments").
			Set("config = jsonb_set(config, '{searcher,name}', ?, true)", `"custom"`).
			Where("id = ?", trial.ExperimentID).
			Exec(ctx)
		require.NoError(t, err)

		for metricName := range expectedMetricsMap {
			childCtx, cancel := context.WithCancel(ctx)
			resp := &mockStream[*apiv1.TrialsSampleResponse]{ctx: childCtx}
			go func() {
				for i := 0; i < 100; i++ {
					if len(resp.data) > 0 {
						cancel()
					}
					time.Sleep(50 * time.Millisecond)
				}
				cancel()
			}()

			err = api.TrialsSample(&apiv1.TrialsSampleRequest{
				ExperimentId:  int32(trial.ExperimentID),
				MetricType:    apiv1.MetricType_METRIC_TYPE_VALIDATION,
				MetricName:    metricName,
				PeriodSeconds: 1,
			}, resp)
			require.NoError(t, err)

			require.Greater(t, len(resp.data), 0)
			require.Len(t, resp.data[0].Trials, 1)
			require.Len(t, resp.data[0].Trials[0].Data, 1)
			require.Equal(t, map[string]any{
				metricName: expectedMetricsMap[metricName],
			}, resp.data[0].Trials[0].Data[0].Values.AsMap())
		}
	})
}

func TestUnusualMetricNames(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	expectedMetricsMap := map[string]any{
		"a.loss": 1.5,
		"b/loss": 2.5,
	}
	asciiSweep := ""
	for i := 1; i <= 255; i++ {
		asciiSweep += fmt.Sprintf("%c", i)
	}
	expectedMetricsMap[asciiSweep] = 3
	expectedMetrics, err := structpb.NewStruct(expectedMetricsMap)
	require.NoError(t, err)

	trial, _ := createTestTrial(t, api, curUser)
	_, err = api.ReportTrialValidationMetrics(ctx, &apiv1.ReportTrialValidationMetricsRequest{
		ValidationMetrics: &trialv1.TrialMetrics{
			TrialId:        int32(trial.ID),
			TrialRunId:     0,
			StepsCompleted: 1,
			Metrics: &commonv1.Metrics{
				AvgMetrics: expectedMetrics,
			},
		},
	})
	require.NoError(t, err)

	req := &apiv1.CompareTrialsRequest{
		TrialIds:      []int32{int32(trial.ID)},
		MaxDatapoints: 3,
		MetricNames:   []string{"a.loss", "b/loss", asciiSweep},
		StartBatches:  0,
		EndBatches:    1000,
		MetricType:    apiv1.MetricType_METRIC_TYPE_VALIDATION,
	}
	_, err = api.CompareTrials(ctx, req)
	require.NoError(t, err)
}

func TestTrialAuthZ(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)
	authZNSC := setupNSCAuthZ()
	trial, _ := createTestTrial(t, api, curUser)

	cases := []struct {
		DenyFuncName   string
		IDToReqCall    func(id int) error
		SkipActionFunc bool
	}{
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.TrialLogs(&apiv1.TrialLogsRequest{
				TrialId: int32(id),
			}, &mockStream[*apiv1.TrialLogsResponse]{ctx: ctx})
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.TrialLogsFields(&apiv1.TrialLogsFieldsRequest{
				TrialId: int32(id),
			}, &mockStream[*apiv1.TrialLogsFieldsResponse]{ctx: ctx})
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
			}, &mockStream[*apiv1.GetTrialProfilerMetricsResponse]{ctx: ctx})
		}, false},
		{"CanGetExperimentArtifacts", func(id int) error {
			return api.GetTrialProfilerAvailableSeries(
				&apiv1.GetTrialProfilerAvailableSeriesRequest{
					TrialId: int32(id),
				}, &mockStream[*apiv1.GetTrialProfilerAvailableSeriesResponse]{ctx: ctx})
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
			authZNSC.On("CanGetTensorboard", mock.Anything, curUser, mock.Anything, mock.Anything,
				mock.Anything).Return(nil).Once()
			_, err := api.LaunchTensorboard(ctx, &apiv1.LaunchTensorboardRequest{
				TrialIds: []int32{int32(id)},
			})
			return err
		}, false},
		{"CanEditExperiment", func(id int) error {
			req := &apiv1.ReportTrialSourceInfoRequest{TrialSourceInfo: &trialv1.TrialSourceInfo{
				TrialId:             int32(id),
				CheckpointUuid:      uuid.NewString(),
				TrialSourceInfoType: trialv1.TrialSourceInfoType_TRIAL_SOURCE_INFO_TYPE_INFERENCE,
			}}
			_, err := api.ReportTrialSourceInfo(ctx, req)
			return err
		}, false},
	}

	for _, curCase := range cases {
		require.ErrorIs(t, curCase.IDToReqCall(-999), apiPkg.NotFoundErrs("trial", "-999", true))
		// Can't view trials experiment gives same error.
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID),
			apiPkg.NotFoundErrs("trial", fmt.Sprint(trial.ID), true))

		// Experiment view error returns error unmodified.
		expectedErr := fmt.Errorf("canGetTrialError")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(expectedErr).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), expectedErr)

		// Action func error returns error in forbidden.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
			Return(nil).Once()
		authZExp.On(curCase.DenyFuncName, mock.Anything, curUser, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.ErrorIs(t, curCase.IDToReqCall(trial.ID), expectedErr)
	}
}

func TestTrialProtoTaskIDs(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	trial, task0 := createTestTrial(t, api, curUser)

	_, err := db.Bun().NewUpdate().Table("experiments").
		Set("best_trial_id = ?", trial.ID).
		Where("id = ?", trial.ExperimentID).Exec(ctx)
	require.NoError(t, err)

	task1 := &model.Task{
		TaskType:   model.TaskTypeTrial,
		LogVersion: model.TaskLogVersion1,
		StartTime:  task0.StartTime.Add(time.Second),
		TaskID:     trialTaskID(trial.ExperimentID, model.NewRequestID(rand.Reader)),
	}
	require.NoError(t, api.m.db.AddTask(task1))

	task2 := &model.Task{
		TaskType:   model.TaskTypeTrial,
		LogVersion: model.TaskLogVersion1,
		StartTime:  task1.StartTime.Add(time.Second),
		TaskID:     trialTaskID(trial.ExperimentID, model.NewRequestID(rand.Reader)),
	}
	require.NoError(t, api.m.db.AddTask(task2))

	_, err = db.Bun().NewInsert().Model(&[]model.TrialTaskID{
		{TrialID: trial.ID, TaskID: task1.TaskID},
		{TrialID: trial.ID, TaskID: task2.TaskID},
	}).Exec(ctx)
	require.NoError(t, err)

	taskIDs := []string{string(task0.TaskID), string(task1.TaskID), string(task2.TaskID)}

	cases := []struct {
		name string
		f    func(t *testing.T) *trialv1.Trial
	}{
		{"GetTrial", func(t *testing.T) *trialv1.Trial {
			resp, err := api.GetTrial(ctx, &apiv1.GetTrialRequest{
				TrialId: int32(trial.ID),
			})
			require.NoError(t, err)
			return resp.Trial
		}},
		{"GetExperimentTrials", func(t *testing.T) *trialv1.Trial {
			resp, err := api.GetExperimentTrials(ctx, &apiv1.GetExperimentTrialsRequest{
				ExperimentId: int32(trial.ExperimentID),
			})
			require.NoError(t, err)
			require.Len(t, resp.Trials, 1)
			return resp.Trials[0]
		}},
		/* CompareTrials previously and now sends TaskID="". We also will send TaskIDs=[].
		{"CompareTrials", func(t *testing.T) *trialv1.Trial {
			resp, err := api.CompareTrials(ctx, &apiv1.CompareTrialsRequest{
				TrialIds: []int32{int32(trial.ID)},
			})
			require.NoError(t, err)
			require.Len(t, resp.Trials, 1)
			return resp.Trials[0].Trial
		}}, */
		{"SearchExperiments", func(t *testing.T) *trialv1.Trial {
			resp, err := api.SearchExperiments(ctx, &apiv1.SearchExperimentsRequest{
				Filter: ptrs.Ptr(
					fmt.Sprintf(
						`{"filterGroup":
	{"children":[{"columnName":"id","kind":"field","operator":"=","value":%d}],
		"conjunction":"and","kind":"group"},"showArchived":false}`, trial.ExperimentID)),
			})
			require.NoError(t, err)
			require.Len(t, resp.Experiments, 1)
			return resp.Experiments[0].BestTrial
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := c.f(t)
			// Still test deprecated field for TaskID.
			require.Equal(t, string(task0.TaskID), resp.TaskId) // nolint: staticcheck
			require.Equal(t, taskIDs, resp.TaskIds)
		})
	}
}

func TestTrialLogs(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	trial, task0 := createTestTrial(t, api, curUser)

	task1 := &model.Task{
		TaskType:   model.TaskTypeTrial,
		LogVersion: model.TaskLogVersion1,
		StartTime:  task0.StartTime.Add(time.Second),
		TaskID:     trialTaskID(trial.ExperimentID, model.NewRequestID(rand.Reader)),
	}
	require.NoError(t, api.m.db.AddTask(task1))

	task2 := &model.Task{
		TaskType:   model.TaskTypeTrial,
		LogVersion: model.TaskLogVersion1,
		StartTime:  task1.StartTime.Add(time.Second),
		TaskID:     trialTaskID(trial.ExperimentID, model.NewRequestID(rand.Reader)),
	}
	require.NoError(t, api.m.db.AddTask(task2))

	_, err := db.Bun().NewInsert().Model(&[]model.TrialTaskID{
		{TrialID: trial.ID, TaskID: task1.TaskID},
		{TrialID: trial.ID, TaskID: task2.TaskID},
	}).Exec(ctx)
	require.NoError(t, err)

	var expected []string
	tasks := []model.TaskID{task0.TaskID, task1.TaskID, task2.TaskID}
	var taskLogs []*model.TaskLog
	for i, taskID := range tasks {
		for j := 0; j < 9; j++ {
			log := fmt.Sprintf("%d-%d\n", i, j)
			taskLogs = append(taskLogs, &model.TaskLog{TaskID: string(taskID), Log: log})
			expected = append(expected, log)
		}
	}
	require.NoError(t, api.m.db.AddTaskLogs(taskLogs))

	stream := &mockStream[*apiv1.TrialLogsResponse]{ctx: ctx}

	err = api.TrialLogs(&apiv1.TrialLogsRequest{
		TrialId: int32(trial.ID),
	}, stream)
	require.NoError(t, err)

	require.Equal(t, len(expected), len(stream.data))
	for i, expected := range expected {
		require.Equal(t, expected, *stream.data[i].Log)
	}

	// Retry with follow.
	newStream := &mockStream[*apiv1.TrialLogsResponse]{ctx: ctx}
	done := make(chan error, 1)
	go func() {
		done <- api.TrialLogs(&apiv1.TrialLogsRequest{
			TrialId: int32(trial.ID),
			Follow:  true,
		}, newStream)
	}()

	// Send a new log.
	log := "new log\n"
	require.NoError(t, api.m.db.AddTaskLogs(
		[]*model.TaskLog{{TaskID: string(task2.TaskID), Log: log}}))
	expected = append(expected, log)

	// Ensure we are still following.
	select {
	case <-done:
		require.NoError(t, err)
		t.Fatal("follow isn't following task logs")
	case <-time.After(10 * time.Second):
		break
	}

	// Note we only update the latest task. We only care about the latest task in following.
	_, err = db.Bun().NewUpdate().Table("tasks").
		Set("end_time = ?", time.Now().Add(-time.Hour)). // An hour ago to avoid termination delay.
		Where("task_id = ?", task2.TaskID).
		Exec(ctx)
	require.NoError(t, err)

	select {
	case <-done:
		break
	case <-time.After(30 * time.Second):
		t.Fatal("follow is following too long task logs")
	}

	require.Equal(t, len(expected), len(newStream.data))
	for i, expected := range expected {
		require.Equal(t, expected, *newStream.data[i].Log)
	}
}

func TestTrialLogFields(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	trial, task0 := createTestTrial(t, api, curUser)

	task1 := &model.Task{
		TaskType:   model.TaskTypeTrial,
		LogVersion: model.TaskLogVersion1,
		StartTime:  task0.StartTime.Add(time.Second),
		TaskID:     trialTaskID(trial.ExperimentID, model.NewRequestID(rand.Reader)),
	}
	require.NoError(t, api.m.db.AddTask(task1))

	task2 := &model.Task{
		TaskType:   model.TaskTypeTrial,
		LogVersion: model.TaskLogVersion1,
		StartTime:  task1.StartTime.Add(time.Second),
		TaskID:     trialTaskID(trial.ExperimentID, model.NewRequestID(rand.Reader)),
	}
	require.NoError(t, api.m.db.AddTask(task2))

	_, err := db.Bun().NewInsert().Model(&[]model.TrialTaskID{
		{TrialID: trial.ID, TaskID: task1.TaskID},
		{TrialID: trial.ID, TaskID: task2.TaskID},
	}).Exec(ctx)
	require.NoError(t, err)

	expectedContainerIDs := make(map[string]bool)
	tasks := []model.TaskID{task0.TaskID, task1.TaskID, task2.TaskID}
	var taskLogs []*model.TaskLog
	for i, taskID := range tasks {
		containerID := fmt.Sprintf("id-%d", i)
		taskLogs = append(taskLogs, &model.TaskLog{
			TaskID:      string(taskID),
			Log:         "test log",
			ContainerID: ptrs.Ptr(containerID),
		})
		expectedContainerIDs[containerID] = true
	}
	require.NoError(t, api.m.db.AddTaskLogs(taskLogs))

	stream := &mockStream[*apiv1.TrialLogsFieldsResponse]{ctx: ctx}

	err = api.TrialLogsFields(&apiv1.TrialLogsFieldsRequest{
		TrialId: int32(trial.ID),
	}, stream)
	require.NoError(t, err)

	actualContainerIDs := make(map[string]bool)
	for _, s := range stream.data {
		for _, containerID := range s.ContainerIds {
			actualContainerIDs[containerID] = true
		}
	}
	require.Equal(t, expectedContainerIDs, actualContainerIDs)

	// Retry with follow.
	newStream := &mockStream[*apiv1.TrialLogsFieldsResponse]{ctx: ctx}
	done := make(chan error, 1)
	go func() {
		done <- api.TrialLogsFields(&apiv1.TrialLogsFieldsRequest{
			TrialId: int32(trial.ID),
			Follow:  true,
		}, newStream)
	}()

	// Send a new log.
	containerID := "newContainerID"
	require.NoError(t, api.m.db.AddTaskLogs(
		[]*model.TaskLog{{
			TaskID:      string(task2.TaskID),
			Log:         "test log",
			ContainerID: ptrs.Ptr(containerID),
		}}))
	expectedContainerIDs[containerID] = true

	// Ensure we are still following.
	select {
	case <-done:
		require.NoError(t, err)
		t.Fatal("follow isn't following task logs")
	case <-time.After(10 * time.Second):
		break
	}

	// Note we only update the latest task. We only care about the latest task in following.
	_, err = db.Bun().NewUpdate().Table("tasks").
		Set("end_time = ?", time.Now().Add(-time.Hour)). // An hour ago to avoid termination delay.
		Where("task_id = ?", task2.TaskID).
		Exec(ctx)
	require.NoError(t, err)

	select {
	case <-done:
		break
	case <-time.After(30 * time.Second):
		t.Fatal("follow is following too long task logs")
	}

	actualContainerIDs = make(map[string]bool)
	for _, s := range newStream.data {
		for _, containerID := range s.ContainerIds {
			actualContainerIDs[containerID] = true
		}
	}
	require.Equal(t, expectedContainerIDs, actualContainerIDs)
}

func compareTrialsResponseToBatches(resp *apiv1.CompareTrialsResponse) []int32 {
	compTrial := resp.Trials[0]
	compMetrics := compTrial.Metrics[0]

	sampleBatches := []int32{}

	for _, m := range compMetrics.Data {
		sampleBatches = append(sampleBatches, m.Batches)
	}

	return sampleBatches
}

func TestCompareTrialsSampling(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	trial, _ := createTestTrialWithMetrics(
		ctx, t, api, curUser, false)

	const DATAPOINTS = 3

	req := &apiv1.CompareTrialsRequest{
		TrialIds:      []int32{int32(trial.ID)},
		MaxDatapoints: DATAPOINTS,
		MetricNames:   []string{"loss"},
		StartBatches:  0,
		EndBatches:    1000,
		MetricType:    apiv1.MetricType_METRIC_TYPE_TRAINING,
	}

	resp, err := api.CompareTrials(ctx, req)
	require.NoError(t, err)

	sampleBatches1 := compareTrialsResponseToBatches(resp)
	require.Equal(t, DATAPOINTS, len(sampleBatches1))

	resp, err = api.CompareTrials(ctx, req)
	require.NoError(t, err)

	sampleBatches2 := compareTrialsResponseToBatches(resp)

	require.Equal(t, sampleBatches1, sampleBatches2)
}

func createTestTrialInferenceMetrics(ctx context.Context, t *testing.T, api *apiServer, id int32) {
	var trialMetrics map[model.MetricGroup][]map[string]any
	require.NoError(t, json.Unmarshal([]byte(
		`{"inference": [{"a":1}, {"b":2}]}`,
	), &trialMetrics))
	for mType, metricsList := range trialMetrics {
		for _, m := range metricsList {
			metrics, err := structpb.NewStruct(m)
			require.NoError(t, err)
			err = api.m.db.AddTrialMetrics(ctx,
				&trialv1.TrialMetrics{
					TrialId:        id,
					TrialRunId:     int32(0),
					StepsCompleted: int32(0),
					Metrics: &commonv1.Metrics{
						AvgMetrics: metrics,
					},
				},
				mType,
			)
			require.NoError(t, err)
		}
	}
}

func TestTrialSourceInfoCheckpoint(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTest(t, nil)
	infTrial, _ := createTestTrial(t, api, curUser)
	infTrial2, _ := createTestTrial(t, api, curUser)
	createTestTrialInferenceMetrics(ctx, t, api, int32(infTrial.ID))

	// Create a checkpoint to index with
	checkpointUUID := createVersionTwoCheckpoint(ctx, t, api, curUser, map[string]int64{"a": 1})

	// Create a TrialSourceInfo associated with each of the two trials.
	resp, err := trials.CreateTrialSourceInfo(
		ctx, &trialv1.TrialSourceInfo{
			TrialId:             int32(infTrial.ID),
			CheckpointUuid:      checkpointUUID,
			TrialSourceInfoType: trialv1.TrialSourceInfoType_TRIAL_SOURCE_INFO_TYPE_INFERENCE,
		},
	)
	require.NoError(t, err)
	require.Equal(t, resp.TrialId, int32(infTrial.ID))
	require.Equal(t, resp.CheckpointUuid, checkpointUUID)

	resp, err = trials.CreateTrialSourceInfo(
		ctx, &trialv1.TrialSourceInfo{
			TrialId:             int32(infTrial2.ID),
			CheckpointUuid:      checkpointUUID,
			TrialSourceInfoType: trialv1.TrialSourceInfoType_TRIAL_SOURCE_INFO_TYPE_INFERENCE,
		},
	)
	require.NoError(t, err)
	require.Equal(t, resp.TrialId, int32(infTrial2.ID))
	require.Equal(t, resp.CheckpointUuid, checkpointUUID)

	authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
		Return(nil).Times(3)
	authZExp.On("CanGetExperimentArtifacts", mock.Anything, curUser, mock.Anything).
		Return(nil).Times(3)

	// If there are no restrictions, we should see all the trials
	getCkptResp, getErr := api.GetTrialMetricsBySourceInfoCheckpoint(
		ctx, &apiv1.GetTrialMetricsBySourceInfoCheckpointRequest{CheckpointUuid: checkpointUUID},
	)
	require.NoError(t, getErr)
	require.Equal(t, len(getCkptResp.Data), 2)

	// Only infTrial should have generic metrics attached.
	for _, tsim := range getCkptResp.Data {
		if tsim.TrialId == int32(infTrial.ID) {
			// One aggregated MetricsReport
			require.Equal(t, len(tsim.MetricReports), 1)
		} else {
			require.Empty(t, tsim.MetricReports)
		}
	}

	infTrialExp, err := db.ExperimentByID(ctx, infTrial.ExperimentID)
	require.NoError(t, err)
	infTrial2Exp, err := db.ExperimentByID(ctx, infTrial2.ExperimentID)
	require.NoError(t, err)

	// All experiments can be seen
	authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).
		Return(nil).Times(3)
	// We can see the experiment that generated the checkpoint
	authZExp.On("CanGetExperimentArtifacts", mock.Anything, curUser, mock.Anything).
		Return(nil).Once()
	// We can't see the experiment for infTrial
	authZExp.On("CanGetExperimentArtifacts", mock.Anything, curUser, infTrialExp).
		Return(authz2.PermissionDeniedError{}).Once()
	// We can see the experiment for infTrial2
	authZExp.On("CanGetExperimentArtifacts", mock.Anything, curUser, infTrial2Exp).
		Return(nil).Once()
	getCkptResp, getErr = api.GetTrialMetricsBySourceInfoCheckpoint(
		ctx, &apiv1.GetTrialMetricsBySourceInfoCheckpointRequest{CheckpointUuid: checkpointUUID},
	)
	require.NoError(t, getErr)
	// Only infTrial2 should be visible
	require.Equal(t, len(getCkptResp.Data), 1)
	require.Equal(t, getCkptResp.Data[0].TrialId, int32(infTrial2.ID))
}

func TestTrialSourceInfoModelVersion(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	infTrial, _ := createTestTrial(t, api, curUser)
	infTrial2, _ := createTestTrial(t, api, curUser)
	createTestTrialInferenceMetrics(ctx, t, api, int32(infTrial.ID))

	// Create a checkpoint to index with
	checkpointUUID := createVersionTwoCheckpoint(ctx, t, api, curUser, map[string]int64{"a": 1})

	// Create a model_version to index with
	conv := &protoconverter.ProtoConverter{}
	modelVersion := RegisterCheckpointAsModelVersion(t, api.m.db, conv.ToUUID(checkpointUUID))

	// Create a TrialSourceInfo associated with each of the two trials.
	resp, err := trials.CreateTrialSourceInfo(
		ctx, &trialv1.TrialSourceInfo{
			TrialId:             int32(infTrial.ID),
			CheckpointUuid:      checkpointUUID,
			TrialSourceInfoType: trialv1.TrialSourceInfoType_TRIAL_SOURCE_INFO_TYPE_INFERENCE,
			ModelId:             &modelVersion.Model.Id,
			ModelVersion:        &modelVersion.Version,
		},
	)
	require.NoError(t, err)
	require.Equal(t, resp.TrialId, int32(infTrial.ID))
	require.Equal(t, resp.CheckpointUuid, checkpointUUID)

	resp, err = trials.CreateTrialSourceInfo(
		ctx, &trialv1.TrialSourceInfo{
			TrialId:             int32(infTrial2.ID),
			CheckpointUuid:      checkpointUUID,
			TrialSourceInfoType: trialv1.TrialSourceInfoType_TRIAL_SOURCE_INFO_TYPE_INFERENCE,
		},
	)
	require.NoError(t, err)
	require.Equal(t, resp.TrialId, int32(infTrial2.ID))
	require.Equal(t, resp.CheckpointUuid, checkpointUUID)

	getMVResp, getMVErr := api.GetTrialSourceInfoMetricsByModelVersion(
		ctx, &apiv1.GetTrialSourceInfoMetricsByModelVersionRequest{
			ModelName:       modelVersion.Model.Name,
			ModelVersionNum: modelVersion.Version,
		},
	)
	require.NoError(t, getMVErr)
	// One trial is valid and it has one aggregated MetricsReport
	require.Equal(t, len(getMVResp.Data), 1)
	require.Equal(t, len(getMVResp.Data[0].MetricReports), 1)
}
