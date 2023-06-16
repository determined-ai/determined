package stream

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Event is an object with a message and a sequence number and json marshal caching.
type Event interface {
	SeqNum() int64
	PreparedMessage() *websocket.PreparedMessage
}

// Update contains an Event and a slice of applicable user ids
type Update[T Event] struct {
	Event T
	Users []int
}
// Streamer aggregates many events and wakeups into a single slice of pre-marshaled messages.
// One streamer may be associated with many Subscription[T]'s, but it should only have at most one
// Subscription per type T.  One Streamer is intended to belong to one websocket connection.
type Streamer[R any] struct {
	Cond *sync.Cond
	Events []*websocket.PreparedMessage
	// ReadEvents are opaque to the streaming API; but they are passed through the Streamer
	// to make writing websocket goroutines easier.
	ReadEvents []R
	// Closed is set externally, and noticed eventually.
	Closed bool
}

func NewStreamer[R any]() *Streamer[R] {
	var lock sync.Mutex
	cond := sync.NewCond(&lock)
	return &Streamer[R]{ Cond: cond }
}

// AddReadEvent passes a read event (presumably from the websocket) to the goroutine that is
// processing streaming events.
func (s *Streamer[R]) AddReadEvent(readEvent R) {
	s.Cond.L.Lock()
	defer s.Cond.L.Unlock()
	s.Cond.Signal()
	s.ReadEvents = append(s.ReadEvents, readEvent)
}

// WaitForSomething returns a tuple of (readEvents, events, closed)
func (s *Streamer[R]) WaitForSomething() ([]R, []*websocket.PreparedMessage, bool) {
	s.Cond.L.Lock()
	defer s.Cond.L.Unlock()
	s.Cond.Wait()
	// steal outputs
	revents := s.ReadEvents
	s.ReadEvents = nil
	events := s.Events
	s.Events = nil
	return revents, events, s.Closed
}

func (s *Streamer[R]) Close() {
	s.Cond.L.Lock()
	defer s.Cond.L.Unlock()
	s.Cond.Signal()
	s.Closed = true
}

type Subscription[T Event, R any] struct {
	// Which streamer is collecting events from this Subscription?
	Streamer *Streamer[R]
	// Which user do we belong to?
	User int
	// Which publisher should we connect to when active?
	Publisher *Publisher[T, R]
	// Decide if the streamer wants this event.
	filter func(T) bool
	// wakeupID prevent duplicate wakeups if multiple events in a single Broadcast are relevant
	wakeupID int64
}

func NewSubscription[T Event, R any](
	streamer *Streamer[R], publisher *Publisher[T, R], user int,
) Subscription[T, R] {
	return Subscription[T, R]{Streamer: streamer, Publisher: publisher, User: user}
}

func (s *Subscription[T, R]) Configure(filter func(T) bool) int64 {
	if filter == nil && s.filter == nil {
		// no change
		return 0 // XXX: seems like (bool, int64) would be appropriate here
	}
	// Changes must be synchronized with our respective publisher.
	s.Publisher.Lock.Lock()
	defer s.Publisher.Lock.Unlock()
	if s.filter == nil {
		// We weren't connected to the publisher before, but now we are.
		usergrp, ok := s.Publisher.UserGroups[s.User]
		if !ok {
			usergrp = &UserGroup[T, R]{}
			s.Publisher.UserGroups[s.User] = usergrp
		}
		usergrp.Subscriptions = append(usergrp.Subscriptions, s)
	} else if filter == nil {
		// Delete an existing registration.
		usergrp := s.Publisher.UserGroups[s.User]
		for i, sub := range usergrp.Subscriptions {
			if sub != s {
				continue
			}
			last := len(usergrp.Subscriptions) - 1
			usergrp.Subscriptions[i] = usergrp.Subscriptions[last]
			usergrp.Subscriptions = usergrp.Subscriptions[:last]
			break
		}
	} else {
		// Modify an existing registraiton.
		// (just save filter, below)
	}
	// Remember the new filter.
	s.filter = filter
	return s.Publisher.NewestPublished
}

// UserGroup is a set of filters belonging to the same user.  It is part of stream rbac enforcement.
type UserGroup[T Event, R any] struct {
	Subscriptions []*Subscription[T, R]
	Events []T

	// a self-pointer for efficient update tracking
	next     *UserGroup[T, R]
	wakeupID int64
}

type Publisher[T Event, R any] struct {
	Lock sync.Mutex
	// map userids to subscriptions matching those userids
	UserGroups map[int]*UserGroup[T, R]
	// The most recent published event (won't be published again)
	NewestPublished int64
	WakeupID int64
}

func NewPublisher[T Event, R any]() *Publisher[T, R]{
	return &Publisher[T, R]{
		UserGroups: make(map[int]*UserGroup[T, R]),
	}
}

func Broadcast[T Event, R any](p *Publisher[T, R], updates []Update[T]) {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	// start with a fresh wakeupid
	p.WakeupID++
	wakeupID := p.WakeupID

	groupSentinel := UserGroup[T, R]{}
	activeGroups := &groupSentinel

	// pass each update to each UserGroup representing users who are allowed to see it
	for _, update := range updates {
		// keep track of the newest published event
		if update.Event.SeqNum() > p.NewestPublished {
			p.NewestPublished = update.Event.SeqNum()
		}
		for _, user := range update.Users {
			// find matching user group
			usergrp, ok := p.UserGroups[user]
			if !ok || len(usergrp.Subscriptions) == 0 {
				continue
			}
			// first event for this user sub?
			if wakeupID != usergrp.wakeupID {
				usergrp.wakeupID = wakeupID
				usergrp.next = activeGroups
				activeGroups = usergrp
				// re-initialize events
				usergrp.Events = nil
			}
			// add event to usergrp
			usergrp.Events = append(usergrp.Events, update.Event)
		}
	}

	// do wakeups, visiting the active usergrps we collected in our list
	usergrp := activeGroups
	next := usergrp.next
	activeGroups = nil
	for ; usergrp != &groupSentinel; usergrp = next {
		// break down the list as we go, so gc is effective
		next = usergrp.next
		usergrp.next = nil

		// Deliver fewer wakeups: any streamer may own many subscriptions in this UserGroup,
		// but since SubscriptionGroups are user-based, no streamer can own subsriptions in two
		// groups.
		func(){
			for _, sub := range usergrp.Subscriptions {
				for _, event := range usergrp.Events {
					// does this subscription want this event?
					if !sub.filter(event){
						continue
					}
					// is it the first event for this Subscription?
					if sub.wakeupID != wakeupID {
						sub.wakeupID = wakeupID
						sub.Streamer.Cond.L.Lock()
						defer sub.Streamer.Cond.L.Unlock()
						sub.Streamer.Cond.Signal()
					}
					// append bytes into the Streamer, which is type-independent
					// TODO: actually marshal with caching

					// sub.Streamer.Events = append(sub.Streamer.Events, event)
					sub.Streamer.Events = append(sub.Streamer.Events, event.PreparedMessage())
				}
			}
		}()
	}
}
