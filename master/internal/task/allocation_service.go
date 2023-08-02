package task

import (
	"context"
	"sync"

	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"golang.org/x/exp/maps"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/allgather"
	"github.com/determined-ai/determined/master/internal/task/preemptible"
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
	syslog      *logrus.Entry
	mu          sync.RWMutex
	allocations map[model.AllocationID]*allocation
}

// newAllocationService creates a new allocationService.
func newAllocationService() *allocationService {
	return &allocationService{
		syslog:      logrus.WithField("component", "allocation-service"),
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
	onExit func(*AllocationExited),
) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	ref, err := newAllocation(logCtx, req, db, rm, specifier, system)
	if err != nil {
		return err
	}
	as.allocations[req.AllocationID] = ref

	go func() {
		_ = ref.awaitTermination()
		if err := ref.Cleanup(); err != nil {
			syslog.WithError(err).Error("cleaning up allocation")
		}

		as.mu.Lock()
		delete(as.allocations, req.AllocationID)
		as.mu.Unlock() // don't defer in case onExit calls back into the service

		onExit(ref.exited)
	}()
	return nil
}

// AwaitTermination waits unilt the given allocation has stopped.
func (as *allocationService) AwaitTermination(id model.AllocationID) {
	ref, err := as.getAllocation(id)
	if err != nil {
		return
	}
	ref.awaitTermination()
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
	ref, err := as.getAllocation(id)
	if err != nil {
		syslog.Warnf("dropped log for unknown allocation: %s", err)
		return
	}
	ref.SendContainerLog(log)
}

// SetReady sets the ready bit and moves the allocation to the running state if it has not
// progressed past it already.
func (as *allocationService) SetReady(ctx context.Context, id model.AllocationID) error {
	ref, err := as.waitForRestore(ctx, id)
	if err != nil {
		return err
	}
	return ref.SetReady(ctx)
}

// SetWaiting moves the allocation to the waiting state if it has not progressed past it yet.
func (as *allocationService) SetWaiting(ctx context.Context, id model.AllocationID) error {
	ref, err := as.waitForRestore(ctx, id)
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
	ref, err := as.waitForRestore(ctx, id)
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
) (*trialv1.RendezvousInfo, error) {
	ref, err := as.waitForRestore(ctx, id)
	if err != nil {
		return nil, err
	}

	w, err := ref.WatchRendezvous(rID)
	if err != nil {
		return nil, err
	}
	defer ref.UnwatchRendezvous(rID)

	select {
	case rsp := <-w.C:
		if rsp.Err != nil {
			return nil, rsp.Err
		}
		return rsp.Info, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SetResourcesAsDaemon marks the resources as daemons. If all non-daemon resources exit, the
// allocation will kill the remaining daemon resources.
func (as *allocationService) SetResourcesAsDaemon(
	ctx context.Context,
	id model.AllocationID,
	rID sproto.ResourcesID,
) error {
	ref, err := as.waitForRestore(ctx, id)
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
	ref, err := as.getAllocation(id)
	if err != nil {
		return err
	}
	ref.Signal(sig, reason)
	return nil
}

// State returns a copy of the current state of the allocation.
// TODO(DET-9698): Just replace this with DB access, easy to do.
func (as *allocationService) State(id model.AllocationID) (AllocationState, error) {
	ref, err := as.getAllocation(id)
	if err != nil {
		return AllocationState{}, err
	}
	return ref.State(), nil
}

// AllGather blocks until `numPeers` with the same `allocationID` are waiting and then returns the
// data from all those peers. It returns an error if the call returns early without data for any
// reason. Only one call may connect per `id`.
func (as *allocationService) AllGather(
	ctx context.Context,
	id model.AllocationID,
	wID uuid.UUID,
	numPeers int,
	data any,
) ([]any, error) {
	_, err := as.waitForRestore(ctx, id)
	if err != nil {
		return nil, err
	}

	readyFn := func() {
		err := as.SetReady(ctx, id)
		if err != nil {
			syslog.WithError(err).Errorf("failed to set ready for %s", id)
		}
	}

	timeoutFn := func(err error) {
		msg := err.Error()
		as.SendLog(ctx, id, &sproto.ContainerLog{AuxMessage: &msg})
	}

	w := allgather.Join(id.String(), wID, numPeers, data, readyFn, timeoutFn)
	defer allgather.Leave(id.String(), wID)
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

// WatchPreemption blocks as long as the context allows to watch for a preemption signal.
func (as *allocationService) WatchPreemption(
	ctx context.Context,
	id model.AllocationID,
) (bool, error) {
	_, err := as.waitForRestore(context.TODO(), id)
	if err != nil {
		// HACK: Swallow the error since contexts with an instant timeout still expect a status.
		return false, nil
	}

	wID := uuid.New()
	w, err := preemptible.Watch(id.String(), wID)
	if err != nil {
		return false, err
	}
	defer preemptible.Unwatch(id.String(), wID)

	select {
	case <-w.C:
		return true, nil
	case <-ctx.Done():
		return false, nil
	}
}

// AckPreemption acknowledges the receipt of a preemption signal. This is used to differentiate
// HPO/user-related early stops with a zero exit code from preemption-related early stopping.
func (as *allocationService) AckPreemption(ctx context.Context, id model.AllocationID) error {
	_, err := as.waitForRestore(ctx, id)
	if err != nil {
		return err
	}
	preemptible.Acknowledge(id.String())
	return nil
}

// waitForRestore waits until the allocation has been restored by the resource manager. The
// allocation must exist otherwise this will return a not found error.
func (as *allocationService) waitForRestore(
	ctx context.Context,
	id model.AllocationID,
) (*allocation, error) {
	ref, err := as.getAllocation(id)
	if err != nil {
		return nil, err
	}
	err = ref.waitForRestore(ctx)
	if err != nil {
		return nil, err
	}
	return ref, nil
}

// getAllocation returns allocation actor by allocation id.
func (as *allocationService) getAllocation(id model.AllocationID) (*allocation, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	ref := as.allocations[id]
	if ref == nil {
		return nil, api.NotFoundErrs("allocation", id.String(), true)
	}
	return ref, nil
}
