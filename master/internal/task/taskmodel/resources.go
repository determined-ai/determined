package taskmodel

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// ResourcesWithState is an sproto.Resources, along with its state that is tracked by the
// allocation. The state is primarily just the updates that come in about the resources.
type ResourcesWithState struct {
	bun.BaseModel `bun:"table:allocation_resources,alias:al_res"`

	sproto.Resources `bun:"-"`
	Rank             int                      `bun:"rank"`
	Started          *sproto.ResourcesStarted `bun:"started"`
	Exited           *sproto.ResourcesStopped `bun:"exited"`
	Daemon           bool                     `bun:"daemon"`
	ResourceID       sproto.ResourcesID       `bun:"resource_id,pk,type:text"` // db only
	AllocationID     model.AllocationID       `bun:"allocation_id,type:text"`  // db only

	// The Container state, if we're using a RM that uses containers and it was given to us.
	// This is a rip in the abstraction, remove eventually. Do not add usages.
	Container *cproto.Container `bun:"-"`
}

// NewResourcesState creates an instance from `sproto.Resources`.
func NewResourcesState(r sproto.Resources, rank int) ResourcesWithState {
	summary := r.Summary()
	return ResourcesWithState{
		Resources:    r,
		Rank:         rank,
		ResourceID:   summary.ResourcesID,
		AllocationID: summary.AllocationID,
		Started:      summary.Started,
		Exited:       summary.Exited,
	}
}

// WipeResourcesState deletes all database contents.
func WipeResourcesState() error {
	// Bun requires at least one WHERE for updates and deletes.
	_, err := db.Bun().NewDelete().Model((*ResourcesWithState)(nil)).Where("1=1").Exec(context.TODO())
	return err
}

// CleanupResourcesState deletes resources for all closed allocations.
func CleanupResourcesState() error {
	// Potentially this may become expensive, however this is a subquery,
	// not materialized in master, and this function only runs once on master startup.
	closedAllocations := db.Bun().NewSelect().Model((*model.Allocation)(nil)).
		Where("end_time IS NOT NULL").
		Column("allocation_id")

	_, err := db.Bun().NewDelete().Model((*ResourcesWithState)(nil)).
		Where("allocation_id IN (?)", closedAllocations).Exec(context.TODO())
	return err
}

// Persist saves the data to the database.
func (r *ResourcesWithState) Persist() error {
	_, err := db.Bun().NewInsert().Model(r).
		On("CONFLICT (resource_id) DO UPDATE").
		Exec(context.TODO())
	return err
}
