package task

import (
	"context"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/task/allgather"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

// AllGather blocks until `numPeers` with the same `allocationID` are waiting and then returns the
// data from all those peers. It returns an error if the call returns early without data for any
// reason. Only one call may connect per `id`.
func AllGather(
	ctx context.Context,
	msgr actor.Messenger,
	allocationID model.AllocationID,
	id uuid.UUID,
	numPeers int,
	data any,
) ([]any, error) {
	w := allgather.Join(
		allocationID.String(), id, numPeers, data,
		func() { SetReady(msgr, allocationID) },
		func(err error) { SendLog(msgr, allocationID, err.Error()) },
	)
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
