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

	ID           int         `bun:"id"`
	Name         string      `bun:"name"`
	Description  string      `bun:"description"`
	ProjectID    int         `bun:"project_id"`
	JobID        string      `bun:"job_id"`
	Archived     bool        `bun:"archived"`
	Username     string      `bun:"username"`
	DisplayName  string      `bun:"display_name"`
	Labels       []string    `bun:"labels"`
	ResourcePool string      `bun:"resource_pool"`
	SearcherType string      `bun:"searcher_type"`
	Notes        string      `bun:"notes"`
	StartTime    *time.Time  `bun:"start_time"`
	EndTime      *time.Time  `bun:"end_time"`
	State        model.State `bun:"state"`
	Progress     float64     `bun:"progress"`
	ForkedFrom   int32       `bun:"forked_from"`
	UserID       int         `bun:"user_id"`
	NumTrials    int32       `bun:"num_trials"`
	TrialIDs     []int32     `bun:"trial_ids"`
}

func (p *ExperimentMetadata) ToProto() (*experimentv1.Experiment, error) {
	conv := protoconverter.ProtoConverter{}
	parsedForkedFrom := wrapperspb.Int32(p.ForkedFrom)
	parsedProgress := wrapperspb.Double(p.Progress)
	out := &experimentv1.Experiment{
		Id:           conv.ToInt32(p.ID),
		Name:         p.Name,
		Description:  p.Description,
		Archived:     p.Archived,
		Username:     p.Username,
		StartTime:    conv.ToTimestamp(p.StartTime),
		EndTime:      conv.ToTimestamp(p.EndTime),
		ProjectId:    conv.ToInt32(p.ProjectID),
		JobId:        p.JobID,
		State:        conv.ToExperimentv1State(string(p.State)),
		Progress:     parsedProgress,
		ForkedFrom:   parsedForkedFrom,
		UserId:       conv.ToInt32(p.UserID),
		ResourcePool: p.ResourcePool,
		Notes:        p.Notes,
		Labels:       p.Labels,
		SearcherType: p.SearcherType,
		TrialIds:     p.TrialIDs,
		NumTrials:    p.NumTrials,
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
	LastExperimentStartedAt *time.Time      `bun:"last_experiment_started_at"`
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
		ColumnExpr("project_metadata.id, project_metadata.name, project_metadata.description").
		ColumnExpr("username, project_metadata.immutable, workspace_id, project_metadata.notes").
		ColumnExpr("(workspaces.archived OR project_metadata.archived) AS archived").
		ColumnExpr("(SELECT COUNT(*) FROM experiments WHERE project_id = project_metadata.id) AS num_experiments").
		ColumnExpr("(SELECT COUNT(*) FROM experiments WHERE project_id = project_metadata.id AND experiments.state = 'ACTIVE') AS num_active_experiments").
		ColumnExpr("(SELECT MAX(start_time) FROM experiments WHERE project_id = project_metadata.id) AS last_experiment_started_at").
		Model(&ps).
		Join("JOIN users ON users.id = project_metadata.user_id").
		Join("JOIN workspaces ON workspaces.id = project_metadata.workspace_id"))
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
		ColumnExpr("start_time, end_time, state, archived, owner_id AS user_id").
		ColumnExpr("job_id, parent_id AS forked_from, progress, project_id").
		ColumnExpr("COALESCE(notes, 'omitted') AS notes").
		ColumnExpr("(SELECT COUNT(*) FROM trials t WHERE experiment_metadata.id = t.experiment_id) AS num_trials").
		ColumnExpr("(SELECT json_agg(id) FROM trials t WHERE experiment_metadata.id = t.experiment_id) AS trial_ids").
		ColumnExpr("config->>'name' AS name, config->>'description' AS description").
		ColumnExpr("config->'resources'->>'resource_pool' AS resource_pool").
		ColumnExpr("config->'labels' AS labels, config->'searcher'->'name' as searcher_type").
		ColumnExpr("users.username, COALESCE(users.display_name, users.username) as display_name").
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

	var s struct {
		NumActiveExperiments    int
		NumExperiments          int
		LastExperimentStartedAt *time.Time
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
		ColumnExpr("0 AS num_experiments, 0 AS num_active_experiments").
		ColumnExpr("now() AS last_experiment_started_at").
		ColumnExpr("description, workspace_id, notes, project_metadata.immutable").
		ColumnExpr("project_metadata.id, project_metadata.name, project_metadata.archived").
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
