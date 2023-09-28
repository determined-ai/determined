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
	cond *sync.Cond
	// these things change each restart of the PublisherSet
	subctx context.Context
	ps     *PublisherSet
}

// NewSupervisor creates a new Supervisor.
func NewSupervisor() *Supervisor {
	var lock sync.Mutex
	return &Supervisor{cond: sync.NewCond(&lock)}
}

func (ssup *Supervisor) runOne(ctx context.Context) error {
	group := errgroupx.WithContext(ctx)
	subctx := group.Context()
	func() {
		ssup.cond.L.Lock()
		defer ssup.cond.L.Unlock()
		ssup.cond.Broadcast()

		ssup.subctx = subctx
		ssup.ps = NewPublisherSet()

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
		time.Sleep(time.Second)
	}
}

// Websocket is the method that we pass to the Echo server, because rb doesn't know how to update
// the backing function for an Echo.GET() call.  So we need something that will live as long as
// the master.
func (ssup *Supervisor) Websocket(socket *websocket.Conn, c echo.Context) error {
	var ssupCtx context.Context
	var ps *PublisherSet
	func() {
		ssup.cond.L.Lock()
		defer ssup.cond.L.Unlock()
		for ssup.ps == nil {
			ssup.cond.Wait()
			break
		}
		ssupCtx = ssup.subctx
		ps = ssup.ps
	}()
	return ps.Websocket(ssupCtx, socket, c)
}
