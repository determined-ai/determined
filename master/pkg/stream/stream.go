package stream

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"

	"github.com/pkg/errors"
)

// Msg is an object with a message and a sequence number and json marshal caching.
type Msg interface {
	GetID() string
	SeqNum() int64
	UpsertMsg() UpsertMsg
	DeleteMsg() DeleteMsg
	Fetch(context.Context) error
}

// Event contains the old and new version a Msg.  Inserts will have Before==nil, deletions will
// have After==nil.
type Event[T Msg] struct {
	Before *T `json:"before"`
	After  *T `json:"after"`

	upsertCache interface{}
	deleteCache interface{}
}

// MarshallableMsg is an intermediary message that is ready to be marshaled and broadcast.
type MarshallableMsg interface {
	MarshalJSON() ([]byte, error)
}

// UpsertMsg represents an upsert in the stream.
type UpsertMsg struct {
	JSONKey string
	Msg     Msg
}

// MarshalJSON returns a json marshaled UpsertMsg.
func (u UpsertMsg) MarshalJSON() ([]byte, error) {
	data := map[string]Msg{
		u.JSONKey: u.Msg,
	}
	return json.Marshal(data)
}

// DeleteMsg represents a deletion in the stream.
type DeleteMsg struct {
	Key     string
	Deleted string
}

// MarshalJSON returns a json marshaled DeleteMsg.
func (d DeleteMsg) MarshalJSON() ([]byte, error) {
	data := map[string]string{
		d.Key: d.Deleted,
	}
	return json.Marshal(data)
}

// SyncMsg is the server response to a StartupMsg once it's been handled.
type SyncMsg struct {
	SyncID   string `json:"sync_id"`
	Complete bool   `json:"complete"`
}

// MarshalJSON returns a json marshaled SyncMsg.
func (sm SyncMsg) MarshalJSON() ([]byte, error) {
	// ensures the infinite json marshaling recursion does not occur
	type syncMsgCopy SyncMsg
	return json.Marshal(syncMsgCopy(sm))
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
	PrepareFn func(message MarshallableMsg) interface{}
}

// NewStreamer creates a new Steamer.
func NewStreamer(prepareFn func(message MarshallableMsg) interface{}) *Streamer {
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
	filterFn func(T) bool,
) Subscription[T] {
	return Subscription[T]{
		Streamer:         streamer,
		Publisher:        publisher,
		permissionFilter: permFilter,
		filter:           filterFn,
	}
}

// Register a Subscription with its Publisher.
func (s *Subscription[T]) Register() {
	s.Publisher.Lock.Lock()
	defer s.Publisher.Lock.Unlock()
	s.Publisher.Subscriptions[s] = struct{}{}
}

// Unregister removes a Subscription from its Publisher.
func (s *Subscription[T]) Unregister() {
	s.Publisher.Lock.Lock()
	defer s.Publisher.Lock.Unlock()
	delete(s.Publisher.Subscriptions, s)
}

// Publisher is responsible for publishing messages of type T
// to streamers associate with active subscriptions.
type Publisher[T Msg] struct {
	Lock          sync.Mutex
	Subscriptions map[*Subscription[T]]struct{}
	WakeupID      int64
}

// NewPublisher creates a new Publisher for message type T.
func NewPublisher[T Msg]() *Publisher[T] {
	return &Publisher[T]{
		Subscriptions: map[*Subscription[T]]struct{}{},
	}
}

// CloseAllStreamers closes all streamers associated with this Publisher.
func (p *Publisher[T]) CloseAllStreamers() {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	seenStreamersSet := make(map[*Streamer]struct{})
	for sub := range p.Subscriptions {
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
func (p *Publisher[T]) Broadcast(ctx context.Context, events []Event[T]) {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	// start with a fresh wakeupid
	p.WakeupID++
	wakeupID := p.WakeupID

	// check each event against each subscription
	for sub := range p.Subscriptions {
		func() {
			// tracks the latest state an entity has been communicated as being
			// enables some events to be skipped if a newer state has already been reported.
			seqCache := make(map[string]int64)
			for _, ev := range events {
				var msg interface{}

				// // if this is an upsert, check if it needs to be hydrated or skipped
				// if ev.After != nil {
				// 	// hold onto in case hydrate overwrites these values
				// 	eventEntityID := (*ev.After).GetID()
				// 	eventEntitySeq :=  (*ev.After).SeqNum()
				// 	// can this event skipped because it's a newer state of the entity has already been communicated
				// 	if lastSeen, ok := seqCache[eventEntityID]; ok && lastSeen >= eventEntitySeq {
				// 		continue EventLoop
				// 	} else {
				// 		err := (*ev.After).Hydrate(ctx)
				// 		// if an error occurs during hydration drop the event
				// 		// TODO (corban): this logic might be flawed
				// 		// if it's a normal deletion, then we can filter it out because there will be a deletion event; however, if it's
				// 		// fallout/disappearance case + a deletion prior the processing the fallout event
				// 		// then we'll see the deletion here, but when we get to the deletion message, then it
				// 		// is possible that we will drop a deletion event won't be communicated because it fails
				// 		// the filter checks?
				// 		// yes, so maybe reorder this...
				// 		if err != nil {
				// 			// did the error occur for any other reason besides deletion?
				// 			if errors.Cause(err) != sql.ErrNoRows {
				// 				log.Errorf("error occured while hydrating message during broadcast: %v", err)
				// 			}
				// 			continue EventLoop
				// 		}
				// 		seqCache[(*ev.After).GetID()] = (*ev.After).SeqNum()
				// 	}
				// // TODO (corban): determine if there's a case when a deletion message can be skipped
				// // what is the seq value we associate with deletion messages? the seq value that was last seen?
				// // or does it get incremented prior to the deletion? Check the migration
				// } else if ev.Before != nil {
				// 	//
				// }

				// determine if and how the event should be communicated with the streamer
				switch {
				case ev.After != nil && sub.filter(*ev.After) && sub.permissionFilter(*ev.After):
					// update, insert, or fallin: send the record to the client.

					eventEntityID := (*ev.After).GetID()
					eventEntitySeq := (*ev.After).SeqNum()
					lastSeq, found := seqCache[eventEntityID]
					// has a more recent state of this entity been reported in this loop
					if found && lastSeq >= eventEntitySeq {
						continue
					}

					// either this entity hasn't been reported, or is a more recent event
					err := (*ev.After).Fetch(ctx)
					if err != nil {
						if errors.Cause(err) != sql.ErrNoRows {
						}
					}

					if ev.upsertCache == nil {
						ev.upsertCache = sub.Streamer.PrepareFn((*ev.After).UpsertMsg())
					}
					msg = ev.upsertCache
				case ev.Before != nil && sub.filter(*ev.Before) && sub.permissionFilter(*ev.Before):
					// deletion or fallout: tell the client the record is deleted.
					if ev.deleteCache == nil {
						ev.deleteCache = sub.Streamer.PrepareFn((*ev.Before).DeleteMsg())
					}
					msg = ev.deleteCache
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
