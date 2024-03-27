package model

import (
	"database/sql"
	"time"
)

// MaintenanceMessage represents a server status from the `maintenance_messages` table.
type MaintenanceMessage struct {
	CreatorID   int
	Message     string
	StartTime   time.Time
	EndTime     sql.NullTime
	CreatedTime sql.NullTime
}
