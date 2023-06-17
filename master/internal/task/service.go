package task

import (
	"github.com/determined-ai/determined/master/internal/actorsystem"
	"github.com/determined-ai/determined/master/internal/rm/allocationmap"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"time"
)

func SetReady(id model.AllocationID) {
	ref := allocationmap.GetAllocation(id)
	if ref == nil {
		return
	}
	actorsystem.DefaultSystem.Tell(ref, AllocationReady{})
}

func SendLog(id model.AllocationID, msg string) {
	ref := allocationmap.GetAllocation(id)
	if ref == nil {
		return
	}
	actorsystem.DefaultSystem.Tell(
		ref,
		sproto.ContainerLog{Timestamp: time.Now().UTC(), AuxMessage: &msg},
	)
}
