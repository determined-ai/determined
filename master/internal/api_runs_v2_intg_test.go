//go:build integration
// +build integration

package internal

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/apiv2"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStreamTrainingMetricsV2(t *testing.T) {
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
			func(runIDs []int32) ([]*trialv1.MetricsReport, error) {
				res := &mockStream[*apiv1.GetTrainingMetricsResponse]{ctx: ctx}
				err := api.GetTrainingMetricsV2(&apiv2.GetTrainingMetricsV2Request{
					RunIds: runIDs,
				}, res)
				if err != nil {
					return nil, err
				}
				var out []*trialv1.MetricsReport
				for _, d := range res.getData() {
					out = append(out, d.Metrics...)
				}
				return out, nil
			}, trainingMetrics, false,
		},
		{
			func(runIDs []int32) ([]*trialv1.MetricsReport, error) {
				res := &mockStream[*apiv1.GetValidationMetricsResponse]{ctx: ctx}
				err := api.GetValidationMetricsV2(&apiv2.GetValidationMetricsV2Request{
					RunIds: runIDs,
				}, res)
				if err != nil {
					return nil, err
				}
				var out []*trialv1.MetricsReport
				for _, d := range res.getData() {
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
		require.Equal(t, codes.InvalidArgument, status.Code(err))

		// Trial IDs not found.
		_, err = curCase.requestFunc([]int32{-1})
		require.Equal(t, codes.NotFound, status.Code(err))

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
