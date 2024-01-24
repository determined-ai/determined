package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"

	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
)

// JSONB is the golang equivalent of the postgres jsonb column type.
type JSONB interface{}

const (
	minReconn   = 1 * time.Second
	maxReconn   = 10 * time.Second
	projectChan = "stream_project_chan"
)

// PublisherSet contains all publishers, and handles all websockets.  It will connect each websocket
// with the appropriate set of publishers, based on that websocket's subscriptions.
//
// There is one PublisherSet for the whole process.  It has one Publisher per streamable type.
type PublisherSet struct {
	DBAddress  string
	Projects   *stream.Publisher[*ProjectMsg]
	bootemChan chan struct{}
	bootLock   sync.Mutex
	readyCond  sync.Cond
	ready      bool
}

// SubscriptionSet is a set of all subscribers for this PublisherSet.
//
// There is one SubscriptionSet for each websocket connection.  It has one SubscriptionManager per
// streamable type.
type SubscriptionSet struct {
	Projects *subscriptionState[*ProjectMsg, ProjectSubscriptionSpec]
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

func newDBListener(dbAddress, channel string) (*pq.Listener, error) {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Errorf("listener on (%s) reported problem: %s", dbAddress, err.Error())
		}
	}
	listener := pq.NewListener(
		dbAddress,
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
func NewPublisherSet(dbAddress string) *PublisherSet {
	lock := sync.Mutex{}
	return &PublisherSet{
		DBAddress:  dbAddress,
		Projects:   stream.NewPublisher[*ProjectMsg](),
		bootemChan: make(chan struct{}),
		readyCond:  *sync.NewCond(&lock),
	}
}

// SyncMsg is the server response to a StartupMsg once it's been handled.
type SyncMsg struct {
	SyncID string `json:"sync_id"`
}

// MarshalJSON converts a sync message into a json representation, implements PreparableMsg.
func (sm SyncMsg) MarshalJSON() ([]byte, error) {
	return json.Marshal(sm)
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
	Trials      string `json:"trials"`
	Metrics     string `json:"metrics"`
	Experiments string `json:"experiments"`
	Checkpoints string `json:"checkpoints"`
	Projects    string `json:"projects"`
	Workspaces  string `json:"workspaces"`
}

// SubscriptionSpecSet is the set of subscription specs that can be sent in startup message.
type SubscriptionSpecSet struct {
	Projects *ProjectSubscriptionSpec `json:"projects"`
}

func start[T stream.Msg](
	ctx context.Context,
	dbAddress,
	channel string,
	publisher *stream.Publisher[T],
	readyChan chan bool,
) error {
	return publishLoop(ctx, dbAddress, channel, publisher, readyChan)
}

// Start starts each Publisher in the PublisherSet.
func (ps *PublisherSet) Start(ctx context.Context) error {
	readyChannels := map[interface{}]chan bool{
		ps.Projects: make(chan bool),
	}

	eg := errgroupx.WithContext(ctx)
	eg.Go(
		func(c context.Context) error {
			return start(c, ps.DBAddress, projectChan, ps.Projects, readyChannels[ps.Projects])
		},
	)

	// wait for all publishers to become ready
	eg.Go(
		func(c context.Context) error {
			for i := range readyChannels {
				select {
				case <-c.Done():
					return nil
				case <-readyChannels[i]:
					continue
				}
			}
			func() {
				ps.readyCond.L.Lock()
				defer ps.readyCond.L.Unlock()
				ps.ready = true
				ps.readyCond.Signal()
			}()
			return nil
		},
	)
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

// processStream processes as startup message, then streams live updates until either:
// - another startup message arrives, in which case it returns it or
// - the streamer is closed gracefully, in which case it returns nil, or
// - an error occurs.
func (ps *PublisherSet) processStream(
	ctx context.Context,
	streamer *stream.Streamer,
	user model.User,
	startupMsg StartupMsg,
	startups *[]StartupMsg,
	prepare func(message stream.PreparableMessage) interface{},
	socket WebsocketLike,
) (
	nextStartup *StartupMsg, err error,
) {
	// create new subscription set
	ss, err := NewSubscriptionSet(ctx, streamer, ps, user, startupMsg.Subscribe)
	if err != nil {
		return nil, fmt.Errorf("creating subscription set: %s", err.Error())
	}
	defer ss.UnregisterAll()

	// startup subscription set
	msgs := []interface{}{}
	offlineMsgs, err := ss.Startup(ctx, user, startupMsg)
	if err != nil {
		return nil, fmt.Errorf("gathering startup messages: %s", err.Error())
	}
	msgs = append(msgs, offlineMsgs...)

	// always include a sync message
	syncMsg := SyncMsg{SyncID: startupMsg.SyncID}
	msgs = append(msgs, prepare(syncMsg))

	// write offline msgs to the websocket
	err = writeAll(socket, msgs)
	if err != nil {
		// don't log broken pipe errors
		if !websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
			log.Errorf("unable to handle startup message: %s", err.Error())
		}
		return nil, fmt.Errorf("error handling startup message: %s", err.Error())
	}

	// startup done, begin streaming supported online events:
	//	- insertions
	//	- updates
	//	- deletions
	//	- fallin
	//	- fallout
	// note: online appearances and disappearances are not supported; we'll detect those
	// situtations and break the connection to the streaming clients

	// waitForSomething Returns a tuple of (first-startup-msg, msgs, closed)
	waitForSomething := func() (*StartupMsg, []interface{}, bool) {
		streamer.Cond.L.Lock()
		defer streamer.Cond.L.Unlock()

		for len(*startups) == 0 && len(streamer.Msgs) == 0 && !streamer.Closed {
			streamer.Cond.Wait()
		}
		// steal outputs
		var startup *StartupMsg
		if len(*startups) > 0 {
			startup = &(*startups)[0]
			*startups = (*startups)[1:]
		}
		msgs := streamer.Msgs
		streamer.Msgs = nil
		return startup, msgs, streamer.Closed
	}
	for {
		startup, msgs, closed := waitForSomething()

		// is the streamer closed?
		if closed {
			return nil, nil
		}

		// soft reset?
		if startup != nil {
			return startup, nil
		}

		// otherwise write all the messages we just got
		err = writeAll(socket, msgs)
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				log.Errorf("unable to handle startup message: %s", err.Error())
			}
			return nil, fmt.Errorf("error writing to socket: %s", err.Error())
		}
	}
}

// entrypoint manages the streamer websocket connection, processing incoming events
// and monitoring for cancellations.
func (ps *PublisherSet) entrypoint(
	publisherSetCtx context.Context,
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

	// block until the publisher set can receive live event
	func() {
		ps.readyCond.L.Lock()
		defer ps.readyCond.L.Unlock()
		for !ps.ready {
			ps.readyCond.Wait()
		}
	}()

	streamer := stream.NewStreamer(prepareFunc)
	// read first startup message
	var startupMsg StartupMsg
	err := socket.ReadJSON(&startupMsg)
	if err != nil {
		return fmt.Errorf("error while reading initial startup message: %s", err.Error())
	}

	// startups is where we collect StartupMsg
	// waitForSomething() in processStream delivers those messages to the websocket goroutine.
	var startups []StartupMsg

	go func() {
		defer streamer.Close()
		for {
			var mods StartupMsg
			err := socket.ReadJSON(&mods)
			if err != nil {
				if websocket.IsUnexpectedCloseError(
					err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
				) {
					log.Errorf("unexpected close error: %s", err.Error())
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

	// detect context cancelation, and bring it into the websocket thread
	go func() {
		select {
		// did the streamer crash?
		case <-ctx.Done():
			log.Tracef("context canceled, closing streamer: %v", streamer)
			streamer.Close()
		// did a publisher crash?
		case <-publisherSetCtx.Done():
			// close streamer if publisher set is down, prepping for restart
			log.Tracef("a publisher crashed, closing streamer: %v", streamer)
			streamer.Close()
		// did permissions change?
		case <-bootemChan:
			// close this streamer if online appearance/disappearance occurred
			log.Tracef("permission scope detected, closing streamer: %v", streamer)
			streamer.Close()
		}
	}()

	for {
		nextStartupMsg, err := ps.processStream(
			ctx,
			streamer,
			user,
			startupMsg,
			&startups,
			prepareFunc,
			socket,
		)
		if err != nil {
			// stream failed
			return err
		}
		if nextStartupMsg == nil {
			// stream closed
			return nil
		}
		// stream soft reset: the last subscription ended due to a user sending another StartupMsg
		startupMsg = *nextStartupMsg
	}
}

// Websocket is the Echo websocket endpoint handler for streaming updates,
// defaulting the prepare function to prepareWebsocketMessage().
func (ps *PublisherSet) Websocket(
	publisherSetCtx context.Context,
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
	return ps.entrypoint(
		publisherSetCtx,
		reqCtx,
		user,
		&WrappedWebsocket{Conn: socket},
		prepareWebsocketMessage,
	)
}

// prepareWebsocketMessage converts the PreparableMessage into a websocket.PreparedMessage.
func prepareWebsocketMessage(obj stream.PreparableMessage) interface{} {
	jbytes, err := json.Marshal(obj)
	if err != nil {
		log.Errorf("error marshaling message for streaming: %s", err.Error())
		return nil
	}
	msg, err := websocket.NewPreparedMessage(websocket.TextMessage, jbytes)
	if err != nil {
		log.Errorf("error preparing message for streaming: %s", err.Error())
		return nil
	}
	return msg
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
		log.Errorf("unable to get permission change listener: %s", err.Error())
		return err
	}
	if permListener == nil {
		// no listener means we don't have permissions configured at all
		return nil
	}
	defer func() {
		err := permListener.Close()
		if err != nil {
			log.Debugf("error occurred while closing permission listener: %s", err.Error())
		}
	}()

	for {
		select {
		// did permissions change?
		case <-permListener.Notify:
			log.Tracef("permission change detected, booting streamers")
			func() {
				ps.bootStreamers()
			}()
		// is the listener still alive?
		case <-time.After(30 * time.Second):
			pingErrChan := make(chan error)
			go func() {
				pingErrChan <- permListener.Ping()
			}()
			if err := <-pingErrChan; err != nil {
				log.Errorf("permission listener failed, no active connection: %s", err.Error())
				return err
			}
		// are we canceled?
		case <-ctx.Done():
			return nil
		}
	}
}

// publishLoop is a wrapper that closes all active streamers in the event that
// an error occurs while listening for and publishing new events.
func publishLoop[T stream.Msg](
	ctx context.Context,
	dbAddress,
	channelName string,
	publisher *stream.Publisher[T],
	readyChan chan bool,
) error {
	err := doPublishLoop(ctx, dbAddress, channelName, publisher, readyChan)
	if err != nil {
		log.Errorf("publishLoop failed: %s", err.Error())
		publisher.CloseAllStreamers()
		return err
	}
	return nil
}

// doPublishLoop watches the channel for new events
// and broadcasts them to the provided Publisher's streamers.
func doPublishLoop[T stream.Msg](
	ctx context.Context,
	dbAddress,
	channelName string,
	publisher *stream.Publisher[T],
	readyChan chan bool,
) error {
	listener, err := newDBListener(dbAddress, channelName)
	if err != nil {
		return fmt.Errorf("failed to listen to %s: %s", channelName, err.Error())
	}

	// let the PublisherSet know this Publisher is ready
	close(readyChan)

	// clean up listener
	defer func() {
		err := listener.Close()
		if err != nil {
			log.Debugf("error while cleaning up %s event listener: %s", channelName, err.Error())
		}
	}()

	for {
		var events []stream.Event[T]
		select {
		// Are we canceled?
		case <-ctx.Done():
			return nil
		// Is the pq listener still active?
		case <-time.After(30 * time.Second):
			pingErrChan := make(chan error)
			go func() {
				pingErrChan <- listener.Ping()
			}()
			if err := <-pingErrChan; err != nil {
				return fmt.Errorf("no active connection: %s", err.Error())
			}

		// Did we get a notification?
		case notification := <-listener.Notify:
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

// newPermFilter creates a new permission filter based on the provided user and filter function
// for populating the streamer filter field.
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

// newFilter creates a filter function based on the provided SubscriptionSpec
// and the streamable type-specific filter constructor.
func newFilter[S any, T stream.Msg](
	spec S,
	filterConstructor func(S) (func(T) bool, error),
	err *error,
) func(T) bool {
	if *err != nil {
		return nil
	}
	out, tempErr := filterConstructor(spec)
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
	var projectSubscriptionState *subscriptionState[*ProjectMsg, ProjectSubscriptionSpec]

	if spec.Projects != nil {
		projectSubscriptionState = &subscriptionState[*ProjectMsg, ProjectSubscriptionSpec]{
			stream.NewSubscription(
				streamer,
				ps.Projects,
				newPermFilter(ctx, user, ProjectMakePermissionFilter, &err),
				newFilter(spec.Projects, ProjectMakeFilter, &err),
			),
			ProjectCollectStartupMsgs,
		}
	}

	return SubscriptionSet{
		Projects: projectSubscriptionState,
	}, err
}

// startup performs the streamer startup process,
// - registering a new subscription with it's type-specific publisher
// - collecting historical msgs that occurred prior to the streamers connection.
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
	if ss.Projects != nil {
		err = startup(
			ctx, user, &msgs, err,
			ss.Projects, known.Projects,
			sub.Projects, ss.Projects.Subscription.Streamer.PrepareFn,
		)
	}
	return msgs, err
}

// UnregisterAll unregisters all Subscription's in the SubscriptionSet.
func (ss *SubscriptionSet) UnregisterAll() {
	if ss.Projects != nil {
		ss.Projects.Subscription.Unregister()
	}
}
