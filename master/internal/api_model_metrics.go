package internal

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) CreateTrainingMetrics(
	_ context.Context, req *apiv1.CreateTrainingMetricsRequest,
) (*apiv1.CreateTrainingMetricsResponse, error) {
	log.Infof("adding training metrics %s (trial %d, batch %d to %d)",
		req.TrainingMetrics.Uuid, req.TrainingMetrics.TrialId,
		req.TrainingMetrics.StartBatch, req.TrainingMetrics.EndBatch)
	modelT, err := model.TrainingMetricsFromProto(req.TrainingMetrics)
	if err != nil {
		return nil, errors.Wrapf(
			err, "error adding training metrics %s (trial %d, batch %d to %d) in database",
			req.TrainingMetrics.Uuid, req.TrainingMetrics.TrialId,
			req.TrainingMetrics.StartBatch, req.TrainingMetrics.EndBatch)
	}
	err = a.m.db.AddStep(modelT)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error adding training metrics %s (trial %d, batch %d to %d) in database",
			req.TrainingMetrics.Uuid, req.TrainingMetrics.TrialId,
			req.TrainingMetrics.StartBatch, req.TrainingMetrics.EndBatch)
	}
	return &apiv1.CreateTrainingMetricsResponse{TrainingMetrics: req.TrainingMetrics}, nil
}

func (a *apiServer) PatchTrainingMetrics(
	_ context.Context, req *apiv1.PatchTrainingMetricsRequest,
) (*apiv1.PatchTrainingMetricsResponse, error) {
	log.Infof("patching training metrics %s (trial %d, batch %d to %d) state %s",
		req.TrainingMetrics.Uuid, req.TrainingMetrics.TrialId,
		req.TrainingMetrics.StartBatch, req.TrainingMetrics.EndBatch, req.TrainingMetrics.State)
	modelS, err := model.TrainingMetricsFromProto(req.TrainingMetrics)
	if err != nil {
		return nil, errors.Wrapf(
			err, "error patching training metrics %s (trial %d, batch %d to %d) in database",
			req.TrainingMetrics.Uuid, req.TrainingMetrics.TrialId,
			req.TrainingMetrics.StartBatch, req.TrainingMetrics.EndBatch)
	}
	err = a.m.db.UpdateStep(
		int(req.TrainingMetrics.TrialId), int(req.TrainingMetrics.EndBatch), *modelS)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error patching training metrics %s (trial %d, batch %d to %d) in database",
			req.TrainingMetrics.Uuid, req.TrainingMetrics.TrialId,
			req.TrainingMetrics.StartBatch, req.TrainingMetrics.EndBatch)
	}
	return &apiv1.PatchTrainingMetricsResponse{TrainingMetrics: req.TrainingMetrics}, nil
}

func (a *apiServer) CreateValidationMetrics(
	_ context.Context, req *apiv1.CreateValidationMetricsRequest,
) (*apiv1.CreateValidationMetricsResponse, error) {
	log.Infof("adding validation metrics %s (trial %d, batch %d)",
		req.ValidationMetrics.Uuid, req.ValidationMetrics.TrialId, req.ValidationMetrics.TotalBatches)
	modelV, err := model.ValidationMetricsFromProto(req.ValidationMetrics)
	if err != nil {
		return nil, errors.Wrapf(
			err, "error adding validation metrics %s (trial %d, batch %d) in database",
			req.ValidationMetrics.Uuid, req.ValidationMetrics.TrialId,
			req.ValidationMetrics.TotalBatches)
	}
	err = a.m.db.AddValidation(modelV)
	if err != nil {
		return nil, errors.Wrapf(
			err, "error adding validation metrics %s (trial %d, batch %d) in database",
			req.ValidationMetrics.Uuid, req.ValidationMetrics.TrialId,
			req.ValidationMetrics.TotalBatches)
	}
	return &apiv1.CreateValidationMetricsResponse{ValidationMetrics: req.ValidationMetrics}, nil
}

func (a *apiServer) PatchValidationMetrics(
	_ context.Context, req *apiv1.PatchValidationMetricsRequest,
) (*apiv1.PatchValidationMetricsResponse, error) {
	log.Infof("patching validation metrics %s (trial %d, batch %d) state %s",
		req.ValidationMetrics.Uuid, req.ValidationMetrics.TrialId,
		req.ValidationMetrics.TotalBatches, req.ValidationMetrics.State)
	modelV, err := model.ValidationMetricsFromProto(req.ValidationMetrics)
	if err != nil {
		return nil, errors.Wrapf(
			err, "error patching validation metrics %s (trial %d, batch %d) in database",
			req.ValidationMetrics.Uuid, req.ValidationMetrics.TrialId,
			req.ValidationMetrics.TotalBatches)
	}
	err = a.m.db.UpdateValidation(
		int(req.ValidationMetrics.TrialId), int(req.ValidationMetrics.TotalBatches), *modelV)
	if err != nil {
		return nil, errors.Wrapf(err,
			"error patching validation metrics %s (trial %d, batch %d) in database",
			req.ValidationMetrics.Uuid, req.ValidationMetrics.TrialId,
			req.ValidationMetrics.TotalBatches)
	}
	return &apiv1.PatchValidationMetricsResponse{ValidationMetrics: req.ValidationMetrics}, nil
}
