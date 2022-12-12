package model

import (
	"time"

	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Project is the bun model of a project.
type Project struct {
	bun.BaseModel           `bun:"table:projects"`
	ID                      int               `bun:"id,pk,autoincrement"`
	Name                    string            `bun:"name"`
	CreatedAt               time.Time         `bun:"created_at,scanonly"`
	Archived                bool              `bun:"archived"`
	WorkspaceID             int               `bun:"workspace_id"`
	WorkspaceName           string            `bun:"workspace_name"`
	UserID                  int               `bun:"user_id"`
	Username                string            `bun:"username"`
	Immutable               bool              `bun:"immutable"`
	Description             string            `bun:"description"`
	Notes                   []*projectv1.Note `bun: "notes,type:jsonb"`
	NumActiveExperiments    int32             `bun:"num_active_experiments"`
	NumExperiments          int32             `bun:"num_experiments"`
	State                   WorkspaceState    `bun:"state"`
	ErrorMessage            string            `bun:"error_message"`
	LastExperimentStartedAt time.Time         `bun:"last_experiment_started_at"`
}

// Projects is an array of project instances
type Projects []*Project

// Proto converts a bun model of a project to a proto object.
func (p Project) Proto() *projectv1.Project {

	return &projectv1.Project{
		Id:                      int32(p.ID),
		Name:                    p.Name,
		Archived:                p.Archived,
		UserId:                  int32(p.UserID),
		Username:                p.Username,
		Immutable:               p.Immutable,
		WorkspaceId:             int32(p.WorkspaceID),
		WorkspaceName:           p.WorkspaceName,
		State:                   p.State.ToProto(),
		Description:             p.Description,
		ErrorMessage:            p.ErrorMessage,
		NumExperiments:          p.NumExperiments,
		NumActiveExperiments:    p.NumActiveExperiments,
		Notes:                   p.Notes,
		LastExperimentStartedAt: timestamppb.New(p.LastExperimentStartedAt),
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
