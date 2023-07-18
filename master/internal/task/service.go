package task

import (
	"time"

	"github.com/determined-ai/determined/master/internal/rm/allocationmap"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

// SetReady asynchronously set the allocation to the ready state.
func SetReady(ctx actor.Messenger, id model.AllocationID) {
	ref := allocationmap.GetAllocation(id)
	if ref == nil {
		return
	}
	ctx.Tell(ref, AllocationReady{})
}

// SendLog inserts a log for the allocation.
func SendLog(ctx actor.Messenger, id model.AllocationID, msg string) {
	ref := allocationmap.GetAllocation(id)
	if ref == nil {
		return
	}
	ctx.Tell(
		ref,
		sproto.ContainerLog{Timestamp: time.Now().UTC(), AuxMessage: &msg},
	)
}
