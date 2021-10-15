package actors

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type (
	// ForwardThroughMock forwards a message (Msg) to another actor (To), using tell and
	// ask appropriately.
	ForwardThroughMock struct {
		To  *actor.Ref
		Msg actor.Message
	}
	// MockResponse sets up a respond to use in respond to a message of a given type.
	MockResponse struct {
		Msg      actor.Message
		Consumed bool
	}
	// MockActor is a convenience actor for testing hierarchies of actors without instantiating
	// off of them.
	MockActor struct {
		Messages  []actor.Message
		Responses map[string]*MockResponse
	}
)

// Receive implements actor.Actor.
func (a *MockActor) Receive(ctx *actor.Context) error {
	a.Messages = append(a.Messages, ctx.Message())
	switch msg := ctx.Message().(type) {
	case error:
		return msg
	case ForwardThroughMock:
		if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Ask(msg.To, msg.Msg).Get())
		} else {
			ctx.Tell(msg.To, msg.Msg)
		}
	default:
		if resp, ok := a.Responses[fmt.Sprintf("%T", msg)]; ok {
			if ctx.ExpectingResponse() {
				ctx.Respond(resp.Msg)
			}
			resp.Consumed = true
		} else if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Message())
		}
	}
	return nil
}

// Expect sets up an expectation to send some response.
func (a *MockActor) Expect(t string, r MockResponse) {
	a.Responses[t] = &r
}

// AssertExpectations asserts mocked expectations were met.
func (a *MockActor) AssertExpectations() error {
	for t, r := range a.Responses {
		if !r.Consumed {
			return fmt.Errorf("expected to reply with %s", t)
		}
	}
	return nil
}
