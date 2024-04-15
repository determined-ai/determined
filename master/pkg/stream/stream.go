package stream

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

// Msg is an object with a message and a sequence number and json marshal caching.
type Msg interface {
	GetID() int
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
	fmt.Printf("msg: %+v\n", u.Msg)
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
	// Hydrate an UpsertMsg.
	hydrator func(int) (T, error)
}

// NewSubscription creates a new Subscription to messages of type T.
func NewSubscription[T Msg](
	streamer *Streamer,
	publisher *Publisher[T],
	permFilter func(T) bool,
	filterFn func(T) bool,
	hydrator func(int) (T, error),
) Subscription[T] {
	return Subscription[T]{
		Streamer:         streamer,
		Publisher:        publisher,
		permissionFilter: permFilter,
		filter:           filterFn,
		hydrator:         hydrator,
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

func (p *Publisher[T]) hydrateMsg(rawMsg T, idToSeq map[int]int64, sub *Subscription[T], ev Event[T]) interface{} {
	var msg interface{}
	// seq < rawMsg.SeqNum() and id doesn't exist in idToSeq
	entityMsg, err := sub.hydrator(rawMsg.GetID())
	if errors.Cause(err) == sql.ErrNoRows {
		// This id has a delete event later
		idToSeq[rawMsg.GetID()] = -1
		return nil
	}

	upsertMsg := entityMsg.UpsertMsg()
	fmt.Println("reach before filters")
	// check filter again in case project move workspace that would be a fall out
	if sub.filter(upsertMsg.Msg.(T)) && sub.permissionFilter(upsertMsg.Msg.(T)) {
		fmt.Println("helper: check filter again")
		ev.upsertCache = sub.Streamer.PrepareFn(upsertMsg)
		// fmt.Println("helper: afterpreparemessage")
		msg = ev.upsertCache
		idToSeq[upsertMsg.Msg.GetID()] = upsertMsg.Msg.SeqNum()
	} else {
		// This id fall out later
		fmt.Println("helper: fallout")
		idToSeq[rawMsg.GetID()] = -1
		return nil
	}
	return msg
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
	idToSeq := map[int]int64{}

	// check each event against each subscription
	for sub := range p.Subscriptions {
		// fmt.Println("NEW SUB")
		func() {
			for _, ev := range events {
				// fmt.Printf("events idx: %v\n", i)
				var msg interface{}
				switch {
				case ev.After != nil && sub.filter(*ev.After) && sub.permissionFilter(*ev.After):
					rawMsg := *ev.After
					// seq, ok := idToSeq[rawMsg.GetID()]
					// fmt.Printf("events idx: %v, seq: %v, ok: %v\n", i, seq, ok)
					if seq, ok := idToSeq[rawMsg.GetID()]; ok {
						// update, insert, or fallin: send the record to the client.
						if seq >= rawMsg.SeqNum() || seq == -1 {
							// ignore this message
							// fmt.Println("ignore this event. has cache.")
							continue
						} else {
							// fmt.Println("upsert event seq > cacahed seq")
							msg = p.hydrateMsg(rawMsg, idToSeq, sub, ev)
							if msg == nil {
								continue
							}

						}
					} else {
						fmt.Println("upsert no cached seq")
						msg = p.hydrateMsg(rawMsg, idToSeq, sub, ev)
						if msg == nil {
							continue
						}
						fmt.Printf("idToSeq: %+v\n", idToSeq)
					}

				case ev.Before != nil && sub.filter(*ev.Before) && sub.permissionFilter(*ev.Before):
					// deletion or fallout: tell the client the record is deleted.
					rawMsg := *ev.Before
					if ev.deleteCache == nil {
						ev.deleteCache = sub.Streamer.PrepareFn(rawMsg.DeleteMsg())
					}
					msg = ev.deleteCache
					// fmt.Println("delete msg")
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
				fmt.Printf("broadcast msg: %#v\n", msg)
				sub.Streamer.Msgs = append(sub.Streamer.Msgs, msg)
			}
		}()
	}
}
