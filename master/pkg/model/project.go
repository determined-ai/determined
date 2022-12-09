package model

import (
	"time"

	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/uptrace/bun"
)

// Project is the bun model of a project.
type Project struct {
	bun.BaseModel `bun:"table:projects"`
	ID            int                    `bun:"id,pk,autoincrement"`
	Name          string                 `bun:"name"`
	CreatedAt     time.Time              `bun:"created_at,scanonly"`
	Archived      bool                   `bun:"archived"`
	WorkspaceID   int                    `bun:"workspace_id"`
	UserID        int                    `bun:"user_id"`
	Immutable     bool                   `bun:"immutable"`
	Username      string                 `bun:"state"`
	Description   string                 `bun:"description"`
	Notes         map[string]interface{} `bun: "notes,type:jsonb"`
	// NumActiveExperiments int32          `bun:"num_active_experiments"`
	// NumExperiments       int32          `bun:"num_experiments"`
	State        WorkspaceState `bun:"state"`
	ErrorMessage string         `bun:"error_message"`
}

// Projects is an array of project instances
type Projects []*Project

// Proto converts a bun model of a project to a proto object.
func (p Project) Proto() *projectv1.Project {

	return &projectv1.Project{
		Id:           int32(p.ID),
		Name:         p.Name,
		Archived:     p.Archived,
		UserId:       int32(p.UserID),
		Immutable:    p.Immutable,
		WorkspaceId:  int32(p.WorkspaceID),
		State:        p.State.ToProto(),
		Description:  p.Description,
		ErrorMessage: p.ErrorMessage,
		// NumExperiments:       p.NumExperiments,
		// NumActiveExperiments: p.NumActiveExperiments,
	}
}

// Proto converts a slice of projects to its protobuf representation.
func ProjectsToProto(ps []*Project) []*projectv1.Project {
	out := make([]*projectv1.Project, len(ps))
	for i, ps := range ps {
		out[i] = ps.Proto()
	}
	return out
}
