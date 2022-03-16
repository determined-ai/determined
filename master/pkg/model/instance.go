package model

import (
	"time"
)

// InstanceStats stores the start/end status of instance.
type InstanceStats struct {
	ResourcePool string     `db:"resource_pool"`
	InstanceID   *string    `db:"instance_id"`
	Slots        int        `db:"slots"`
	StartTime    time.Time  `db:"start_time"`
	EndTime      *time.Time `db:"end_time"`
}
