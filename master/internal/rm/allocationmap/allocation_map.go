package allocationmap

import (
	"sync"

	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

var (
	allocationMap      = map[model.AllocationID]*actor.Ref{}
	allocationMapMutex sync.RWMutex
)

// GetAllocation returns allocation actor by allocation id.
func GetAllocation(allocationID model.AllocationID) *actor.Ref {
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
func RegisterAllocation(allocationID model.AllocationID, ref *actor.Ref) {
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
