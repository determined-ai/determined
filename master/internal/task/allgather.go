package task

import (
	"context"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/task/allgather"
	"github.com/determined-ai/determined/master/pkg/model"
)

// AllGather blocks until `numPeers` with the same `allocationID` are waiting and then returns the
// data from all those peers. It returns an error if the call returns early without data for any
// reason. Only one call may connect per `id`.
func AllGather(
	ctx context.Context,
	allocationID model.AllocationID,
	id uuid.UUID,
	numPeers int,
	data any,
) ([]any, error) {
	err := WaitForRestore(ctx, allocationID)
	if err != nil {
		return nil, err
	}

	readyFn := func() {
		err := SetReady(ctx, allocationID)
		if err != nil {
			logrus.WithError(err).Error("failed to set ready for %s", allocationID)
		}
	}

	timeoutFn := func(err error) {
		msg := err.Error()
		SendLog(ctx, allocationID, &sproto.ContainerLog{AuxMessage: &msg})
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
