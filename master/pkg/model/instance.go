package model

// InstanceStats stores the start/end status of instance.
type InstanceStats struct {
	ResourcePool string `db:"resource_pool"`
	InstanceID   string `db:"instance_id"`
	Slots        int    `db:"slots"`
}
