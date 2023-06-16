package stream

import (
	"context"
	"time"
	"fmt"
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/stream"
)

type JsonB []byte

// PubSubSystem contains all publishers, and handles all websockets.  It will connect each websocket
// with the appropriate set of publishers, based on that websocket's subscriptions.
//
// There is one PubSubSystem for the whole process.  It has one Publisher per streamable type.
type PubSubSystem struct {
	Trials *stream.Publisher[*TrialMsg, FilterModSet]
	// Experiments *stream.Publisher[*ExperimentMsg, FilterModSet]
}

// SubscriptionSet is a set of all subscribers for this PubSubSystem.
//
// There is one SubscriptionSet for each websocket connection.  It has one SubscriptionManager per
// streamable type.
type SubscriptionSet struct {
	Trials SubscriptionManager[*TrialMsg, TrialFilterMod]
	// Experiments SubscriptionManager[*ExperimentMsg, ExperimentFilterMod]
}

// StartupMsg is the first message a streaming client sends.
//
// It declares initially known keys and also configures the initial subscriptions for the stream.
type StartupMsg struct {
	Known KeySet `json:"known"`
	Subscribe AddOrDropSet `json:"subscribe"`
}

// FilterModSet is a subsequent message from a streaming client.
//
// It allows removing old subscriptions and adding new ones.
type FilterModSet struct {
	Add AddOrDropSet `json:"add"`
	Drop AddOrDropSet `json:"drop"`
}

// KeySet allows a client to describe which primary keys it knows of as existing, so the server
// can respond with a different KeySet of deleted messages of client-known keys that don't exist.
//
// Each field of a KeySet is a comma-separated list of int64s and ranges like "a,b-c,d".
type KeySet struct {
	Trials string `json:"trials"`
	// Experiments string `json:"experiments"`
}

// AddOrDropSet is both the type for .Add and .Drop of the FilterModSet type that a user can
// write to the websocket to change their message type.
type AddOrDropSet struct {
	Trials *TrialFilterMod `json:"trials"`
	// Experiments *ExperimentFilterMod `json:"experiments"`
}

// FilterMaker is a stateful object for building efficient filters.
//
// For example, if users can subscribe to a type Thing by it's primary key, the ThingFilterMaker
// should probably generate a filter function that check if a given ThingMsg.ID appears in a map,
// for O(1) lookups during filtering.
type FilterMaker[T stream.Event] interface {
	AddSpec(spec FilterMod)
	DropSpec(spec FilterMod)
	// MakeFilter should return a nil function if it would always return false.
	MakeFilter() func(T) bool
}

// FilterMod is what a user specifies through the REST API.  There should be one FilterMod
// implementation per streamable type.
type FilterMod interface {
	// Startup emits deletion and update messages for known ids and subscription.  Startup is
	// expected to be called only for the startup message from the streaming clientww.
	Startup(known string, ctx context.Context) ([]*websocket.PreparedMessage, error)
	// Modify emits events matching newly-added subcscriptions.  Modify is meant to be called once
	// per FilterModSet message from the streaming client.
	Modify(ctx context.Context) ([]*websocket.PreparedMessage, error)
}

func NewPubSubSystem() PubSubSystem {
	return PubSubSystem {
		Trials: stream.NewPublisher[*TrialMsg, FilterModSet](),
	}
}

func (pss PubSubSystem) Start(ctx context.Context) {
	// start each publisher
	go publishLoop(ctx, "stream_trial_chan", newTrialMsgs, pss.Trials)
}

func writeAll(socket *websocket.Conn, events []*websocket.PreparedMessage) error {
	for _, ev := range events {
		err := socket.WritePreparedMessage(ev)
		if err != nil {
			return err
		}
	}
	return nil
}

// Websocket is an Echo websocket endpoint.
func (pss PubSubSystem) Websocket(socket *websocket.Conn, c echo.Context) error {
	ctx := c.Request().Context()
	streamer := stream.NewStreamer[FilterModSet]()

	user := 1

	ss := NewSubscriptionSet(streamer, pss, user)
	defer ss.UnsubscribeAll()

	// First read the startup message.
	var startupMsg StartupMsg
	err := socket.ReadJSON(&startupMsg)
	// XXX: errors here don't seem to appear on the websocket side...?
	if err != nil {
		return errors.Wrap(err, "reading startup message")
	}
	// Process deletions, disappearances, and appearances.
	events, err := ss.Startup(startupMsg, ctx)
	if err != nil {
		return errors.Wrapf(err, "gathering startup messages")
	}
	err = writeAll(socket, events)
	if err != nil {
		return errors.Wrapf(err, "writing startup messages")
	}

	// detect context cancelation, and bring it into the websocket thread
	go func() {
		<-ctx.Done()
		streamer.Close()
	}()

	// always be reading for new subscriptions
	go func() {
		// TODO: close streamer if reader goroutine dies?
		for {
			var mods FilterModSet
			err := socket.ReadJSON(&mods)
			if err != nil {
				if websocket.IsUnexpectedCloseError(
					err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
				) {
					log.Errorf("unexpected close error: %v", err)
				}
				break
			}
			// wake up streamer goroutine with the newly-read FilterModSet
			streamer.AddReadEvent(mods)
		}
	}()

	// stream events until the cows come home

	for {
		mods, events, closed := streamer.WaitForSomething()
		// were we closed?
		if closed {
			return nil
		}
		// any modifications to our subscriptions?
		if len(mods) > 0 {
			temp, err := ss.Apply(mods, ctx)
			if err != nil {
				return errors.Wrapf(err, "error modifying subscriptions")
			}
			events = append(events, temp...)
			// TODO: also append a sync message (or one sync per FilterModSet)
		}
		// write events to the websocket
		err = writeAll(socket, events)
		if err != nil {
			// TODO: don't log broken pipe errors.
			if err != nil {
				return errors.Wrapf(err, "error writing to socket")
			}
		}
	}

	return nil
}

func publishLoop[T stream.Event](
	ctx context.Context,
	channelName string,
	rescanFn func(int64, context.Context) (int64, []T, error),
	publisher *stream.Publisher[T, FilterModSet],
) error {
	minReconn := 1 * time.Second
	maxReconn := 10 * time.Second

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Printf("reportProblem: %v\n", err.Error())
		}
	}

	listener := pq.NewListener(
		"postgresql://postgres:postgres@localhost/determined?sslmode=disable",
		minReconn,
		maxReconn,
		reportProblem,
	)

	// start listening
	err := listener.Listen(channelName)
	if err != nil {
		return errors.Wrapf(err, "failed to listen: %v", channelName)
	}

	// scan for initial since
	// TODO: actually just ask for the maximum seq directly.
	since, _, err := rescanFn(0, ctx)
	if err != nil {
		return errors.Wrap(err, "failed initial scan")
	}

	for {
		select {
		// Are we canceled?
		case <-ctx.Done():
			fmt.Printf("publishTrials canceled\n")
			return nil

		// Is there work to do?
		case <-listener.Notify:
			break

		// The pq listener example includes a timeout case, so we do too.
		// (https://pkg.go.dev/github.com/lib/pq/example/listen)
		case <-time.After(30 * time.Second):
			go listener.Ping()
		}

		var evs []T
		since, evs, err = rescanFn(since, ctx)
		if err != nil {
			return errors.Wrap(err, "failed wakeup scan")
		}
		// noop?
		if len(evs) == 0 {
			continue
		}
		// generate updates
		var updates []stream.Update[T]
		for _, ev := range evs {
			update := stream.Update[T]{
				Event: ev,
				// TODO: get valid uids from database instead.
				Users: []int{1, 2},
			}
			updates = append(updates, update)
		}
		stream.Broadcast(publisher, updates)
	}

	return nil
}

func NewSubscriptionSet(
	streamer *stream.Streamer[FilterModSet], pss PubSubSystem, user int,
) SubscriptionSet {
	return SubscriptionSet{
		Trials: NewSubscriptionManager[*TrialMsg, TrialFilterMod](
			streamer, pss.Trials, user, NewTrialFilterMaker(),
		),
	}
}

func (ss *SubscriptionSet) UnsubscribeAll() {
	ss.Trials.Unsubscribe()
	// ss.Exps.Unsubscribe()
}

func (ss *SubscriptionSet) Startup(startupMsg StartupMsg, ctx context.Context) (
	[]*websocket.PreparedMessage, error,
) {
	known := startupMsg.Known
	sub := startupMsg.Subscribe

	// Configure startup subscriptions.
	ss.Trials.Apply(sub.Trials, nil)
	// ss.Experiments.Apply(sub.Experiments, nil)

	// Sync subscription updates with publishers.  Do this before initial scan so that we don't
	// miss any updates.
	ss.Trials.Flush()
	// ss.Expermients.Flush()

	// Do initial startup message scans, which includes detecting removed and added messages.
	var msgs []*websocket.PreparedMessage
	var err error
	msgs, err = ss.Trials.Startup(msgs, err, known.Trials, sub.Trials, ctx)
	// msgs, err = ss.Experiments.Startup(msgs, err, known.Experiments, sub.Experiments, ctx)
	return msgs, err
}

func (ss *SubscriptionSet) Apply(mods []FilterModSet, ctx context.Context) (
	[]*websocket.PreparedMessage, error,
) {
	// apply subscription changes first
	for _, m := range mods {
		ss.Trials.Apply(m.Add.Trials, m.Drop.Trials)
		// ss.Experiments.Apply(m.Add.Experiments, m.Drop.Experiments)
	}

	// Sync subscription updates with publishers.  Do this before initial scan so that we don't
	// miss any updates.
	ss.Trials.Flush()
	// ss.Expermients.Flush()

	// Do initial scans for newly-added subscriptions.
	var msgs []*websocket.PreparedMessage
	var err error
	for _, m := range mods {
		msgs, err = ss.Trials.Modify(msgs, err, m.Add.Trials, ctx)
		// msgs, err = ss.Experiments.Modify(msgs, err, m.Add.Experiments, ctx)
	}
	return msgs, err
}

// SubscriptionManager is a helper function to automate logic around:
// - Running initial db scans after the StartupMsg.
// - Running additional db scans when new subscriptions are added in a FilterModSet message.
// - Passing FilterMod objects to update
// - Updating the filter function for the stream.Subscription.
type SubscriptionManager[T stream.Event, C FilterMod] struct {
	FilterMaker FilterMaker[T]
	StreamSubscription stream.Subscription[T, FilterModSet]
	dirty bool
}

func NewSubscriptionManager[T stream.Event, C FilterMod](
	streamer *stream.Streamer[FilterModSet],
	publisher *stream.Publisher[T, FilterModSet],
	user int,
	filterMaker FilterMaker[T],
) SubscriptionManager[T, C] {
	return SubscriptionManager[T, C]{
		FilterMaker: filterMaker,
		StreamSubscription: stream.NewSubscription(streamer, publisher, user),
	}
}

func (sm *SubscriptionManager[T, C]) Unsubscribe() {
	sm.StreamSubscription.Configure(nil)
}

func (sm *SubscriptionManager[T, C]) Apply(add *C, drop *C) {
	if add != nil {
		sm.FilterMaker.AddSpec(*add)
		sm.dirty = true
	}
	if drop != nil {
		sm.FilterMaker.DropSpec(*drop)
		sm.dirty = true
	}
}

func (sm *SubscriptionManager[T, C]) Flush() {
	if !sm.dirty {
		return
	}
	sm.dirty = false
	sm.StreamSubscription.Configure(sm.FilterMaker.MakeFilter())
}

func (sm *SubscriptionManager[T, C]) Startup(
	msgs []*websocket.PreparedMessage, err error, known string, subscribe *C, ctx context.Context,
) ([]*websocket.PreparedMessage, error) {
	if err != nil || subscribe == nil {
		return msgs, err
	}
	var newMsgs []*websocket.PreparedMessage
	newMsgs, err = (*subscribe).Startup(known, ctx)
	if err != nil {
		return msgs, err
	}
	return append(msgs, newMsgs...), nil
}

func (sm *SubscriptionManager[T, C]) Modify(
	msgs []*websocket.PreparedMessage, err error, add *C, ctx context.Context,
) ([]*websocket.PreparedMessage, error) {
	if err != nil || add == nil {
		return msgs, err
	}
	var newMsgs []*websocket.PreparedMessage
	newMsgs, err = (*add).Modify(ctx)
	if err != nil {
		return msgs, err
	}
	return append(msgs, newMsgs...), nil
}

func prepareMessageWithCache(
	obj interface{}, cache **websocket.PreparedMessage,
) *websocket.PreparedMessage {
	if *cache != nil {
		return *cache
	}
	jbytes, err := json.Marshal(obj)
	if err != nil {
		log.Errorf("error marshaling message for streaming: %v", err.Error())
		return nil
	}
	*cache, err = websocket.NewPreparedMessage(websocket.BinaryMessage, jbytes)
	if err != nil {
		log.Errorf("error preparing message for streaming: %v", err.Error())
		return nil
	}
	return *cache
}
