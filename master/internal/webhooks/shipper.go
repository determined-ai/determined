package webhooks

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	numWorkers     = 4
	maxBatchSize   = 10
	maxAttempts    = 2
	attemptBackoff = 2 * time.Second
)

type shipper struct {
	log *logrus.Entry
	cl  *http.Client

	wake   chan<- struct{}
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func newShipper(ctx context.Context) *shipper {
	ctx, cancel := context.WithCancel(ctx)
	wake := make(chan struct{}, 1)
	wake <- struct{}{}
	s := &shipper{
		log:    logrus.WithField("componenet", "webhook-shipper"),
		cl:     &http.Client{},
		wake:   wake,
		cancel: cancel,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.process(ctx, wake)
	}()

	return s
}

func (s *shipper) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
		// If the channel is full, then we will wake up after this instant, and this is after
		// we persisted the events. Note, processors must empty the queue for this to be correct.
	}
}

func (s *shipper) Close() {
	s.cancel()
	s.wg.Wait()
}

func (s *shipper) process(ctx context.Context, wake <-chan struct{}) {
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
	for {
		switch n, err := s.shipBatch(ctx); {
		case err != nil:
			s.log.WithError(err).Error("shipping batch")
			return
		case n == 0:
			return
		}
	}
}

func (s *shipper) shipBatch(ctx context.Context) (int, error) {
	b, err := dequeueEvents(ctx, maxBatchSize)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := b.close(); err != nil {
			s.log.WithError(err).Warn("finalizing batch")
		}
	}()

	for _, e := range b.events {
		s.deliverWithRetries(ctx, e)
	}

	if err := b.consume(ctx); err != nil {
		return 0, err
	}
	return len(b.events), nil
}

func (s *shipper) deliverWithRetries(ctx context.Context, e Event) {
	for i := 0; i < maxAttempts; i++ {
		if err := s.deliver(ctx, e); err != nil {
			s.log.WithError(err).Warn("delivering %v on try %d/%d", e.ID, i, maxAttempts)
			time.Sleep(attemptBackoff)
		} else {
			return
		}
	}
	s.log.Errorf("exhausted tries to deliver %v", e.ID)
}

func (s *shipper) deliver(ctx context.Context, e Event) error {
	return nil
}
