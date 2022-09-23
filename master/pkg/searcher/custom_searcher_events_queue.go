package searcher

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

type (
	// SearcherEventQueue stores the list of custom searcher events and the event
	// that was event that was processed last by client and acknowledged by master.
	SearcherEventQueue struct {
		events     []*experimentv1.SearcherEvent
		eventCount int32 // stores the number of events in the queue.
		watchers   map[uuid.UUID]chan<- []*experimentv1.SearcherEvent
	}

	searcherEventQueueJSON struct {
		EventsJSON []json.RawMessage `json:"custom_searcher_events"`
		EventCount int32             `json:"custom_searcher_event_count"`
	}

	EventsWatcher struct {
		C <-chan []*experimentv1.SearcherEvent
	}
)

func newSearcherEventQueue() *SearcherEventQueue {
	events := make([]*experimentv1.SearcherEvent, 0)
	return &SearcherEventQueue{
		events: events, eventCount: 0,
		watchers: map[uuid.UUID]chan<- []*experimentv1.SearcherEvent{},
	}
}

// Create a eventsWatcher. If events are available add events and close it.
func (q *SearcherEventQueue) Watch(id uuid.UUID) (EventsWatcher, error) {
	// buffer size is 1 because we don't want to block
	//  until another goroutine recieves from this channel.
	// and only one event list can be sent to a channel.
	w := make(chan []*experimentv1.SearcherEvent, 1)
	q.watchers[id] = w

	if len(q.events) > 0 {
		w <- q.events
		close(w)
		delete(q.watchers, id)
	}
	return EventsWatcher{C: w}, nil
}

// Unwatch unregisters a eventsWatcher.
func (p *SearcherEventQueue) Unwatch(id uuid.UUID) {
	if p == nil {
		return
	}
	delete(p.watchers, id)
}

// Enqueue an event.
func (q *SearcherEventQueue) Enqueue(event *experimentv1.SearcherEvent) {
	q.eventCount++
	event.Id = q.eventCount
	q.events = append(q.events, event)

	// add events to all watcher channels.
	for id, w := range q.watchers {
		w <- q.events
		close(w)
		delete(q.watchers, id)
	}
	print("In enqueue")
	print(len(q.events))
}

// GetEvents returns all the events.
func (q *SearcherEventQueue) GetEvents() []*experimentv1.SearcherEvent {
	return q.events
}

// RemoveUpTo the given event Id.
func (q *SearcherEventQueue) RemoveUpTo(eventID int) error {
	for i, v := range q.events {
		if v.Id == int32(eventID) {
			q.events = q.events[i+1:]
			return nil
		}
	}
	return fmt.Errorf("event %d not found", eventID)
}

// MarshalJSON returns a marshaled searcherEventQueueJSON.
func (q *SearcherEventQueue) MarshalJSON() ([]byte, error) {
	marshaledPBEvents, err := marshalPBEvents(q.events)
	if err != nil {
		return nil, err
	}

	return json.Marshal(searcherEventQueueJSON{
		EventsJSON: marshaledPBEvents,
		EventCount: q.eventCount,
	})
}

// UnmarshalJSON unmarshals searcherEventQueueJSON.
func (q *SearcherEventQueue) UnmarshalJSON(sJSON []byte) error {
	var searcherEQJSON searcherEventQueueJSON
	if err := json.Unmarshal(sJSON, &searcherEQJSON); err != nil {
		return err
	}
	events, err := unmarshalPBEvents(searcherEQJSON.EventsJSON)
	if err != nil {
		return err
	}
	q.events = events
	q.eventCount = searcherEQJSON.EventCount
	q.watchers = map[uuid.UUID]chan<- []*experimentv1.SearcherEvent{}
	return nil
}

func marshalPBEvents(pbEvents []*experimentv1.SearcherEvent) ([]json.RawMessage, error) {
	marshaledPBEvents := make([]json.RawMessage, 0)
	for _, event := range pbEvents {
		mEvent, err := protojson.Marshal(event)
		if err != nil {
			return nil,
				errors.Wrap(err, "failed to marshal protobuf events list in (custom) SearcherEventQueue")
		}
		marshaledPBEvents = append(marshaledPBEvents, mEvent)
	}
	return marshaledPBEvents, nil
}

func unmarshalPBEvents(mEvents []json.RawMessage) ([]*experimentv1.SearcherEvent, error) {
	unmarshaledPBEvents := make([]*experimentv1.SearcherEvent, 0)
	for _, mEvent := range mEvents {
		var pbEvent experimentv1.SearcherEvent
		if err := protojson.Unmarshal(mEvent, &pbEvent); err != nil {
			return nil,
				errors.Wrap(err, "failed to save unmarshal events list in (custom) SearcherEventQueue")
		}
		unmarshaledPBEvents = append(unmarshaledPBEvents, &pbEvent)
	}
	return unmarshaledPBEvents, nil
}
