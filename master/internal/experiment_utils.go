package internal

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/workload"
)

func checkpointFromTrialIDOrUUID(
	db *db.PgDB, trialID *int, checkpointUUIDStr *string,
) (*model.Checkpoint, error) {
	var checkpoint *model.Checkpoint
	var err error

	// Attempt to find a Checkpoint object from the given IDs.
	if trialID != nil {
		checkpoint, err = db.LatestCheckpointForTrial(*trialID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get checkpoint for source trial %d", *trialID)
		}
		if checkpoint == nil {
			return nil, errors.Errorf("no checkpoint found for source trial %d", *trialID)
		}
	} else if checkpointUUIDStr != nil {
		checkpointUUID, err := uuid.Parse(*checkpointUUIDStr)
		if err != nil {
			return nil, errors.Wrap(err, "invalid source checkpoint UUID")
		}
		checkpoint, err = db.CheckpointByUUID(checkpointUUID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get source checkpoint %v", checkpointUUID)
		}
		if checkpoint == nil {
			return nil, errors.Errorf("no checkpoint found with UUID %v", checkpointUUID)
		}
	}
	return checkpoint, nil
}

// checkpointFromCheckpointMetrics converts a workload.CheckpointMetrics into a model.Checkpoint
// with the UUID, and Resources fields filled out.
func checkpointFromCheckpointMetrics(metrics workload.CheckpointMetrics) model.Checkpoint {
	resources := model.JSONObj{}
	for key, value := range metrics.Resources {
		resources[key] = value
	}

	id := metrics.UUID.String()
	return model.Checkpoint{
		UUID:      &id,
		Resources: resources,
		Framework: metrics.Framework,
		Format:    metrics.Format,
	}
}

func markWorkloadErrored(db *db.PgDB, w workload.Workload) error {
	switch w.Kind {
	case workload.RunStep:
		return db.UpdateStep(w.TrialID, w.TotalBatches(), model.Step{State: model.ErrorState})
	case workload.CheckpointModel:
		return db.UpdateCheckpoint(w.TrialID, w.TotalBatches(), model.Checkpoint{State: model.ErrorState})
	case workload.ComputeValidationMetrics:
		return db.UpdateValidation(w.TrialID, w.TotalBatches(), model.Validation{State: model.ErrorState})
	default:
		return errors.Errorf("unexpected workload in markWorkloadErrored: %v", w)
	}
}
