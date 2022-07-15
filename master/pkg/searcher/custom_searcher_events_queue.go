package searcher

import (
	"fmt"

	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

// SearcherEventQueue stores the list of custom searcher events and the event
// that was event that was processed last by client and acknowledged by master.
type SearcherEventQueue struct {
	events               []*experimentv1.SearcherEvent
	lastProcessedEventID int32 // this indicates that the client has received the event
	// and has sent the operations to master and is acknowledged by master.
}

func newSearcherEventQueue() *SearcherEventQueue {
	events := make([]*experimentv1.SearcherEvent, 0)
	return &SearcherEventQueue{events: events, lastProcessedEventID: -1}
}

// GetLastProcessedEventID get last processed event id.
func (q *SearcherEventQueue) GetLastProcessedEventID() int32 {
	return q.lastProcessedEventID
}

// SetLastProcessedEventID set last processed event id.
func (q *SearcherEventQueue) SetLastProcessedEventID(processedEvent *experimentv1.SearcherEvent) {
	q.lastProcessedEventID = processedEvent.Id
}

// Enqueue an event.
func (q *SearcherEventQueue) Enqueue(event *experimentv1.SearcherEvent) error {
	q.events = append(q.events, event)
	return nil
}

// GetEvents returns all the events.
func (q *SearcherEventQueue) GetEvents() []*experimentv1.SearcherEvent {
	return q.events
}

// RemoveUpTo the given event Id.
func (q *SearcherEventQueue) RemoveUpTo(eventID int) error {
	for i, v := range q.events {
		if v.Id == int32(eventID) {
			q.events = q.events[i:]
			return nil
		}
	}
	return fmt.Errorf("event %d not found", eventID)
}
