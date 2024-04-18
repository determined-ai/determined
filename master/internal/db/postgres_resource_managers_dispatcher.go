package db

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// Dispatch is the Determined-persisted representation for dispatch existence.
type Dispatch struct {
	bun.BaseModel `bun:"table:resourcemanagers_dispatcher_dispatches"`

	DispatchID       string             `bun:"dispatch_id"`
	ResourceID       sproto.ResourcesID `bun:"resource_id"`
	AllocationID     model.AllocationID `bun:"allocation_id"`
	ImpersonatedUser string             `bun:"impersonated_user"`
}

// InsertDispatch persists the existence for a dispatch.
func InsertDispatch(ctx context.Context, r *Dispatch) error {
	_, err := Bun().NewInsert().Model(r).Exec(ctx)
	if err != nil {
		return fmt.Errorf("inserting dispatch: %w", err)
	}
	return nil
}

// DispatchByID retrieves a dispatch by its ID.
func DispatchByID(
	ctx context.Context,
	id string,
) (*Dispatch, error) {
	d := Dispatch{}
	err := Bun().NewSelect().Model(&d).Where("dispatch_id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("scanning dispatch by ID (%s): %w", id, err)
	}
	return &d, nil
}

// ListDispatchesByJobID returns a list of dispatches associated with the specified job.
func ListDispatchesByJobID(
	ctx context.Context,
	jobID string,
) ([]*Dispatch, error) {
	ds := []*Dispatch{}
	err := Bun().NewSelect().Model(&ds).Join(
		"join allocations on allocations.allocation_id = dispatch.allocation_id").Join(
		"join tasks on tasks.task_id = allocations.task_id").Where("job_id = ?", jobID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("scanning dispatch by job ID (%s): %w", jobID, err)
	}
	return ds, nil
}

// ListAllDispatches lists all dispatches in the DB.
func ListAllDispatches(ctx context.Context) ([]*Dispatch, error) {
	return ListDispatches(ctx, func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
		return q, nil
	})
}

// ListDispatchesByAllocationID lists all dispatches for an allocation ID.
func ListDispatchesByAllocationID(
	ctx context.Context,
	id model.AllocationID,
) ([]*Dispatch, error) {
	return ListDispatches(ctx, func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
		return q.Where("allocation_id = ?", id), nil
	})
}

// ListDispatches lists all dispatches according to the options provided.
func ListDispatches(
	ctx context.Context,
	opts func(*bun.SelectQuery) (*bun.SelectQuery, error),
) ([]*Dispatch, error) {
	var ds []*Dispatch

	q, err := opts(Bun().NewSelect().Model(&ds))
	if err != nil {
		return nil, fmt.Errorf("building dispatch model query: %w", err)
	}

	if err = q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("scanning dispatch models: %w", err)
	}

	return ds, nil
}

// DeleteDispatch deletes the specified dispatch and returns the number deleted.
func DeleteDispatch(
	ctx context.Context,
	id string,
) (int64, error) {
	return DeleteDispatches(ctx, func(q *bun.DeleteQuery) *bun.DeleteQuery {
		return q.Where("dispatch_id = ?", id)
	})
}

// DeleteDispatches deletes all dispatches for the specified query
// and returns the number deleted.
func DeleteDispatches(
	ctx context.Context,
	opts func(*bun.DeleteQuery) *bun.DeleteQuery,
) (int64, error) {
	var ds []*Dispatch
	res, err := opts(Bun().NewDelete().Model(&ds)).Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete dispatch exec: %w", err)
	}
	count, _ := res.RowsAffected()
	return count, err
}
