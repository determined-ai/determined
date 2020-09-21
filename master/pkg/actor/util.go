package actor

import (
	"context"
	"encoding/json"
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
