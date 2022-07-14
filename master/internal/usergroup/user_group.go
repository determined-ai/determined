package usergroup

import (
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// Group represents a user group as it's stored in the database.
type Group struct {
	bun.BaseModel `bun:"table:groups,alias:groups"`

	ID     int          `bun:"id,pk,autoincrement" json:"id"`
	Name   string       `bun:"group_name,notnull"  json:"name"`
	UserID model.UserID `bun:"user_id,nullzero"    json:"userId,omitempty"`
}

// GroupMembership represents a user's membership to a group as it's stored in the database.
type GroupMembership struct {
	bun.BaseModel `bun:"table:user_group_membership"`

	UserID  model.UserID `bun:"user_id,notnull"`
	GroupID int          `bun:"group_id,notnull"`
}
