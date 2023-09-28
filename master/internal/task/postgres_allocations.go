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
