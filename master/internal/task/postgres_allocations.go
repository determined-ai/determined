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
	workspaceInfo := model.Workspace{}
	err := db.Bun().NewSelect().Model(&workspaceInfo).
		Join("INNER JOIN projects p on workspace.id = p.workspace_id").
		Join("INNER JOIN experiments e on p.id = e.project_id").
		Where("e.id = ?", experimentID).Scan(ctx)
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
		WorkspaceID:   workspaceInfo.ID,
		WorkspaceName: workspaceInfo.Name,
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
) error {
	workspaceInfo := model.Workspace{}
	err := db.Bun().NewSelect().Model(&workspaceInfo).
		Join("INNER JOIN command_state c ON (c.generic_command_spec->'Metadata'->>'workspace_id')::int = workspace.id").
		Join("INNER JOIN tasks t ON c.task_id = t.task_id").
		Join("INNER JOIN allocations a ON t.task_id = a.task_id").
		Where("a.allocation_id = ?", allocationID).Scan(ctx)
	if err != nil {
		return fmt.Errorf(
			"unable to retrieve workspace information for NTSC allocation (%s): %w",
			allocationID,
			err,
		)
	}

	_, err = db.Bun().NewInsert().Model(&model.AllocationWorkspaceRecord{
		AllocationID:  allocationID,
		WorkspaceID:   workspaceInfo.ID,
		WorkspaceName: workspaceInfo.Name,
	}).Exec(ctx)
	if err != nil {
		return fmt.Errorf("inserting NTSC allocation workspace record: %w", err)
	}
	return nil
}
