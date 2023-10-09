package stream

import (
	"encoding/json"
	"sync"
)

// UpsertFunc is a function that overrides the default upsert
type UpsertFunc func(interface{}) interface{}

// DeleteFunc is a function that overrides the default deletion
type DeleteFunc func(string, string) interface{}

// Msg is an object with a message and a sequence number and json marshal caching.
type Msg interface {
	SeqNum() int64
	UpsertMsg() UpsertMsg
	DeleteMsg() DeleteMsg
}

// Event contains the old and new version a Msg.  Inserts will have Before==nil, deletions will
// have After==nil.
type Event[T Msg] struct {
	Before *T `json:"before"`
	After  *T `json:"after"`

	upsertCache interface{}
	deleteCache interface{}
}

type PreparableMessage interface{}

type UpsertMsg struct {
	PreparableMessage
	JsonKey string
	Msg     Msg
}

func (u *UpsertMsg) MarshalJSON() ([]byte, error) {
	data := map[string]Msg{
		u.JsonKey: u.Msg,
	}
	return json.Marshal(data)
}

type DeleteMsg struct {
	PreparableMessage
	Key     string
	Deleted string
}

func (d *DeleteMsg) MarshalJSON() ([]byte, error) {
	data := map[string]string{
		d.Key: d.Deleted,
	}
	return json.Marshal(data)
}

// Streamer aggregates many events and wakeups into a single slice of pre-marshaled messages.
// One streamer may be associated with many Subscription[T]'s, but it should only have at most one
// Subscription per type T.  One Streamer is intended to belong to one websocket connection.
type Streamer struct {
	Cond *sync.Cond
	// Msgs are pre-marshaled messages to send to the streaming client.
	Msgs []interface{}
	// Closed is set externally, and noticed eventually.
	Closed bool
	// PrepareFn is a user defined function that prepares Msgs for broadcast
	PrepareFn func(message PreparableMessage) interface{}
}

// NewStreamer creates a new Steamer.
func NewStreamer(prepareFn func(message PreparableMessage) interface{}) *Streamer {
	var lock sync.Mutex
	cond := sync.NewCond(&lock)
	if prepareFn == nil {
		panic("PrepareFn must be provided")
	}
	return &Streamer{Cond: cond, PrepareFn: prepareFn}
}

// Close closes a streamer.
func (s *Streamer) Close() {
	s.Cond.L.Lock()
	defer s.Cond.L.Unlock()
	s.Cond.Signal()
	s.Closed = true
}

// Subscription manages a streamer's subscription to messages of type T.
type Subscription[T Msg] struct {
	// Which streamer is collecting messages from this Subscription?
	Streamer *Streamer
	// Which publisher should we connect to when active?
	Publisher *Publisher[T]
	// Decide if the streamer wants this message.
	filter func(T) bool
	// Decide if the streamer has permission to view this message.
	permissionFilter func(T) bool
	// wakeupID prevent duplicate wakeups if multiple events in a single Broadcast are relevant
	wakeupID int64
}

// NewSubscription creates a new Subscription to messages of type T.
func NewSubscription[T Msg](
	streamer *Streamer,
	publisher *Publisher[T],
	permFilter func(T) bool,
) Subscription[T] {
	return Subscription[T]{
		Streamer:         streamer,
		Publisher:        publisher,
		permissionFilter: permFilter,
	}
}

// Configure updates a Subscription's filters and updates the associated Publisher
// in the event of creating or deleting a Subscription.
func (s *Subscription[T]) Configure(filter func(T) bool) {
	if filter == nil && s.filter == nil {
		// no change, no synchronization needed
		return
	}
	// Changes must be synchronized with our respective publisher.
	s.Publisher.Lock.Lock()
	defer s.Publisher.Lock.Unlock()
	if s.filter == nil {
		// We weren't connected to the publisher before, but now we are.
		s.Publisher.Subscriptions = append(s.Publisher.Subscriptions, s)
	} else if filter == nil {
		// Delete an existing registration.
		for i, sub := range s.Publisher.Subscriptions {
			if sub != s {
				continue
			}
			last := len(s.Publisher.Subscriptions) - 1
			s.Publisher.Subscriptions[i] = s.Publisher.Subscriptions[last]
			s.Publisher.Subscriptions = s.Publisher.Subscriptions[:last]
			break
		}
	} // else modify an existing registration, update subscription filter

	// Remember the new filter.
	s.filter = filter
}

// Publisher is responsible for publishing messages of type T
// to streamers associate with active subscriptions.
type Publisher[T Msg] struct {
	Lock          sync.Mutex
	Subscriptions []*Subscription[T]
	WakeupID      int64
}

// NewPublisher creates a new Publisher for message type T.
func NewPublisher[T Msg]() *Publisher[T] {
	return &Publisher[T]{}
}

// CloseAllStreamers closes all streamers associated with this Publisher.
func (p *Publisher[T]) CloseAllStreamers() {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	seenStreamersSet := make(map[*Streamer]struct{})
	for _, sub := range p.Subscriptions {
		if _, ok := seenStreamersSet[sub.Streamer]; !ok {
			sub.Streamer.Close()
			seenStreamersSet[sub.Streamer] = struct{}{}
		}
	}
	p.Subscriptions = nil
}

// Broadcast receives a list of events, determines if they are
// applicable to the publisher's subscriptions, and sends
// appropriate messages to corresponding streamers.
func (p *Publisher[T]) Broadcast(events []Event[T]) {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	// start with a fresh wakeupid
	p.WakeupID++
	wakeupID := p.WakeupID

	// check each event against each subscription
	for _, sub := range p.Subscriptions {
		func() {
			for _, ev := range events {
				var msg interface{}
				switch {
				case ev.After != nil && sub.filter(*ev.After) && sub.permissionFilter(*ev.After):
					// update, insert, or fallin: send the record to the client.
					msg = (*ev.After).UpsertMsg()
				case ev.Before != nil && sub.filter(*ev.Before) && sub.permissionFilter(*ev.Before):
					// deletion or fallout: tell the client the record is deleted.
					msg = (*ev.Before).DeleteMsg()
				default:
					// ignore this message
					continue
				}
				// is this the first match for this Subscription during this Broadcast?
				if sub.wakeupID != wakeupID {
					sub.wakeupID = wakeupID
					sub.Streamer.Cond.L.Lock()
					defer sub.Streamer.Cond.L.Unlock()
					sub.Streamer.Cond.Signal()
				}
				sub.Streamer.Msgs = append(sub.Streamer.Msgs, msg)
			}
		}()
	}
}
