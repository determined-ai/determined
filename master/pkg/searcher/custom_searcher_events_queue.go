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
	// SearcherEventQueue stores the list of custom searcher events and the event that was event that
	// was processed last by client and acknowledged by master.
	SearcherEventQueue struct {
		events     []*experimentv1.SearcherEvent
		eventCount int32
		watchers   map[uuid.UUID]chan<- []*experimentv1.SearcherEvent
	}

	// searcherEventQueueJSON is used internally for JSON marshaling purposes.
	searcherEventQueueJSON struct {
		Events     []json.RawMessage `json:"custom_searcher_events"`
		EventCount int32             `json:"custom_searcher_event_count"`
	}

	// EventsWatcher has a channel which allows communication to the GET searcher events API.
	EventsWatcher struct {
		ID uuid.UUID
		C  <-chan []*experimentv1.SearcherEvent
	}
)

func newSearcherEventQueue() *SearcherEventQueue {
	return &SearcherEventQueue{
		events:     nil,
		eventCount: 0,
		watchers:   map[uuid.UUID]chan<- []*experimentv1.SearcherEvent{},
	}
}

func (q *SearcherEventQueue) sendEventsToWatcher(
	id uuid.UUID,
	w chan<- []*experimentv1.SearcherEvent,
) {
	events := make([]*experimentv1.SearcherEvent, len(q.events))
	copy(events, q.events)
	w <- events
	close(w)
	delete(q.watchers, id)
}

// Watch creates an eventsWatcher. If any events are currently in the queue, they are immediately
// sent; otherwise, the channel in the result will block until an event comes in.
func (q *SearcherEventQueue) Watch() (EventsWatcher, error) {
	// Buffer size is 1 because we don't want to block until another goroutine receives from this
	// channel and only one event list can be sent to a channel.
	w := make(chan []*experimentv1.SearcherEvent, 1)
	id := uuid.New()
	q.watchers[id] = w

	if len(q.events) > 0 {
		q.sendEventsToWatcher(id, w)
	}
	return EventsWatcher{ID: id, C: w}, nil
}

// Unwatch unregisters an eventsWatcher.
func (q *SearcherEventQueue) Unwatch(id uuid.UUID) {
	if q == nil {
		return
	}
	delete(q.watchers, id)
}

// Enqueue adds an event to the queue, setting its ID automatically.
func (q *SearcherEventQueue) Enqueue(event *experimentv1.SearcherEvent) {
	q.eventCount++
	event.Id = q.eventCount
	q.events = append(q.events, event)

	// Add events to all watcher channels.
	for id, w := range q.watchers {
		q.sendEventsToWatcher(id, w)
	}
}

// GetEvents returns all the events.
func (q *SearcherEventQueue) GetEvents() []*experimentv1.SearcherEvent {
	return q.events
}

// RemoveUpTo removes all events up to and including the one with the given event ID.
func (q *SearcherEventQueue) RemoveUpTo(eventID int) error {
	maxID := int(q.eventCount)
	minID := maxID - (len(q.events) - 1)
	if !(minID <= eventID && eventID <= maxID) {
		return fmt.Errorf("event %d not found", eventID)
	}
	q.events = q.events[eventID-minID+1:]
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (q *SearcherEventQueue) MarshalJSON() ([]byte, error) {
	events, err := marshalEvents(q.events)
	if err != nil {
		return nil, err
	}

	return json.Marshal(searcherEventQueueJSON{
		Events:     events,
		EventCount: q.eventCount,
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (q *SearcherEventQueue) UnmarshalJSON(data []byte) error {
	var js searcherEventQueueJSON
	if err := json.Unmarshal(data, &js); err != nil {
		return err
	}
	events, err := unmarshalEvents(js.Events)
	if err != nil {
		return err
	}
	q.events = events
	q.eventCount = js.EventCount
	q.watchers = map[uuid.UUID]chan<- []*experimentv1.SearcherEvent{}
	return nil
}

func marshalEvents(pbEvents []*experimentv1.SearcherEvent) ([]json.RawMessage, error) {
	var events []json.RawMessage
	for _, pbEvent := range pbEvents {
		event, err := protojson.Marshal(pbEvent)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal searcher event")
		}
		events = append(events, event)
	}
	return events, nil
}

func unmarshalEvents(events []json.RawMessage) ([]*experimentv1.SearcherEvent, error) {
	var pbEvents []*experimentv1.SearcherEvent
	for _, event := range events {
		var pbEvent experimentv1.SearcherEvent
		if err := protojson.Unmarshal(event, &pbEvent); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal searcher event")
		}
		pbEvents = append(pbEvents, &pbEvent)
	}
	return pbEvents, nil
}
