package rmevents

import (
	"sync/atomic"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
)

var syslog = logrus.WithField("component", "rmevents")

const eventBufferSize = 1024

type subscribeRequest struct {
	topic model.AllocationID
	id    int64
	inbox *queue.Queue[sproto.ResourcesEvent]
}

type unsubscribeRequest struct {
	topic model.AllocationID
	id    int64
}

type eventWithTopic struct {
	topic model.AllocationID
	event sproto.ResourcesEvent
}

type manager struct {
	id          atomic.Int64
	events      chan<- eventWithTopic
	subEvents   chan<- subscribeRequest
	unsubEvents chan<- unsubscribeRequest
}

func newManager() *manager {
	in := make(chan eventWithTopic, eventBufferSize)
	// This channel is used to synchronize receipt of unsubscription
	// with draining our updates channel, do not buffer it.
	subs := make(chan subscribeRequest)
	unsubs := make(chan unsubscribeRequest)
	go fanOut(in, subs, unsubs)
	return &manager{events: in, subEvents: subs, unsubEvents: unsubs}
}

func (m *manager) subscribe(topic model.AllocationID) *sproto.ResourcesSubscription {
	id := m.id.Add(1)
	inbox := queue.New[sproto.ResourcesEvent]()
	m.subEvents <- subscribeRequest{topic: topic, id: id, inbox: inbox}
	return sproto.NewAllocationSubscription(inbox, func() {
		m.unsubEvents <- unsubscribeRequest{topic: topic, id: id}
	})
}

func (m *manager) publish(topic model.AllocationID, event sproto.ResourcesEvent) {
	m.events <- eventWithTopic{topic: topic, event: event}
}

func fanOut(
	in <-chan eventWithTopic,
	subs <-chan subscribeRequest,
	unsubs <-chan unsubscribeRequest,
) {
	subsByTopicByID := map[model.AllocationID]map[int64]*queue.Queue[sproto.ResourcesEvent]{}
	for {
		select {
		case msg := <-in:
			syslog.Tracef("sending %T to %s", msg.event, msg.topic)
			send(subsByTopicByID, msg)
		case msg := <-subs:
			syslog.Tracef("subscribing %s:%d", msg.topic, msg.id)
			sub(subsByTopicByID, msg)
		case msg := <-unsubs:
			syslog.Tracef("unsubscribing %s:%d", msg.topic, msg.id)
			unsub(subsByTopicByID, msg)
		}
	}
}

func send(
	subsByTopicByID map[model.AllocationID]map[int64]*queue.Queue[sproto.ResourcesEvent],
	msg eventWithTopic,
) {
	subs, ok := subsByTopicByID[msg.topic]
	if !ok {
		syslog.Warnf("dropping message for %s with no subs", msg.topic)
		return
	}
	for _, c := range subs {
		c.Put(msg.event)
	}
}

func sub(
	subsByTopicByID map[model.AllocationID]map[int64]*queue.Queue[sproto.ResourcesEvent],
	msg subscribeRequest,
) {
	if _, ok := subsByTopicByID[msg.topic]; !ok {
		subsByTopicByID[msg.topic] = map[int64]*queue.Queue[sproto.ResourcesEvent]{}
	}
	subsByTopicByID[msg.topic][msg.id] = msg.inbox
}

func unsub(
	subsByTopicByID map[model.AllocationID]map[int64]*queue.Queue[sproto.ResourcesEvent],
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
