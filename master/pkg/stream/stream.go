package stream

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

type EntityCache struct {
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

func (p *Publisher[T]) hydrateMsg(rawMsg T, idToEventCache map[int]EntityCache, sub *Subscription[T], ev Event[T]) interface{} {
	fmt.Println("hydrate message")
	var msg interface{}
	entityCache := EntityCache{}

	entityMsg, err := sub.hydrator(rawMsg.GetID())
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		// This id has a delete event later
		entityCache.Seq = -1
		entityCache.DeleteCache = sub.Streamer.PrepareFn(entityMsg.DeleteMsg())
		idToEventCache[rawMsg.GetID()] = entityCache
		return nil
	} else if err != nil {
		log.Debugf("failed to hydrate message: %s", err.Error())
		return nil
	}

	upsertMsg := entityMsg.UpsertMsg()
	entityCache.Seq = upsertMsg.Msg.SeqNum()
	entityCache.UpsertCache = sub.Streamer.PrepareFn(upsertMsg)
	// check filter again in case project move workspace that would be a fall out
	if sub.filter(upsertMsg.Msg.(T)) && sub.permissionFilter(upsertMsg.Msg.(T)) {
		// fmt.Println("helper: check filters again")
		if entityCache.Seq == rawMsg.SeqNum() {
			msg = entityCache.UpsertCache
		}
	} else {
		// This id fall out
		fmt.Println("detected fallout in hydration")
		entityCache.DeleteCache = sub.Streamer.PrepareFn(entityMsg.DeleteMsg())
		if entityCache.Seq == rawMsg.SeqNum() {
			fmt.Println("send fallout in hydration")
			msg = entityCache.DeleteCache
		}
	}
	idToEventCache[rawMsg.GetID()] = entityCache

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
	idToEventCache := map[int]EntityCache{}

	// check each event against each subscription
	for sub := range p.Subscriptions {
		// fmt.Println("NEW SUB")
		func() {
			for i, ev := range events {
				fmt.Printf("events idx: %v\n", i)
				var msg interface{}
				switch {
				case ev.After != nil && sub.filter(*ev.After) && sub.permissionFilter(*ev.After):
					rawMsg := *ev.After
					entityCache, ok := idToEventCache[rawMsg.GetID()]
					fmt.Printf("events idx: %v, entityCache: %+v, ok: %v, filter: %v\n", i, entityCache, ok, sub.filter(*ev.After))
					if entityCache, ok := idToEventCache[rawMsg.GetID()]; ok {
						// update, insert, or fallin: send the record to the client.
						cachedSeq := entityCache.Seq
						if cachedSeq > rawMsg.SeqNum() || cachedSeq == -1 {
							// ignore this message
							// fmt.Println("ignore this event. has cache.")
							continue
						} else if cachedSeq == rawMsg.SeqNum() && entityCache.UpsertCache != nil {
							// entityCache.UpsertCache can be nil if the previous event is a fallout.
							// It doesn't have UpsertCache.
							msg = entityCache.UpsertCache
							fmt.Printf("cached msg sent: %#v\n", msg)
						} else {
							msg = p.hydrateMsg(rawMsg, idToEventCache, sub, ev)
							fmt.Printf("updated cached msg sent: %#v\n", msg)
							if msg == nil {
								continue
							}
						}
					} else {
						// fmt.Println("upsert no cached seq")
						msg = p.hydrateMsg(rawMsg, idToEventCache, sub, ev)
						fmt.Printf("no cached msg sent: %#v\n", msg)
						if msg == nil {
							continue
						}
						// fmt.Printf("idToCachedSeq: %+v\n", idToEventCache)
					}

				case ev.Before != nil && ev.After != nil && sub.filter(*ev.Before) && sub.permissionFilter(*ev.Before) &&
					(!sub.filter(*ev.After) || !sub.permissionFilter(*ev.After)):
					// fallout: tell the client the record is deleted.
					fmt.Println("fallout")
					afterMsg := *ev.After

					if entityCache, ok := idToEventCache[afterMsg.GetID()]; ok {
						if entityCache.DeleteCache == nil {
							entityCache.DeleteCache = sub.Streamer.PrepareFn(afterMsg.DeleteMsg())
						}
						msg = entityCache.DeleteCache
						idToEventCache[afterMsg.GetID()] = entityCache
					} else {
						entityCache = EntityCache{}
						entityCache.DeleteCache = sub.Streamer.PrepareFn(afterMsg.DeleteMsg())
						msg = entityCache.DeleteCache
						idToEventCache[afterMsg.GetID()] = entityCache
					}

				case ev.Before != nil && sub.filter(*ev.Before) && sub.permissionFilter(*ev.Before):
					// deletion: tell the client the record is deleted.
					beforeMsg := *ev.Before
					fmt.Println("delete msg")

					if entityCache, ok := idToEventCache[beforeMsg.GetID()]; ok {
						if entityCache.DeleteCache == nil {
							entityCache.DeleteCache = sub.Streamer.PrepareFn(beforeMsg.DeleteMsg())
						}
						entityCache.Seq = -1
						msg = entityCache.DeleteCache
						idToEventCache[beforeMsg.GetID()] = entityCache
					} else {
						entityCache = EntityCache{}
						entityCache.Seq = -1
						entityCache.DeleteCache = sub.Streamer.PrepareFn(beforeMsg.DeleteMsg())
						msg = entityCache.DeleteCache
						idToEventCache[beforeMsg.GetID()] = entityCache
					}

				default:
					fmt.Printf("This message is not relavent to the subscriber. Ignore it.\n")
					log.Tracef("This message is not relavent to the subscriber. Ignore it.\n")
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
