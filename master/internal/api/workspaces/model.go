package workspaces

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// WorkspaceMetadata matches rows returned from workspace queries.
type WorkspaceMetadata struct {
	bun.BaseModel `bun:"select:workspaces"`

	ID          int    `bun:"id"`
	Name        string `bun:"name"`
	Archived    bool   `bun:"archived"`
	Username    string `bun:"username"`
	DisplayName string `bun:"display_name"`
	UserID      int    `bun:"user_id"`
	NumProjects int32  `bun:"num_projects"`
	Immutable   bool   `bun:"immutable"`
}

// ToProto converts each WorkspaceMetadata model into API-accessible format.
func (p *WorkspaceMetadata) ToProto() (*workspacev1.Workspace, error) {
	conv := protoconverter.ProtoConverter{}
	out := &workspacev1.Workspace{
		Id:          conv.ToInt32(p.ID),
		Name:        p.Name,
		Archived:    p.Archived,
		Username:    p.Username,
		NumProjects: p.NumProjects,
		Immutable:   p.Immutable,
	}
	if err := conv.Error(); err != nil {
		return nil, fmt.Errorf("converting workspace to proto: %w", err)
	}
	return out, nil
}

// WorkspaceList fetches multiple Workspaces.
func WorkspaceList(ctx context.Context, opts db.SelectExtension) ([]*WorkspaceMetadata, error) {
	ps := []*WorkspaceMetadata{}
	q, err := opts(db.Bun().
		NewSelect().
		ColumnExpr("workspace_metadata.id, name, archived, immutable").
		ColumnExpr("(SELECT COUNT(*) FROM projects WHERE" +
			" workspace_id = workspace_metadata.id) AS num_projects").
		ColumnExpr("users.username AS username").
		Model(&ps).
		Join("JOIN users ON users.id = workspace_metadata.user_id"))
	if err != nil {
		return nil, fmt.Errorf("building query: %w", err)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}
	return ps, nil
}

// ByID returns a single Workspace by its ID.
func ByID(ctx context.Context, id int32) (*WorkspaceMetadata, error) {
	p := WorkspaceMetadata{}

	q := db.Bun().
		NewSelect().
		ColumnExpr("workspace_metadata.id, name, archived, immutable").
		ColumnExpr("(SELECT COUNT(*) FROM projects WHERE workspace_id = ?) AS num_projects", id).
		ColumnExpr("users.username AS username").
		Join("JOIN users ON users.id = workspace_metadata.user_id").
		Model(&p).
		Where("workspace_metadata.id = ?", id)

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("getting single workspace: %w", err)
	}
	return &p, nil
}
