package actor

import (
	"sync"
	"time"
)

// System is a hierarchical group of actors.
type System struct {
	id string
	*Ref

	refsLock sync.RWMutex
	refs     map[Address]*Ref
}

// NewSystem constructs a new actor system and starts it.
func NewSystem(id string) *System {
	return NewSystemWithRoot(id, &rootActor{})
}

// NewSystemWithRoot constructs a new actor system with the specified root actor and starts it.
func NewSystemWithRoot(id string, actor Actor) *System {
	system := &System{id: id, refs: make(map[Address]*Ref)}
	system.Ref = newRef(system, nil, rootAddress, actor)
	return system
}

// Tell sends the specified message to the actor (fire-and-forget semantics). The context's sender
// is set to `nil`.
func (s *System) Tell(actor *Ref, message Message) {
	if actor == nil {
		return
	}
	actor.tell(nil, message)
}

// TellAt sends the specified message to the actor (fire-and-forget semantics) at the provided
// address. The context's sender is set to `nil`.
func (s *System) TellAt(addr Address, message Message) {
	s.Tell(s.Get(addr), message)
}

// Ask sends the specified message to the actor, returning a future to the result of the call. The
// context's sender is set to `nil`.
func (s *System) Ask(actor *Ref, message Message) Response {
	if actor == nil {
		return emptyResponse(nil)
	}
	return actor.ask(nil, message)
}

// AskAt sends the specified message to the actor at the provided address, returning a future to the
// result of the call. The context's sender is set to `nil`.
func (s *System) AskAt(addr Address, message Message) Response {
	return s.Ask(s.Get(addr), message)
}

// AskAll sends the specified message to all actors, returning a future to all results of the call.
// Results are returned in arbitrary order. The result channel is closed after all actors respond.
// The context's sender is set to `nil`.
func (s *System) AskAll(message Message, actors ...*Ref) Responses {
	return askAll(message, nil, nil, actors)
}

// AskAllTimeout sends the specified message to all actors, returning a future to all results of the
// call. Results are returned in arbitrary order. The result channel is closed after all actors
// respond. The context's sender is set to `nil`. If the timeout is reached, nil responses are
// returned.
func (s *System) AskAllTimeout(message Message, timeout time.Duration, actors ...*Ref) Responses {
	return askAll(message, &timeout, nil, actors)
}

// Get returns the actor reference with the id, or nil if no actor with that id is found.
func (s *System) Get(address Address) *Ref {
	if address == rootAddress {
		return s.Ref
	}

	s.refsLock.RLock()
	defer s.refsLock.RUnlock()

	return s.refs[address]
}

// ActorOf adds the actor with the provided address.
// The second return value denotes whether a new actor was created or not.
func (s *System) ActorOf(address Address, actor Actor) (*Ref, bool) {
	parent := s.Get(address.Parent())
	if parent == nil {
		return nil, false
	}
	resp := s.Ask(parent, createChild{address: address, actor: actor})
	if resp.Empty() {
		return nil, false
	}
	created := resp.Get().(childCreated)
	return created.child, created.created
}

// MustActorOf adds the actor with the provided address.
// It panics if a new actor was not created.
func (s *System) MustActorOf(address Address, actor Actor) *Ref {
	parent := s.Get(address.Parent())
	if parent == nil {
		panic("address has no parent")
	}
	resp := s.Ask(parent, createChild{address: address, actor: actor})
	if resp.Empty() {
		panic("createChild had empty response")
	}
	created := resp.Get().(childCreated)
	return created.child
}
