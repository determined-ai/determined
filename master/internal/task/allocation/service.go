package allocation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/pkg/model"
)

var (
	allocationMap      = map[model.AllocationID]*AllocationHandle{}
	allocationMapMutex sync.RWMutex
)

// GetAllocation returns allocation actor by allocation id.
func GetAllocation(allocationID model.AllocationID) (*AllocationHandle, error) {
	allocationMapMutex.RLock()
	defer allocationMapMutex.RUnlock()
	a, ok := allocationMap[allocationID]
	if !ok {
		return nil, ErrAllocationNotFound
	}
	return a, nil
}

func GetRestoredAllocation(ctx context.Context, id model.AllocationID) (*AllocationHandle, error) {
	h, err := GetAllocation(id)
	if err != nil {
		return nil, err
	}

	err = waitForAllocationToBeRestored(ctx, h)
	if err != nil {
		return nil, err
	}
	return h, nil
}

// GetAllAllocationIds returns all registered allocation ids.
func GetAllAllocationIds() []model.AllocationID {
	allocationMapMutex.RLock()
	defer allocationMapMutex.RUnlock()
	return maps.Keys(allocationMap)
}

// registerAllocation inserts the new allocation into the map.
func registerAllocation(allocationID model.AllocationID, ref *AllocationHandle) {
	allocationMapMutex.Lock()
	defer allocationMapMutex.Unlock()
	allocationMap[allocationID] = ref
}

// UnregisterAllocation deletes an allocation from the map.
func UnregisterAllocation(allocationID model.AllocationID) {
	allocationMapMutex.Lock()
	defer allocationMapMutex.Unlock()
	delete(allocationMap, allocationID)
}

func waitForAllocationToBeRestored(ctx context.Context, alloc *AllocationHandle) error {
	for i := 0; i < 60; i++ {
		if !alloc.Restoring() {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("allocation stuck restoring after one minute of retrying")
}
