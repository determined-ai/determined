package task

import (
	"context"

	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// AllocationService allows callers to launch, direct and query allocations.
type AllocationService interface {
	GetAllAllocationIDs() []model.AllocationID
	StartAllocation(
		logCtx logger.Context,
		req sproto.AllocateRequest,
		db db.DB,
		rm rm.ResourceManager,
		specifier tasks.TaskSpecifier,
		system *actor.System,
		onExit func(*AllocationExited),
	)
	Signal(
		id model.AllocationID,
		sig AllocationSignal,
		reason string,
	) error
	State(id model.AllocationID) (AllocationState, error)
	SetReady(ctx context.Context, id model.AllocationID) error
	SetWaiting(ctx context.Context, id model.AllocationID) error
	SetProxyAddress(
		ctx context.Context,
		id model.AllocationID,
		addr string,
	) error
	WatchRendezvous(
		ctx context.Context,
		id model.AllocationID,
		rID sproto.ResourcesID,
	) (*trialv1.RendezvousInfo, error)
	SetResourcesAsDaemon(
		ctx context.Context,
		id model.AllocationID,
		rID sproto.ResourcesID,
	) error
	AllGather(
		ctx context.Context,
		allocationID model.AllocationID,
		id uuid.UUID,
		numPeers int,
		data any,
	) ([]any, error)
	WatchPreemption(ctx context.Context, id model.AllocationID) (bool, error)
	AckPreemption(ctx context.Context, id model.AllocationID) error
	SendLog(
		ctx context.Context,
		id model.AllocationID,
		log *sproto.ContainerLog,
	)
}
