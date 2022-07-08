package actor

import (
	"encoding/json"
	"sync"
	"time"
)

// Response holds a reference to the future result of an `Ask` of an actor. Responses are not thread
// safe.
type Response interface {
	// Source returns the source of the response.
	Source() *Ref
	// Get returns the result of the `Ask` or nil if the actor did not respond.
	Get() Message
	// GetOrTimeout returns the result of the `Ask` or nil if the actor did not respond. If the
	// timeout is reached, nil is returned and the second result returns false.
	GetOrTimeout(timeout time.Duration) (Message, bool)
	// GetOrElse returns the result of the `Ask` or the provided default if the actor did not
	// respond.
	GetOrElse(defaultValue Message) Message
	// GetOrElseTimeout returns the result of the `Ask` or the provided default if the actor did not
	// respond. If the timeout is reached, the defaultValue is returned and the second result
	// returns false.
	GetOrElseTimeout(defaultValue Message, timeout time.Duration) (Message, bool)
	// Empty returns true if the actor did not respond and false otherwise.
	Empty() (empty bool)
	// Error returns the error if the actor returned an error response and nil otherwise.
	Error() (err error)
	// ErrorOrTimeout returns the error, or nil if there is none, if the actor returned an error
	// response within the deadline. If the deadline is exceeded, the bool returned false.
	ErrorOrTimeout(timeout time.Duration) (bool, error)
}

type response struct {
	lock    sync.Mutex
	source  *Ref
	fetched bool
	future  chan Message
	result  Message
}

func emptyResponse(source *Ref) Response {
	return &response{
		source:  source,
		fetched: true,
		future:  nil,
		result:  errNoResponse,
	}
}

func (r *response) Source() *Ref {
	return r.source
}

func (r *response) get(cancel <-chan bool) Message {
	if r.fetched {
		return r.result
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.fetched = true
	select {
	case r.result = <-r.future:
		return r.result
	case <-cancel:
		return nil
	}
}

func (r *response) Get() Message {
	return r.GetOrElse(nil)
}

func (r *response) GetOrTimeout(timeout time.Duration) (Message, bool) {
	return r.GetOrElseTimeout(nil, timeout)
}

func (r *response) GetOrElse(defaultValue Message) Message {
	if r.Empty() {
		return defaultValue
	}
	return r.get(nil)
}

func (r *response) GetOrElseTimeout(defaultValue Message, timeout time.Duration) (Message, bool) {
	future := make(chan Message, 1)
	cancel := make(chan bool, 1)

	go func() {
		future <- r.get(cancel)
	}()
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case result := <-future:
		return result, true
	case <-t.C:
		cancel <- true
		r.lock.Lock()
		defer r.lock.Unlock()
		r.fetched = true
		r.result = errNoResponse
		return defaultValue, false
	}
}

func (r *response) Empty() bool {
	return r.get(nil) == errNoResponse
}

func (r *response) Error() error {
	err, ok := r.get(nil).(error)
	if r.Empty() || !ok {
		return nil
	}
	return err
}

func (r *response) ErrorOrTimeout(timeout time.Duration) (bool, error) {
	msg, ok := r.GetOrElseTimeout(nil, timeout)
	if !ok {
		return false, nil
	}
	err, ok := msg.(error)
	if r.Empty() || !ok {
		return true, nil
	}
	return true, err
}

func (r *response) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Get())
}
