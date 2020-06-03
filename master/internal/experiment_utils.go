package internal

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
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

func convertSearcherEvent(id int, event searcher.Event) (
	*model.SearcherEvent, bool, error,
) {
	var eventType string
	var content model.JSONObj

	// In order to not lose any work in case of crashing and to keep the state of the database
	// consistent with the searcher state, we indicate to the experiment that this event must be
	// flushed to the log under the following conditions:
	//  - We have a checkpoint that has occurred
	//  - We have a trial created
	//  - We have computed validation metrics
	var flush bool

	switch event := event.(type) {
	case searcher.TrialCreatedEvent:
		createBytes, err := json.Marshal(event)
		if err != nil {
			return nil, false, err
		}
		eventType = "TrialCreated"
		content = model.JSONObj{
			"trial_id":   event.TrialID,
			"request_id": uuid.UUID(event.Create.RequestID).String(),
			"operation":  json.RawMessage(createBytes),
		}
		flush = true

	case searcher.TrialClosedEvent:
		eventType = "TrialClosed"
		content = model.JSONObj{
			"request_id": event.RequestID.String(),
		}

	case searcher.CompletedMessage:
		switch event.Workload.Kind {
		case searcher.RunStep:
			event.RawMetrics = json.RawMessage("")
		case searcher.CheckpointModel:
			flush = true
		case searcher.ComputeValidationMetrics:
			flush = true
		}
		workloadBytes, err := json.Marshal(event)
		if err != nil {
			return nil, false, err
		}

		eventType = "WorkloadCompleted"
		content = model.JSONObj{
			"msg": json.RawMessage(workloadBytes),
		}
	}
	modelEvent := &model.SearcherEvent{
		ExperimentID: id,
		EventType:    eventType,
		Content:      content,
	}
	return modelEvent, flush, nil
}

// checkpointFromCheckpointMetrics converts a workload.CheckpointMetrics into a model.Checkpoint
// with the UUID, and Resources fields filled out.
func checkpointFromCheckpointMetrics(metrics searcher.CheckpointMetrics) model.Checkpoint {
	resources := model.JSONObj{}
	for key, value := range metrics.Resources {
		resources[key] = value
	}

	id := metrics.UUID.String()
	return model.Checkpoint{
		UUID:      &id,
		Resources: resources,
	}
}

func saveWorkload(db *db.PgDB, w searcher.Workload) error {
	switch w.Kind {
	case searcher.RunStep:
		return db.AddStep(model.NewStep(w.TrialID, w.StepID))
	case searcher.CheckpointModel:
		return db.AddCheckpoint(model.NewCheckpoint(w.TrialID, w.StepID))
	case searcher.ComputeValidationMetrics:
		return db.AddValidation(model.NewValidation(w.TrialID, w.StepID))
	default:
		return errors.Errorf("unexpected workload: %v", w)
	}
}

func markWorkloadErrored(db *db.PgDB, w searcher.Workload) error {
	switch w.Kind {
	case searcher.RunStep:
		return db.UpdateStep(w.TrialID, w.StepID, model.ErrorState, nil)
	case searcher.CheckpointModel:
		return db.UpdateCheckpoint(w.TrialID, w.StepID, model.ErrorState, "", nil, nil)
	case searcher.ComputeValidationMetrics:
		return db.UpdateValidation(w.TrialID, w.StepID, model.ErrorState, nil)
	default:
		return errors.Errorf("unexpected workload: %v", w)
	}
}

func markWorkloadCompleted(db *db.PgDB, msg searcher.CompletedMessage) error {
	switch msg.Workload.Kind {
	case searcher.RunStep:
		return db.UpdateStep(
			msg.Workload.TrialID, msg.Workload.StepID, model.CompletedState, msg.RunMetrics)
	case searcher.CheckpointModel:
		checkpoint := checkpointFromCheckpointMetrics(*msg.CheckpointMetrics)
		return db.UpdateCheckpoint(
			msg.Workload.TrialID, msg.Workload.StepID, model.CompletedState,
			*checkpoint.UUID, checkpoint.Resources, checkpoint.Metadata)
	case searcher.ComputeValidationMetrics:
		metrics := make(model.JSONObj)
		metrics["num_inputs"] = msg.ValidationMetrics.NumInputs
		metrics["validation_metrics"] = msg.ValidationMetrics.Metrics
		return db.UpdateValidation(
			msg.Workload.TrialID, msg.Workload.StepID, model.CompletedState, metrics)
	default:
		return errors.Errorf("unexpected workload: %v", msg.Workload)
	}
}
