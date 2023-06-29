package task

import (
	"context"
	"sync"

	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TODO(!!!): TBH, a service struct is probably better long term.
var (
	allocationMap      = map[model.AllocationID]*Allocation{}
	allocationMapMutex sync.RWMutex
)

// GetAllocation returns allocation actor by allocation id.
// TODO(!!!): IDK if this should be public.
func GetAllocation(allocationID model.AllocationID) *Allocation {
	allocationMapMutex.RLock()
	defer allocationMapMutex.RUnlock()
	return allocationMap[allocationID]
}

// GetAllAllocationIDs returns all registered allocation ids.
func GetAllAllocationIDs() []model.AllocationID {
	allocationMapMutex.RLock()
	defer allocationMapMutex.RUnlock()
	return maps.Keys(allocationMap)
}

func SendLog(ctx context.Context, id model.AllocationID, log *sproto.ContainerLog) {
	ref := GetAllocation(id)
	if ref == nil {
		// TODO(!!!): Something, something errors.
		return
	}
	ref.SendLog(log)
}

func SetReady(ctx context.Context, id model.AllocationID) error {
	ref := GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.SetReady(ctx)
}

func SetWaiting(ctx context.Context, id model.AllocationID) error {
	ref := GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.SetWaiting(ctx)
}

func SetProxyAddress(ctx context.Context, id model.AllocationID, addr string) error {
	ref := GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.SetProxyAddress(ctx, addr)
}

func WatchRendezvous(
	ctx context.Context,
	id model.AllocationID,
	rID sproto.ResourcesID,
) (RendezvousWatcher, error) {
	ref := GetAllocation(id)
	if ref == nil {
		return RendezvousWatcher{}, api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return RendezvousWatcher{}, err
	}

	return ref.WatchRendezvous(rID)
}

func UnwatchRendezvous(
	ctx context.Context,
	id model.AllocationID,
	rID sproto.ResourcesID,
) error {
	ref := GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.UnwatchRendezvous(rID)
}

func MarkResourcesDaemon(ctx context.Context, id model.AllocationID, rID sproto.ResourcesID) error {
	ref := GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.SetResourcesAsDaemon(ctx, rID)
}

func WaitForRestore(ctx context.Context, id model.AllocationID) error {
	ref := GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}
	return ref.WaitForRestore(ctx)
}

// RegisterAllocation inserts the new allocation into the map.
func registerAllocation(allocationID model.AllocationID, ref *Allocation) {
	allocationMapMutex.Lock()
	defer allocationMapMutex.Unlock()
	allocationMap[allocationID] = ref
}

// UnregisterAllocation deletes an allocation from the map.
func unregisterAllocation(allocationID model.AllocationID) {
	allocationMapMutex.Lock()
	defer allocationMapMutex.Unlock()
	delete(allocationMap, allocationID)
}
