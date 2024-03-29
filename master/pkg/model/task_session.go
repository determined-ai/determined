package model

import "github.com/uptrace/bun"

// AllocationSession corresponds to a row in the "allocation_sessions" DB table.
type AllocationSession struct {
	bun.BaseModel `bun:"table:allocation_sessions"`
	ID            SessionID    `db:"id" bun:"id,pk,autoincrement" json:"id"`
	AllocationID  AllocationID `db:"allocation_id" bun:"allocation_id" json:"allocation_id"`
	OwnerID       *UserID      `db:"owner_id" bun:"owner_id" json:"owner_id"`
}
