package rmevents

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
)

// TODO(!!!): Must add tests and review intensely.

const (
	mainBufferSize = 1024
)

type subscribeRequest struct {
	topic model.AllocationID
	id    int
	inbox *queue.Queue[sproto.AllocationEvent]
}

type unsubscribeRequest struct {
	topic model.AllocationID
	id    int
}

type eventWithTopic struct {
	topic model.AllocationID
	event sproto.AllocationEvent
}

type manager struct {
	id          sequence
	events      chan<- eventWithTopic
	subEvents   chan<- subscribeRequest
	unsubEvents chan<- unsubscribeRequest
}

func newManager() *manager {
	in := make(chan eventWithTopic, mainBufferSize)
	// This channel is used to synchronize receipt of unsubscription
	// with draining our updates channel, do not buffer it.
	subs := make(chan subscribeRequest)
	unsubs := make(chan unsubscribeRequest)
	go fanOut(in, subs, unsubs)
	return &manager{events: in, subEvents: subs, unsubEvents: unsubs}
}

func (m *manager) subscribe(topic model.AllocationID) *sproto.AllocationSubscription {
	id := m.id.next()
	inbox := queue.New[sproto.AllocationEvent]()
	m.subEvents <- subscribeRequest{topic: topic, id: id, inbox: inbox}
	return sproto.NewAllocationSubscription(inbox, func() {
		m.unsubEvents <- unsubscribeRequest{topic: topic, id: id}
	})
}

func (m *manager) publish(topic model.AllocationID, event sproto.AllocationEvent) {
	m.events <- eventWithTopic{topic: topic, event: event}
}

func fanOut(
	in <-chan eventWithTopic,
	subs <-chan subscribeRequest,
	unsubs <-chan unsubscribeRequest,
) {
	subsByTopicByID := map[model.AllocationID]map[int]*queue.Queue[sproto.AllocationEvent]{}
	for {
		select {
		case msg := <-in:
			send(subsByTopicByID, msg)
		case msg := <-subs:
			sub(subsByTopicByID, msg)
		case msg := <-unsubs:
			unsub(subsByTopicByID, msg)
		}
	}
}

func send(
	subsByTopicByID map[model.AllocationID]map[int]*queue.Queue[sproto.AllocationEvent],
	msg eventWithTopic,
) {
	subs, ok := subsByTopicByID[msg.topic]
	if !ok {
		logrus.Warnf("dropping message for %s with no subs", msg.topic)
		return
	}
	for _, c := range subs {
		c.Put(msg.event)
	}
}

func sub(
	subsByTopicByID map[model.AllocationID]map[int]*queue.Queue[sproto.AllocationEvent],
	msg subscribeRequest,
) {
	if _, ok := subsByTopicByID[msg.topic]; !ok {
		subsByTopicByID[msg.topic] = map[int]*queue.Queue[sproto.AllocationEvent]{}
	}
	subsByTopicByID[msg.topic][msg.id] = msg.inbox
}

func unsub(
	subsByTopicByID map[model.AllocationID]map[int]*queue.Queue[sproto.AllocationEvent],
	msg unsubscribeRequest,
) {
	_, ok := subsByTopicByID[msg.topic][msg.id]
	if !ok {
		return
	}

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
