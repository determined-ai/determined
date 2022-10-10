package webhooks

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const (
	maxWorkers        = 3
	maxEventBatchSize = 10
	maxPendingEvents  = 100
	maxAttempts       = 2
	attemptBackoff    = 2 * time.Second
)

type shipper struct {
	// System dependencies.
	logger *log.Entry
	cl     *http.Client

	// Internal state.
	wake   chan<- struct{}
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func newShipper(ctx context.Context) *shipper {
	ctx, cancel := context.WithCancel(ctx)
	wake := make(chan struct{}, 1)
	wake <- struct{}{} // Always attempt to process existing events.
	s := &shipper{
		logger: log.WithField("component", "webhook-sender"),
		cl:     &http.Client{},

		cancel: cancel,
		wake:   wake,
	}

	for i := 0; i < maxWorkers; i++ {
		s.logger.Infof("creating webhook worker: %d", i)
		w := worker{log: s.logger.WithField("worker-id", i), cl: &http.Client{}}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			w.work(ctx, wake)
		}()
	}

	return s
}

// Wake attempts to wake the sender.
func (s *shipper) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
		// If the channel is full, we will forgo sending. We will already wake, and that will
		// be after we persisted the event that caused this wake.
		// The only critical correctness condition to be aware of is: if you wake, you must consume
		// all events. If you stop short, they may be delay until the next wake.
	}
}

func (s *shipper) Close() {
	s.cancel()
	s.wg.Wait()
}

type worker struct {
	// System dependencies.
	log *logrus.Entry
	cl  *http.Client
}

func (w *worker) work(ctx context.Context, wake <-chan struct{}) {
	for {
		select {
		case <-wake:
			w.ship(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (w *worker) ship(ctx context.Context) {
loop:
	for {
		switch n, err := w.shipBatch(ctx); {
		case err != nil:
			w.log.WithError(err).Warn("error shipping webhook events")
			return
		case n <= 0:
			return
		default:
			// Continue until events are exhausted.
			continue loop
		}
	}
}

func (w *worker) shipBatch(ctx context.Context) (int, error) {
	b, err := dequeueEvents(ctx, maxEventBatchSize)
	if err != nil {
		return 0, fmt.Errorf("getting events: %w", err)
	}
	ids := []int{}
	for _, ev := range b.events {
		ids = append(ids, int(ev.ID))
	}
	w.log.Infof("dequeued events: %v", ids)
	defer func() {
		if err := b.close(); err != nil {
			w.log.WithError(err).Error("failed to finalize batch")
		}
	}()

	for _, e := range b.events {
		w.deliverWithRetries(ctx, e)
	}

	if err := b.consume(); err != nil {
		return 0, fmt.Errorf("consuming batch %v: %v", ids, err)
	}
	return len(b.events), nil
}

func (w *worker) deliverWithRetries(ctx context.Context, e Event) {
	for i := 0; i < maxAttempts; i++ {
		if err := w.deliver(ctx, e); err != nil {
			w.log.WithError(err).Warnf("couldn't deliver %v (%d/%d)", e.ID, i, maxAttempts)
			time.Sleep(attemptBackoff)
			continue
		}
		return
	}
	w.log.Errorf("exhausted tries to deliver %v, giving up", e.ID)
}

var url string // TODO(Brad): Hack.

func (w *worker) deliver(ctx context.Context, e Event) error {
	resp, err := w.cl.Post(
		url,
		"application/json; charset=UTF-8",
		bytes.NewBuffer(e.Payload),
	)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
