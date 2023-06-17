package allgather

import (
	"github.com/determined-ai/determined/master/pkg/syncx/mapx"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
)

var processGroups = mapx.New[string, *allGather]()

func Watch(
	groupID string,
	id uuid.UUID,
	numPeers int,
	data *structpb.Struct,
	ready func(),
	timeout func(error),
) Watcher {
	var ag *allGather
	processGroups.WithLock(func(m map[string]*allGather) {
		existing, ok := m[groupID]
		if !ok {
			ag = New(ready, timeout)
			m[groupID] = ag
		} else {
			ag = existing
		}
	})

	return ag.watch(WatchRequest{
		WatcherID: id,
		NumPeers:  numPeers,
		Data:      data,
	})
}

func Unwatch(groupID string, id uuid.UUID) {
	ag, ok := processGroups.Load(groupID)
	if !ok {
		return
	}
	ag.unwatch(UnwatchRequest{WatcherID: id})
}
