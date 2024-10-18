package internal

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/apiv2"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func (a *apiServer) GetMetricsV2(
	req *apiv2.GetMetricsV2Request, resp apiv1.Determined_GetMetricsServer,
) error {
	sendFunc := func(m []*trialv1.MetricsReport) error {
		return resp.Send(&apiv1.GetMetricsResponse{Metrics: m})
	}
	if err := a.streamMetrics(resp.Context(), req.RunIds, sendFunc,
		model.MetricGroup(req.Group)); err != nil {
		return err
	}

	return nil
}

func (a *apiServer) GetTrainingMetricsV2(
	req *apiv2.GetTrainingMetricsV2Request, resp apiv1.Determined_GetTrainingMetricsServer,
) error {
	sendFunc := func(m []*trialv1.MetricsReport) error {
		return resp.Send(&apiv1.GetTrainingMetricsResponse{Metrics: m})
	}
	if err := a.streamMetrics(resp.Context(), req.RunIds, sendFunc,
		model.TrainingMetricGroup); err != nil {
		return err
	}

	return nil
}

func (a *apiServer) GetValidationMetricsV2(
	req *apiv2.GetValidationMetricsV2Request, resp apiv1.Determined_GetValidationMetricsServer,
) error {
	sendFunc := func(m []*trialv1.MetricsReport) error {
		return resp.Send(&apiv1.GetValidationMetricsResponse{Metrics: m})
	}
	if err := a.streamMetrics(resp.Context(), req.RunIds, sendFunc,
		model.ValidationMetricGroup); err != nil {
		return err
	}

	return nil
}
