package model

import (
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/proto/pkg/groupv1"
)

// Group represents a user group as it's stored in the database.
type Group struct {
	bun.BaseModel `bun:"table:groups,alias:groups"`

	ID      int    `bun:"id,pk,autoincrement" json:"id"`
	Name    string `bun:"group_name,notnull"  json:"name"`
	OwnerID UserID `bun:"user_id,nullzero"    json:"userId,omitempty"`
}

// Proto converts a group to its protobuf representation.
func (g *Group) Proto() *groupv1.Group {
	return &groupv1.Group{
		GroupId: int32(g.ID),
		Name:    g.Name,
	}
}

// Groups is a slice of Group objects—primarily useful for its methods.
type Groups []Group

// Proto converts Groups into its protobuf representation.
func (gs Groups) Proto() []*groupv1.Group {
	out := make([]*groupv1.Group, len(gs))
	for i, g := range gs {
		out[i] = g.Proto()
	}
	return out
}

// GroupMembership represents a user's membership to a group as it's stored in the database.
type GroupMembership struct {
	bun.BaseModel `bun:"table:user_group_membership"`

	UserID  UserID `bun:"user_id,notnull"`
	GroupID int    `bun:"group_id,notnull"`
}
