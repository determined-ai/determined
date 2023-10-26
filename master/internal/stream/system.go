package stream

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
)

// JSONB is the golang equivalent of the postgres jsonb column type.
type JSONB interface{}

const (
	minReconn  = 1 * time.Second
	maxReconn  = 10 * time.Second
	trialChan  = "stream_trial_chan"
	metricChan = "stream_metric_chan"
)

// PublisherSet contains all publishers, and handles all websockets.  It will connect each websocket
// with the appropriate set of publishers, based on that websocket's subscriptions.
//
// There is one PublisherSet for the whole process.  It has one Publisher per streamable type.
type PublisherSet struct {
	DBAddress string
	Trials    *stream.Publisher[*TrialMsg]
	Metrics   *stream.Publisher[*MetricMsg]
	// Experiments *stream.Publisher[*ExperimentMsg]
	bootemChan chan struct{}
	bootLock   sync.Mutex
}

// SubscriptionSet is a set of all subscribers for this PublisherSet.
//
// There is one SubscriptionSet for each websocket connection.  It has one SubscriptionManager per
// streamable type.
type SubscriptionSet struct {
	Trials  *subscriptionState[*TrialMsg, TrialSubscriptionSpec]
	Metrics *subscriptionState[*MetricMsg, MetricSubscriptionSpec]
	// Experiments *subscriptionState[*ExperimentMsg, ExperimentSubscriptionSpec]
}

// subscriptionState contains per-type subscription state.
type subscriptionState[T stream.Msg, S any] struct {
	Subscription       stream.Subscription[T]
	CollectStartupMsgs CollectStartupMsgsFunc[S]
}

// CollectStartupMsgsFunc collects messages that were missed prior to startup.
type CollectStartupMsgsFunc[S any] func(
	ctx context.Context,
	user model.User,
	known string,
	spec S,
) (
	[]stream.PreparableMessage, error,
)

func newDBListener(address, channel string) (*pq.Listener, error) {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Errorf("reportProblem: %v\n", err.Error())
		}
	}
	listener := pq.NewListener(
		// XXX: update this to use master config rather than hardcoded for a local db
		address,
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
func NewPublisherSet() *PublisherSet {
	return &PublisherSet{
		DBAddress: "postgresql://postgres:postgres@localhost/determined?sslmode=disable",
		Trials:    stream.NewPublisher[*TrialMsg](),
		Metrics:   stream.NewPublisher[*MetricMsg](),
		// Experiments: stream.NewPublisher[*ExperimentMsg](),
		bootemChan: make(chan struct{}),
	}
}

// SyncMsg is the server response to a StartupMsg once it's been handled.
type SyncMsg struct {
	SyncID string `json:"sync_id"`
}

// toPreparedMessage converts as SyncMsg to a PreparedMessage.
func (sm SyncMsg) toPreparedMessage() *websocket.PreparedMessage {
	jbytes, err := json.Marshal(sm)
	if err != nil {
		log.Errorf("error marshaling sync message for streaming: %v", err.Error())
		return nil
	}
	msg, err := websocket.NewPreparedMessage(websocket.TextMessage, jbytes)
	if err != nil {
		log.Errorf("error preparing sync message for streaming: %v", err.Error())
		return nil
	}
	return msg
}

// StartupMsg is the first message a streaming client sends.
//
// It declares initially known keys and also configures the initial subscriptions for the stream.
type StartupMsg struct {
	SyncID    string              `json:"sync_id"`
	Known     KnownKeySet         `json:"known"`
	Subscribe SubscriptionSpecSet `json:"subscribe"`
}

// KnownKeySet allows a client to describe which primary keys it knows of as existing, so the server
// can respond with a different KnownKeySet of deleted messages of client-known keys that don't
// exist.
//
// Each field of a KnownKeySet is a comma-separated list of int64s and ranges like "a,b-c,d".
type KnownKeySet struct {
	Trials  string `json:"trials"`
	Metrics string `json:"metrics"`
	// Experiments string `json:"experiments"`
}

// SubscriptionSpecSet is the set of subscription specs that can be sent in startup message.
type SubscriptionSpecSet struct {
	Trials  *TrialSubscriptionSpec  `json:"trials"`
	Metrics *MetricSubscriptionSpec `json:"metrics"`
	// Experiments *ExperimentSubscriptionSpec `json:"experiments"`
}

func start[T stream.Msg](
	ctx context.Context,
	pgAddress,
	channel string,
	publisher *stream.Publisher[T],
) error {
	return publishLoop(ctx, pgAddress, channel, publisher)
}

// Start starts each Publisher in the PublisherSet.
func (ps *PublisherSet) Start(ctx context.Context) error {
	eg := errgroupx.WithContext(ctx)

	eg.Go(
		func(c context.Context) error {
			return start(ctx, ps.DBAddress, trialChan, ps.Trials)
		},
	)
	eg.Go(
		func(c context.Context) error {
			return start(ctx, ps.DBAddress, metricChan, ps.Metrics)
		},
	)
	// eg.Go(start(ctx, "stream_experiment_chan", ps.Experiments))
	return eg.Wait()
}

func writeAll(socketLike WebsocketLike, msgs []interface{}) error {
	for _, msg := range msgs {
		err := socketLike.Write(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

// Websocket is an Echo websocket endpoint.
func (ps *PublisherSet) entrypoint(
	ssupCtx context.Context,
	ctx context.Context,
	user model.User,
	socket WebsocketLike,
	prepareFunc func(message stream.PreparableMessage) interface{},
) error {
	// get permission change channel
	var bootemChan chan struct{}
	func() {
		ps.bootLock.Lock()
		defer ps.bootLock.Unlock()
		bootemChan = ps.bootemChan
	}()

	streamer := stream.NewStreamer(prepareFunc)
	// read and handle in initial startup message
	var startupMsg StartupMsg
	err := socket.ReadJSON(&startupMsg)
	if err != nil {
		return errors.Wrapf(err, "error while reading initial startup message")
	}
	var ss SubscriptionSet
	defer ss.UnsubscribeAll()
	msgs, err := startupMsgHandler(ctx, startupMsg, ps, &ss, user, streamer)
	if err != nil {
		if !websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
			log.Debugf("unable to handle startup message: %s", err)
		}
		return errors.Wrapf(err, "error handling startup message")
	}
	err = writeAll(socket, msgs)
	if err != nil {
		return errors.Wrapf(err, "writing startup messages")
	}

	// startups is where we collect StartupMsg messages we read from the websocket until
	// waitForSomething() delivers those messages to the websocket goroutine.
	var startups []StartupMsg

	// always be reading for new startup messages
	go func() {
		defer streamer.Close()
		for {
			var mods StartupMsg
			err := socket.ReadJSON(&mods)
			if err != nil {
				if websocket.IsUnexpectedCloseError(
					err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
				) {
					log.Errorf("unexpected close error: %v", err)
				}
				break
			}
			// wake up streamer goroutine with the newly-read StartupMsg
			func() {
				streamer.Cond.L.Lock()
				defer streamer.Cond.L.Unlock()
				streamer.Cond.Signal()
				startups = append(startups, mods)
			}()
		}
	}()

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
		select {
		case <-ctx.Done():
			streamer.Close()
		case <-ssupCtx.Done():
			// close streamer if supervisor is down
			streamer.Close()
		case <-bootemChan:
			// close this streamer if online appearance/disappearance occurred
			streamer.Close()
		}
	}()

	// waitForSomething returns a tuple of (msgs, closed)
	waitForSomething := func() ([]StartupMsg, []interface{}, bool) {
		streamer.Cond.L.Lock()
		defer streamer.Cond.L.Unlock()
		streamer.Cond.Wait()
		// steal outputs
		starts := startups
		startups = nil
		msgs := streamer.Msgs
		streamer.Msgs = nil
		return starts, msgs, streamer.Closed
	}

	for {
		starts, msgs, closed := waitForSomething()

		// is the streamer closed?
		if closed {
			return nil
		}

		// any new startups?
		for _, start := range starts {
			temp, err := startupMsgHandler(ctx, start, ps, &ss, user, streamer)
			if err != nil {
				return errors.Wrapf(err, "error handling additional startup messages")
			}
			msgs = append(msgs, temp...)
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

// Websocket is an Echo websocket endpoint.
func (ps *PublisherSet) Websocket(
	ssupCtx context.Context,
	socket *websocket.Conn,
	c echo.Context,
) error {
	reqCtx := c.Request().Context()
	detCtx, ok := c.(*detContext.DetContext)
	if !ok {
		log.Errorf("unable to run PublisherSet: expected DetContext but received %t",
			reflect.TypeOf(c))
	}
	user := detCtx.MustGetUser()
	return ps.entrypoint(ssupCtx, reqCtx, user, &WrappedWebsocket{Conn: socket},
		prepareWebsocketMessage)
}

func prepareWebsocketMessage(obj stream.PreparableMessage) interface{} {
	jbytes, err := json.Marshal(obj)
	if err != nil {
		log.Errorf("error marshaling message for streaming: %v", err.Error())
		return nil
	}
	msg, err := websocket.NewPreparedMessage(websocket.TextMessage, jbytes)
	if err != nil {
		log.Errorf("error preparing message for streaming: %v", err.Error())
		return nil
	}
	return msg
}

// startupMsgHandler reads and responds to startup messages sent by streaming clients.
func startupMsgHandler(
	ctx context.Context,
	startupMsg StartupMsg,
	ps *PublisherSet,
	ss *SubscriptionSet,
	user model.User,
	streamer *stream.Streamer,
) (msgs []interface{}, err error) {
	// check for active subscriptions
	if ss != nil {
		ss.UnsubscribeAll()
	}

	// create new subscription set
	tempSS, err := NewSubscriptionSet(ctx, streamer, ps, user, startupMsg.Subscribe)
	*ss = tempSS
	if err != nil {
		return msgs, errors.Wrap(err, "creating subscription set")
	}

	// startup subscription set
	offlineMsgs, err := ss.Startup(ctx, user, startupMsg)
	if err != nil {
		return msgs, errors.Wrapf(err, "gathering startup messages")
	}
	msgs = append(msgs, offlineMsgs...)

	// add sync message if everything else went correctly
	syncMsg := SyncMsg{SyncID: startupMsg.SyncID}
	msgs = append(msgs, syncMsg.toPreparedMessage())

	return msgs, nil
}

// bootStreamers closes and replaces the bootem channel with a new channel.
func (ps *PublisherSet) bootStreamers() {
	ps.bootLock.Lock()
	defer ps.bootLock.Unlock()
	close(ps.bootemChan)
	ps.bootemChan = make(chan struct{})
}

// BootemLoop listens for permission changes, updates the PublisherSet
// to signal to boot streamers, returns an error in the event of a failure to listen.
func BootemLoop(ctx context.Context, ps *PublisherSet) error {
	permListener, err := AuthZProvider.Get().GetPermissionChangeListener()
	if err != nil {
		log.Errorf("unable to get permission change listener: %s", err)
		return err
	}
	if permListener == nil {
		// no listener means we don't have permissions configured at all
		return nil
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
			log.Debugf("permission change detected, booting streamers")
			func() {
				ps.bootStreamers()
			}()
		// is the listener still alive?
		case <-time.After(30 * time.Second):
			pingErrChan := make(chan error)
			go func() {
				err = permListener.Ping()
				pingErrChan <- errors.Wrap(err, "no active connection")
			}()
			if err := <-pingErrChan; err != nil {
				log.Errorf("permission listener failed %s", err)
				return err
			}
		// are we canceled?
		case <-ctx.Done():
			return nil
		}
	}
}

// publishLoop monitors for new events and broadcasts them to Publishers.
func publishLoop[T stream.Msg](
	ctx context.Context,
	pgAddress,
	channelName string,
	publisher *stream.Publisher[T],
) error {
	err := doPublishLoop(ctx, pgAddress, channelName, publisher)
	if err != nil {
		log.Errorf("publishLoop failed: %s", err)
		publisher.CloseAllStreamers()
		return err
	}
	return nil
}

func doPublishLoop[T stream.Msg](
	ctx context.Context,
	pgAddress,
	channelName string,
	publisher *stream.Publisher[T],
) error {
	listener, err := newDBListener(pgAddress, channelName)
	if err != nil {
		return errors.Wrapf(err, "failed to listen: %v", channelName)
	}
	// clean up listener
	defer func() {
		err := listener.Close()
		if err != nil {
			log.Debugf("error while cleaning up %s event listener: %s", channelName, err)
		}
	}()

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

func newPermFilter[T stream.Msg](
	ctx context.Context,
	user model.User,
	permFilterFn func(context.Context, model.User) (func(T) bool, error),
	err *error,
) func(T) bool {
	if *err != nil {
		return nil
	}
	out, tempErr := permFilterFn(ctx, user)
	if tempErr != nil {
		*err = tempErr
		return nil
	}
	return out
}

func newFilter[S any, T stream.Msg](
	spec S,
	filterFn func(S) (func(T) bool, error),
	err *error,
) func(T) bool {
	if *err != nil {
		return nil
	}
	out, tempErr := filterFn(spec)
	if tempErr != nil {
		*err = tempErr
		return nil
	}
	return out
}

// NewSubscriptionSet constructor for SubscriptionSet.
func NewSubscriptionSet(
	ctx context.Context,
	streamer *stream.Streamer,
	ps *PublisherSet,
	user model.User,
	spec SubscriptionSpecSet,
) (SubscriptionSet, error) {
	var err error
	return SubscriptionSet{
		Trials: &subscriptionState[*TrialMsg, TrialSubscriptionSpec]{
			stream.NewSubscription(
				streamer,
				ps.Trials,
				newPermFilter(ctx, user, TrialMakePermissionFilter, &err),
				newFilter(spec.Trials, TrialMakeFilter, &err),
			),
			TrialCollectStartupMsgs,
		},
		Metrics: &subscriptionState[*MetricMsg, MetricSubscriptionSpec]{
			stream.NewSubscription(
				streamer,
				ps.Metrics,
				newPermFilter(ctx, user, MetricMakePermissionFilter, &err),
				newFilter(spec.Metrics, MetricMakeFilter, &err),
			),
			MetricCollectStartupMsgs,
		},
	}, err
}

func startup[T stream.Msg, S any](
	ctx context.Context,
	user model.User,
	msgs *[]interface{},
	err error,
	state *subscriptionState[T, S],
	known string,
	spec *S,
	prepare func(message stream.PreparableMessage) interface{},
) error {
	if err != nil {
		return err
	}
	if spec == nil {
		// no change
		return nil
	}
	// Sync subscription with publishers.  Do this before initial scan so that we don't
	// miss any events.
	state.Subscription.Register()

	// Scan for historical msgs matching newly-added subscriptions.
	newmsgs, err := state.CollectStartupMsgs(ctx, user, known, *spec)
	if err != nil {
		return echo.ErrCookieNotFound
	}
	for _, msg := range newmsgs {
		*msgs = append(*msgs, prepare(msg))
	}
	return nil
}

// Startup handles starting up the Subscription objects in the SubscriptionSet.
func (ss *SubscriptionSet) Startup(ctx context.Context, user model.User, startupMsg StartupMsg) (
	[]interface{}, error,
) {
	known := startupMsg.Known
	sub := startupMsg.Subscribe

	var msgs []interface{}
	var err error
	err = startup(
		ctx, user, &msgs, err,
		ss.Trials, known.Trials,
		sub.Trials, ss.Trials.Subscription.Streamer.PrepareFn,
	)
	err = startup(
		ctx, user, &msgs, err,
		ss.Metrics, known.Metrics,
		sub.Metrics, ss.Metrics.Subscription.Streamer.PrepareFn,
	)
	return msgs, err
}

// UnsubscribeAll unsubscribes all Subscription's in the SubscriptionSet.
func (ss *SubscriptionSet) UnsubscribeAll() {
	if ss.Trials != nil {
		ss.Trials.Subscription.Unsubscribe()
	}
	if ss.Metrics != nil {
		ss.Metrics.Subscription.Unsubscribe()
	}
	// ss.Experiments.Subscription.Unsubscribe()
}
