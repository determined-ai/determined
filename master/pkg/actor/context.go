package actor

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/logger"
)

// Context holds contextual information for the context's recipient and the current message.
type Context struct {
	inner      context.Context
	message    Message
	sender     *Ref
	recipient  *Ref
	result     chan<- Message
	resultSent bool
	forwarded  bool
}

// Message returns the underlying message.
func (c *Context) Message() Message {
	return c.message
}

// Log returns the context's recipient's logger.
func (c *Context) Log() *log.Entry {
	return c.recipient.log
}

// AddLabel adds a new label to the context's recipient's logger.
func (c *Context) AddLabel(key string, value interface{}) {
	c.recipient.log = c.recipient.log.WithField(key, value)
}

// AddLabels adds new labels to the context's recipient's logger.
func (c *Context) AddLabels(ctx logger.Context) {
	c.recipient.log = c.recipient.log.WithFields(ctx.Fields())
}

// Tell sends the specified message to the actor (fire-and-forget semantics). The new context's
// sender is set to the recipient of this context.
func (c *Context) Tell(actor *Ref, message Message) {
	actor.tell(c.inner, c.recipient, message)
}

// TellAll sends the specified message to all actors (fire-and-forget semantics).
func (c *Context) TellAll(message Message, actors ...*Ref) {
	for _, ref := range actors {
		ref.tell(c.inner, ref, message)
	}
}

// Ask sends the specified message to the actor, returning a future to the result of the call. The
// new context's sender is set to the recipient of this context.
func (c *Context) Ask(actor *Ref, message Message) Response {
	return actor.ask(c.inner, c.recipient, message)
}

// AskAll sends the specified message to all actors, returning a future to all results of the call.
// Results are returned in arbitrary order. The result channel is closed after all actors respond.
// The new context's sender is set to recipient of this context.
func (c *Context) AskAll(message Message, actors ...*Ref) Responses {
	return askAll(c.inner, message, nil, c.recipient, actors)
}

// ActorOf adds the actor to the system as a child of the context's recipient. If an actor with that
// ID already exists, that actor's reference is returned instead. The second argument is true if the
// actor reference was created and false otherwise.
func (c *Context) ActorOf(id interface{}, actor Actor) (*Ref, bool) {
	return c.recipient.createChild(c.recipient.address.Child(id), actor)
}

// MustActorOf adds the actor with the provided address. It panics if a new actor was not created.
func (c *Context) MustActorOf(id interface{}, actor Actor) *Ref {
	ref, created := c.ActorOf(id, actor)
	if !created {
		panic("actor was not created")
	}
	return ref
}

// ActorOfFromFactory behaves the same as ActorOf but will only create the actor instance if it's
// needed. It is intended for cases where an actor needs to be looked up many times safely but
// usually exists.
func (c *Context) ActorOfFromFactory(id interface{}, factory func() Actor) (*Ref, bool) {
	return c.recipient.createChildFromFactory(c.recipient.address.Child(id), factory)
}

// Self returns the reference to the context's recipient.
func (c *Context) Self() *Ref {
	return c.recipient
}

// Children returns a list of references to the context's recipient's children.
func (c *Context) Children() []*Ref {
	return c.recipient.Children()
}

// Child returns the child with the given local ID.
func (c *Context) Child(id interface{}) *Ref {
	return c.recipient.Child(id)
}

// ExpectingResponse returns true if the sender is expecting a response and false otherwise.
func (c *Context) ExpectingResponse() bool {
	return c.result != nil && !c.resultSent && !c.forwarded
}

// Respond returns a response message for this request message back to the sender.
func (c *Context) Respond(message Message) {
	if c.result == nil {
		panic("sender is not expecting a response")
	}
	if c.forwarded {
		panic("message forwarded to another actor")
	}
	c.resultSent = true
	c.result <- message
	close(c.result)
}

// RespondCheckError returns a response message for this request message back to the sender. If the
// response has an error send that instead.
func (c *Context) RespondCheckError(message Message, err error) {
	if err != nil {
		c.Respond(err)
	} else {
		c.Respond(message)
	}
}

// Kill removes the child with the given local ID from this parent. All messages from this child to
// this actor are ignored.
func (c *Context) Kill(id interface{}) bool {
	if child := c.Child(id); child != nil {
		delete(c.recipient.children, child.Address())
		c.recipient.deadChildren[child.Address()] = true
		child.Stop()
		return true
	}
	return false
}
