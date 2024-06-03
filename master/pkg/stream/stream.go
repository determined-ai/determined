package stream

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

const insertSeq = int64(1)

type RecordCache struct {
	InsertCache interface{}
	FallinSeq   int64
	FallinCache interface{}
	UpdateSeq   int64
	UpdateCache interface{}
	hasDeleted  bool
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

func (p *Publisher[T]) hydrateMsgInsert(rawMsg T, idToRecordCache map[int]RecordCache, sub *Subscription[T], ev Event[T]) interface{} {
	var msg interface{}
	recordCache := RecordCache{}

	hydratedMsg, err := sub.hydrator(rawMsg.GetID())
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		// This id has deleted.
		recordCache.hasDeleted = true
		recordCache.DeleteCache = sub.Streamer.PrepareFn(hydratedMsg.DeleteMsg())
		idToRecordCache[rawMsg.GetID()] = recordCache
		return nil
	} else if err != nil {
		log.Debugf("failed to hydrate message: %s", err.Error())
		return nil
	}

	upsertMsg := hydratedMsg.UpsertMsg()

	// check filter again to see if the record has changed.
	if sub.filter(upsertMsg.Msg.(T)) && sub.permissionFilter(upsertMsg.Msg.(T)) {
		// Filter check pass. It's insert, update, or fallin.
		if rawMsg.SeqNum() == insertSeq {
			// The hydarte Msg has data from the insert event.
			recordCache.InsertCache = sub.Streamer.PrepareFn(upsertMsg)
			msg = recordCache.InsertCache
		} else {
			// The hydarted msg has data from update or fallin event.
			recordCache.UpdateSeq = rawMsg.SeqNum()
			recordCache.UpdateCache = sub.Streamer.PrepareFn(upsertMsg)
		}
	}
	idToRecordCache[rawMsg.GetID()] = recordCache

	return msg
}

// hydrateMsg queries the DB by the ID from rawMsg of a upsert or fallin event
// and grabs the fields(hydrated message) that we care about.
// Here are the different scenarios of an event in this function:
// 1. The record with id x has been deleted.
// 2. The record with id x still has the same Seq as the rawMsg.
// 3. The record with id x has a Seq greater than the rawMsg.
// The function returns an upsert message scenarios 2.
func (p *Publisher[T]) hydrateMsg(rawMsg T, idToRecordCache map[int]RecordCache, sub *Subscription[T], ev Event[T]) interface{} {
	var msg interface{}
	recordCache := RecordCache{}

	hydratedMsg, err := sub.hydrator(rawMsg.GetID())
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		// This id has deleted.
		recordCache.hasDeleted = true
		recordCache.DeleteCache = sub.Streamer.PrepareFn(hydratedMsg.DeleteMsg())
		idToRecordCache[rawMsg.GetID()] = recordCache
		return nil
	} else if err != nil {
		log.Debugf("failed to hydrate message: %s", err.Error())
		return nil
	}

	upsertMsg := hydratedMsg.UpsertMsg()
	recordCache.UpdateSeq = upsertMsg.Msg.SeqNum()
	recordCache.UpdateCache = sub.Streamer.PrepareFn(upsertMsg)
	// check filter again to see if the record has changed.
	if sub.filter(upsertMsg.Msg.(T)) && sub.permissionFilter(upsertMsg.Msg.(T)) {
		// Filter check pass. It's update, or fallin.
		if recordCache.UpdateSeq == rawMsg.SeqNum() {
			// The hydarte Msg has data from the original update event.
			msg = recordCache.UpdateCache
		}
	}
	idToRecordCache[rawMsg.GetID()] = recordCache

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
	idToRecordCache := map[int]RecordCache{}

	// check each event against each subscription
	for _, sub := range p.Subscriptions {
		fmt.Printf("sub: %+v\n", sub)
		func() {
			for i, ev := range events {
				var msg interface{}
				switch {
				case ev.Before == nil && ev.After != nil && sub.filter(*ev.After) && sub.permissionFilter(*ev.After):
					// insert: send the record to the client.
					fmt.Println("insert")
					afterMsg := *ev.After

					msg = p.hydrateMsgInsert(afterMsg, idToRecordCache, sub, ev)
					if msg == nil {
						continue
					}

				case ev.Before != nil && ev.After != nil && (!sub.filter(*ev.Before) || !sub.permissionFilter(*ev.Before)) &&
					sub.filter(*ev.After) && sub.permissionFilter(*ev.After):
					// fallin: send the record to the client.
					fmt.Println("fallin")
					afterMsg := *ev.After

					if recordCache, ok := idToRecordCache[afterMsg.GetID()]; ok {
						cachedSeq := recordCache.UpdateSeq
						if cachedSeq > afterMsg.SeqNum() || recordCache.hasDeleted {
							// ignore this message
							continue
						} else if cachedSeq == afterMsg.SeqNum() && recordCache.UpdateCache != nil {
							// recordCache.UpsertCache can be nil if this event is a fallout for previous subscribers.
							// It doesn't have UpdateCache.
							msg = recordCache.UpdateCache
						} else {
							msg = p.hydrateMsg(afterMsg, idToRecordCache, sub, ev)
							if msg == nil {
								continue
							}
						}
					} else {
						msg = p.hydrateMsg(afterMsg, idToRecordCache, sub, ev)
						if msg == nil {
							continue
						}
					}

				case ev.Before != nil && ev.After != nil && sub.filter(*ev.Before) && sub.permissionFilter(*ev.Before) &&
					sub.filter(*ev.After) && sub.permissionFilter(*ev.After):
					// update: send the record to the client.
					fmt.Println("update")
					fmt.Printf("index: %+v, afterMsg: %+v\n", i, *ev.After)
					afterMsg := *ev.After

					if recordCache, ok := idToRecordCache[afterMsg.GetID()]; ok {
						cachedSeq := recordCache.UpdateSeq
						if cachedSeq > afterMsg.SeqNum() || recordCache.hasDeleted {
							// ignore this message
							continue
						} else if cachedSeq == afterMsg.SeqNum() && recordCache.UpdateCache != nil {
							// recordCache.UpsertCache can be nil if this event is a fallout for previous subscribers.
							// It doesn't have UpdateCache.
							msg = recordCache.UpdateCache
							fmt.Printf("send msg: %+v\n", msg)
						} else {
							msg = p.hydrateMsg(afterMsg, idToRecordCache, sub, ev)
							if msg == nil {
								continue
							}
							fmt.Printf("send msg: %+v\n", msg)
						}
					} else {
						msg = p.hydrateMsg(afterMsg, idToRecordCache, sub, ev)
						if msg == nil {
							continue
						}
						fmt.Printf("send msg: %+v\n", msg)
					}

				case ev.Before != nil && ev.After != nil && sub.filter(*ev.Before) && sub.permissionFilter(*ev.Before) &&
					(!sub.filter(*ev.After) || !sub.permissionFilter(*ev.After)):
					// fallout: tell the client the record is deleted.
					fmt.Println("fallout")
					afterMsg := *ev.After

					if recordCache, ok := idToRecordCache[afterMsg.GetID()]; ok {
						if recordCache.DeleteCache == nil {
							recordCache.DeleteCache = sub.Streamer.PrepareFn(afterMsg.DeleteMsg())
						}
						msg = recordCache.DeleteCache
						idToRecordCache[afterMsg.GetID()] = recordCache
					} else {
						recordCache = RecordCache{}
						recordCache.DeleteCache = sub.Streamer.PrepareFn(afterMsg.DeleteMsg())
						msg = recordCache.DeleteCache
						idToRecordCache[afterMsg.GetID()] = recordCache
					}

				case ev.Before != nil && sub.filter(*ev.Before) && sub.permissionFilter(*ev.Before):
					// deletion: tell the client the record is deleted.
					fmt.Println("delete")
					beforeMsg := *ev.Before

					if recordCache, ok := idToRecordCache[beforeMsg.GetID()]; ok {
						if recordCache.DeleteCache == nil {
							recordCache.DeleteCache = sub.Streamer.PrepareFn(beforeMsg.DeleteMsg())
						}
						recordCache.hasDeleted = true
						msg = recordCache.DeleteCache
						idToRecordCache[beforeMsg.GetID()] = recordCache
					} else {
						recordCache = RecordCache{}
						recordCache.hasDeleted = true
						recordCache.DeleteCache = sub.Streamer.PrepareFn(beforeMsg.DeleteMsg())
						msg = recordCache.DeleteCache
						idToRecordCache[beforeMsg.GetID()] = recordCache
					}

				default:
					// ignore this message
					fmt.Println("ignore")
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
