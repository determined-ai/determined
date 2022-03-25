package model

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

// Resources maps filenames to file sizes.
type Resources map[string]int64

// Scan converts jsonb from postgres into a Resources object.
// TODO: Combine all json.unmarshal-based Scanners into a single Scan implementation.
func (r *Resources) Scan(src interface{}) error {
	if src == nil {
		*r = nil
		return nil
	}
	bytes, ok := src.([]byte)
	if !ok {
		return errors.Errorf("unable to convert to []byte: %v", src)
	}
	obj := make(map[string]int64)
	if err := json.Unmarshal(bytes, &obj); err != nil {
		return errors.Wrapf(err, "unable to unmarshal Resources: %v", src)
	}
	*r = Resources(obj)
	return nil
}

// Checkpoint represents a row from the `checkpoints` table.
type Checkpoint struct {
	bun.BaseModel

	ID                int        `db:"id" json:"id"`
	TrialID           int        `db:"trial_id" json:"trial_id"`
	TrialRunID        int        `db:"trial_run_id" json:"-"`
	TotalBatches      int        `db:"total_batches" json:"total_batches"`
	State             State      `db:"state" json:"state"`
	EndTime           *time.Time `db:"end_time" json:"end_time"`
	UUID              *string    `db:"uuid" json:"uuid"`
	Resources         Resources  `db:"resources" json:"resources"`
	Metadata          JSONObj    `db:"metadata" json:"metadata"`
	Framework         string     `db:"framework" json:"framework"`
	Format            string     `db:"format" json:"format"`
	DeterminedVersion string     `db:"determined_version" json:"determined_version"`
}

// ValidationMetrics is based on the checkpointsv1.Metrics protobuf message.
type ValidationMetrics struct {
	NumInputs         int     `json:"num_inputs"`
	ValidationMetrics JSONObj `json:"validation_metrics"`
}

// CheckpointExpanded represents a row from the `checkpoints_expanded` view.  It is called
// "expanded" because it includes various data from non-checkpoint tables that our system
// auto-associates with checkpoints.  Likely this object is only useful to REST API endpoint code;
// most of the rest of the system will prefer the more specific Checkpoint object.
type CheckpointExpanded struct {
	bun.BaseModel

	// CheckpointExpanded is not json-serialized, so no `json:""` struct tags.
	// CheckpointExpanded is only used by bun code, so no `db:""` struct tags.

	ID                int
	TrialID           int
	TrialRunID        int
	TotalBatches      int
	State             State
	EndTime           *time.Time
	UUID              string
	Resources         Resources
	Metadata          JSONObj
	Framework         string
	Format            string
	DeterminedVersion string

	ExperimentConfig  JSONObj
	ExperimentID      int
	Hparams           JSONObj
	Metadata          JSONObj
	ValidationMetrics ValidationMetrics
	ValidationState   State
	SearcherMetric    float64
}

func (c CheckpointExpanded) ToProto(pc *protoutils.ProtoConverter) checkpointv1.Checkpoint {
	if pc.Error() != nil {
		return checkpointv1.Checkpoint{}
	}

	out := checkpointv1.Checkpoint{
		uuid:              c.UUID,
		ExperimentConfig:  pc.ToStruct(c.ExperimentConfig),
		ExperimentID:      pc.ToInt32(c.ExperimentID),
		TrialId:           c.TrialID,
		Hparams:           pc.ToStruct(c.Hparams),
		BatchNumber:       pc.ToInt32(c.TotalBatches),
		EndTime:           pc.ToTimestamp(c.EndTime),
		Resources:         c.Resources,
		Metadata:          pc.ToStruct(c.Metadata),
		Framework:         c.Framework,
		Format:            c.Format,
		DeterminedVersion: c.DeterminedVersion,
		Metrics:           c.ValidationMetrics.ToProto(pc),
		ValidationState:   pc.ToCheckpointv1State(c.ValidationState),
		State:             pc.ToCheckpointv1State(c.ValidationState),
		SearcherMetric:    c.SearcherMetric,
	}

	return out
}
