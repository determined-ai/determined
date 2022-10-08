package webhooks

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

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
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.worker(ctx, wake)
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

func (s *shipper) worker(ctx context.Context, wake <-chan struct{}) {
	for {
		select {
		case <-wake:
			s.ship(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *shipper) ship(ctx context.Context) {
loop:
	for {
		switch n, err := s.shipBatch(ctx); {
		case err != nil:
			s.logger.WithError(err).Warn("error shipping webhook events")
			return
		case n <= 0:
			return
		default:
			// Continue until events are exhausted.
			continue loop
		}
	}
}

func (s *shipper) shipBatch(ctx context.Context) (int, error) {
	b, err := dequeueEvents(ctx, maxEventBatchSize)
	if err != nil {
		return 0, fmt.Errorf("getting events: %w", err)
	}
	defer func() {
		if err := b.close(); err != nil {
			s.logger.WithError(err).Error("failed to finalize batch")
		}
	}()

	for _, e := range b.events {
		s.deliverWithRetries(ctx, e)
	}

	if err := b.consume(); err != nil {
		return 0, fmt.Errorf("consuming batch: %v", err)
	}
	return len(b.events), nil
}

func (s *shipper) deliverWithRetries(ctx context.Context, e Event) {
	for i := 0; i < maxAttempts; i++ {
		if err := s.deliver(ctx, e); err != nil {
			s.logger.WithError(err).Warnf("couldn't deliver %v (%d/%d)", e.ID, i, maxAttempts)
			time.Sleep(attemptBackoff)
			continue
		}
		return
	}
	s.logger.Errorf("exhausted tries to deliver %v, giving up", e.ID)
}

var url string

func (s *shipper) deliver(ctx context.Context, e Event) error {
	resp, err := s.cl.Post(
		url,
		"application/json; charset=UTF-8",
		bytes.NewBuffer(e.Payload),
	)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}
