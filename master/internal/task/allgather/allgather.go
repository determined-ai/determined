package allgather

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

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
		g.closeWatcher(id, c, Result{Err: ErrReconnected})
	}

	// Channel is size 1 since data info will only ever be sent once and we'd rather not block.
	c := make(chan Result, 1)
	g.watchers[id] = c
	g.numPeers = &numPeers
	g.data[id] = data
	if g.done() {
		g.push()
	}
	return Watcher{C: c}
}

func (g *allGather) leave(id uuid.UUID) (empty bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	c, ok := g.watchers[id]
	if !ok {
		return
	}

	g.closeWatcher(id, c, Result{Err: ErrClosed})
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
		g.closeWatcher(id, c, Result{Data: res})
	}
}

func (g *allGather) closeWatcher(id uuid.UUID, c chan Result, r Result) {
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
