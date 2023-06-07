package allocationmap

import (
	"sync"

	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/model"
)

var (
	allocationMap      map[model.AllocationID]*task.Allocation
	allocationMapMutex sync.RWMutex
)

// InitAllocationMap initializes the global allocation_id -> allocation actor map.
func InitAllocationMap() {
	allocationMap = map[model.AllocationID]*task.Allocation{}
}

// GetAllocation returns allocation actor by allocation id.
func GetAllocation(allocationID model.AllocationID) *task.Allocation {
	allocationMapMutex.RLock()
	defer allocationMapMutex.RUnlock()
	return allocationMap[allocationID]
}

// GetAllAllocationIds returns all registered allocation ids.
func GetAllAllocationIds() []model.AllocationID {
	allocationMapMutex.RLock()
	defer allocationMapMutex.RUnlock()
	return maps.Keys(allocationMap)
}

// RegisterAllocation inserts the new allocation into the map.
func RegisterAllocation(allocationID model.AllocationID, a *task.Allocation) {
	allocationMapMutex.Lock()
	defer allocationMapMutex.Unlock()
	allocationMap[allocationID] = a
}

// UnregisterAllocation deletes an allocation from the map.
func UnregisterAllocation(allocationID model.AllocationID) {
	allocationMapMutex.Lock()
	defer allocationMapMutex.Unlock()
	delete(allocationMap, allocationID)
}
