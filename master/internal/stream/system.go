package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
)

// JSONB is the golang equivalent of the postgres jsonb column type.
type JSONB interface{}

const (
	minReconn = 1 * time.Second
	maxReconn = 10 * time.Second
)

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

// subscriptionState contains per-type subscription state.
type subscriptionState[T stream.Msg, S any] struct {
	Subscription               stream.Subscription[T]
	FilterMaker                FilterMaker[T, S]
	CollectStartupMsgs         CollectStartupMsgsFunc[S]
	CollectSubscriptionModMsgs CollectSubscriptionModMsgsFunc[S]
}

// CollectStartupMsgsFunc collects messages that were missed prior to startup.
type CollectStartupMsgsFunc[S any] func(
	ctx context.Context,
	user model.User,
	known string,
	spec S,
) (
	[]*websocket.PreparedMessage, error,
)

// CollectSubscriptionModMsgsFunc collects messages that are missed due to modifying a subscription.
type CollectSubscriptionModMsgsFunc[S any] func(ctx context.Context, addSpec S) (
	[]*websocket.PreparedMessage, error,
)

// Restart restarts underlying publishers and closes all active websocket connections.
func (ps *PublisherSet) Restart() (errs []error) {
	ps.Trials.CloseAllStreamers()
	// ps.Experiments.CloseAllStreamers()
	return errs
}

func newDBListener(channel string) (*pq.Listener, error) {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Errorf("reportProblem: %v\n", err.Error())
		}
	}
	listener := pq.NewListener(
		// XXX: update this to use master config rather than hardcoded for a local db
		"postgresql://postgres:postgres@localhost/determined?sslmode=disable",
		minReconn,
		maxReconn,
		reportProblem,
	)
	err := listener.Listen(channel)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

// NewPublisherSet constructor for PublisherSet.
func NewPublisherSet() PublisherSet {
	return PublisherSet{
		Trials: stream.NewPublisher[*TrialMsg](),
		// Experiments: stream.NewPublisher[*ExperimentMsg](),
	}
}

// StartupMsg is the first message a streaming client sends.
//
// It declares initially known keys and also configures the initial subscriptions for the stream.
type StartupMsg struct {
	Known     KnownKeySet         `json:"known"`
	Subscribe SubscriptionSpecSet `json:"subscribe"`
}

// SubscriptionModMsg is a subsequent message from a streaming client.
//
// It allows removing old subscriptions and adding new ones.
type SubscriptionModMsg struct {
	Add  SubscriptionSpecSet `json:"add"`
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

// SubscriptionSpecSet is both the type for .Add and .Drop of
// the SubscriptionModMsg type that a streaming client
// can write to the websocket to change their message type.
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

func (ps *PublisherSet) permissionChangeLoop(ctx context.Context) {
	permListener, err := AuthZProvider.Get().GetPermissionChangeListener()
	switch {
	case err != nil:
		log.Errorf("error occurred while getting permission change listener: %s", err)
		fallthrough
	case permListener == nil:
		// no need to loop if there's no listener
		return
	}
	defer func() {
		err := permListener.Close()
		if err != nil {
			log.Debugf("error occurred while closing permission listener: %s", err)
		}
	}()

	for {
		select {
		// did permissions change?
		case <-permListener.Notify:
			log.Debugf("permission change detected, restarting publisher set")
			for _, err := range ps.Restart() {
				if err != nil {
					log.Debugf("error restarting publisher set: %v", err)
				}
			}
		// is the listener still alive?
		case <-time.After(30 * time.Second):
			pingErrChan := make(chan error)
			go func() {
				err = permListener.Ping()
				pingErrChan <- errors.Wrap(err, "no active connection")
			}()
			if err := <-pingErrChan; err != nil {
				log.Errorf("permission listener failed: %s", err)
			}
		// are we canceled?
		case <-ctx.Done():
			return
		}
	}
}

// Start starts each Publisher in the PublisherSet.
func (ps *PublisherSet) Start(ctx context.Context) {
	// start listening for permission changes
	go ps.permissionChangeLoop(ctx)

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
func (ps *PublisherSet) Websocket(socket *websocket.Conn, c echo.Context) error {
	// clean up socket
	defer func() {
		if err := socket.Close(); err != nil {
			log.Debugf("error while cleaning up socket: %s", err)
		}
	}()

	ctx := c.Request().Context()
	streamer := stream.NewStreamer()
	user := c.(*detContext.DetContext).MustGetUser()

	ss, err := NewSubscriptionSet(ctx, streamer, ps, user)
	if err != nil {
		return errors.Wrap(err, "creating subscription set")
	}
	defer ss.UnsubscribeAll()

	// First read the startup message.
	var startupMsg StartupMsg
	err = socket.ReadJSON(&startupMsg)
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
	msgs, err := ss.Startup(ctx, user, startupMsg)
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
	// (note that online appearances and disappearances are not supported; we'll detect those
	// situations and just break the connection to streaming clients).

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
			func() {
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
			temp, err := ss.SubscriptionMod(ctx, mod)
			if err != nil {
				return errors.Wrapf(err, "error modifying subscriptions")
			}
			msgs = append(msgs, temp...)
			// XXX: also append a sync message (or one sync per SubscriptionModMsg)
		}

		// write msgs to the websocket
		err = writeAll(socket, msgs)
		if err != nil {
			// XXX: don't log broken pipe errors.
			if err != nil {
				return errors.Wrapf(err, "error writing to socket")
			}
		}
	}
}

func publishLoop[T stream.Msg](
	ctx context.Context,
	channelName string,
	publisher *stream.Publisher[T],
) {
	// XXX: is there a better recovery technique than this?
	// XXX: at least boot all the connected streamers, they'll all be invalid now
	for {
		err := doPublishLoop(ctx, channelName, publisher)
		if err != nil {
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
	listener, err := newDBListener(channelName)
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
			pingErrChan := make(chan error)
			go func() {
				err = listener.Ping()
				pingErrChan <- errors.Wrap(err, "no active connection")
			}()
			if err := <-pingErrChan; err != nil {
				return err
			}

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
				case notification = <-listener.Notify:
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
		}
	}
}

// NewSubscriptionSet constructor for SubscriptionSet.
func NewSubscriptionSet(
	ctx context.Context,
	streamer *stream.Streamer,
	ps *PublisherSet,
	user model.User,
) (SubscriptionSet, error) {
	trialPermissionFilter, err := TrialMakePermissionFilter(ctx, user)
	if err != nil {
		return SubscriptionSet{}, err
	}
	return SubscriptionSet{
		Trials: &subscriptionState[*TrialMsg, TrialSubscriptionSpec]{
			stream.NewSubscription(
				streamer,
				ps.Trials,
				trialPermissionFilter,
			),
			NewTrialFilterMaker(),
			TrialCollectStartupMsgs,
			TrialCollectSubscriptionModMsgs,
		},
	}, nil
}

func startup[T stream.Msg, S any](
	ctx context.Context,
	user model.User,
	msgs []*websocket.PreparedMessage,
	err error,
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

	// configure initial filter
	state.FilterMaker.AddSpec(*spec)

	// Sync subscription with publishers.  Do this before initial scan so that we don't
	// miss any events.
	filter := state.FilterMaker.MakeFilter()
	state.Subscription.Configure(filter)

	// Scan for historical msgs matching newly-added subscriptions.
	var newmsgs []*websocket.PreparedMessage
	newmsgs, err = state.CollectStartupMsgs(ctx, user, known, *spec)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, newmsgs...)
	return msgs, nil
}

// Startup handles starting up the Subscription objects in the SubscriptionSet.
func (ss *SubscriptionSet) Startup(ctx context.Context, user model.User, startupMsg StartupMsg) (
	[]*websocket.PreparedMessage, error,
) {
	known := startupMsg.Known
	sub := startupMsg.Subscribe

	var msgs []*websocket.PreparedMessage
	var err error
	msgs, err = startup(ctx, user, msgs, err, ss.Trials, known.Trials, sub.Trials)
	// msgs, err = startup(msgs, err, ctx, ss.Experiments, known.Experiments, sub.Experiments)
	return msgs, err
}

func subMod[T stream.Msg, S any](
	ctx context.Context,
	msgs []*websocket.PreparedMessage,
	err error,
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
		newmsgs, err = state.CollectSubscriptionModMsgs(ctx, *addSpec)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, newmsgs...)
	}
	return msgs, nil
}

// SubscriptionMod modifies a subscription based on the SubscriptionModMsg.
func (ss *SubscriptionSet) SubscriptionMod(ctx context.Context, msg SubscriptionModMsg) (
	[]*websocket.PreparedMessage, error,
) {
	add := msg.Add
	drop := msg.Drop

	var msgs []*websocket.PreparedMessage
	var err error
	msgs, err = subMod(ctx, msgs, err, ss.Trials, add.Trials, drop.Trials)
	// msgs, err = subMod(msgs, err, ctx, ss.Experiments, add.Experiments, drop.Experiments)
	return msgs, err
}

// UnsubscribeAll unsubscribes all Subscription's in the SubscriptionSet.
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
