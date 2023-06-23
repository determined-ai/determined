package allgather

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/syncx/mapx"
)

// ErrAllGatherTimeoutExceeded indicates that we not halt within the expected deadline.
var ErrAllGatherTimeoutExceeded = fmt.Errorf(
	"some ranks are taking a long time to connect to master" +
		"during all gather; when running on kubernetes this may happen " +
		"because only some of the pods have been scheduled; it is possible " +
		"that some pods will never be scheduled without adding compute " +
		"resources or pausing / killing other experiments in the cluster")

// ErrClosed is returned from a closed and incomplete allgather.
var ErrClosed = fmt.Errorf("left or closed")

// ErrReconnected indicates another watcher connected with the same ID. Only
// one watcher should connect per ID. Anyone attempted to synchronize more things
// should use more `numPeers` and different IDs.
var ErrReconnected = fmt.Errorf("another watcher with the same ID connected")

// DefaultTimeout is the default timeout for all gather.
var DefaultTimeout = 10 * time.Minute

// Watcher contains a channel which can be polled for all gather completion.
type Watcher struct {
	C <-chan Result
}

// Result contains the information from a completed all gather.
type Result struct {
	Data []any
	Err  error
}

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
