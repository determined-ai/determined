package task

import (
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
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
	// AllGather performs an all gather for an allocation.
	AllGather struct {
		id       uuid.UUID
		watchers map[uuid.UUID]chan AllGatherInfoOrError
		data     []*structpb.Struct
		numPeers *int

		readyPassed bool
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

// NewAllGather returns a new all gather component.
func NewAllGather(ctx *actor.Context) *AllGather {
	id := uuid.New()
	if ctx != nil {
		actors.NotifyAfter(ctx, DefaultAllGatherTimeout, allGatherTimeout{id: id})
	}
	return &AllGather{
		id:       id,
		watchers: map[uuid.UUID]chan AllGatherInfoOrError{},
	}
}

// ReceiveMsg receives rendezvous-specific messages.
func (g *AllGather) ReceiveMsg(ctx *actor.Context) (bool, error) {
	switch msg := ctx.Message().(type) {
	case WatchAllGather:
		ctx.Respond(g.watch(msg.WatcherID, msg.NumPeers, msg.Data))
	case UnwatchAllGather:
		g.unwatch(msg.WatcherID)
	case allGatherTimeout:
		if msg.id == g.id {
			return false, ErrTimeoutExceeded{Message: AllGatherTimeoutMessage}
		}
	default:
		return false, actor.ErrUnexpectedMessage(ctx)
	}
	return g.ready(), nil
}

func (g *AllGather) watch(id uuid.UUID, count int, data *structpb.Struct) AllGatherWatcher {
	if _, ok := g.watchers[id]; ok {
		// If this peer has already connected, just respond with the watcher again. This is only
		// possible if it disconnects and reconnects since the original actor ask blocks forever.
		return AllGatherWatcher{C: g.watchers[id]}
	}

	// Channel is size 1 since data info will only ever be sent once and we'd rather not block.
	w := make(chan AllGatherInfoOrError, 1)
	g.watchers[id] = w
	g.data = append(g.data, data)
	g.numPeers = ptrs.IntPtr(count)
	if g.ready() {
		g.push()
	}
	return AllGatherWatcher{C: w}
}

func (g *AllGather) unwatch(id uuid.UUID) {
	delete(g.watchers, id)
}

// ready returns true if and only if all the containers are reported to be started with the
// ContainerStarted message and their sockets to be connected with the containerConnected
// message. The two messages are not guaranteed to come in-order. During each run of the
// trial, once all the containers are ready this function will return true afterward because this
// function is used in deciding if the trial should be forcibly killed when terminating.
func (g *AllGather) ready() bool {
	if g == nil {
		return false
	}

	if g.readyPassed {
		return true
	}

	g.readyPassed = g.numPeers != nil && len(g.watchers) == *g.numPeers
	return g.readyPassed
}

// push gathers up the external addresses for the exposed ports and sends them to all the
// containers in the trial.
func (g AllGather) push() bool {
	if !g.ready() {
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
func (g *AllGather) Close() {
	if g == nil {
		return
	}

	for cID, w := range g.watchers {
		w <- AllGatherInfoOrError{Err: errors.New("task terminated")}
		close(w)
		delete(g.watchers, cID)
	}
}
