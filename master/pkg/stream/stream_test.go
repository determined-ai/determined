package stream

import (
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	TestMsgAUpsertKey = "testmsgA"
	TestMsgADeleteKey = "testmsgA_deleted"
	TestMsgBUpsertKey = "testmsgB"
	TestMsgBDeleteKey = "testmsgB_deleted"
)

type TestMsgTypeA struct {
	Seq int64
	ID  int
}

func (tm TestMsgTypeA) GetID() int {
	return tm.ID
}

func (tm TestMsgTypeA) SeqNum() int64 {
	return tm.Seq
}

func (tm TestMsgTypeA) UpsertMsg() UpsertMsg {
	return UpsertMsg{
		JSONKey: TestMsgAUpsertKey,
		Msg:     tm,
	}
}

func (tm TestMsgTypeA) DeleteMsg() DeleteMsg {
	deleted := strconv.FormatInt(int64(tm.ID), 10)
	return DeleteMsg{
		Key:     TestMsgADeleteKey,
		Deleted: deleted,
	}
}

// A second test message type to help test that msgs are being distinguished from each other.
type TestMsgTypeB struct {
	Seq int64
	ID  int
}

func (tm TestMsgTypeB) GetID() int {
	return tm.ID
}

func (em TestMsgTypeB) SeqNum() int64 {
	return em.Seq
}

func (em TestMsgTypeB) UpsertMsg() UpsertMsg {
	return UpsertMsg{
		JSONKey: TestMsgBUpsertKey,
		Msg:     em,
	}
}

func (em TestMsgTypeB) DeleteMsg() DeleteMsg {
	deleted := strconv.FormatInt(int64(em.ID), 10)
	return DeleteMsg{
		Key:     TestMsgBDeleteKey,
		Deleted: deleted,
	}
}

type TestEvent struct {
	Type          string
	FallinUserID  []int
	FalloutUserID []int
	BeforeSeq     int64
	AfterSeq      int64
}

func alwaysTrue[T Msg](msg T) bool {
	return true
}

func trueAfterN[T Msg](n int) func(T) bool {
	var msgCount int
	return func(T) bool {
		msgCount++
		return msgCount > n
	}
}

func falseAtAndAfterSeq[T Msg](seq int64) func(T) bool {
	return func(msg T) bool {
		return msg.SeqNum() < seq
	}
}

func alwaysFalse[T Msg](msg T) bool {
	return false
}

type TestSubscriber struct {
	ID       int
	Streamer *Streamer
}

func prepareNothing(message MarshallableMsg) interface{} {
	return message
}

func TestConfigureSubscription(t *testing.T) {
	dummyFilter := func(msg TestMsgTypeA) bool {
		return true
	}
	dummyHydrator := func(ID int) (TestMsgTypeA, error) {
		return TestMsgTypeA{}, nil
	}
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsgTypeA]()
	sub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], dummyFilter, dummyHydrator)
	require.True(t, sub.filter != nil, "subscription filter is nil after instantiation")
	require.True(t, len(publisher.Subscriptions) == 0,
		"publisher's subscriptions are non-nil after instantiation")

	sub.Register()
	require.True(t, sub.filter != nil, "subscription filter is nil after configuration")
	require.True(t, sub.filter(TestMsgTypeA{}), "set filter does not work")
	require.True(t, len(publisher.Subscriptions) == 1,
		"publisher's subscriptions are nil after configuration")
	for _, subscription := range publisher.Subscriptions {
		require.True(t, subscription.filter(TestMsgTypeA{}),
			"publisher's subscription has the wrong filter")
	}

	sub2 := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA], dummyHydrator)
	require.True(t, sub2.filter != nil, "subscription filter is nil after instantiation")

	sub2.Register()
	require.True(t, sub2.filter != nil, "subscription filter is nil after configuration")
	require.False(t, sub2.filter(TestMsgTypeA{}), "set filter does not work")
	require.True(t, len(publisher.Subscriptions) == 2,
		"publisher's subscriptions are nil after configuration")

	ok := slices.Contains(publisher.Subscriptions, &sub2)
	require.True(t, ok, "publisher has correct new subscription")

	sub.Unregister()
	require.True(t, len(publisher.Subscriptions) == 1,
		"publisher's still has subscriptions after deletion")
	ok = slices.Contains(publisher.Subscriptions, &sub)
	require.False(t, ok, "publisher removed the wrong subscription")
}

func TestBroadcast(t *testing.T) {
	hydrator := func(ID int) (TestMsgTypeA, error) {
		return TestMsgTypeA{
			Seq: int64(ID),
			ID:  ID,
		}, nil
	}
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsgTypeA]()
	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA], hydrator)
	falseSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA], hydrator)
	trueSub.Register()
	falseSub.Register()
	afterMsg := TestMsgTypeA{
		Seq: 0,
		ID:  0,
	}
	event := Event[TestMsgTypeA]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsgTypeA]{event})

	require.Equal(t, 1, len(streamer.Msgs), "upsert message was not upserted")
	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
	require.True(t, ok, "message was not an upsert type")
	require.Equal(t, 0, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

	beforeMsg := TestMsgTypeA{
		Seq: 1,
		ID:  1,
	}
	event = Event[TestMsgTypeA]{Before: &beforeMsg}
	publisher.Broadcast([]Event[TestMsgTypeA]{event})
	require.Equal(t, 2, len(streamer.Msgs), "delete message was not upsert")
	deleteMsg, ok := streamer.Msgs[1].(DeleteMsg)
	require.True(t, ok, "message was not a delete type")
	require.Equal(t, "1", deleteMsg.Deleted)
}

func TestBroadcastWithFilters(t *testing.T) {
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsgTypeA]()
	publisherTwo := NewPublisher[TestMsgTypeA]()
	hydrator := func(ID int) (TestMsgTypeA, error) {
		return TestMsgTypeA{
			Seq: int64(ID),
			ID:  ID,
		}, nil
	}
	// oneSub's filter expects to return true after receiving trueAfterCount messages
	trueAfterCount := 2
	oneSub := NewSubscription[TestMsgTypeA](
		streamer,
		publisherTwo,
		alwaysTrue[TestMsgTypeA],
		trueAfterN[TestMsgTypeA](trueAfterCount),
		hydrator,
	)
	falseSub := NewSubscription[TestMsgTypeA](
		streamer,
		publisher,
		alwaysTrue[TestMsgTypeA],
		alwaysFalse[TestMsgTypeA],
		hydrator,
	)
	oneSub.Register()
	falseSub.Register()

	// Msgs sent on publisher should not be sent.
	afterMsg := TestMsgTypeA{
		Seq: 0,
		ID:  0,
	}
	event := Event[TestMsgTypeA]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsgTypeA]{event})
	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	beforeMsg := TestMsgTypeA{
		Seq: 1,
		ID:  1,
	}
	event = Event[TestMsgTypeA]{Before: &beforeMsg}
	publisher.Broadcast([]Event[TestMsgTypeA]{event})
	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	afterMsg = TestMsgTypeA{
		Seq: 20,
		ID:  20,
	}
	event = Event[TestMsgTypeA]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsgTypeA]{event})
	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	// Msgs sent on publisherTwo should be conditionally sent.
	afterMsg = TestMsgTypeA{
		Seq: 1,
		ID:  1,
	}
	event = Event[TestMsgTypeA]{After: &afterMsg}
	publisherTwo.Broadcast([]Event[TestMsgTypeA]{event})
	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	beforeMsg = TestMsgTypeA{
		Seq: 2,
		ID:  2,
	}
	event = Event[TestMsgTypeA]{Before: &beforeMsg}
	publisherTwo.Broadcast([]Event[TestMsgTypeA]{event})
	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	afterMsg = TestMsgTypeA{
		Seq: 3,
		ID:  3,
	}
	event = Event[TestMsgTypeA]{After: &afterMsg}
	publisherTwo.Broadcast([]Event[TestMsgTypeA]{event})

	require.Equal(t, 1, len(streamer.Msgs), "upsert message was not upserted")
	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
	require.True(t, ok, "message was not an upsert type")
	require.Equal(t, 3, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

	beforeMsg = TestMsgTypeA{
		Seq: 4,
		ID:  4,
	}
	event = Event[TestMsgTypeA]{Before: &beforeMsg}
	publisherTwo.Broadcast([]Event[TestMsgTypeA]{event})

	require.Equal(t, 2, len(streamer.Msgs), "upsert message was not upserted")
	deleteMsg, ok := streamer.Msgs[1].(DeleteMsg)
	require.True(t, ok, "message was not a delete type")
	require.Equal(t, "4", deleteMsg.Deleted, "Deleted number incorrect")

	// Msgs on publisher should not be sent
	afterMsg = TestMsgTypeA{
		Seq: 30,
		ID:  30,
	}
	event = Event[TestMsgTypeA]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsgTypeA]{event})

	require.Equal(t, 2, len(streamer.Msgs), "upsert message was not upserted")
}

func TestBroadcastWithPermissionFilters(t *testing.T) {
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsgTypeA]()
	publisherTwo := NewPublisher[TestMsgTypeA]()
	hydrator := func(ID int) (TestMsgTypeA, error) {
		return TestMsgTypeA{
			Seq: int64(ID),
			ID:  ID,
		}, nil
	}
	// oneSub's permission filter will return true after receiving trueAfterCount messages
	trueAfterCount := 2
	oneSub := NewSubscription[TestMsgTypeA](
		streamer,
		publisherTwo,
		trueAfterN[TestMsgTypeA](trueAfterCount),
		alwaysTrue[TestMsgTypeA],
		hydrator,
	)
	falseSub := NewSubscription[TestMsgTypeA](
		streamer,
		publisher,
		alwaysFalse[TestMsgTypeA],
		alwaysTrue[TestMsgTypeA],
		hydrator,
	)
	oneSub.Register()
	falseSub.Register()

	// Msgs sent on publisherTwo should be conditionally sent.
	afterMsg := TestMsgTypeA{
		Seq: 1,
		ID:  1,
	}
	event := Event[TestMsgTypeA]{After: &afterMsg}
	publisherTwo.Broadcast([]Event[TestMsgTypeA]{event})

	beforeMsg := TestMsgTypeA{
		Seq: 2,
		ID:  2,
	}
	event = Event[TestMsgTypeA]{Before: &beforeMsg}
	publisherTwo.Broadcast([]Event[TestMsgTypeA]{event})

	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	afterMsg = TestMsgTypeA{
		Seq: 3,
		ID:  3,
	}
	event = Event[TestMsgTypeA]{After: &afterMsg}
	publisherTwo.Broadcast([]Event[TestMsgTypeA]{event})

	require.Equal(t, 1, len(streamer.Msgs), "upsert message was not upserted")
	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
	require.True(t, ok, "message was not an upsert type")
	require.Equal(t, 3, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

	beforeMsg = TestMsgTypeA{
		Seq: 4,
		ID:  4,
	}
	event = Event[TestMsgTypeA]{Before: &beforeMsg}
	publisherTwo.Broadcast([]Event[TestMsgTypeA]{event})

	require.Equal(t, 2, len(streamer.Msgs), "upsert message was not upserted")
	deleteMsg, ok := streamer.Msgs[1].(DeleteMsg)
	require.True(t, ok, "message was not a delete type")
	require.Equal(t, "4", deleteMsg.Deleted, "Deleted number incorrect")

	// Msgs on publisher should not be sent.
	afterMsg = TestMsgTypeA{
		Seq: 3,
		ID:  3,
	}
	event = Event[TestMsgTypeA]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsgTypeA]{event})

	require.Equal(t, 2, len(streamer.Msgs), "upsert message was not upserted")
}

func TestBroadcastSeparateEvents(t *testing.T) {
	streamer := NewStreamer(prepareNothing)
	streamerTwo := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsgTypeA]()
	publisherTwo := NewPublisher[TestMsgTypeB]()
	publisherThree := NewPublisher[TestMsgTypeB]()
	hydratorA := func(ID int) (TestMsgTypeA, error) {
		return TestMsgTypeA{
			Seq: int64(ID),
			ID:  ID,
		}, nil
	}
	hydratorB := func(ID int) (TestMsgTypeB, error) {
		return TestMsgTypeB{
			Seq: int64(ID),
			ID:  ID,
		}, nil
	}
	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA], hydratorA)
	separateSub := NewSubscription[TestMsgTypeB](
		streamerTwo, publisherTwo, alwaysTrue[TestMsgTypeB], alwaysTrue[TestMsgTypeB], hydratorB)
	togetherSub := NewSubscription[TestMsgTypeB](
		streamer, publisherThree, alwaysTrue[TestMsgTypeB], alwaysTrue[TestMsgTypeB], hydratorB)
	trueSub.Register()
	separateSub.Register()
	togetherSub.Register()

	// Msgs sent on publisher should be picked up.
	afterMsg := TestMsgTypeA{
		Seq: 0,
		ID:  0,
	}
	event := Event[TestMsgTypeA]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsgTypeA]{event})

	require.Equal(t, 1, len(streamer.Msgs), "picked up message we don't want")
	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
	require.True(t, ok, "message was not an upsert type")
	require.Equal(t, 0, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

	beforeMsg := TestMsgTypeA{
		Seq: 1,
		ID:  1,
	}
	event = Event[TestMsgTypeA]{Before: &beforeMsg}
	publisher.Broadcast([]Event[TestMsgTypeA]{event})
	require.Equal(t, 2, len(streamer.Msgs), "picked up message we don't want")
	deleteMsg, ok := streamer.Msgs[1].(DeleteMsg)
	require.True(t, ok, "message was not a delete type")
	require.Equal(t, "1", deleteMsg.Deleted, "Deleted number incorrect")

	// Msgs sent on publisherTwo should not be picked up.
	afterMsgB := TestMsgTypeB{
		Seq: 2,
		ID:  2,
	}
	eventB := Event[TestMsgTypeB]{After: &afterMsgB}
	publisherTwo.Broadcast([]Event[TestMsgTypeB]{eventB})

	require.Equal(t, 2, len(streamer.Msgs), "picked up message we don't want")

	beforeMsgB := TestMsgTypeB{
		Seq: 3,
		ID:  3,
	}
	eventB = Event[TestMsgTypeB]{Before: &beforeMsgB}
	publisherTwo.Broadcast([]Event[TestMsgTypeB]{eventB})

	require.Equal(t, 2, len(streamer.Msgs), "picked up message we don't want")

	// Msgs sent onf publisherthree should be picked up.
	afterMsgB = TestMsgTypeB{
		Seq: 4,
		ID:  4,
	}
	eventB = Event[TestMsgTypeB]{After: &afterMsgB}
	publisherThree.Broadcast([]Event[TestMsgTypeB]{eventB})

	require.Equal(t, 3, len(streamer.Msgs), "upsert message was not upserted")
	upsertMsg, ok = streamer.Msgs[2].(UpsertMsg)
	require.True(t, ok, "message was not an upsert type")
	require.Equal(t, 4, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

	beforeMsgB = TestMsgTypeB{
		Seq: 5,
		ID:  5,
	}
	eventB = Event[TestMsgTypeB]{Before: &beforeMsgB}
	publisherThree.Broadcast([]Event[TestMsgTypeB]{eventB})

	require.Equal(t, 4, len(streamer.Msgs), "upsert message was not upserted")
	deleteMsg, ok = streamer.Msgs[3].(DeleteMsg)
	require.True(t, ok, "message was not a delete type")
	require.Equal(t, "5", deleteMsg.Deleted, "Deleted number incorrect")
}

// func TestMultipleUpserts(t *testing.T) {
// 	streamer := NewStreamer(prepareNothing)
// 	publisher := NewPublisher[TestMsgTypeA]()

// 	afterMsg1 := TestMsgTypeA{
// 		Seq: 0,
// 		ID:  0,
// 	}
// 	afterMsg2 := TestMsgTypeA{
// 		Seq: 1,
// 		ID:  0,
// 	}
// 	afterMsg3 := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	afterMsg4 := TestMsgTypeA{
// 		Seq: 1,
// 		ID:  1,
// 	}
// 	// Two update messages, afterMsg1 and afterMsg2.
// 	// Hydrate the first one getting the second update's content.
// 	hydrator := func(ID int) (TestMsgTypeA, error) {
// 		return TestMsgTypeA{
// 			Seq: 1,
// 			ID:  ID,
// 		}, nil
// 	}
// 	event := Event[TestMsgTypeA]{After: &afterMsg1}
// 	event2 := Event[TestMsgTypeA]{After: &afterMsg2}
// 	event3 := Event[TestMsgTypeA]{After: &afterMsg3}
// 	event4 := Event[TestMsgTypeA]{After: &afterMsg4}
// 	events := []Event[TestMsgTypeA]{event, event2}

// 	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA], hydrator)
// 	falseSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA], hydrator)
// 	trueSub.Register()
// 	falseSub.Register()

// 	publisher.Broadcast(events)

// 	require.Equal(t, 1, len(streamer.Msgs), "streamer.Msgs length incorrect")
// 	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 1, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

// 	// Three update messages, afterMsg1, afterMsg2 and afterMsg3.
// 	// Expect hydrator to be called twice.
// 	// Hydrate afterMsg1 getting afterMsg2's content.
// 	// Hydrate afterMsg3 getting afterMsg3's content.
// 	fmt.Println("testing second case")
// 	streamer = NewStreamer(prepareNothing)
// 	publisher = NewPublisher[TestMsgTypeA]()

// 	seqs := []int64{1, 2}
// 	hydrator = func(ID int) (TestMsgTypeA, error) {
// 		seq := seqs[0]
// 		seqs = seqs[1:]
// 		return TestMsgTypeA{
// 			Seq: seq,
// 			ID:  ID,
// 		}, nil
// 	}
// 	trueSub = NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA], hydrator)
// 	falseSub = NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA], hydrator)
// 	trueSub.Register()
// 	falseSub.Register()

// 	events = []Event[TestMsgTypeA]{event, event2, event3}
// 	publisher.Broadcast(events)

// 	require.Equal(t, 2, len(streamer.Msgs), "streamer.Msgs length incorrect")
// 	upsertMsg, ok = streamer.Msgs[0].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 1, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")
// 	upsertMsg, ok = streamer.Msgs[1].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 2, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

// 	// Three update messages for ID 0, afterMsg1, afterMsg2 and afterMsg3.
// 	// One update message for ID 1
// 	// Special hydrate handling only applies to ID 0: afterMsg1 getting afterMsg3's content.
// 	// Expect hydrator to be called once for ID 0.
// 	fmt.Println("----------")
// 	streamer = NewStreamer(prepareNothing)
// 	publisher = NewPublisher[TestMsgTypeA]()

// 	hydrator = func(ID int) (TestMsgTypeA, error) {
// 		if ID == 0 {
// 			return TestMsgTypeA{
// 				Seq: 2,
// 				ID:  ID,
// 			}, nil
// 		} else {
// 			return TestMsgTypeA{
// 				Seq: int64(ID),
// 				ID:  ID,
// 			}, nil
// 		}
// 	}
// 	trueSub = NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA], hydrator)
// 	falseSub = NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA], hydrator)
// 	trueSub.Register()
// 	falseSub.Register()

// 	events = []Event[TestMsgTypeA]{event4, event, event2, event3}
// 	publisher.Broadcast(events)

// 	require.Equal(t, 2, len(streamer.Msgs), "streamer.Msgs length incorrect")
// 	upsertMsg, ok = streamer.Msgs[0].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 1, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")
// 	upsertMsg, ok = streamer.Msgs[1].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 2, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")
// }

// func TestUpsertsAndDelete(t *testing.T) {
// 	streamer := NewStreamer(prepareNothing)
// 	publisher := NewPublisher[TestMsgTypeA]()

// 	afterMsg1 := TestMsgTypeA{
// 		Seq: 1,
// 		ID:  0,
// 	}
// 	afterMsg2 := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	// afterMsg3 := TestMsgTypeA{
// 	// 	Seq: 2,
// 	// 	ID:  0,
// 	// }
// 	// afterMsg4 := TestMsgTypeA{
// 	// 	Seq: 1,
// 	// 	ID:  1,
// 	// }
// 	beforeMsg := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	// 4. Update(1) Update(2) Delete(2) on the same project.
// 	// 	1. Update(1) -> check map false -> hydrate on Update(1) -> store map[id] = 1
// 	// 	2. Update(2) -> check map true -> seq > cached seq -> hydrate on Update(2) -> store map[id] = 2
// 	// 	3. Delete(2)
// 	seqs := []int64{1, 2}
// 	hydrator := func(ID int) (TestMsgTypeA, error) {
// 		seq := seqs[0]
// 		// fmt.Printf("hydrator seq: %+v\n", seq)
// 		seqs = seqs[1:]
// 		// fmt.Printf("hydrator seqs: %+v\n", seqs)
// 		return TestMsgTypeA{
// 			Seq: seq,
// 			ID:  ID,
// 		}, nil
// 	}

// 	event := Event[TestMsgTypeA]{After: &afterMsg1}
// 	event2 := Event[TestMsgTypeA]{After: &afterMsg2}
// 	event5 := Event[TestMsgTypeA]{Before: &beforeMsg}
// 	events := []Event[TestMsgTypeA]{event, event2, event5}

// 	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA], hydrator)
// 	falseSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA], hydrator)
// 	trueSub.Register()
// 	falseSub.Register()

// 	publisher.Broadcast(events)

// 	require.Equal(t, 3, len(streamer.Msgs), "streamer.Msgs length incorrect")
// 	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 1, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")
// 	upsertMsg, ok = streamer.Msgs[1].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 2, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")
// 	deleteMsg, ok := streamer.Msgs[2].(DeleteMsg)
// 	require.True(t, ok, "message was not a delete type")
// 	require.Equal(t, "0", deleteMsg.Deleted)
// }

// func TestUpsertsAndDelete2(t *testing.T) {
// 	// 5. Update(1) Update(2) Delete(2) on the same project.
// 	// 	1. Update(1) -> check map false -> hydrate on Delete(2) -> store map[id] = -1
// 	// 	2. Update(2) -> check map true -> seq == -1 -> skip
// 	// 	3. Delete(2)
// 	streamer := NewStreamer(prepareNothing)
// 	publisher := NewPublisher[TestMsgTypeA]()
// 	afterMsg1 := TestMsgTypeA{
// 		Seq: 1,
// 		ID:  0,
// 	}
// 	afterMsg2 := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	beforeMsg := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	hydrator := func(ID int) (TestMsgTypeA, error) {
// 		return TestMsgTypeA{}, sql.ErrNoRows
// 	}

// 	event := Event[TestMsgTypeA]{After: &afterMsg1}
// 	event2 := Event[TestMsgTypeA]{After: &afterMsg2}
// 	event5 := Event[TestMsgTypeA]{Before: &beforeMsg}
// 	events := []Event[TestMsgTypeA]{event, event2, event5}

// 	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA], hydrator)
// 	falseSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA], hydrator)
// 	trueSub.Register()
// 	falseSub.Register()

// 	publisher.Broadcast(events)

// 	require.Equal(t, 1, len(streamer.Msgs), "streamer.Msgs length incorrect")
// 	deleteMsg, ok := streamer.Msgs[0].(DeleteMsg)
// 	require.True(t, ok, "message was not a delete type")
// 	require.Equal(t, "0", deleteMsg.Deleted)
// }

// func TestUpsertsAndFallout(t *testing.T) {
// 	// 6. Update(1) Update(2) Fallout(3) (Fallout is a special upsert event)
// 	// 	1. Update(1) -> check map false -> hydrate on Fallout(3) -> store map[id] = -1
// 	// 	2. Update(2) -> check map true -> seq == -1 -> skip
// 	// 	3. Fallout(3)
// 	streamer := NewStreamer(prepareNothing)
// 	publisher := NewPublisher[TestMsgTypeA]()
// 	afterMsg1 := TestMsgTypeA{
// 		Seq: 1,
// 		ID:  0,
// 	}
// 	afterMsg2 := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	beforeMsg := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	hydrator := func(ID int) (TestMsgTypeA, error) {
// 		return TestMsgTypeA{
// 			Seq: 3,
// 			ID:  0,
// 		}, nil
// 	}

// 	event := Event[TestMsgTypeA]{After: &afterMsg1}
// 	event2 := Event[TestMsgTypeA]{After: &afterMsg2}
// 	event5 := Event[TestMsgTypeA]{Before: &beforeMsg}
// 	events := []Event[TestMsgTypeA]{event, event2, event5}

// 	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, trueAtNs[TestMsgTypeA]([]int{1, 3, 4}), alwaysTrue[TestMsgTypeA], hydrator)
// 	falseSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA], hydrator)
// 	trueSub.Register()
// 	falseSub.Register()

// 	publisher.Broadcast(events)

// 	require.Equal(t, 1, len(streamer.Msgs), "streamer.Msgs length incorrect")
// 	deleteMsg, ok := streamer.Msgs[0].(DeleteMsg)
// 	require.True(t, ok, "message was not a delete type")
// 	require.Equal(t, "0", deleteMsg.Deleted)
// }

// func TestUpsertsAndFallout2(t *testing.T) {
// 	// 7. Update(1) Update(2) Fallout(3) (Fallout is a special upsert event) on id 0, Fallin(1) (Fallin is a special upsert event) on id 1
// 	// 	1. Update(1) -> check map false -> hydrate on Update(2) -> store map[0] = 2
// 	// 	2. Update(2) -> check map true -> seq <= cached seq -> skip
// 	// 	3. Fallout(3)
// 	// 	4. Fallin(2) -> check map false -> hydrate on Fallin(1) -> store map[1] = 2
// 	streamer := NewStreamer(prepareNothing)
// 	publisher := NewPublisher[TestMsgTypeA]()
// 	afterMsg1 := TestMsgTypeA{
// 		Seq: 1,
// 		ID:  0,
// 	}
// 	afterMsg2 := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	beforeMsg := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	afterMsg3 := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  1,
// 	}
// 	seqs := []int{2, 2, 2}
// 	hydrator := func(ID int) (TestMsgTypeA, error) {
// 		seq := seqs[0]
// 		seqs = seqs[1:]
// 		return TestMsgTypeA{
// 			Seq: int64(seq),
// 			ID:  ID,
// 		}, nil
// 	}

// 	event := Event[TestMsgTypeA]{After: &afterMsg1}
// 	event2 := Event[TestMsgTypeA]{After: &afterMsg2}
// 	event3 := Event[TestMsgTypeA]{Before: &beforeMsg}
// 	event4 := Event[TestMsgTypeA]{After: &afterMsg3}
// 	events := []Event[TestMsgTypeA]{event, event2, event3, event4}

// 	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA], hydrator)
// 	falseSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA], hydrator)
// 	trueSub.Register()
// 	falseSub.Register()

// 	publisher.Broadcast(events)

// 	require.Equal(t, 3, len(streamer.Msgs), "streamer.Msgs length incorrect")
// 	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 2, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")
// 	deleteMsg, ok := streamer.Msgs[1].(DeleteMsg)
// 	require.True(t, ok, "message was not a delete type")
// 	require.Equal(t, "0", deleteMsg.Deleted)
// 	upsertMsg, ok = streamer.Msgs[2].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 2, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")
// }

// // 8. Update(1) Update(2) Fallout(3) (Fallout is a special upsert event) Fallin(4) (Fallin is a special upsert event) on id 0
// //  1. Update(1) -> check map false -> hydrate on Fallout(3) -> store map[0] = 3
// //  2. Update(2) -> check map true -> cached seq > seq -> skip
// //  3. Fallout(3)
// //  4. Fallin(4) -> check map true -> cached seq < seq -> hydrate on Fallin(4) -> store map[0] = 4
// func TestUpsertsAndFallout3(t *testing.T) {
// 	streamer := NewStreamer(prepareNothing)
// 	publisher := NewPublisher[TestMsgTypeA]()
// 	afterMsg1 := TestMsgTypeA{
// 		Seq: 1,
// 		ID:  0,
// 	}
// 	afterMsg2 := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	beforeMsg3 := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	afterMsg3 := TestMsgTypeA{
// 		Seq: 3,
// 		ID:  0,
// 	}
// 	beforeMsg4 := TestMsgTypeA{
// 		Seq: 3,
// 		ID:  0,
// 	}
// 	afterMsg4 := TestMsgTypeA{
// 		Seq: 4,
// 		ID:  0,
// 	}
// 	seqs := []int{3, 4}
// 	hydrator := func(ID int) (TestMsgTypeA, error) {
// 		seq := seqs[0]
// 		seqs = seqs[1:]
// 		return TestMsgTypeA{
// 			Seq: int64(seq),
// 			ID:  ID,
// 		}, nil
// 	}

// 	event := Event[TestMsgTypeA]{After: &afterMsg1}
// 	event2 := Event[TestMsgTypeA]{After: &afterMsg2}
// 	event3 := Event[TestMsgTypeA]{Before: &beforeMsg3, After: &afterMsg3}
// 	event4 := Event[TestMsgTypeA]{Before: &beforeMsg4, After: &afterMsg4}
// 	events := []Event[TestMsgTypeA]{event, event2, event3, event4}

// 	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, trueAtNs[TestMsgTypeA]([]int{1, 3, 5, 6, 7}), alwaysTrue[TestMsgTypeA], hydrator)
// 	falseSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA], hydrator)
// 	trueSub.Register()
// 	falseSub.Register()

// 	publisher.Broadcast(events)

// 	require.Equal(t, 2, len(streamer.Msgs), "streamer.Msgs length incorrect")
// 	deleteMsg, ok := streamer.Msgs[0].(DeleteMsg)
// 	require.True(t, ok, "message was not an delete type")
// 	require.Equal(t, "0", deleteMsg.Deleted)
// 	upsertMsg, ok := streamer.Msgs[1].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 4, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")
// }

// // 9. Update(1) on id 0, Update(2) on id 0.
// // Subscriber1:
// //  1. Update(1) -> check cache false -> hydrate on Update(1) -> save cache and send message. Filter 1 true, 2 true
// //  2. Update(2) -> check cache true -> seq > cached seq -> hydrate on Update(2), found it's a Fallout
// //     and send message. Filter 3 true, 4 false.
// //
// // Subscriber2:
// //  1. Update(1) -> check cache true -> seq < cached seq -> do nothing. Filter 5 true.
// //  2. Update(2) -> check cache true -> seq = cached seq -> use cache, send message. Filter 6 true.
// func TestUpdateAndFallout(t *testing.T) {
// 	streamer := NewStreamer(prepareNothing)
// 	publisher := NewPublisher[TestMsgTypeA]()
// 	afterMsg1 := TestMsgTypeA{
// 		Seq: 1,
// 		ID:  0,
// 	}
// 	afterMsg2 := TestMsgTypeA{
// 		Seq: 2,
// 		ID:  0,
// 	}
// 	seqs := []int{1, 2}
// 	hydrator := func(ID int) (TestMsgTypeA, error) {
// 		seq := seqs[0]
// 		seqs = seqs[1:]
// 		return TestMsgTypeA{
// 			Seq: int64(seq),
// 			ID:  ID,
// 		}, nil
// 	}

// 	event := Event[TestMsgTypeA]{After: &afterMsg1}
// 	event2 := Event[TestMsgTypeA]{After: &afterMsg2}
// 	events := []Event[TestMsgTypeA]{event, event2}

// 	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, trueAtNs[TestMsgTypeA]([]int{1, 2, 3, 5, 6}), alwaysTrue[TestMsgTypeA], hydrator)
// 	falseSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA], hydrator)
// 	trueSub.Register()
// 	falseSub.Register()

// 	publisher.Broadcast(events)

// 	require.Equal(t, 3, len(streamer.Msgs), "streamer.Msgs length incorrect")
// 	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 1, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")
// 	deleteMsg, ok := streamer.Msgs[1].(DeleteMsg)
// 	require.True(t, ok, "message was not an delete type")
// 	require.Equal(t, "0", deleteMsg.Deleted)
// 	upsertMsg, ok = streamer.Msgs[2].(UpsertMsg)
// 	require.True(t, ok, "message was not an upsert type")
// 	require.Equal(t, 2, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")
// }

func setup(testEvents []TestEvent, testSubscribers []TestSubscriber) {
	publisher := NewPublisher[TestMsgTypeA]()

	var events []Event[TestMsgTypeA]
	userToFalloutSeq := make(map[int]int64)

	for _, testEvent := range testEvents {
		var event Event[TestMsgTypeA]
		switch testEvent.Type {
		case "update", "insert", "fallin":
			event = Event[TestMsgTypeA]{
				Before: &TestMsgTypeA{
					Seq: int64(testEvent.AfterSeq - 1),
					ID:  0,
				},
				After: &TestMsgTypeA{
					Seq: int64(testEvent.AfterSeq),
					ID:  0,
				}}
		case "fallout":
			event = Event[TestMsgTypeA]{
				Before: &TestMsgTypeA{
					Seq: int64(testEvent.AfterSeq - 1),
					ID:  0,
				},
				After: &TestMsgTypeA{
					Seq: int64(testEvent.AfterSeq),
					ID:  0,
				}}
			for _, userID := range testEvent.FalloutUserID {
				userToFalloutSeq[userID] = int64(testEvent.AfterSeq)
			}
		case "delete":
			event = Event[TestMsgTypeA]{Before: &TestMsgTypeA{
				Seq: int64(testEvent.BeforeSeq),
				ID:  0,
			}}
		}
		events = append(events, event)
	}

	// Setting fallout seq for users do not have a fallout event. It's for creating subscription filter.
	for _, ts := range testSubscribers {
		if _, ok := userToFalloutSeq[ts.ID]; !ok {
			userToFalloutSeq[ts.ID] = int64(len(testEvents) + 1)
		}
	}
	fmt.Printf("userToFalloutSeq: %+v\n", userToFalloutSeq)

	// Setting seqs for the mocked hydrator
	var seqs []int64
	entityDeleted := false
	for _, ts := range testSubscribers {
		numOfEvents := len(testEvents)

		hasFellout := false
		index := 0
		for range numOfEvents {
			testEvent := testEvents[index]

			if testEvent.Type == "insert" {
				seqs = append(seqs, testEvent.AfterSeq)
				testEvents = append(testEvents[:index], testEvents[index+1:]...)

			} else if testEvent.Type == "update" && !hasFellout {
				if testEvent.FalloutUserID != nil && len(testEvent.FalloutUserID) > 0 {
					if ts.ID != testEvent.UserID {
						if !slices.Contains(seqs, testEvent.AfterSeq) {
							seqs = append(seqs, testEvent.AfterSeq)
						}
						index += 1
					} else {
						hasFellout = true
						// This entity has fell out. The remaining events are not relavent to the
						// user.
						// TODO: this is not true when we have tests with fallin events.
						break
					}

				}

			} else if testEvent.Type == "delete" {
				entityDeleted = true
				testEvents = append(testEvents[:index], testEvents[index+1:]...)
				break
			}
		}
		if entityDeleted {
			// break because entity deleted, doesn't need hydration anymore.
			break
		}
	}
	fmt.Printf("seqs: %+v\n", seqs)

	hydrator := func(ID int) (TestMsgTypeA, error) {
		seq := seqs[0]
		seqs = seqs[1:]
		return TestMsgTypeA{
			Seq: int64(seq),
			ID:  ID,
		}, nil
	}

	for _, ts := range testSubscribers {
		subscriber := NewSubscription[TestMsgTypeA](ts.Streamer, publisher, falseAtAndAfterSeq[TestMsgTypeA](userToFalloutSeq[ts.ID]), alwaysTrue[TestMsgTypeA], hydrator)
		subscriber.Register()
	}

	publisher.Broadcast(events)
}

// Up to four DB events are included in the TestTwoSubscribers. Update on id 0, subscriber1 fallout on id 0,
// subsriber2 fallout on id 0, delete id 0. Permutate these events to generate the tests in this function.
func TestTwoSubscribers(t *testing.T) {
	type testCase struct {
		description  string
		dBEvents     []TestEvent
		outGoingMsgs []interface{}
	}

	tcs := []testCase{
		{
			description: "1. insert id 0(1), update on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "insert", AfterSeq: 1},
				{Type: "update", AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "2. insert id 0(1), subscriber1 fallin on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "insert", AfterSeq: 1},
				{Type: "update", FallinUserID: []int{1}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "3. insert id 0(1), subscriber2 fallin on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "insert", AfterSeq: 1},
				{Type: "update", FallinUserID: []int{2}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "4. insert id 0(1), subscriber1 fallout on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "insert", AfterSeq: 1},
				{Type: "update", FalloutUserID: []int{1}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "5. insert id 0(1), subscriber2 fallout on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "insert", AfterSeq: 1},
				{Type: "update", FalloutUserID: []int{2}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "6. insert id 0(1), delete id 0(2)",
			dBEvents: []TestEvent{
				{Type: "insert", AfterSeq: 1},
				{Type: "delete", BeforeSeq: 1},
			},
			outGoingMsgs: []interface{}{
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "7. update on id 0(1), subscriber1 fallin on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", AfterSeq: 1},
				{Type: "update", FallinUserID: []int{1}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "8. update on id 0(1), subscriber2 fallin on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", AfterSeq: 1},
				{Type: "update", FallinUserID: []int{2}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "9. update on id 0(1), subscriber1 fallout on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", AfterSeq: 1},
				{Type: "update", FalloutUserID: []int{1}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "10. update on id 0(1), subscriber2 fallout on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", AfterSeq: 1},
				{Type: "update", FalloutUserID: []int{2}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "11. update on id 0(1), delete id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", AfterSeq: 1},
				{Type: "delete", BeforeSeq: 1},
			},
			outGoingMsgs: []interface{}{
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "12. subscriber1 fallin on id 0(1), update on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FallinUserID: []int{1}, AfterSeq: 1},
				{Type: "update", AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "13. subscriber1 fallin on id 0(1), subscriber2 fallin on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FallinUserID: []int{1}, AfterSeq: 1},
				{Type: "update", FallinUserID: []int{2}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "14. subscriber1 fallin on id 0(1), subscriber1 fallout on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FallinUserID: []int{1}, AfterSeq: 1},
				{Type: "update", FalloutUserID: []int{1}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "15. subscriber1 fallin on id 0(1), subscriber2 fallout on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FallinUserID: []int{1}, AfterSeq: 1},
				{Type: "update", FalloutUserID: []int{2}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "16. subscriber1 fallin on id 0(1), delete id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FallinUserID: []int{1}, AfterSeq: 1},
				{Type: "delete", BeforeSeq: 1},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "17. subscriber2 fallin on id 0(1), update on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FallinUserID: []int{2}, AfterSeq: 1},
				{Type: "update", AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "18. subscriber2 fallin on id 0(1), subscriber1 fallin on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FallinUserID: []int{2}, AfterSeq: 1},
				{Type: "update", FallinUserID: []int{1}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "19. subscriber2 fallin on id 0(1), subscriber1 fallout on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FallinUserID: []int{2}, AfterSeq: 1},
				{Type: "update", FalloutUserID: []int{1}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "20. subscriber2 fallin on id 0(1), subscriber2 fallout on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FallinUserID: []int{2}, AfterSeq: 1},
				{Type: "update", FalloutUserID: []int{2}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "21. subscriber2 fallin on id 0(1), delete id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FallinUserID: []int{2}, AfterSeq: 1},
				{Type: "delete", BeforeSeq: 1},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "22. subscriber1 fallout on id 0(1), update on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FalloutUserID: []int{1}, AfterSeq: 1},
				{Type: "update", AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "23. subscriber1 fallout on id 0(1), subscriber1 fallin on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FalloutUserID: []int{1}, AfterSeq: 1},
				{Type: "update", FallinUserID: []int{1}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "24. subscriber1 fallout on id 0(1), subscriber2 fallin on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FalloutUserID: []int{1}, AfterSeq: 1},
				{Type: "update", FallinUserID: []int{2}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "25. subscriber1 fallout on id 0(1), subscriber2 fallout on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FalloutUserID: []int{1}, AfterSeq: 1},
				{Type: "update", FalloutUserID: []int{2}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "26. subscriber1 fallout on id 0(1), delete id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FalloutUserID: []int{1}, AfterSeq: 1},
				{Type: "delete", BeforeSeq: 1},
			},
			outGoingMsgs: []interface{}{
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "27. subscriber2 fallout on id 0(1), update on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FalloutUserID: []int{2}, AfterSeq: 1},
				{Type: "update", AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "28. subscriber2 fallout on id 0(1), subscriber1 fallin on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FalloutUserID: []int{2}, AfterSeq: 1},
				{Type: "update", FallinUserID: []int{1}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "29. subscriber2 fallout on id 0(1), subscriber2 fallin on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FalloutUserID: []int{2}, AfterSeq: 1},
				{Type: "update", FallinUserID: []int{2}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{

				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
				DeleteMsg{Deleted: "0"},
				UpsertMsg{Msg: TestMsgTypeA{2, 0}},
			},
		},
		{
			description: "30. subscriber2 fallout on id 0(1), subscriber1 fallout on id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FalloutUserID: []int{2}, AfterSeq: 1},
				{Type: "update", FalloutUserID: []int{1}, AfterSeq: 2},
			},
			outGoingMsgs: []interface{}{
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
				DeleteMsg{Deleted: "0"},
			},
		},
		{
			description: "31. subscriber2 fallout on id 0(1), delete id 0(2)",
			dBEvents: []TestEvent{
				{Type: "update", FalloutUserID: []int{2}, AfterSeq: 1},
				{Type: "delete", BeforeSeq: 1},
			},
			outGoingMsgs: []interface{}{
				UpsertMsg{Msg: TestMsgTypeA{1, 0}},
				DeleteMsg{Deleted: "0"},
				DeleteMsg{Deleted: "0"},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.description, func(t *testing.T) {
			testSubscribers := []TestSubscriber{{1, NewStreamer(prepareNothing)}, {2, NewStreamer(prepareNothing)}}
			setup(
				tc.dBEvents,
				testSubscribers,
			)

			var streamerMsgs []interface{}
			for _, ts := range testSubscribers {
				streamerMsgs = append(streamerMsgs, ts.Streamer.Msgs...)
			}
			require.Equal(t, len(tc.outGoingMsgs), len(streamerMsgs), "streamer.Msgs length incorrect")

			for i, o := range tc.outGoingMsgs {
				switch o.(type) {
				case UpsertMsg:
					upsertMsg, ok := streamerMsgs[i].(UpsertMsg)
					require.True(t, ok, "message was not an upsert type")
					require.Equal(t, o.(UpsertMsg).Msg.SeqNum(), upsertMsg.Msg.SeqNum(), "Sequence number incorrect")
				case DeleteMsg:
					deleteMsg, ok := streamerMsgs[i].(DeleteMsg)
					require.True(t, ok, "message was not an delete type")
					require.Equal(t, "0", deleteMsg.Deleted)
				}
			}
		})
	}
}
