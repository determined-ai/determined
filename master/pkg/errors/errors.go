package errors

import (
	"sync"
	"time"
)

// StickyError is a helper struct that can be used to retry an error for a given number of
// retries and timeout.
type StickyError struct {
	timeout    time.Duration
	maxRetries int

	mu sync.RWMutex

	err     error
	time    time.Time
	retries int
}

// NewStickyError returns a new ErrorTimeoutRetry.
func NewStickyError(timeout time.Duration, maxRetries int) *StickyError {
	return &StickyError{
		timeout:    timeout,
		maxRetries: maxRetries,
	}
}

func (e *StickyError) isExpired(t time.Time) bool {
	return e.timeout <= 0 || t.After(e.time.Add(e.timeout))
}

// Error returns an error after max retries has been met and we are within the timeout duration.
func (e *StickyError) Error() error {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.error(time.Now())
}

func (e *StickyError) error(t time.Time) error {
	if e.retries < e.maxRetries || e.isExpired(t) {
		return nil
	}
	return e.err
}

// SetError increments or resets the number of retries if the last error was within the timeout
// duration or the current error is nil. Returns the error provided by GetError.
func (e *StickyError) SetError(err error) error {
	if e == nil {
		panic("cannot set error on nil ErrorTimeoutRetry")
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.setError(time.Now(), err)
}

func (e *StickyError) setError(t time.Time, err error) error {
	if err == nil || e.isExpired(t) {
		e.retries = 0
	} else if e.err != nil {
		e.retries++
	}
	e.err = err
	e.time = t
	return e.error(t)
}
