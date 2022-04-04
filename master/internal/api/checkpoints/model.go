package checkpoints

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

type CheckpointVersion int

const (
	CheckpointVersion0       CheckpointVersion = 1
	CheckpointVersion1       CheckpointVersion = 2
	CurrentCheckpointVersion CheckpointVersion = CheckpointVersion1
)

type CheckpointMetadata struct {
	bun.BaseModel `bun:"select:checkpoints_view"`

	ID           int                `bun:"id,nullzero"`
	UUID         uuid.UUID          `bun:"uuid"`
	TaskID       model.TaskID       `bun:"task_id"`
	AllocationID model.AllocationID `bun:"allocation_id"`
	ReportTime   time.Time          `bun:"report_time"`
	State        model.State        `bun:"state"`
	Resources    map[string]int64   `bun:"resources"`
	Metadata     model.JSONObj      `bun:"metadata"`

	CheckpointVersion CheckpointVersion `bun:"checkpoint_version"`

	CheckpointTrainingMetadata
}

type CheckpointTrainingMetadata struct {
	TrialID           int                    `bun:"trial_id"`
	ExperimentID      int                    `bun:"experiment_id"`
	ExperimentConfig  map[string]interface{} `bun:"experiment_config"`
	HParams           map[string]interface{} `bun:"hparams"`
	TrainingMetrics   map[string]interface{} `bun:"training_metrics"`
	ValidationMetrics map[string]interface{} `bun:"validation_metrics"`
	SearcherMetric    float64                `bun:"searcher_metric"`
}

func FromProto(in *checkpointv1.CheckpointMetadata) (*CheckpointMetadata, error) {
	conv := protoconverter.ProtoConverter{}
	out := &CheckpointMetadata{
		UUID:         conv.ToUUID(in.Uuid),
		TaskID:       model.TaskID(in.TaskId),
		AllocationID: model.AllocationID(in.AllocationId),
		ReportTime:   in.ReportTime.AsTime(),
		State:        conv.ToCheckpointState(in.State),
		Resources:    in.Resources,
		Metadata:     in.Metadata.AsMap(),
	}
	if err := conv.Error(); err != nil {
		return nil, fmt.Errorf("converting proto checkpoint: %w", err)
	}
	return out, nil
}

func (c *CheckpointMetadata) ToProto() (*checkpointv1.CheckpointMetadata, error) {
	conv := protoconverter.ProtoConverter{}
	out := &checkpointv1.CheckpointMetadata{
		TaskId:       c.TaskID.String(),
		AllocationId: c.AllocationID.String(),
		Uuid:         c.UUID.String(),
		ReportTime:   conv.ToTimestamp(c.ReportTime),
		Resources:    c.Resources,
		Metadata:     conv.ToStruct(c.Metadata, "metadata"),
		State:        conv.ToCheckpointv1State(string(c.State)),
		Training: &checkpointv1.CheckpointTrainingMetadata{
			TrialId:           conv.ToInt32(c.CheckpointTrainingMetadata.TrialID),
			ExperimentId:      conv.ToInt32(c.CheckpointTrainingMetadata.ExperimentID),
			ExperimentConfig:  conv.ToStruct(c.CheckpointTrainingMetadata.ExperimentConfig, "experiment_config"),
			Hparams:           conv.ToStruct(c.CheckpointTrainingMetadata.HParams, "hparams"),
			TrainingMetrics:   conv.ToStruct(c.CheckpointTrainingMetadata.TrainingMetrics, "training_metrics"),
			ValidationMetrics: conv.ToStruct(c.CheckpointTrainingMetadata.ValidationMetrics, "validation_metrics"),
			SearcherMetric:    conv.ToDoubleWrapper(c.CheckpointTrainingMetadata.SearcherMetric),
		},
	}
	if err := conv.Error(); err != nil {
		return nil, fmt.Errorf("converting checkpoint to proto: %w", err)
	}
	return out, nil
}

func (c *CheckpointMetadata) Insert(ctx context.Context) error {
	_, err := db.Bun().
		NewInsert().
		Table("checkpoints_v2").
		Model(c).
		Exec(ctx)
	return err
}

func (c *CheckpointMetadata) Upsert(ctx context.Context) error {
	switch c.CheckpointVersion {
	case CheckpointVersion0:
		_, err := db.Bun().
			NewInsert().
			Table("checkpoints").
			Model(c).
			On("CONFLICT (uuid) DO UPDATE").
			Set("state = EXCLUDED.state").
			Set("resources = EXCLUDED.resources").
			Set("metadata = EXCLUDED.metadata").
			Exec(ctx)
		return err
	case CheckpointVersion1:
		_, err := db.Bun().
			NewInsert().
			Model(c).
			Table("checkpoints_v2").
			On("CONFLICT (uuid) DO UPDATE").
			Set("task_id = EXCLUDED.task_id").
			Set("allocation_id = EXCLUDED.allocation_id").
			Set("report_time = EXCLUDED.report_time").
			Set("state = EXCLUDED.state").
			Set("resources = EXCLUDED.resources").
			Set("metadata = EXCLUDED.metadata").
			Exec(ctx)
		return err
	default:
		return fmt.Errorf("upsert with unsupported checkpoint version: %d", c.CheckpointVersion)
	}
}

func Single(ctx context.Context, opts db.SelectExtension) (*CheckpointMetadata, error) {
	c := CheckpointMetadata{}

	q, err := opts(db.Bun().
		NewSelect().
		Model(&c))
	if err != nil {
		return nil, fmt.Errorf("building query: %w", err)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("getting single checkpoint: %w", err)
	}
	return &c, nil
}

func ByUUID(ctx context.Context, uuid uuid.UUID) (*CheckpointMetadata, error) {
	return Single(ctx, func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
		return q.Where("uuid = ?", uuid), nil
	})
}

func List(ctx context.Context, opts db.SelectExtension) ([]*CheckpointMetadata, error) {
	cs := []*CheckpointMetadata{}

	q, err := opts(db.Bun().
		NewSelect().
		Model(&cs))
	if err != nil {
		return nil, fmt.Errorf("building query: %w", err)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("listing checkpoints: %w", err)
	}
	return cs, nil
}
