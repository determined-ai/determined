package model

import (
	"database/sql"

	"github.com/uptrace/bun"
)

// RoleAssignmentScope represents a RoleAssignmentScope as it's stored in the database.
type RoleAssignmentScope struct {
	bun.BaseModel `bun:"table:role_assignment_scopes"`

	ID          int           `bun:"id,pk,autoincrement" json:"id"`
	WorkspaceID sql.NullInt32 `bun:"scope_workspace_id"  json:"workspace_id"`
}
