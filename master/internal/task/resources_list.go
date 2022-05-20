package task

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

type (
	// ResourcesWithState is an sproto.Resources, along with its state that is tracked by the
	// allocation. The state is primarily just the updates that come in about the resources.
	ResourcesWithState struct {
		bun.BaseModel `bun:"table:allocation_resources,alias:al_res"`

		sproto.Resources `bun:"-"`
		Rank             int                      `bun:"rank"`
		Started          *sproto.ResourcesStarted `bun:"started"`
		Exited           *sproto.ResourcesStopped `bun:"exited"`
		Daemon           bool                     `bun:"daemon"`
		ResourceID       sproto.ResourcesID       `bun:"resource_id,type:text"`   // db only
		AllocationID     model.AllocationID       `bun:"allocation_id,type:text"` // db only

		// The container state, if we're using a RM that uses containers and it was given to us.
		// This is a rip in the abstraction, remove eventually. Do not add usages.
		container *cproto.Container
	}

	// resourcesList tracks resourcesList with their state.
	resourcesList map[sproto.ResourcesID]*ResourcesWithState
)

// NewResourcesState creates an instance from `sproto.Resources`.
func NewResourcesState(r sproto.Resources, rank int) ResourcesWithState {
	summary := r.Summary()
	return ResourcesWithState{
		Resources:    r,
		Rank:         rank,
		ResourceID:   summary.ResourcesID,
		AllocationID: summary.AllocationID,
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

func (rs resourcesList) append(ars []sproto.Resources) error {
	start := len(rs)
	for rank, r := range ars {
		summary := r.Summary()
		state := NewResourcesState(r, start+rank)
		if err := state.Persist(); err != nil {
			return err
		}
		rs[summary.ResourcesID] = &state
	}

	return nil
}

func (rs resourcesList) first() *ResourcesWithState {
	for _, r := range rs {
		return r
	}
	return nil
}

func (rs resourcesList) firstDevice() *device.Device {
	for _, r := range rs {
		if r.container != nil && len(r.container.Devices) > 0 {
			return &r.container.Devices[0]
		}
	}
	return nil
}

func (rs resourcesList) daemons() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.Daemon {
			nrs[id] = r
		}
	}
	return nrs
}

func (rs resourcesList) started() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.Started != nil {
			nrs[id] = r
		}
	}
	return nrs
}

func (rs resourcesList) exited() resourcesList {
	nrs := resourcesList{}
	for id, r := range rs {
		if r.Exited != nil {
			nrs[id] = r
		}
	}
	return nrs
}
