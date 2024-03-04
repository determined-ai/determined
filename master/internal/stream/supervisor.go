package stream

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"

	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
)

// Supervisor manages the context for underlying PublisherSet.
type Supervisor struct {
	cond      sync.Cond
	dbAddress string

	// this context is responsible for monitoring the life time of the
	// the active PublisherSet, in the event that a Publisher fails, this context is canceled
	// and all active websockets are disconnected.
	// nolint
	publisherSetCtx context.Context
	// currently active PublisherSet
	ps *PublisherSet
}

// NewSupervisor creates a new Supervisor.
func NewSupervisor(dbAddress string) *Supervisor {
	return &Supervisor{
		cond:      *sync.NewCond(&sync.Mutex{}),
		dbAddress: dbAddress,
	}
}

// runOne attempts to start up an instance of the publishing system.
func (ssup *Supervisor) runOne(ctx context.Context) error {
	group := errgroupx.WithContext(ctx)
	subctx := group.Context()
	func() {
		ssup.cond.L.Lock()
		defer ssup.cond.L.Unlock()
		// Broadcast in case this is the first PublisherSet for this supervisor.
		defer ssup.cond.Broadcast()
		ssup.publisherSetCtx = subctx
		ssup.ps = NewPublisherSet(ssup.dbAddress)

		// start monitoring permission scope changes
		group.Go(
			func(c context.Context) error {
				return BootemLoop(c, ssup.ps)
			},
		)
		// start up all publishers
		group.Go(ssup.ps.Run)
	}()
	return group.Wait()
}

// Run attempts to start up the publisher system and recovers in the event of a failure.
func (ssup *Supervisor) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			// we got canceled.
			return nil
		default:
		}
		err := ssup.runOne(ctx)
		if err != nil {
			log.Errorf("publisher system failed (will restart): %s", err)
		}
		time.Sleep(time.Second * 2)
	}
}

// Websocket is the Echo endpoint handler for streaming updates.  It wraps the business logic in
// doWebsocket with some IO-specific code.
func (ssup *Supervisor) Websocket(socket *websocket.Conn, c echo.Context) error {

	defer func() {
		if err := socket.Close(); err != nil {
			log.Debugf("error while cleaning up socket: %s", err)
		}
	}()

	reqCtx := c.Request().Context()
	detCtx, ok := c.(*detContext.DetContext)
	if !ok {
		log.Errorf("unable to run PublisherSet: expected DetContext but received %T", c)
	}
	user := detCtx.MustGetUser()

	return ssup.doWebsocket(
		reqCtx,
		user,
		&WrappedWebsocket{Conn: socket},
		prepareWebsocketMessage,
	)
}

// doWebsocket is a mockable entrypoint for the streaming updates system.  Websocket requests are
// attached to the Supervisor instead of the PublisherSet because the Supervior is meant to be
// always alive, while the PublisherSet may crash and get restarted from time to time.
//
// doWebsocket will grab the current PublisherSet and point this connection at it.
func (ssup *Supervisor) doWebsocket (
	reqCtx context.Context,
	user model.User,
	socket WebsocketLike,
	prepareFunc func(message stream.MarshallableMsg) interface{},
) error {
	// Grab the current publisher set and its context.
	var publisherSetCtx context.Context
	var ps *PublisherSet
	func() {
		ssup.cond.L.Lock()
		defer ssup.cond.L.Unlock()
		for ssup.ps == nil {
			// Wait for the first publisher set to start.
			ssup.cond.Wait()
		}
		publisherSetCtx = ssup.publisherSetCtx
		ps = ssup.ps
	}()

	return ps.streamHandler(
		publisherSetCtx,
		reqCtx,
		user,
		socket,
		prepareFunc,
	)
}
