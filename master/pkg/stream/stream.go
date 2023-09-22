package stream

import (
	"sync"
)

// UpsertFunc is a function that overrides the default upsert
type UpsertFunc func(interface{}) interface{}

// DeleteFunc is a function that overrides the default deletion
type DeleteFunc func(string, string) interface{}

// Msg is an object with a message and a sequence number and json marshal caching.
type Msg interface {
	SeqNum() int64
	//PrepareFn(upsert UpsertMsg, delete DeleteMsg) interface{}
	//UpsertMsg(prepare UpsertFunc) interface{} // cache aversion of yourself for the purpose of upsert on the wire. This would have to take this prep function and pass the prep function to the upsert pmsg fn
	//// publisher stream in internal, pass each publisher object the prep function. Know it works if you don't have to import websockets into the pkg stream object. When testing, just write in whatever is most useful for the test. Could be the full object.
	//DeleteMsg(prepare DeleteFunc) interface{}
}

// Event contains the old and new version a Msg.  Inserts will have Before==nil, deletions will
// have After==nil.
type Event[T Msg] struct {
	Before *T `json:"before"`
	After  *T `json:"after"`

	upsertCache interface{}
	deleteCache interface{}
}

type UpsertMsg struct {
	JsonKey string
	Msg     Msg
}

func NewUpsertMsg(msg Msg, jsonKey string) UpsertMsg {
	return UpsertMsg{
		JsonKey: jsonKey,
		Msg:     msg,
	}
}

type DeleteMsg struct {
	Key     string
	Deleted string
}

func NewDeleteMsg(key, deleted string) DeleteMsg {
	return DeleteMsg{
		Key:     key,
		Deleted: deleted,
	}
}

// Streamer aggregates many events and wakeups into a single slice of pre-marshaled messages.
// One streamer may be associated with many Subscription[T]'s, but it should only have at most one
// Subscription per type T.  One Streamer is intended to belong to one websocket connection.
type Streamer struct {
	Cond *sync.Cond
	// Msgs are pre-marshalled messages to send to the streaming client.
	Msgs []interface{}
	// Closed is set externally, and noticed eventually.
	Closed bool
	// PrepareFn is a user defined function that prepares Msgs for broadcast
	PrepareFn func(interface{}) interface{}
}

func NewStreamer(prepareFn func(interface{}) interface{}) *Streamer {
	var lock sync.Mutex
	cond := sync.NewCond(&lock)
	if prepareFn == nil {
		panic("PrepareFn must be provided")
	}
	return &Streamer{Cond: cond, PrepareFn: prepareFn}
}

func (s *Streamer) Close() {
	s.Cond.L.Lock()
	defer s.Cond.L.Unlock()
	s.Cond.Signal()
	s.Closed = true
}

type Subscription[T Msg] struct {
	// Which streamer is collecting messages from this Subscription?
	Streamer *Streamer
	// Which publisher should we connect to when active?
	Publisher *Publisher[T]
	// Decide if the streamer wants this message.
	filter func(T) bool
	// wakeupID prevent duplicate wakeups if multiple events in a single Broadcast are relevant
	wakeupID int64
}

func NewSubscription[T Msg](streamer *Streamer, publisher *Publisher[T]) Subscription[T] {
	return Subscription[T]{Streamer: streamer, Publisher: publisher}
}

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
	} else {
		// Modify an existing registraiton.
		// (just save filter, below)
	}
	// Remember the new filter.
	s.filter = filter
}

type Publisher[T Msg] struct {
	Lock          sync.Mutex
	Subscriptions []*Subscription[T]
	WakeupID      int64
}

func NewPublisher[T Msg]() *Publisher[T] {
	return &Publisher[T]{}
}

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
				if ev.After != nil && sub.filter(*ev.After) {
					if ev.upsertCache == nil {
						ev.upsertCache = sub.Streamer.PrepareFn(ev.After)
					}
					// update, insert, or fallin: send the record to the client.
					// msg = (*ev.After).UpsertMsg(sub.UpsertFunc)
					msg = ev.upsertCache
				} else if ev.Before != nil && sub.filter(*ev.Before) {
					if ev.deleteCache == nil {
						ev.deleteCache = sub.Streamer.PrepareFn(ev.Before)
					}
					// deletion or fallout: tell the client the record is deleted.
					// msg = (*ev.Before).DeleteMsg(sub.DeleteFunc)
					msg = ev.deleteCache
				} else {
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
