package task

import (
	"context"
	"github.com/determined-ai/determined/master/internal/task/allgather"
	"github.com/determined-ai/determined/master/pkg/model"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/google/uuid"
)

func WatchAllGather(
	ctx context.Context,
	allocationID model.AllocationID,
	id uuid.UUID,
	numPeers int,
	data *structpb.Struct,
) ([]*structpb.Struct, error) {
	w := allgather.Watch(
		allocationID.String(), id, numPeers, data,
		func() { SetReady(allocationID) },
		func(err error) { SendLog(allocationID, err.Error()) },
	)
	defer allgather.Unwatch(allocationID.String(), id)

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
