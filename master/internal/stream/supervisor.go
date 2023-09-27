package stream

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
)

const (
	minInterval = time.Second
	maxInterval = time.Minute
	maxRetries  = 5
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

// monitorPermissionChanges listens for permission changes, updates the PublisherSet
// to signal to boot streamers, returns an error in the event of a failure to listen.
func monitorPermissionChanges(ctx context.Context, ps *PublisherSet) error {
	permListener, err := AuthZProvider.Get().GetPermissionChangeListener()
	if err != nil {
		log.Errorf("unable to get permission change listener: %s", err)
		// XXX: investigate better recovery mechanism
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
				ps.permissionLock.Lock()
				defer ps.permissionLock.Unlock()
				close(ps.permissionChangeChan)
				ps.permissionChangeChan = make(chan struct{})
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
				return monitorPermissionChanges(c, ssup.ps)
			},
		)
		// start up all publishers
		// XXX: should these be passing errors backup?
		group.Go(ssup.ps.Start)
		// group.Go(PublishLoop(ctx, "experiment_trial_chan", ps.Experiments))
	}()
	return group.Wait()
}

// Run attempts to start up the publisher system and recovers in the event of a failure.
func (ssup *Supervisor) Run(ctx context.Context) error {
	backoffSettings := backoff.NewExponentialBackOff()
	backoffSettings.InitialInterval = minInterval
	backoffSettings.MaxInterval = maxInterval

	run := func() error {
		err := ssup.runOne(ctx)
		if err != nil {
			log.Errorf("restarting publisher system after failure: %s", err)
		}
		return err
	}

	err := backoff.Retry(run, backoff.WithContext(backoffSettings, ctx))

	if err != nil && err != backoff.Permanent(err) {
		return errors.Wrap(err, "maximum number of retries reached")
	}
	return err
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
