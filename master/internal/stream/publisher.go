package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
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

// Start starts each Publisher in the PublisherSet.
func (ps *PublisherSet) Run(ctx context.Context) error {
	readyChannels := map[interface{}]chan bool{
		ps.Projects: make(chan bool),
	}

	eg := errgroupx.WithContext(ctx)
	eg.Go(
		func(c context.Context) error {
			err := publishLoop(
				c,
				ps.DBAddress,
				projectChannel,
				ps.Projects,
				readyChannels[ps.Projects],
			)
			if err != nil {
				return fmt.Errorf("project publishLoop failed: %s", err.Error())
			}
			return nil
		},
	)

	// Wait for all publishers to become ready.
	eg.Go(
		func(c context.Context) error {
			// Always set ready=true, even if the publisher loop crashes and this goroutine is
			// canceled, because there might be streamers waiting for this PublisherSet to become
			// ready, and we don't want them to hang.
			defer func() {
				ps.readyCond.L.Lock()
				defer ps.readyCond.L.Unlock()
				ps.ready = true
				ps.readyCond.Broadcast()
			}()

			for i := range readyChannels {
				select {
				case <-c.Done():
					return nil
				case <-readyChannels[i]:
					continue
				}
			}
			return nil
		},
	)
	return eg.Wait()
}

// entrypoint manages the streamer websocket connection, processing incoming events
// and monitoring for cancellations.
func (ps *PublisherSet) streamHandler(
	publisherSetCtx context.Context,
	ctx context.Context,
	user model.User,
	socket WebsocketLike,
	prepareFunc func(message stream.MarshallableMsg) interface{},
) error {
	// get permission change channel
	var bootemChan chan struct{}
	func() {
		ps.bootLock.Lock()
		defer ps.bootLock.Unlock()
		bootemChan = ps.bootemChan
	}()

	// block until the PublisherSet can receive live events
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
		defer streamer.Close()
		select {
		// did the streamer crash?
		case <-ctx.Done():
			log.Tracef("context canceled, closing streamer: %v", streamer)
		// did a publisher crash?
		case <-publisherSetCtx.Done():
			// close streamer if PublisherSet is down, prepping for restart
			log.Tracef("a publisher crashed, closing streamer: %v", streamer)
		// did permissions change?
		case <-bootemChan:
			// close this streamer if online appearance/disappearance occurred
			log.Tracef("permission scope detected, closing streamer: %v", streamer)
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
	prepare func(message stream.MarshallableMsg) interface{},
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
	msgs = append(msgs, prepare(stream.SyncMsg{SyncID: startupMsg.SyncID, Complete: false}))
	offlineMsgs, err := ss.Startup(ctx, user, startupMsg)
	if err != nil {
		return nil, fmt.Errorf("gathering startup messages: %s", err.Error())
	}
	msgs = append(msgs, offlineMsgs...)

	// always include a sync message
	syncMsg := stream.SyncMsg{SyncID: startupMsg.SyncID, Complete: true}
	msgs = append(msgs, prepare(syncMsg))

	// write offline msgs to the websocket
	err = WriteAll(socket, msgs)
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
		err = WriteAll(socket, msgs)
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				log.Errorf("unable to handle startup message: %s", err.Error())
			}
			return nil, fmt.Errorf("error writing to socket: %s", err.Error())
		}
	}
}

// publishLoop watches the channel for new events and broadcasts them to the provided Publisher's
// streamers.
func publishLoop[T stream.Msg](
	ctx context.Context,
	dbAddress,
	channelName string,
	publisher *stream.Publisher[T],
	readyChan chan bool,
) error {
	// Always close all active streamers when we shut down, because no subscription can be valid
	// across the transition from an old PublisherSet to a new PublisherSet; there would always be
	// a risk of dropped events between the two.
	defer publisher.CloseAllStreamers()

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

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		var events []stream.Event[T]
		select {
		// Are we canceled?
		case <-ctx.Done():
			return nil
		// Is the pq listener still active?
		case <-ticker.C:
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

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		// did permissions change?
		case <-permListener.Notify:
			log.Tracef("permission change detected, booting streamers")
			func() {
				ps.bootStreamers()
			}()
		// is the listener still alive?
		case <-ticker.C:
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
