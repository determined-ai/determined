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
