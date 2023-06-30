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

// TODO(!!!): TBH, a service struct is probably better long term.
var DefaultService = NewAllocationService()

type AllocationService struct {
	mu          sync.RWMutex
	allocations map[model.AllocationID]*Allocation
}

func NewAllocationService() *AllocationService {
	return &AllocationService{
		allocations: map[model.AllocationID]*Allocation{},
	}
}

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

func (as *AllocationService) SendLog(
	ctx context.Context,
	id model.AllocationID,
	log *sproto.ContainerLog,
) {
	ref := as.GetAllocation(id)
	if ref == nil {
		// TODO(!!!): Something, something errors.
		return
	}
	ref.SendLog(log)
}

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

func (as *AllocationService) MarkResourcesDaemon(
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
			syslog.WithError(err).Error("failed to set ready for %s", allocationID)
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

func (as *AllocationService) WaitForRestore(ctx context.Context, id model.AllocationID) error {
	ref := as.GetAllocation(id)
	if ref == nil {
		return api.NotFoundErrs("allocation", id.String(), true)
	}
	return ref.WaitForRestore(ctx)
}
