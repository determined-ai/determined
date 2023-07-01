package task

import (
	"context"
	"sync"

	"golang.org/x/exp/maps"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/allgather"
	"github.com/determined-ai/determined/master/pkg/actor"
	detLogger "github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

var syslog = logrus.WithField("component", "allocation_service")

// DefaultService is the singleton default AllocationService.
var DefaultService = NewAllocationService()

// AllocationService is used to launch, track and interact with allocations.
type AllocationService struct {
	mu          sync.RWMutex
	allocations map[model.AllocationID]*Allocation
}

// NewAllocationService creates a new AllocationService.
func NewAllocationService() *AllocationService {
	return &AllocationService{
		allocations: map[model.AllocationID]*Allocation{},
	}
}

// StartAllocation starts an allocation and returns a handle to it.
func (as *AllocationService) StartAllocation(
	logCtx detLogger.Context,
	req sproto.AllocateRequest,
	db db.DB,
	rm rm.ResourceManager,
	specifier tasks.TaskSpecifier,
	system *actor.System,
	parent *actor.Ref,
) *Allocation {
	as.mu.Lock()
	defer as.mu.Unlock()

	ref := startAllocation(logCtx, req, db, rm, specifier, system, parent)
	as.allocations[req.AllocationID] = ref

	go func() {
		_ = ref.AwaitTermination()
		if err := ref.Close(); err != nil {
			syslog.WithError(err).Error("cleaning up allocation")
		}

		as.mu.Lock()
		defer as.mu.Unlock()
		delete(as.allocations, req.AllocationID)
	}()

	return ref
}

// GetAllocation returns allocation actor by allocation id.
// TODO(!!!): IDK if this should be public.
func (as *AllocationService) GetAllocation(allocationID model.AllocationID) *Allocation {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.allocations[allocationID]
}

// GetAllAllocationIDs returns all registered allocation ids.
func (as *AllocationService) GetAllAllocationIDs() []model.AllocationID {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return maps.Keys(as.allocations)
}

// SendLog sends a container log, enriched with metadata from the allocation.
func (as *AllocationService) SendLog(
	ctx context.Context,
	id model.AllocationID,
	log *sproto.ContainerLog,
) {
	ref := as.GetAllocation(id)
	if ref == nil {
		syslog.Warnf("dropped log for unknown allocation: %s", id)
		return
	}
	ref.SendLog(log)
}

// SetReady sets the ready bit and moves the allocation to the running state if it has not
// progressed past it already.
func (as *AllocationService) SetReady(ctx context.Context, id model.AllocationID) error {
	ref := as.GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.SetReady(ctx)
}

// SetWaiting moves the allocation to the waiting state if it has not progressed past it yet.
func (as *AllocationService) SetWaiting(ctx context.Context, id model.AllocationID) error {
	ref := as.GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.SetWaiting(ctx)
}

// SetProxyAddress sets the proxy address of the allocation and sets up proxies for any services
// it provides.
func (as *AllocationService) SetProxyAddress(
	ctx context.Context,
	id model.AllocationID,
	addr string,
) error {
	ref := as.GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.SetProxyAddress(ctx, addr)
}

// WatchRendezvous returns a watcher for the caller to wait for rendezvous to complete. When a
// process from each resource in the allocation connects and the resource manager sends each
// resource's state, each watcher will receive a copy of the rendezvous info for communicating
// with its peers.
func (as *AllocationService) WatchRendezvous(
	ctx context.Context,
	id model.AllocationID,
	rID sproto.ResourcesID,
) (RendezvousWatcher, error) {
	ref := as.GetAllocation(id)
	if ref == nil {
		return RendezvousWatcher{}, api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return RendezvousWatcher{}, err
	}

	return ref.WatchRendezvous(rID)
}

// UnwatchRendezvous removes a rendezvous watcher.
func (as *AllocationService) UnwatchRendezvous(
	ctx context.Context,
	id model.AllocationID,
	rID sproto.ResourcesID,
) error {
	ref := as.GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.UnwatchRendezvous(rID)
}

// SetResourcesAsDaemon marks the resources as daemons. If all non-daemon resources exit, the
// allocation will kill the remaining daemon resources.
func (as *AllocationService) SetResourcesAsDaemon(
	ctx context.Context,
	id model.AllocationID,
	rID sproto.ResourcesID,
) error {
	ref := as.GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.SetResourcesAsDaemon(ctx, rID)
}

// State returns a copy of the current state of the allocation.
func (as *AllocationService) State(id model.AllocationID) (AllocationState, error) {
	ref := as.GetAllocation(id)
	if ref == nil {
		return AllocationState{}, api.NotFoundErrs("allocation", id.String(), true)
	}
	return ref.State(), nil
}

// AllGather blocks until `numPeers` with the same `allocationID` are waiting and then returns the
// data from all those peers. It returns an error if the call returns early without data for any
// reason. Only one call may connect per `id`.
func (as *AllocationService) AllGather(
	ctx context.Context,
	allocationID model.AllocationID,
	id uuid.UUID,
	numPeers int,
	data any,
) ([]any, error) {
	err := as.WaitForRestore(ctx, allocationID)
	if err != nil {
		return nil, err
	}

	readyFn := func() {
		err := as.SetReady(ctx, allocationID)
		if err != nil {
			syslog.WithError(err).Errorf("failed to set ready for %s", allocationID)
		}
	}

	timeoutFn := func(err error) {
		msg := err.Error()
		as.SendLog(ctx, allocationID, &sproto.ContainerLog{AuxMessage: &msg})
	}

	w := allgather.Join(allocationID.String(), id, numPeers, data, readyFn, timeoutFn)
	defer allgather.Leave(allocationID.String(), id)

	select {
	case res := <-w.C:
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// WaitForRestore waits until the allocation has been restored by the resource manager. The
// allocation must exist otherwise this will return a not found error.
func (as *AllocationService) WaitForRestore(ctx context.Context, id model.AllocationID) error {
	ref := as.GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}
	return ref.WaitForRestore(ctx)
}
