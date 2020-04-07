package actors

import (
	"github.com/determined-ai/determined/master/pkg/actor"
)

// Group is an actor that manages a set of child actors.
type Group struct{}

type (
	// Children asks the actor for the list of actor references to all children.
	Children struct{}
	// NewChild creates a new child actor in the group.
	NewChild struct {
		ID    string
		Actor actor.Actor
	}
	// TellChild tells the child with the provided ID about the provided message. If the ID is not
	// set, all children are told about the message.
	TellChild struct {
		ID      *string
		Message actor.Message
	}
	// AskChild asks the child with the provided ID about the provided message. If the ID is not
	// set, all children are asked about the message.
	AskChild struct {
		ID      *string
		Message actor.Message
	}
)

// Receive implements the actor.Actor interface.
func (g Group) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case NewChild:
		ctx.ActorOf(msg.ID, msg.Actor)
	case TellChild:
		if msg.ID == nil {
			for _, child := range ctx.Children() {
				ctx.Tell(child, msg.Message)
			}
		} else {
			ctx.Tell(ctx.Child(*msg.ID), msg.Message)
		}
	case AskChild:
		if msg.ID == nil {
			ctx.Respond(ctx.AskAll(msg.Message, ctx.Children()...))
		} else {
			ctx.Tell(ctx.Child(*msg.ID), msg.Message)
		}
	case Children:
		ctx.Respond(ctx.Children())
	}
	return nil
}
