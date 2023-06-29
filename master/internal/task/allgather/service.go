package allgather

import (
	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/syncx/mapx"
)

var groups = mapx.New[string, *allGather]()

// Join adds the member with `id` to the allgather group `groupID`. The allgather
// waits until `numPeers` members are waiting then gives all members all the submitted `data`
// and fires the `ready` callback. If the `DefaultTimeout` is exceeded before `ready`, the
// `timeout` callback fires. Note, the data is not a copy, it should not be mutated.
func Join(
	groupID string,
	id uuid.UUID,
	numPeers int,
	data any,
	ready func(),
	timeout func(error),
) Watcher {
	var ag *allGather
	groups.WithLock(func(m map[string]*allGather) {
		existing, ok := m[groupID]
		if !ok {
			ag = newAllGather(ready, timeout)
			m[groupID] = ag
		} else {
			ag = existing
		}
	})

	return ag.join(id, numPeers, data)
}

// Leave removes from member with `id` from the allgather group `groupID`.
// The last member out closes the group.
func Leave(groupID string, id uuid.UUID) {
	groups.WithLock(func(m map[string]*allGather) {
		ag, ok := m[groupID]
		if !ok {
			return
		}

		empty := ag.leave(id)
		if empty {
			delete(m, groupID)
		}
	})
}
