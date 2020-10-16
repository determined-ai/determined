package actor

import (
	"context"

	log "github.com/sirupsen/logrus"
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

// Tell sends the specified message to the actor (fire-and-forget semantics). The new context's
// sender is set to the recipient of this context.
func (c *Context) Tell(actor *Ref, message Message) {
	actor.tell(c.inner, c.recipient, message)
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

// Sender returns the reference to the actor that sent the message.
func (c *Context) Sender() *Ref {
	return c.sender
}

// Self returns the reference to the context's recipient.
func (c *Context) Self() *Ref {
	return c.recipient
}

// Children returns a list of references to the context's recipient's children.
func (c *Context) Children() []*Ref {
	children := make([]*Ref, 0, len(c.recipient.children))
	for _, child := range c.recipient.children {
		children = append(children, child)
	}
	return children
}

// Child returns the child with the given local ID.
func (c *Context) Child(id interface{}) *Ref {
	return c.recipient.children[c.recipient.address.Child(id)]
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
