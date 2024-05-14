package stream

import (
	"database/sql"
	"encoding/json"
	"slices"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
}

type EventCache struct {
	Seq         int64
	UpsertCache interface{}
	DeleteCache interface{}
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
	s.Publisher.Subscriptions = append(s.Publisher.Subscriptions, s)
}

// Unregister removes a Subscription from its Publisher.
func (s *Subscription[T]) Unregister() {
	s.Publisher.Lock.Lock()
	defer s.Publisher.Lock.Unlock()
	subscriptions := s.Publisher.Subscriptions
	i := slices.Index(subscriptions, s)
	if i == -1 {
		log.Errorf("failed to unregister subscription.")
		return
	}
	subscriptions[i] = subscriptions[len(subscriptions)-1]
	subscriptions = subscriptions[:len(subscriptions)-1]
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
	return &Publisher[T]{
		Subscriptions: []*Subscription[T]{},
	}
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

// hydrateMsg queries the DB by the ID from rawMsg of a upsert or fallin event
// and grabs the fields(hydrated message) that we care about.
// Here are the different scenarios of an event in this function:
// 1. It checks if the event has been deleted.
// 2. It checks if the event still has the same Seq as the rawMsg.
// 3. It checks if the event has a Seq greater than the rawMsg.
// 4. It checks if the event is now considered as a fallout event for the subscriber and still has the same Seq as the rawMsg.
// 5. It checks if the event is now considered as a fallout event for the subscriber and has a Seq greater than the rawMsg.
// The function returns an upsert message scenarios 2 and a delete message in scenario 4.
func (p *Publisher[T]) hydrateMsg(rawMsg T, idToEventCache map[int]EventCache, sub *Subscription[T], ev Event[T]) interface{} {
	var msg interface{}
	eventCache := EventCache{}

	hydratedMsg, err := sub.hydrator(rawMsg.GetID())
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		// This id has a delete event later
		eventCache.Seq = -1
		eventCache.DeleteCache = sub.Streamer.PrepareFn(hydratedMsg.DeleteMsg())
		idToEventCache[rawMsg.GetID()] = eventCache
		return nil
	} else if err != nil {
		log.Debugf("failed to hydrate message: %s", err.Error())
		return nil
	}

	upsertMsg := hydratedMsg.UpsertMsg()
	eventCache.Seq = upsertMsg.Msg.SeqNum()
	eventCache.UpsertCache = sub.Streamer.PrepareFn(upsertMsg)
	// check filter again to see if the original event has become a fallout event.
	if sub.filter(upsertMsg.Msg.(T)) && sub.permissionFilter(upsertMsg.Msg.(T)) {
		if eventCache.Seq == rawMsg.SeqNum() {
			msg = eventCache.UpsertCache
		}
	} else {
		// It's a fallout event.
		eventCache.DeleteCache = sub.Streamer.PrepareFn(hydratedMsg.DeleteMsg())
		if eventCache.Seq == rawMsg.SeqNum() {
			msg = eventCache.DeleteCache
		}
	}
	idToEventCache[rawMsg.GetID()] = eventCache

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
	idToEventCache := map[int]EventCache{}

	// check each event against each subscription
	for _, sub := range p.Subscriptions {
		func() {
			for _, ev := range events {
				var msg interface{}
				switch {
				case ev.After != nil && sub.filter(*ev.After) && sub.permissionFilter(*ev.After):
					// update, insert, or fallin: send the record to the client.
					rawMsg := *ev.After

					if eventCache, ok := idToEventCache[rawMsg.GetID()]; ok {
						cachedSeq := eventCache.Seq
						if cachedSeq > rawMsg.SeqNum() || cachedSeq == -1 {
							// ignore this message
							continue
						} else if cachedSeq == rawMsg.SeqNum() && eventCache.UpsertCache != nil {
							// eventCache.UpsertCache can be nil if the previous event is a fallout.
							// It doesn't have UpsertCache.
							msg = eventCache.UpsertCache
						} else {
							msg = p.hydrateMsg(rawMsg, idToEventCache, sub, ev)
							if msg == nil {
								continue
							}
						}
					} else {
						msg = p.hydrateMsg(rawMsg, idToEventCache, sub, ev)
						if msg == nil {
							continue
						}
					}

				case ev.Before != nil && ev.After != nil && sub.filter(*ev.Before) && sub.permissionFilter(*ev.Before) &&
					(!sub.filter(*ev.After) || !sub.permissionFilter(*ev.After)):
					// fallout: tell the client the record is deleted.
					afterMsg := *ev.After

					if eventCache, ok := idToEventCache[afterMsg.GetID()]; ok {
						if eventCache.DeleteCache == nil {
							eventCache.DeleteCache = sub.Streamer.PrepareFn(afterMsg.DeleteMsg())
						}
						msg = eventCache.DeleteCache
						idToEventCache[afterMsg.GetID()] = eventCache
					} else {
						eventCache = EventCache{}
						eventCache.DeleteCache = sub.Streamer.PrepareFn(afterMsg.DeleteMsg())
						msg = eventCache.DeleteCache
						idToEventCache[afterMsg.GetID()] = eventCache
					}

				case ev.Before != nil && sub.filter(*ev.Before) && sub.permissionFilter(*ev.Before):
					// deletion: tell the client the record is deleted.
					beforeMsg := *ev.Before

					if eventCache, ok := idToEventCache[beforeMsg.GetID()]; ok {
						if eventCache.DeleteCache == nil {
							eventCache.DeleteCache = sub.Streamer.PrepareFn(beforeMsg.DeleteMsg())
						}
						eventCache.Seq = -1
						msg = eventCache.DeleteCache
						idToEventCache[beforeMsg.GetID()] = eventCache
					} else {
						eventCache = EventCache{}
						eventCache.Seq = -1
						eventCache.DeleteCache = sub.Streamer.PrepareFn(beforeMsg.DeleteMsg())
						msg = eventCache.DeleteCache
						idToEventCache[beforeMsg.GetID()] = eventCache
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
