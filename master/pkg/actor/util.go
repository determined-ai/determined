package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Responses wraps a collection of response objects from different actors.
type Responses <-chan Response

// MarshalJSON implements the json.Marshaler interface.
func (r Responses) MarshalJSON() ([]byte, error) {
	responses := r.GetAll()
	results := make(map[string]Message, len(responses))
	for source, response := range responses {
		results[source.Address().String()] = response
	}
	return json.Marshal(results)
}

// GetAll waits for all actors to respond and returns a mapping of all actors and their
// corresponding responses.
func (r Responses) GetAll() map[*Ref]Message {
	results := make(map[*Ref]Message, cap(r))
	for response := range r {
		if !response.Empty() {
			results[response.Source()] = response.Get()
		}
	}
	return results
}

func askAll(
	ctx context.Context, message Message, timeout *time.Duration, sender *Ref, actors []*Ref,
) Responses {
	results := make(chan Response, len(actors))
	wg := sync.WaitGroup{}
	wg.Add(len(actors))
	for _, actor := range actors {
		resp := actor.ask(ctx, sender, message)
		go func() {
			defer wg.Done()
			// Wait for the response to be ready before putting into the result channel.
			if timeout == nil {
				resp.Get()
			} else {
				resp.GetOrTimeout(*timeout)
			}
			results <- resp
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}

// UnpackResponse ... asks at addr the req and puts the response into what v points at. When
// appropriate, errors are converted appropriate for an API response. Error cases are enumerated
// below:
//  * If v points to an unsettable value, a 500 is returned.
//  * If the actor cannot be found, a 404 is returned.
//  * If v is settable and the actor didn't respond or responded with nil, a 404 is returned.
//  * If the actor returned an error and it is a well-known error type, it is coalesced to gRPC.
//  * If the actor returned plain error, a 500 is returned.
//  * Finally, if the response's type is OK, it is put into v.
//  * Else, a 500 is returned.
// TODO(Brad): use this.
func UnpackResponse(resp Response, ref *Ref, req interface{}, v interface{}) error {
	if reflect.ValueOf(v).IsValid() && !reflect.ValueOf(v).Elem().CanSet() {
		return fmt.Errorf(
			"ask to actor %s contains valid but unsettable response holder %T", ref, v,
		)
	}
	expectingResponse := reflect.ValueOf(v).IsValid() && reflect.ValueOf(v).Elem().CanSet()
	switch {
	case resp.Source() == nil:
		return fmt.Errorf("actor %s could not be found", ref)
	case expectingResponse && resp.Empty(), expectingResponse && resp.Get() == nil:
		return fmt.Errorf("actor %s did not respond", ref)
	case resp.Error() != nil:
		return resp.Error()
	default:
		if expectingResponse {
			if reflect.ValueOf(v).Elem().Type() != reflect.ValueOf(resp.Get()).Type() {
				return fmt.Errorf(
					"actor %s returned unexpected message (%T): %v", ref, resp, resp,
				)
			}
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(resp.Get()))
		}
		return nil
	}
}
