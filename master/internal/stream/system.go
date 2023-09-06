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

// JsonB is the golang equivalent of the postgres jsonb column type.
type JsonB interface{}

// PublisherSet contains all publishers, and handles all websockets.  It will connect each websocket
// with the appropriate set of publishers, based on that websocket's subscriptions.
//
// There is one PublisherSet for the whole process.  It has one Publisher per streamable type.
type PublisherSet struct {
	Trials *stream.Publisher[*TrialMsg]
	// Experiments *stream.Publisher[*ExperimentMsg]
}

// SubscriptionSet is a set of all subscribers for this PublisherSet.
//
// There is one SubscriptionSet for each websocket connection.  It has one SubscriptionManager per
// streamable type.
type SubscriptionSet struct {
	Trials *subscriptionState[*TrialMsg, TrialSubscriptionSpec]
	// Experiments *subscriptionState[*ExperimentMsg, ExperimentSubscriptionSpec]
}

// subscriptionState contains per-type subscription state
type subscriptionState[T stream.Msg, S any] struct {
	Subscription stream.Subscription[T]
	FilterMaker FilterMaker[T, S]
	CollectStartupMsgs CollectStartupMsgsFunc[S]
	CollectSubscriptionModMsgs CollectSubscriptionModMsgsFunc[S]
}

type CollectStartupMsgsFunc[S any] func(known string, spec S, ctx context.Context) (
	[]*websocket.PreparedMessage, error,
)

type CollectSubscriptionModMsgsFunc[S any] func(addSpec S, ctx context.Context) (
	[]*websocket.PreparedMessage, error,
)

func NewPublisherSet() PublisherSet {
	return PublisherSet {
		Trials: stream.NewPublisher[*TrialMsg](),
	}
}

// StartupMsg is the first message a streaming client sends.
//
// It declares initially known keys and also configures the initial subscriptions for the stream.
type StartupMsg struct {
	Known KnownKeySet `json:"known"`
	Subscribe SubscriptionSpecSet `json:"subscribe"`
}

// SubscriptionModMsg is a subsequent message from a streaming client.
//
// It allows removing old subscriptions and adding new ones.
type SubscriptionModMsg struct {
	Add SubscriptionSpecSet `json:"add"`
	Drop SubscriptionSpecSet `json:"drop"`
}

// KnownKeySet allows a client to describe which primary keys it knows of as existing, so the server
// can respond with a different KnownKeySet of deleted messages of client-known keys that don't
// exist.
//
// Each field of a KnownKeySet is a comma-separated list of int64s and ranges like "a,b-c,d".
type KnownKeySet struct {
	Trials string `json:"trials"`
	// Experiments string `json:"experiments"`
}

// SubscriptionSpecSet is both the type for .Add and .Drop of the SubscriptionModMsg type that a streaming
// client can write to the websocket to change their message type.
type SubscriptionSpecSet struct {
	Trials *TrialSubscriptionSpec `json:"trials"`
	// Experiments *ExperimentSubscriptionSpec `json:"experiments"`
}

// FilterMaker is a stateful object for building efficient filters.
//
// For example, if streaming clients can subscribe to a type Thing by it's primary key, the
// ThingFilterMaker should probably generate a filter function that check if a given ThingMsg.ID
// appears in a map, for O(1) lookups during filtering.
type FilterMaker[T stream.Msg, S any] interface {
	AddSpec(spec S)
	DropSpec(spec S)
	// MakeFilter should return a nil function if it would always return false.
	MakeFilter() func(T) bool
}

func (ps PublisherSet) Start(ctx context.Context) {
	// start each publisher
	go publishLoop(ctx, "stream_trial_chan", ps.Trials)
	// go publishLoop(ctx, "stream_experiment_chan", ps.Experiments)
}

func writeAll(socket *websocket.Conn, msgs []*websocket.PreparedMessage) error {
	for _, msg := range msgs {
		err := socket.WritePreparedMessage(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

// Websocket is an Echo websocket endpoint.
func (ps PublisherSet) Websocket(socket *websocket.Conn, c echo.Context) error {
	ctx := c.Request().Context()
	streamer := stream.NewStreamer()

	ss := NewSubscriptionSet(streamer, ps)
	defer ss.UnsubscribeAll()

	// First read the startup message.
	var startupMsg StartupMsg
	err := socket.ReadJSON(&startupMsg)
	// XXX: errors here don't seem to appear on the websocket side...?
	if err != nil {
		return errors.Wrap(err, "reading startup message")
	}
	// Use the declarative strategy to process all offline events:
	//   - insertions
	//   - updates
	//   - deletions
	//   - appearances
	//   - disappearances
	//   - fallin
	//   - fallout
	msgs, err := ss.Startup(startupMsg, ctx)
	if err != nil {
		return errors.Wrapf(err, "gathering startup messages")
	}
	err = writeAll(socket, msgs)
	if err != nil {
		return errors.Wrapf(err, "writing startup messages")
	}

	// startup done, begin streaming of supported online events:
	//   - insertions
	//   - updates
	//   - deletions
	//   - fallin
	//   - fallout
	//
	// (note that online appearences and disappearances are not supported; we'll detect those
	// situations and just break the connection to the relevant streaming clients).

	// detect context cancelation, and bring it into the websocket thread
	go func() {
		<-ctx.Done()
		streamer.Close()
	}()

	// reads is where we collect SubscriptionModMsg messages we read from the websocket until
	// waitForSomething() delivers those messages to the websocket goroutine.
	var reads []SubscriptionModMsg

	// always be reading for new subscriptions
	go func() {
		defer streamer.Close()
		for {
			var mods SubscriptionModMsg
			err := socket.ReadJSON(&mods)
			if err != nil {
				if websocket.IsUnexpectedCloseError(
					err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
				) {
					log.Errorf("unexpected close error: %v", err)
				}
				break
			}
			// wake up streamer goroutine with the newly-read SubscriptionModMsg
			func(){
				streamer.Cond.L.Lock()
				defer streamer.Cond.L.Unlock()
				streamer.Cond.Signal()
				reads = append(reads, mods)
			}()
		}
	}()

	// waitForSomething returns a tuple of (mods, msgs, closed)
	waitForSomething := func() ([]SubscriptionModMsg, []*websocket.PreparedMessage, bool) {
		streamer.Cond.L.Lock()
		defer streamer.Cond.L.Unlock()
		streamer.Cond.Wait()
		// steal outputs
		mods := reads
		reads = nil
		msgs := streamer.Msgs
		streamer.Msgs = nil
		return mods, msgs, streamer.Closed
	}

	for {
		mods, msgs, closed := waitForSomething()

		// were we closed?
		if closed {
			return nil
		}

		// any modifications to our subscriptions?
		for _, mod := range mods {
			temp, err := ss.SubscriptionMod(mod, ctx)
			if err != nil {
				return errors.Wrapf(err, "error modifying subscriptions")
			}
			msgs = append(msgs, temp...)
			// TODO: also append a sync message (or one sync per SubscriptionModMsg)
		}

		// write msgs to the websocket
		err = writeAll(socket, msgs)
		if err != nil {
			// TODO: don't log broken pipe errors.
			if err != nil {
				return errors.Wrapf(err, "error writing to socket")
			}
		}
	}

	return nil
}

func publishLoop[T stream.Msg](
	ctx context.Context,
	channelName string,
	publisher *stream.Publisher[T],
) {
	// TODO: is there a better recovery technique than this?
	// XXX: at least boot all the connected streamers, they'll all be invalid now
	for {
		err := doPublishLoop(ctx, channelName, publisher)
		if err != nil{
			log.Errorf("publishLoop failed (will restart): %v", err.Error())
			continue
		}
		// exited without error
		break
	}
}

func doPublishLoop[T stream.Msg](
	ctx context.Context,
	channelName string,
	publisher *stream.Publisher[T],
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

	for {
		var events []stream.Event[T]
		select {
		// Are we canceled?
		case <-ctx.Done():
			// fmt.Printf("publishTrials canceled\n")
			return nil

		// The pq listener example includes a timeout case, so we do too.
		// (https://pkg.go.dev/github.com/lib/pq/example/listen)
		case <-time.After(30 * time.Second):
			go listener.Ping()

		// Did we get a notification?
		case notification := <-listener.Notify:
			// fmt.Printf("Notify: %v\n", notification.Extra)
			var event stream.Event[T]
			err = json.Unmarshal([]byte(notification.Extra), &event)
			if err != nil {
				return err
			}
			events = append(events, event)
			// Collect all available notifications before proceeding.
			keepGoing := true
			for keepGoing {
				select {
				case notification = <- listener.Notify:
					// fmt.Printf("More Notify: %v\n", notification.Extra)
					var event stream.Event[T]
					err = json.Unmarshal([]byte(notification.Extra), &event)
					if err != nil {
						return err
					}
					events = append(events, event)
				default:
					keepGoing = false
				}
			}
			// Broadcast all the events.
			publisher.Broadcast(events)
			break
		}
	}

	return nil
}

func NewSubscriptionSet(streamer *stream.Streamer, ps PublisherSet) SubscriptionSet {
	return SubscriptionSet{
		Trials: &subscriptionState[*TrialMsg, TrialSubscriptionSpec]{
			stream.NewSubscription(streamer, ps.Trials),
			NewTrialFilterMaker(),
			TrialCollectStartupMsgs,
			TrialCollectSubscriptionModMsgs,
		},
	}
}

func startup[T stream.Msg, S any](
	msgs []*websocket.PreparedMessage,
	err error,
	ctx context.Context,
	state *subscriptionState[T, S],
	known string,
	spec *S,
) ([]*websocket.PreparedMessage, error) {
	if err != nil {
		return nil, err
	}
	if spec == nil {
		// no change
		return msgs, nil
	}

	// configure intial filter
	state.FilterMaker.AddSpec(*spec)

	// Sync subscription with publishers.  Do this before initial scan so that we don't
	// miss any events.
	filter := state.FilterMaker.MakeFilter()
	state.Subscription.Configure(filter)

	// Scan for historical msgs matching newly-added subscriptions.
	var newmsgs []*websocket.PreparedMessage
	newmsgs, err = state.CollectStartupMsgs(known, *spec, ctx)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, newmsgs...)
	return msgs, nil
}

func (ss *SubscriptionSet) Startup(startupMsg StartupMsg, ctx context.Context) (
	[]*websocket.PreparedMessage, error,
) {
	known := startupMsg.Known
	sub := startupMsg.Subscribe

	var msgs []*websocket.PreparedMessage
	var err error
	msgs, err = startup(msgs, err, ctx, ss.Trials, known.Trials, sub.Trials)
	// msgs, err = startup(msgs, err, ctx, ss.Experiments, known.Experiments, sub.Experiments)
	return msgs, err
}

func subMod[T stream.Msg, S any](
	msgs []*websocket.PreparedMessage,
	err error,
	ctx context.Context,
	state *subscriptionState[T, S],
	addSpec *S,
	dropSpec *S,
) ([]*websocket.PreparedMessage, error) {
	if err != nil {
		return nil, err
	}
	if addSpec == nil && dropSpec == nil {
		// no change
		return msgs, nil
	}

	// apply SubscriptionSpec changes
	if addSpec != nil {
		state.FilterMaker.AddSpec(*addSpec)
	}
	if dropSpec != nil {
		state.FilterMaker.DropSpec(*dropSpec)
	}

	// Sync subscription changes with publishers.  Do this before initial scan so that we don't
	// miss any events.
	filter := state.FilterMaker.MakeFilter()
	state.Subscription.Configure(filter)

	if addSpec != nil {
		// Scan for historical msgs matching newly-added subscriptions.
		var newmsgs []*websocket.PreparedMessage
		newmsgs, err = state.CollectSubscriptionModMsgs(*addSpec, ctx)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, newmsgs...)
	}
	return msgs, nil
}

func (ss *SubscriptionSet) SubscriptionMod(msg SubscriptionModMsg, ctx context.Context) (
	[]*websocket.PreparedMessage, error,
) {
	add := msg.Add
	drop := msg.Drop

	var msgs []*websocket.PreparedMessage
	var err error
	msgs, err = subMod(msgs, err, ctx, ss.Trials, add.Trials, drop.Trials)
	// msgs, err = subMod(msgs, err, ctx, ss.Experiments, add.Experiments, drop.Experiments)
	return msgs, err
}

func (ss *SubscriptionSet) UnsubscribeAll() {
	ss.Trials.Subscription.Configure(nil)
	// ss.Experiments.Subscription.Configure(nil)
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
	*cache, err = websocket.NewPreparedMessage(websocket.TextMessage, jbytes)
	if err != nil {
		log.Errorf("error preparing message for streaming: %v", err.Error())
		return nil
	}
	return *cache
}

func newDeletedMsg(key string, deleted string) *websocket.PreparedMessage {
	strMsg := fmt.Sprintf("{\"%v\": \"%v\"}", key, deleted)
	msg, err := websocket.NewPreparedMessage(websocket.TextMessage, []byte(strMsg))
	if err != nil {
		log.Errorf("error marshaling deletion message for streaming: %v", err.Error())
		return nil
	}
	return msg
}

func newDeletedMsgWithCache(
	key string, deleted string, cache **websocket.PreparedMessage,
) *websocket.PreparedMessage {
	if *cache != nil {
		return *cache
	}
	*cache = newDeletedMsg(key, deleted)
	return *cache
}
