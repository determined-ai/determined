package projects

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

type ExperimentMetadata struct {
	bun.BaseModel `bun:"select:experiments"`

	ID           int       `bun:"id"`
	Name         string    `bun:"config"->>'name'`
	Description  string    `bun:"config"->>'description'`
	ProjectID    int       `bun:"project_id"`
	JobID        string    `bun:"job_id"`
	Archived     bool      `bun:"archived"`
	Username     string    `bun:"username"`
	Labels       []string  `bun:"config"->'labels'`
	ResourcePool string    `bun:"config"->'resources'->>'resource_pool'`
	SearcherType string    `bun:"config"->'searcher'->'name'`
	Notes        string    `bun:"notes"`
	StartTime    time.Time `bun:"start_time"`
	EndTime      time.Time `bun:"end_time"`
	State        string    `bun:"state"`
	Progress     float64   `bun:"progress"`
	ForkedFrom   int32     `bun:"forked_from"`
	UserId       int       `bun:"user_id"`
}

func (p *ExperimentMetadata) ToProto() (*experimentv1.Experiment, error) {
	conv := protoconverter.ProtoConverter{}
	var trialIDs []int32
	parsedForkedFrom := wrapperspb.Int32(p.ForkedFrom)
	parsedProgress := wrapperspb.Double(p.Progress)
	out := &experimentv1.Experiment{
		Id:          conv.ToInt32(p.ID),
		Name:        p.Name,
		Description: p.Description,
		Archived:    p.Archived,
		Username:    p.Username,
		StartTime:   conv.ToTimestamp(p.StartTime),
		EndTime:     conv.ToTimestamp(p.EndTime),
		ProjectId:   conv.ToInt32(p.ProjectID),
		JobId:       p.JobID,
		// State:        experimentv1.State,
		Progress:     parsedProgress,
		ForkedFrom:   parsedForkedFrom,
		UserId:       conv.ToInt32(p.UserId),
		ResourcePool: p.ResourcePool,
		Notes:        p.Notes,
		Labels:       p.Labels,
		SearcherType: p.SearcherType,
		TrialIds:     trialIDs,
		NumTrials:    0,
	}
	if err := conv.Error(); err != nil {
		return nil, fmt.Errorf("converting experiment to proto: %w", err)
	}
	return out, nil
}

type ProjectMetadata struct {
	bun.BaseModel `bun:"select:projects"`

	ID                      int             `bun:"id"`
	Name                    string          `bun:"name"`
	Description             string          `bun:"description"`
	WorkspaceID             int             `bun:"workspace_id"`
	LastExperimentStartedAt time.Time       `bun:"last_experiment_started_at"`
	Notes                   []model.JSONObj `bun:"notes"`
	NumExperiments          int             `bun:"num_experiments"`
	NumActiveExperiments    int             `bun:"num_active_experiments"`
	Archived                bool            `bun:"archived"`
	Username                string          `bun:"username"`
	Immutable               bool            `bun:"immutable"`
}

func (p *ProjectMetadata) ToProto() (*projectv1.Project, error) {
	conv := protoconverter.ProtoConverter{}
	notes, err := conv.ToProjectNotes(p.Notes)
	if err != nil {
		return nil, fmt.Errorf("conversion of notes: %w", err)
	}

	out := &projectv1.Project{
		Id:                      conv.ToInt32(p.ID),
		Name:                    p.Name,
		Description:             p.Description,
		WorkspaceId:             conv.ToInt32(p.WorkspaceID),
		LastExperimentStartedAt: conv.ToTimestamp(p.LastExperimentStartedAt),
		Notes:                   notes,
		NumExperiments:          conv.ToInt32(p.NumExperiments),
		NumActiveExperiments:    conv.ToInt32(p.NumActiveExperiments),
		Archived:                p.Archived,
		Username:                p.Username,
		Immutable:               p.Immutable,
	}
	if err = conv.Error(); err != nil {
		return nil, fmt.Errorf("converting project to proto: %w", err)
	}
	return out, nil
}

// Fetch list of Projects
func List(ctx context.Context, opts db.SelectExtension) ([]*ProjectMetadata, error) {
	ps := []*ProjectMetadata{}

	q, err := opts(db.Bun().
		NewSelect().
		Model(&ps))
	if err != nil {
		return nil, fmt.Errorf("building query: %w", err)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	return ps, nil
}

// Fetch list of Experiments
func ExperimentList(ctx context.Context, opts db.SelectExtension) ([]*ExperimentMetadata,
	error) {
	exps := []*ExperimentMetadata{}

	q, err := opts(db.Bun().
		NewSelect().
		Model(&exps).
		ColumnExpr("experiment_metadata.id").
		ColumnExpr("users.username AS username").
		Join("JOIN users ON users.id = experiment_metadata.owner_id"))
	if err != nil {
		return nil, fmt.Errorf("building query: %w", err)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("listing experiments: %w", err)
	}
	return exps, nil
}

// Fetch single Project by its ID
func ByID(ctx context.Context, id int32) (*ProjectMetadata, error) {
	p := ProjectMetadata{}

	var s struct{
		NumActiveExperiments	int
		NumExperiments				int
		LastExperimentStartedAt time.Time
	}
	experiment := db.Bun().NewSelect().
		ColumnExpr("COUNT(*) AS num_experiments").
		ColumnExpr("SUM(case when state = 'ACTIVE' then 1 else 0 end) AS num_active_experiments").
		ColumnExpr("MAX(start_time) AS last_experiment_started_at").
		TableExpr("experiments").
		Where("project_id = ?", id).
		Model(&s)
	if err := experiment.Scan(ctx); err != nil {
		return nil, fmt.Errorf("getting single project: %w", err)
	}

	q := db.Bun().
		NewSelect().
		ColumnExpr("project_metadata.id, project_metadata.name, project_metadata.immutable").
		ColumnExpr("0 AS num_experiments").
		ColumnExpr("0 AS num_active_experiments").
		ColumnExpr("now() AS last_experiment_started_at").
		ColumnExpr("users.username AS username").
		Join("JOIN users ON users.id = project_metadata.user_id").
		Join("JOIN workspaces ON workspaces.id = project_metadata.workspace_id").
		Model(&p).
		Where("project_metadata.id = ?", id)

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("getting single project: %w", err)
	}
	p.NumActiveExperiments = s.NumActiveExperiments
	p.NumExperiments = s.NumExperiments
	p.LastExperimentStartedAt = s.LastExperimentStartedAt

	return &p, nil
}
