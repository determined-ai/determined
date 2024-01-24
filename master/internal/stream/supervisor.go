package stream

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
)

// Supervisor manages the context for underlying PublisherSet.
type Supervisor struct {
	lock      sync.Mutex
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
	// initialize with a valid PublisherSet and canceled supervisor context,
	// so connections prior to runOne() can at least send messages collected
	// during startup.
	ctx, cancelFn := context.WithCancel(context.Background())
	cancelFn()

	return &Supervisor{
		dbAddress:       dbAddress,
		ps:              NewPublisherSet(dbAddress),
		publisherSetCtx: ctx,
	}
}

// runOne attempts to start up an instance of the publishing system.
func (ssup *Supervisor) runOne(ctx context.Context) error {
	group := errgroupx.WithContext(ctx)
	subctx := group.Context()
	func() {
		ssup.lock.Lock()
		defer ssup.lock.Unlock()
		ssup.publisherSetCtx = subctx
		ssup.ps = NewPublisherSet(ssup.dbAddress)

		// start monitoring permission scope changes
		group.Go(
			func(c context.Context) error {
				return BootemLoop(c, ssup.ps)
			},
		)
		// start up all publishers
		group.Go(ssup.ps.Start)
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

// Websocket passes incoming stream request to the active PublisherSet's websocket handler,
// ensuring that in the event of a PublisherSet failure, stream requests can still routed
// during recovery.
func (ssup *Supervisor) Websocket(socket *websocket.Conn, c echo.Context) error {
	var publisherSetCtx context.Context
	var ps *PublisherSet
	defer func() {
		if err := socket.Close(); err != nil {
			log.Debugf("error while cleaning up socket: %s", err)
		}
	}()
	func() {
		ssup.lock.Lock()
		defer ssup.lock.Unlock()
		publisherSetCtx = ssup.publisherSetCtx
		ps = ssup.ps
	}()
	return ps.StreamHandler(publisherSetCtx, socket, c)
}
