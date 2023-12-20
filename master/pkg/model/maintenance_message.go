package model

import (
	"time"
)

// MaintenanceMessage represents a server status from the `maintenance_messages` table.
type MaintenanceMessage struct {
	CreatorID int        `db:"creator_id" json:"creator_id"`
	Message   string     `db:"message" json:"message"`
	StartTime *time.Time `db:"start_time" json:"start_time"`
	EndTime   *time.Time `db:"end_time" json:"end_time"`
}
