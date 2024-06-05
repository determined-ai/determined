package stream

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"sync"

	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Msg is an object with a message and a sequence number and json marshal caching.
type Msg interface {
	GetID() int
	SeqNum() int64
	UpsertMsg() *UpsertMsg
	DeleteMsg() *DeleteMsg
}

// Event contains the old and new version a Msg.  Inserts will have Before==nil, deletions will
// have After==nil.
type Event[T Msg] struct {
	Before T `json:"before"`
	After  T `json:"after"`
}

// MarshallableMsg is an intermediary message that is ready to be marshaled and broadcast.
type MarshallableMsg interface {
	MarshalJSON() ([]byte, error)
}

// UpsertMsg represents an upsert in the stream.
type UpsertMsg struct {
	JSONKey string
	Msg
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
	s.Publisher.Subscriptions.Insert(s)
}

// Unregister removes a Subscription from its Publisher.
func (s *Subscription[T]) Unregister() {
	s.Publisher.Lock.Lock()
	defer s.Publisher.Lock.Unlock()
	subscriptions := s.Publisher.Subscriptions
	subscriptions.Remove(s)
}

// Publisher is responsible for publishing messages of type T
// to streamers associate with active subscriptions.
type Publisher[T Msg] struct {
	Lock          sync.Mutex
	Subscriptions set.Set[*Subscription[T]]
	WakeupID      int64
	// Hydrate an UpsertMsg.
	Hydrator func(T) (T, error)
}

// NewPublisher creates a new Publisher for message type T.
func NewPublisher[T Msg](hydrator func(T) (T, error)) *Publisher[T] {
	return &Publisher[T]{
		Subscriptions: set.Set[*Subscription[T]]{},
		Hydrator:      hydrator,
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

// HydrateMsg queries the DB by the ID from rawMsg of a upsert or fallin event
// and get the full record.
func (p *Publisher[T]) HydrateMsg(msg T, idToSaturatedMsg map[int]*UpsertMsg) {
	if reflect.ValueOf(msg).IsNil() {
		return
	}

	hydratedMsg, err := p.Hydrator(msg)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Errorf("failed to hydrate message: %s", err.Error())
		return
	}

	idToSaturatedMsg[msg.GetID()] = hydratedMsg.UpsertMsg()
}

// Broadcast receives a list of events, determines if they are
// applicable to the publisher's subscriptions, and sends
// appropriate messages to corresponding streamers.
func (p *Publisher[T]) Broadcast(events []Event[T], idToSaturatedMsg map[int]*UpsertMsg) {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	// start with a fresh wakeupid
	p.WakeupID++
	wakeupID := p.WakeupID

	// check each event against each subscription
	for sub := range p.Subscriptions {
		userNotKnownIDs := set.New[int]()
		func() {
			for _, ev := range events {
				var msg interface{}
				switch {
				case !reflect.ValueOf(ev.After).IsNil() && sub.filter(ev.After) && sub.permissionFilter(ev.After):
					// update, insert, or fallin: send the record to the client.
					afterMsg := ev.After
					isInsert := reflect.ValueOf(ev.Before).IsNil()
					isFallin := !reflect.ValueOf(ev.Before).IsNil() && (!sub.filter(ev.Before) || !sub.permissionFilter(ev.Before))

					if saturatedMsg, ok := idToSaturatedMsg[afterMsg.GetID()]; ok {
						cachedSeq := saturatedMsg.SeqNum()
						if cachedSeq == afterMsg.SeqNum() {
							msg = sub.Streamer.PrepareFn(saturatedMsg)
						} else {
							if isInsert || isFallin {
								userNotKnownIDs.Insert(afterMsg.GetID())
							}
							continue
						}
					} else {
						if isInsert || isFallin {
							userNotKnownIDs.Insert(afterMsg.GetID())
						}
						continue
					}
				case !reflect.ValueOf(ev.Before).IsNil() && sub.filter(ev.Before) && sub.permissionFilter(ev.Before):
					// deletion or fallout: tell the client the record is deleted.
					beforeMsg := ev.Before
					if !userNotKnownIDs.Contains(beforeMsg.GetID()) {
						msg = sub.Streamer.PrepareFn(beforeMsg.DeleteMsg())
						userNotKnownIDs.Remove(beforeMsg.GetID())
					} else {
						continue
					}
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
