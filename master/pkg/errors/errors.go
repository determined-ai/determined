package errors

import (
	"sync"
	"time"
)

type ErrorTimeoutRetry struct {
	timeout    time.Duration
	maxRetries int

	mu sync.RWMutex

	err     error
	time    time.Time
	retries int
}

func NewErrorTimeoutRetry(timeout time.Duration, maxRetries int) *ErrorTimeoutRetry {
	return &ErrorTimeoutRetry{
		timeout:    timeout,
		maxRetries: maxRetries,
	}
}

// GetError returns an error after max retries has been met and we are within the timeout duration.
func (e *ErrorTimeoutRetry) GetError() error {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	if time.Now().After(e.time.Add(e.timeout)) {
		e = nil
		return nil
	}
	if e.retries < e.maxRetries {
		return nil
	}
	return e.err
}

// SetError increments or resets the number of retries if the last error was within the timeout
// duration or the current error is nil.
func (e *ErrorTimeoutRetry) SetError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	if err == nil ||
		e.timeout <= 0 ||
		now.After(e.time.Add(e.timeout)) {
		e.retries = 0
	} else {
		e.retries++
	}
	e.err = err
	e.time = now
}
