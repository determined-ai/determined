package allocation

import (
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/pkg/errors"
)

var (
	// DefaultAllGatherTimeout is the default timeout for all gather.
	DefaultAllGatherTimeout = 10 * time.Minute
	// AllGatherTimeoutMessage is the error returned when an all gather times out.
	AllGatherTimeoutMessage = "some ranks are taking a long time to connect to master" +
		"during all gather; when running on kubernetes this may happen " +
		"because only some of the pods have been scheduled; it is possible " +
		"that some pods will never be scheduled without adding compute " +
		"resources or pausing / killing other experiments in the cluster"
)

type (
	allGather struct {
		id       uuid.UUID
		watchers map[uuid.UUID]chan AllGatherInfoOrError
		data     []*structpb.Struct
		numPeers *int

		alreadyDone bool
	}

	// AllGatherWatcher contains a channel which can be polled for all gather completion.
	AllGatherWatcher struct {
		C <-chan AllGatherInfoOrError
	}

	// AllGatherInfoOrError contains the information from a completed all gather.
	AllGatherInfoOrError struct {
		Data []*structpb.Struct
		Err  error
	}

	// Messages for all gathering.

	// WatchAllGather begins or joins an all gather.
	WatchAllGather struct {
		WatcherID uuid.UUID
		NumPeers  int
		Data      *structpb.Struct
	}
	// UnwatchAllGather indicates the peer has disconnected.
	UnwatchAllGather struct {
		WatcherID uuid.UUID
	}
	// Indicates the all gather has timed out.
	allGatherTimeout struct {
		id uuid.UUID
	}
)

func newAllGather(deadlineExceeded func(err error)) *allGather {
	g := &allGather{
		id:       uuid.New(),
		watchers: map[uuid.UUID]chan AllGatherInfoOrError{},
	}

	go func() {
		time.Sleep(DefaultAllGatherTimeout)
		if !g.alreadyDone {
			deadlineExceeded(ErrAllGatherTimeoutExceeded)
		}
	}()

	return g
}

func (g *allGather) watch(msg WatchAllGather) AllGatherWatcher {
	if _, ok := g.watchers[msg.WatcherID]; ok {
		// If this peer has already connected, just respond with the watcher again. This is only
		// possible if it disconnects and reconnects since the original actor ask blocks forever.
		return AllGatherWatcher{C: g.watchers[msg.WatcherID]}
	}

	// Channel is size 1 since data info will only ever be sent once and we'd rather not block.
	w := make(chan AllGatherInfoOrError, 1)
	g.watchers[msg.WatcherID] = w
	g.data = append(g.data, msg.Data)
	g.numPeers = ptrs.Ptr(msg.NumPeers)
	if g.done() {
		g.push()
	}
	return AllGatherWatcher{C: w}
}

func (g *allGather) unwatch(msg UnwatchAllGather) {
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

	g.alreadyDone = g.numPeers != nil && len(g.watchers) == *g.numPeers
	return g.alreadyDone
}

// push gathers up the external addresses for the exposed ports and sends them to all the
// containers in the trial.
func (g *allGather) push() bool {
	if !g.done() {
		return false
	}
	for id, c := range g.watchers {
		c <- AllGatherInfoOrError{Data: g.data}
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
		w <- AllGatherInfoOrError{Err: errors.New("task terminated")}
		close(w)
		delete(g.watchers, cID)
	}
}
