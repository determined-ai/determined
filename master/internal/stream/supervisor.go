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
	// and all active webscockets are disconnected.
	// nolint
	publisherSetCtx context.Context
	// currently active publisher set
	ps *PublisherSet
}

// NewSupervisor creates a new Supervisor.
func NewSupervisor(dbAddress string) *Supervisor {
	// initialize with a valid publisher set and canceled supervisor context,
	// so connections prior to runOne() can at least send startup messages.
	ctx, cancelFn := context.WithCancel(context.Background())
	// this context is immediately canceled to enable "limb" mode, allowing connections that come in
	// prior to the publisher set being fully started to receive offline events.
	cancelFn()
	return &Supervisor{
		dbAddress:       dbAddress,
		ps:              NewPublisherSet(dbAddress),
		publisherSetCtx: ctx,
	}
}

func (ssup *Supervisor) runOne(ctx context.Context) error {
	group := errgroupx.WithContext(ctx)
	subctx := group.Context()
	func() {
		ssup.lock.Lock()
		defer ssup.lock.Unlock()
		ssup.publisherSetCtx = subctx
		ssup.ps = NewPublisherSet(ssup.dbAddress)

		// start monitoring permissions
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

// Websocket is the method that we pass to the Echo server, because rb doesn't know how to update
// the backing function for an Echo.GET() call.  So we need something that will live as long as
// the master.
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
	return ps.Websocket(publisherSetCtx, socket, c)
}
