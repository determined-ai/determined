package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	back "github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-cleanhttp"
	log "github.com/sirupsen/logrus"

	conf "github.com/determined-ai/determined/master/internal/config"
)

const (
	maxWorkers        = 3
	maxEventBatchSize = 10

	backoffAttempts = 2
	backoffInterval = time.Second
	backoffMax      = time.Minute
)

var singletonShipper *shipper

// Init creates a shipper singleton.
func Init() {
	singletonShipper = newShipper()
}

// Deinit closes a shipper.
func Deinit() {
	singletonShipper.Close()
}

type shipper struct {
	// System dependencies.
	log *log.Entry

	// Internal state.
	wake   chan<- struct{}
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func newShipper() *shipper {
	ctx, cancel := context.WithCancel(context.Background()) // Shipper-lifetime scoped context.

	wake := make(chan struct{}, 1)
	wake <- struct{}{} // Always attempt to process existing events.
	s := &shipper{
		log:    log.WithField("component", "webhook-sender"),
		wake:   wake,
		cancel: cancel,
	}

	for i := 0; i < maxWorkers; i++ {
		s.log.Debugf("creating webhook worker: %d", i)
		w := newWorker(i)
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

func newWorker(id int) *worker {
	return &worker{
		log: log.WithFields(log.Fields{"component": "webhook-shipper-worker", "id": id}),
		cl:  cleanhttp.DefaultClient(),
	}
}

type worker struct {
	// System dependencies.
	log *log.Entry
	cl  *http.Client
}

func (w *worker) work(ctx context.Context, wake <-chan struct{}) {
	defer func() {
		if rec := recover(); rec != nil {
			w.log.Errorf("uncaught error, webhook worker crashed: %v", rec)
		}
	}()

	for {
		select {
		case <-wake:
			if err := w.ship(ctx); err != nil {
				w.log.WithError(err).Error("failed to ship batch")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *worker) ship(ctx context.Context) error {
loop:
	for {
		switch n, err := w.shipBatch(ctx); {
		case err != nil:
			return err
		case n <= 0:
			return nil
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
	defer func() {
		if err := b.rollback(); err != nil {
			w.log.WithError(err).Warn("failed to finalize batch")
		}
	}()

	var wg sync.WaitGroup
	for _, e := range b.events {
		wg.Add(1)
		go func(e Event) {
			defer wg.Done()
			if err := back.Retry(
				func() error { return w.deliver(ctx, e) },
				backoff(),
			); err != nil {
				w.log.WithError(err).Error("failed to deliver webhook")
			}
		}(e)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if err := b.commit(); err != nil {
		return 0, fmt.Errorf("consuming batch: %w", err)
	}
	return len(b.events), nil
}

func backoff() back.BackOff {
	bf := back.NewExponentialBackOff()
	bf.InitialInterval = backoffInterval
	bf.MaxInterval = backoffMax
	return back.WithMaxRetries(bf, backoffAttempts)
}

func (w *worker) deliver(ctx context.Context, e Event) error {
	req, err := generateWebhookRequest(ctx, e.URL, e.Payload, time.Now().Unix())
	if err != nil {
		return err
	}

	resp, err := w.cl.Do(req)
	if err != nil {
		return fmt.Errorf("sending webhook request: %w", err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			w.log.WithError(err).Warn("failed to close response body")
		}
	}()

	switch {
	case resp.StatusCode >= 500:
		return fmt.Errorf("request returned %v: %w", resp.StatusCode, err)
	case resp.StatusCode >= 400:
		return back.Permanent(fmt.Errorf("request returned %v: %w", resp.StatusCode, err))
	default:
		return nil
	}
}

func generateWebhookRequest(
	ctx context.Context,
	url string,
	payload []byte,
	t int64,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed creating webhook request: %w", err)
	}
	key := []byte(conf.GetMasterConfig().Webhooks.SigningKey)
	signedPayload := generateSignedPayload(req, t, key)
	req.Header.Add("X-Determined-AI-Signature-Timestamp", fmt.Sprintf("%v", t))
	req.Header.Add("X-Determined-AI-Signature", signedPayload)
	req.Header.Add("Content-Type", "application/json; charset=UTF-8")
	return req, nil
}

func generateSignedPayload(req *http.Request, t int64, key []byte) string {
	body := req.GetBody
	bodyCopy, _ := body()
	buf, _ := io.ReadAll(bodyCopy)
	message := []byte(fmt.Sprintf(`%v,%s`, t, buf))
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}
