package task

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// AddAllocationAcceleratorData stores acceleration data for an allocation.
func AddAllocationAcceleratorData(ctx context.Context, accData model.AcceleratorData,
) error {
	_, err := db.Bun().NewInsert().Model(&accData).Exec(ctx)
	if err != nil {
		return fmt.Errorf("adding allocation acceleration data: %w", err)
	}
	return nil
}

// GetAllocation stores acceleration data for an allocation.
func getAllocation(ctx context.Context, allocationID string,
) (*model.Allocation, error) {
	var allocation model.Allocation
	err := db.Bun().NewRaw(`
SELECT allocation_id, task_id, state, slots, is_ready, start_time, 
end_time, exit_reason, exit_error, status_code
FROM allocations
WHERE allocation_id = ?
	`, allocationID).Scan(ctx, &allocation)
	if err != nil {
		return nil, fmt.Errorf("querying allocation %s: %w", allocationID, err)
	}

	return &allocation, nil
}

// InsertTrialAllocationWorkspaceRecord inserts a record linking an trial's allocation
// to a trial to it's respective workspace & experiment.
func InsertTrialAllocationWorkspaceRecord(
	ctx context.Context,
	experimentID int,
	allocationID model.AllocationID,
) error {
	var workspaceInfo struct {
		WorkspaceID   int    `bun:"id"`
		WorkspaceName string `bun:"name"`
	}
	err := db.Bun().NewSelect().
		Table("workspaces").
		Join("INNER JOIN projects ON workspaces.id = projects.workspace_id").
		Join("INNER JOIN experiments ON projects.id = experiments.project_id").
		Column("workspaces.id", "workspaces.name").
		Where("experiments.id = ?", experimentID).
		Scan(ctx, &workspaceInfo)
	if err != nil {
		return fmt.Errorf(
			"unable to retrieve workspace information for allocation (%s) associated with experiment %d: %w",
			allocationID,
			experimentID,
			err,
		)
	}

	_, err = db.Bun().NewInsert().Model(&model.AllocationWorkspaceRecord{
		AllocationID:  allocationID,
		ExperimentID:  experimentID,
		WorkspaceID:   workspaceInfo.WorkspaceID,
		WorkspaceName: workspaceInfo.WorkspaceName,
	}).Exec(ctx)
	if err != nil {
		return fmt.Errorf("inserting allocation workspace record: %w", err)
	}
	return nil
}

// InsertNTSCAllocationWorkspaceRecord inserts a record linking
// an NTSC tasks' allocation to it's respective workspace.
func InsertNTSCAllocationWorkspaceRecord(
	ctx context.Context,
	allocationID model.AllocationID,
	workspaceID int,
	workspaceName string,
) error {
	_, err := db.Bun().NewInsert().Model(&model.AllocationWorkspaceRecord{
		AllocationID:  allocationID,
		WorkspaceID:   workspaceID,
		WorkspaceName: workspaceName,
	}).Exec(ctx)
	if err != nil {
		return fmt.Errorf("inserting NTSC allocation workspace record: %w", err)
	}
	return nil
}
