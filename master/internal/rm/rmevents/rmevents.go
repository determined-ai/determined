package rmevents

import (
	"sync"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TODO(!!!): Must add tests and review intensely.

const bufferSize = 64

type eventWithTopic struct {
	topic model.AllocationID
	event sproto.AllocationEvent
}

type subscribeRequest struct {
	topic   model.AllocationID
	id      int
	updates chan<- sproto.AllocationEvent
}

type unsubscribeRequest struct {
	topic model.AllocationID
	id    int
}

type changeSubRequest interface{ SubEvent() }

func (subscribeRequest) SubEvent()   {}
func (unsubscribeRequest) SubEvent() {}

type manager struct {
	id        sequence
	events    chan<- eventWithTopic
	subEvents chan<- changeSubRequest // sub or unsub request
}

func newManager() *manager {
	in := make(chan eventWithTopic, bufferSize)
	// This channel is used to synchronize receipt of unsubscription
	// with draining our updates channel, do not buffer it.
	subs := make(chan changeSubRequest)
	m := &manager{events: in, subEvents: subs}
	go fanOut(in, subs)
	return m
}

func (m *manager) subscribe(topic model.AllocationID) *sproto.AllocationSubscription {
	id := m.id.next()
	updates := make(chan sproto.AllocationEvent, bufferSize)
	m.subEvents <- subscribeRequest{topic: topic, id: id, updates: updates}
	return sproto.NewAllocationSubscription(updates, func() {
		// fire off the unsub request asynchronously and drain the channel, in the event
		// we stopped consuming, our channel was full, and the fanOut routine is blocked
		// sending to us.
		done := make(chan struct{})
		go func() {
			m.subEvents <- unsubscribeRequest{topic: topic, id: id}
			close(done)
		}()
		for {
			select {
			case <-updates:
			case <-done:
				return
			}
		}
	})
}

func (m *manager) publish(topic model.AllocationID, event sproto.AllocationEvent) {
	m.events <- eventWithTopic{topic: topic, event: event}
}

func fanOut(in <-chan eventWithTopic, subs <-chan changeSubRequest) {
	subsByTopicByID := map[model.AllocationID]map[int]chan<- sproto.AllocationEvent{}
	for {
		select {
		case msg := <-in:
			send(subsByTopicByID, msg)
		case msg := <-subs:
			changeSubs(subsByTopicByID, msg)
		}
	}
}

func send(subsByTopicByID map[model.AllocationID]map[int]chan<- sproto.AllocationEvent, msg eventWithTopic) {
	subs, ok := subsByTopicByID[msg.topic]
	if !ok {
		return
	}
	for _, c := range subs {
		c <- msg.event // TODO: some kind of fail-safe. Timeout?
	}
}

func changeSubs(subsByTopicByID map[model.AllocationID]map[int]chan<- sproto.AllocationEvent, msg changeSubRequest) {
	switch msg := msg.(type) {
	case subscribeRequest:
		sub(subsByTopicByID, msg)
	case unsubscribeRequest:
		unsub(subsByTopicByID, msg)
	}
}

func sub(subsByTopicByID map[model.AllocationID]map[int]chan<- sproto.AllocationEvent, msg subscribeRequest) {
	if _, ok := subsByTopicByID[msg.topic]; !ok {
		subsByTopicByID[msg.topic] = map[int]chan<- sproto.AllocationEvent{}
	}
	subsByTopicByID[msg.topic][msg.id] = msg.updates
}

func unsub(subsByTopicByID map[model.AllocationID]map[int]chan<- sproto.AllocationEvent, msg unsubscribeRequest) {
	updates, ok := subsByTopicByID[msg.topic][msg.id]
	if !ok {
		return
	}

	close(updates)
	delete(subsByTopicByID[msg.topic], msg.id)
	if len(subsByTopicByID[msg.topic]) == 0 {
		delete(subsByTopicByID, msg.topic)
	}
	return
}

type sequence struct {
	mu sync.Mutex
	i  int
}

func (s *sequence) next() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.i++
	return s.i
}
