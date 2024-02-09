package stream

import (
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

func alwaysFalse[T Msg](msg T) bool {
	return false
}

func prepareNothing(message MarshallableMsg) interface{} {
	return message
}

func TestConfigureSubscription(t *testing.T) {
	dummyFilter := func(msg TestMsgTypeA) bool {
		return true
	}
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsgTypeA]()
	sub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], dummyFilter)
	require.True(t, sub.filter != nil, "subscription filter is nil after instantiation")
	require.True(t, len(publisher.Subscriptions) == 0,
		"publisher's subscriptions are non-nil after instantiation")

	sub.Register()
	require.True(t, sub.filter != nil, "subscription filter is nil after configuration")
	require.True(t, sub.filter(TestMsgTypeA{}), "set filter does not work")
	require.True(t, len(publisher.Subscriptions) == 1,
		"publisher's subscriptions are nil after configuration")
	for subscription := range publisher.Subscriptions {
		require.True(t, subscription.filter(TestMsgTypeA{}),
			"publisher's subscription has the wrong filter")
	}

	sub2 := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA])
	require.True(t, sub2.filter != nil, "subscription filter is nil after instantiation")

	sub2.Register()
	require.True(t, sub2.filter != nil, "subscription filter is nil after configuration")
	require.False(t, sub2.filter(TestMsgTypeA{}), "set filter does not work")
	require.True(t, len(publisher.Subscriptions) == 2,
		"publisher's subscriptions are nil after configuration")

	_, ok := publisher.Subscriptions[&sub2]
	require.True(t, ok, "publisher has correct new subscription")

	sub.Unregister()
	require.True(t, len(publisher.Subscriptions) == 1,
		"publisher's still has subscriptions after deletion")
	_, ok = publisher.Subscriptions[&sub]
	require.False(t, ok, "publisher removed the wrong subscription")
}

func TestBroadcast(t *testing.T) {
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsgTypeA]()
	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA])
	falseSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysFalse[TestMsgTypeA])
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
	// oneSub's filter expects to return true after receiving trueAfterCount messages
	trueAfterCount := 2
	oneSub := NewSubscription[TestMsgTypeA](
		streamer,
		publisherTwo,
		alwaysTrue[TestMsgTypeA],
		trueAfterN[TestMsgTypeA](trueAfterCount),
	)
	falseSub := NewSubscription[TestMsgTypeA](
		streamer,
		publisher,
		alwaysTrue[TestMsgTypeA],
		alwaysFalse[TestMsgTypeA],
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
	// oneSub's permission filter will return true after receiving trueAfterCount messages
	trueAfterCount := 2
	oneSub := NewSubscription[TestMsgTypeA](
		streamer,
		publisherTwo,
		trueAfterN[TestMsgTypeA](trueAfterCount),
		alwaysTrue[TestMsgTypeA],
	)
	falseSub := NewSubscription[TestMsgTypeA](
		streamer,
		publisher,
		alwaysFalse[TestMsgTypeA],
		alwaysTrue[TestMsgTypeA],
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
	trueSub := NewSubscription[TestMsgTypeA](streamer, publisher, alwaysTrue[TestMsgTypeA], alwaysTrue[TestMsgTypeA])
	separateSub := NewSubscription[TestMsgTypeB](
		streamerTwo, publisherTwo, alwaysTrue[TestMsgTypeB], alwaysTrue[TestMsgTypeB])
	togetherSub := NewSubscription[TestMsgTypeB](
		streamer, publisherThree, alwaysTrue[TestMsgTypeB], alwaysTrue[TestMsgTypeB])
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
