package webhooks

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	maxConcurrentSenders = 10
	maxAttempts          = 5
	attemptBackoff       = 5 * time.Second
)

var theOneSender = sender{
	wakeC: make(chan struct{}, 1),
}

func Init(ctx context.Context) {
	theOneSender.g.SetLimit(maxConcurrentSenders)
	go theOneSender.run(ctx)
}

type sender struct {
	logger *log.Entry

	g     errgroup.Group
	wakeC chan struct{}
}

func (s *sender) run(parent context.Context) error {
	for {
		es, err := getEvents(parent)
		if err != nil {
			s.logger.WithError(err).Error("failed to get events")
			continue
		}

		eg, ctx := errgroup.WithContext(parent)
		eg.SetLimit(maxConcurrentSenders)
		for _, e := range es {
			eg.Go(func() error {
				switch err := s.send(ctx, e); {
				case errors.Is(err, context.Canceled):

				case err != nil && e.Attempts >= maxAttempts:
					s.logger.WithError(err).Error("webhook unsuccessful after %d tries", maxAttempts)
					// delete webhook
				case err != nil:
					s.logger.WithError(err).Error("webhook unsuccessful after %d tries", e.Attempts)
				default:
					// delete webhook
				}
			})
		}

		select {
		case <-s.wakeC:
		case <-parent.Done():
			return parent.Err()
		}
	}
	return nil
}

func (s *sender) send(ctx context.Context, e Event) error {
	// implement me
}

// wake attempts to wake the sender.
func (s *sender) wake() {
	select {
	case s.wakeC <- struct{}{}:
	default:
		// If the channel is full, we will forgo sending since we will already wake.
		// To reason about the correctness of all this, we order happens before relations (~>) and
		// synchronization points (<~>):
		// For buffered channels, write ~> read. For unbuffered, write <~> read. Assumed buffered.
		//
		// IF we have one write A:
		//  write A ~> write wake A,
		//  write wake A ~> read wake A, and
		//  read wake A ~> read A, clearly
		// THEN
		//  write A ~> write wake A ~> read wake A ~> read A
		//
		// So, A is not missed by the send
		//
		// IF we have two writes and write A ~> write B:
		//   write A to STORAGE ~> write wake A
		//   write B to STORAGE ~> give up write wake B
		//   give up write wake B ~> read wake A (other wise we would've written)
		//   write wake A ~> read wake A
		//   read wake A ~> read STORAGE
		// THEN
		//   write A to STORAGE ~> write wake A ~> read wake A ~> read STORAGE, which has A
		//   write B to STORAGE ~> give up write wake B ~> read wake A ~> read STORAGE, which has B
		//
		// So, A and B are both not missed by the send and all cases are the same as these w.l.o.g.
	}
}
