package allgather

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

// ErrAllGatherTimeoutExceeded indicates that we not halt within the expected deadline.
var ErrAllGatherTimeoutExceeded = fmt.Errorf(
	"some ranks are taking a long time to connect to master " +
		"during all gather; when running on kubernetes this may happen " +
		"because only some of the pods have been scheduled; it is possible " +
		"that some pods will never be scheduled without adding compute " +
		"resources or pausing / killing other experiments in the cluster",
)

// ErrClosed is returned from a closed and incomplete allgather.
var ErrClosed = fmt.Errorf("left or closed")

// ErrReconnected indicates another watcher connected with the same ID. Only
// one watcher should connect per ID. Anyone attempted to synchronize more things
// should use more `numPeers` and different IDs.
var ErrReconnected = fmt.Errorf("another watcher with the same ID connected")

// DefaultTimeout is the default timeout for all gather.
var DefaultTimeout = 10 * time.Minute

// Watcher signals all gather completion via a channel which is closed upon said completion.
type Watcher struct {
	C <-chan Result
}

// Result contains the information from a completed all gather.
type Result struct {
	Data []any
	Err  error
}

type allGather struct {
	// Configuration details.
	readyCallback   func()
	timeoutCallback func(error)

	// Mutable internal state.
	mu          sync.Mutex
	wg          waitgroupx.Group
	watchers    map[uuid.UUID]chan Result
	data        map[uuid.UUID]any
	numPeers    *int
	alreadyDone bool
}

func newAllGather(readyCallback func(), timeoutCallback func(error)) *allGather {
	g := &allGather{
		readyCallback:   readyCallback,
		timeoutCallback: timeoutCallback,

		wg:       waitgroupx.WithContext(context.Background()),
		watchers: map[uuid.UUID]chan Result{},
		data:     map[uuid.UUID]any{},
	}

	g.wg.Go(g.run)

	return g
}

func (g *allGather) run(ctx context.Context) {
	// don't acquire a lock in here without changing close to not lock while it waits.
	t := time.NewTimer(DefaultTimeout)
	defer t.Stop()

	select {
	case <-t.C:
		if g.timeoutCallback != nil {
			g.timeoutCallback(ErrAllGatherTimeoutExceeded)
		}
	case <-ctx.Done():
	}
}

func (g *allGather) join(id uuid.UUID, numPeers int, data any) Watcher {
	g.mu.Lock()
	defer g.mu.Unlock()

	if c, ok := g.watchers[id]; ok {
		g.removeWatcher(id, c, Result{Err: ErrReconnected})
	}

	w := g.addWatcher(id, numPeers, data)
	if g.done() {
		g.push()
	}
	return w
}

func (g *allGather) leave(id uuid.UUID) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	c, ok := g.watchers[id]
	if !ok {
		return len(g.watchers) == 0
	}

	g.removeWatcher(id, c, Result{Err: ErrClosed})
	return len(g.watchers) == 0
}

// done returns true if and only if all peers are connected.
func (g *allGather) done() bool {
	if g.alreadyDone {
		return true
	}

	ready := g.numPeers != nil && len(g.data) == *g.numPeers
	if !ready {
		return false
	}

	g.alreadyDone = true
	if g.readyCallback != nil {
		g.readyCallback()
	}
	g.wg.Close()
	return true
}

// push gathers the external addresses for the exposed ports and sends them to all the
// containers in the trial.
func (g *allGather) push() {
	res := g.dataSlice()
	for id, c := range g.watchers {
		g.removeWatcher(id, c, Result{Data: res})
	}
}

// addWatcher must be undone by removeWatcher.
func (g *allGather) addWatcher(id uuid.UUID, numPeers int, data any) Watcher {
	// Channel is size 1 since data info will only ever be sent once and we'd rather not block.
	c := make(chan Result, 1)
	g.watchers[id] = c
	g.numPeers = &numPeers
	g.data[id] = data
	return Watcher{C: c}
}

// removeWatcher must undo what addWatcher does.
func (g *allGather) removeWatcher(id uuid.UUID, c chan Result, r Result) {
	c <- r
	close(c)
	delete(g.watchers, id)
	delete(g.data, id)
}

func (g *allGather) dataSlice() []any {
	var res []any
	for _, c := range g.data {
		res = append(res, c)
	}
	return res
}
