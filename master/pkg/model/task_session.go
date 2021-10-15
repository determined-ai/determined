package model

// AllocationSession corresponds to a row in the "allocation_sessions" DB table.
type AllocationSession struct {
	ID           SessionID    `db:"id" json:"id"`
	AllocationID AllocationID `db:"allocation_id" json:"allocation_id"`
}
