package allgather

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"

	"github.com/pkg/errors"
)

// ErrAllGatherTimeoutExceeded indicates that we not halt within the expected deadline.
var ErrAllGatherTimeoutExceeded = fmt.Errorf(
	"some ranks are taking a long time to connect to master" +
		"during all gather; when running on kubernetes this may happen " +
		"because only some of the pods have been scheduled; it is possible " +
		"that some pods will never be scheduled without adding compute " +
		"resources or pausing / killing other experiments in the cluster")

// DefaultTimeout is the default timeout for all gather.
var DefaultTimeout = 10 * time.Minute

// Watcher contains a channel which can be polled for all gather completion.
type Watcher struct {
	C <-chan Result
}

// Result contains the information from a completed all gather.
type Result struct {
	Data []*structpb.Struct
	Err  error
}

// Messages for all gathering.

// WatchRequest begins or joins an all gather.
type WatchRequest struct {
	WatcherID uuid.UUID
	NumPeers  int
	Data      *structpb.Struct
}

// UnwatchRequest indicates the peer has disconnected.
type UnwatchRequest struct {
	WatcherID uuid.UUID
}

type allGather struct {
	id              uuid.UUID
	readyCallback   func()
	timeoutCallback func(error)
	watchers        map[uuid.UUID]chan Result
	data            []*structpb.Struct
	numPeers        *int

	alreadyDone bool
	wg          waitgroupx.Group
}

func New(readyCallback func(), timeoutCallback func(error)) *allGather {
	a := &allGather{
		id:              uuid.New(),
		readyCallback:   readyCallback,
		timeoutCallback: timeoutCallback,
		watchers:        map[uuid.UUID]chan Result{},
		wg:              waitgroupx.WithContext(context.Background()),
	}

	a.wg.Go(a.run)

	return a
}

func (ag *allGather) run(ctx context.Context) {
	// don't acquire a lock in here without changing close to not lock while it waits.
	t := time.NewTimer(DefaultTimeout)
	defer t.Stop()

	select {
	case <-t.C:
		if ag.timeoutCallback != nil {
			ag.timeoutCallback(ErrAllGatherTimeoutExceeded)
		}
	case <-ctx.Done():
	}
}

func (g *allGather) watch(msg WatchRequest) Watcher {
	if _, ok := g.watchers[msg.WatcherID]; ok {
		// If this peer has already connected, just respond with the watcher again. This is only
		// possible if it disconnects and reconnects since the original actor ask blocks forever.
		return Watcher{C: g.watchers[msg.WatcherID]}
	}

	// Channel is size 1 since data info will only ever be sent once and we'd rather not block.
	w := make(chan Result, 1)
	g.watchers[msg.WatcherID] = w
	g.data = append(g.data, msg.Data)
	g.numPeers = ptrs.Ptr(msg.NumPeers)
	if g.done() {
		g.push()
	}
	return Watcher{C: w}
}

func (g *allGather) unwatch(msg UnwatchRequest) {
	delete(g.watchers, msg.WatcherID)
}

// done returns true if and only if all peers are connected.
func (g *allGather) done() bool {
	if g == nil {
		return false
	}

	if g.alreadyDone {
		return true
	}

	ready := g.numPeers != nil && len(g.watchers) == *g.numPeers
	if !ready {
		return false
	}

	g.alreadyDone = true
	if g.readyCallback != nil {
		g.readyCallback()
	}
	return true
}

// push gathers up the external addresses for the exposed ports and sends them to all the
// containers in the trial.
func (g allGather) push() bool {
	if !g.done() {
		return false
	}
	for id, c := range g.watchers {
		c <- Result{Data: g.data}
		close(c)
		delete(g.watchers, id)
	}
	return true
}

// Close closes rendezvous by letting still active watchers know they were terminated.
func (g *allGather) Close() {
	if g == nil {
		return
	}

	for cID, w := range g.watchers {
		w <- Result{Err: errors.New("task terminated")}
		close(w)
		delete(g.watchers, cID)
	}
}
