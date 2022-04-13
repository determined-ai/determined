package projects

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

type ProjectMetadata struct {
	bun.BaseModel `bun:"select:projects_view"`

	Id                      int             `bun:"id,nullzero"`
	Name                    string          `bun:"name"`
	Description             string          `bun:"description"`
	WorkspaceId             int             `bun:"workspace_id"`
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
		Id:                      conv.ToInt32(p.Id),
		Name:                    p.Name,
		Description:             p.Description,
		WorkspaceId:             conv.ToInt32(p.WorkspaceId),
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

func Single(ctx context.Context, opts db.SelectExtension) (*ProjectMetadata, error) {
	p := ProjectMetadata{}

	q, err := opts(db.Bun().
		NewSelect().
		Model(&p))
	if err != nil {
		return nil, fmt.Errorf("building query: %w", err)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("getting single project: %w", err)
	}
	return &p, nil
}

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

func ByID(ctx context.Context, id int32) (*ProjectMetadata, error) {
	return Single(ctx, func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
		return q.Where("id = ?", id), nil
	})
}
