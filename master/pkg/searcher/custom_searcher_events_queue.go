package searcher

import (
	"encoding/json"
	"fmt"

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
		// Will help with uniquely identifying an event.
		searcherEQJSON searcherEventQueueJSON
	}

	searcherEventQueueJSON struct {
		EventsJSON []json.RawMessage `json:"custom_searcher_events"`
		EventCount int32             `json:"custom_searcher_event_count"`
	}
)

func newSearcherEventQueue() *SearcherEventQueue {
	events := make([]*experimentv1.SearcherEvent, 0)
	return &SearcherEventQueue{events: events, eventCount: 0}
}

// Enqueue an event.
func (q *SearcherEventQueue) Enqueue(event *experimentv1.SearcherEvent) {
	q.eventCount++
	event.Id = q.eventCount
	q.events = append(q.events, event)
}

// GetEvents returns all the events.
func (q *SearcherEventQueue) GetEvents() []*experimentv1.SearcherEvent {
	return q.events
}

// SetEvents sets the events.
func (q *SearcherEventQueue) SetEvents(events []*experimentv1.SearcherEvent) {
	q.events = events
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
	q.searcherEQJSON.EventsJSON = marshaledPBEvents
	q.searcherEQJSON.EventCount = q.eventCount
	return json.Marshal(q.searcherEQJSON)
}

// UnmarshalJSON unmarshals searcherEventQueueJSON.
func (q *SearcherEventQueue) UnmarshalJSON(sJSON []byte) error {
	err := json.Unmarshal(sJSON, &q.searcherEQJSON)
	if err != nil {
		return err
	}
	q.events, err = unmarshalPBEvents(q.searcherEQJSON.EventsJSON)
	if err != nil {
		return err
	}
	q.eventCount = q.searcherEQJSON.EventCount
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
		err := protojson.Unmarshal(mEvent, &pbEvent)
		if err != nil {
			return nil,
				errors.Wrap(err, "failed to save unmarshal events list in (custom) SearcherEventQueue")
		}
		unmarshaledPBEvents = append(unmarshaledPBEvents, &pbEvent)
	}
	return unmarshaledPBEvents, nil
}
