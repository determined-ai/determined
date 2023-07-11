package rmevents

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestRMEvents(t *testing.T) {
	rand.Seed(1337)

	numTopics, numSubsPerTopic := 10, 10

	var topics []model.AllocationID
	for topicID := 0; topicID < numTopics; topicID++ {
		topics = append(topics, model.AllocationID(strconv.Itoa(topicID)))
	}

	t.Logf("starting %d subs each for %d topics", numSubsPerTopic, numTopics)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var subs []*sproto.AllocationSubscription
	results := map[model.AllocationID]map[int][]sproto.AllocationEvent{}
	for _, topic := range topics {
		topic := topic

		mu.Lock()
		results[topic] = map[int][]sproto.AllocationEvent{}
		mu.Unlock()

		for subID := 0; subID < numSubsPerTopic; subID++ {
			subID := subID
			sub := Subscribe(topic)
			subs = append(subs, sub)

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer sub.Close()
				for {
					ev := sub.Get()
					t.Logf("%s:%d got %T", topic, subID, ev)
					if ev == (sproto.SentinelAllocationEvent{}) {
						return
					}
					mu.Lock()
					results[topic][subID] = append(results[topic][subID], ev)
					mu.Unlock()
				}
			}()
		}
	}

	iterations := 1000
	t.Logf("sending %d messages on random topics", iterations)
	expected := map[model.AllocationID][]sproto.AllocationEvent{}
	for i := 0; i < iterations; i++ {
		topicID := rand.Int63n(int64(len(topics)))
		topic := model.AllocationID(strconv.Itoa(int(topicID)))

		log := strconv.Itoa(i)
		msg := sproto.ContainerLog{AuxMessage: &log}
		Publish(topic, msg)
		t.Logf("published %T to %s", msg, topic)
		expected[topic] = append(expected[topic], msg)
	}

	t.Log("closing subs and waiting on background goroutines")
	for _, topic := range topics {
		Publish(topic, sproto.SentinelAllocationEvent{})
	}
	wg.Wait()

	t.Log("checking results")
	for topic, subResults := range results {
		for _, actual := range subResults {
			require.Len(t, actual, len(expected[topic]))
			require.ElementsMatch(t, expected[topic], actual)
		}
	}
}
