package model

import (
	"github.com/uptrace/bun"
)

type Group struct {
	bun.BaseModel `bun:"table:groups"`

	ID     int    `bun:"id,pk,autoincrement" json:"id"`
	Name   string `bun:"group_name,notnull"  json:"name"`
	UserID UserID `bun:"user_id,nullzero"    json:"userId,omitempty"`
}

type GroupMembership struct {
	bun.BaseModel `bun:"table:user_group_membership"`

	UserID  UserID `bun:"user_id,notnull"`
	GroupID int    `bun:"group_id,notnull"`
}
