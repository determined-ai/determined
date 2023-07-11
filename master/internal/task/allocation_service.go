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

// DefaultService is the singleton default allocationService.
var DefaultService AllocationService = newAllocationService()

// allocationService is used to launch, track and interact with allocations.
type allocationService struct {
	mu          sync.RWMutex
	allocations map[model.AllocationID]*allocation
}

// newAllocationService creates a new allocationService.
func newAllocationService() *allocationService {
	return &allocationService{
		allocations: map[model.AllocationID]*allocation{},
	}
}

// StartAllocation starts an allocation and returns a handle to it.
func (as *allocationService) StartAllocation(
	logCtx detLogger.Context,
	req sproto.AllocateRequest,
	db db.DB,
	rm rm.ResourceManager,
	specifier tasks.TaskSpecifier,
	system *actor.System,
	parent *actor.Ref,
) {
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
}

// GetAllAllocationIDs returns all registered allocation ids.
func (as *allocationService) GetAllAllocationIDs() []model.AllocationID {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return maps.Keys(as.allocations)
}

// SendLog sends a container log, enriched with metadata from the allocation.
func (as *allocationService) SendLog(
	ctx context.Context,
	id model.AllocationID,
	log *sproto.ContainerLog,
) {
	ref := as.getAllocation(id)
	if ref == nil {
		syslog.Warnf("dropped log for unknown allocation: %s", id)
		return
	}
	ref.SendLog(log)
}

// SetReady sets the ready bit and moves the allocation to the running state if it has not
// progressed past it already.
func (as *allocationService) SetReady(ctx context.Context, id model.AllocationID) error {
	ref := as.getAllocation(id)
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
func (as *allocationService) SetWaiting(ctx context.Context, id model.AllocationID) error {
	ref := as.getAllocation(id)
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
func (as *allocationService) SetProxyAddress(
	ctx context.Context,
	id model.AllocationID,
	addr string,
) error {
	ref := as.getAllocation(id)
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
func (as *allocationService) WatchRendezvous(
	ctx context.Context,
	id model.AllocationID,
	rID sproto.ResourcesID,
) (RendezvousWatcher, error) {
	ref := as.getAllocation(id)
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
func (as *allocationService) UnwatchRendezvous(
	ctx context.Context,
	id model.AllocationID,
	rID sproto.ResourcesID,
) error {
	ref := as.getAllocation(id)
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
func (as *allocationService) SetResourcesAsDaemon(
	ctx context.Context,
	id model.AllocationID,
	rID sproto.ResourcesID,
) error {
	ref := as.getAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}

	err := ref.WaitForRestore(ctx)
	if err != nil {
		return err
	}

	return ref.SetResourcesAsDaemon(ctx, rID)
}

// Signal the allocation with the given signal.
func (as *allocationService) Signal(
	id model.AllocationID,
	sig AllocationSignal,
	reason string,
) error {
	ref := as.getAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}
	ref.HandleSignal(sig, reason) // TODO: public/private methods on allocation itself.
	return nil
}

// State returns a copy of the current state of the allocation.
func (as *allocationService) State(id model.AllocationID) (AllocationState, error) {
	ref := as.getAllocation(id)
	if ref == nil {
		return AllocationState{}, api.NotFoundErrs("allocation", id.String(), true)
	}
	return ref.State(), nil
}

// AllGather blocks until `numPeers` with the same `allocationID` are waiting and then returns the
// data from all those peers. It returns an error if the call returns early without data for any
// reason. Only one call may connect per `id`.
func (as *allocationService) AllGather(
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
func (as *allocationService) WaitForRestore(ctx context.Context, id model.AllocationID) error {
	ref := as.getAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}
	return ref.WaitForRestore(ctx)
}

// getAllocation returns allocation actor by allocation id.
// TODO(!!!): IDK if this should be public.
func (as *allocationService) getAllocation(allocationID model.AllocationID) *allocation {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.allocations[allocationID]
}
