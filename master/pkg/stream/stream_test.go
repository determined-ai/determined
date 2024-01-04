package stream

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	TestMsgUpsertKey   = "testmsg"
	TestMsgDeleteKey   = "testmsg_deleted"
	TestMsgJrUpsertKey = "testmsgjr"
	TestMsgJrDeleteKey = "testmsgjr_deleted"
)

type TestMsg struct {
	Seq int64
	ID  int
}

func (tm TestMsg) SeqNum() int64 {
	return tm.Seq
}

func (tm TestMsg) UpsertMsg() UpsertMsg {
	return UpsertMsg{
		JSONKey: TestMsgUpsertKey,
		Msg:     tm,
	}
}

func (tm TestMsg) DeleteMsg() DeleteMsg {
	deleted := strconv.FormatInt(int64(tm.ID), 10)
	return DeleteMsg{
		Key:     TestMsgDeleteKey,
		Deleted: deleted,
	}
}

// A second test message type to help test that msgs are being distinguished from each other.
type TestMsgJr struct {
	Seq int64
	ID  int
}

func (em TestMsgJr) SeqNum() int64 {
	return em.Seq
}

func (em TestMsgJr) UpsertMsg() UpsertMsg {
	return UpsertMsg{
		JSONKey: TestMsgJrUpsertKey,
		Msg:     em,
	}
}

func (em TestMsgJr) DeleteMsg() DeleteMsg {
	deleted := strconv.FormatInt(int64(em.ID), 10)
	return DeleteMsg{
		Key:     TestMsgJrDeleteKey,
		Deleted: deleted,
	}
}

func alwaysTrue(msg TestMsg) bool {
	return true
}

func alwaysTrueJr(msg TestMsgJr) bool {
	return true
}

func trueAfterTwo(msg TestMsg) bool {
	if msg.ID > 2 {
		return true
	}
	return false
}

func alwaysFalse(msg TestMsg) bool {
	return false
}

func prepareNothing(message PreparableMessage) interface{} {
	return message
}

func TestConfigureSubscription(t *testing.T) {
	dummyFilter := func(msg TestMsg) bool {
		return true
	}
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsg]()
	sub := NewSubscription[TestMsg](streamer, publisher, alwaysTrue, dummyFilter)
	require.True(t, sub.filter != nil, "subscription filter is nil after instantiation")
	require.True(t, len(publisher.Subscriptions) == 0,
		"publisher's subscriptions are non-nil after instantiation")

	sub.Register()
	require.True(t, sub.filter != nil, "subscription filter is nil after configuration")
	require.True(t, sub.filter(TestMsg{}), "set filter does not work")
	require.True(t, len(publisher.Subscriptions) == 1,
		"publisher's subscriptions are nil after configuration")
	for subscription := range publisher.Subscriptions {
		require.True(t, subscription.filter(TestMsg{}),
			"publisher's subscription has the wrong filter")
	}

	sub2 := NewSubscription[TestMsg](streamer, publisher, alwaysTrue, alwaysFalse)
	require.True(t, sub2.filter != nil, "subscription filter is nil after instantiation")

	sub2.Register()
	require.True(t, sub2.filter != nil, "subscription filter is nil after configuration")
	require.False(t, sub2.filter(TestMsg{}), "set filter does not work")
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
	publisher := NewPublisher[TestMsg]()
	trueSub := NewSubscription[TestMsg](streamer, publisher, alwaysTrue, alwaysTrue)
	falseSub := NewSubscription[TestMsg](streamer, publisher, alwaysTrue, alwaysFalse)
	trueSub.Register()
	falseSub.Register()
	afterMsg := TestMsg{
		Seq: 0,
		ID:  0,
	}
	event := Event[TestMsg]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsg]{event})

	require.Equal(t, 1, len(streamer.Msgs), "upsert message was not upserted")
	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
	require.True(t, ok, "message was not an upsert type")
	require.Equal(t, 0, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

	beforeMsg := TestMsg{
		Seq: 1,
		ID:  1,
	}
	event = Event[TestMsg]{Before: &beforeMsg}
	publisher.Broadcast([]Event[TestMsg]{event})
	require.Equal(t, 2, len(streamer.Msgs), "delete message was not upsert")
	deleteMsg, ok := streamer.Msgs[1].(DeleteMsg)
	require.True(t, ok, "message was not a delete type")
	require.Equal(t, "1", deleteMsg.Deleted)
}

func TestBroadcastWithFilters(t *testing.T) {
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsg]()
	publisherTwo := NewPublisher[TestMsg]()
	oneSub := NewSubscription[TestMsg](streamer, publisherTwo, alwaysTrue, trueAfterTwo)
	falseSub := NewSubscription[TestMsg](streamer, publisher, alwaysTrue, alwaysFalse)
	oneSub.Register()
	falseSub.Register()

	// Msgs sent on publisher should not be sent.
	afterMsg := TestMsg{
		Seq: 0,
		ID:  0,
	}
	event := Event[TestMsg]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsg]{event})
	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	beforeMsg := TestMsg{
		Seq: 1,
		ID:  1,
	}
	event = Event[TestMsg]{Before: &beforeMsg}
	publisher.Broadcast([]Event[TestMsg]{event})
	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	afterMsg = TestMsg{
		Seq: 20,
		ID:  20,
	}
	event = Event[TestMsg]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsg]{event})
	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	// Msgs sent on publisherTwo should be conditionally sent.
	afterMsg = TestMsg{
		Seq: 1,
		ID:  1,
	}
	event = Event[TestMsg]{After: &afterMsg}
	publisherTwo.Broadcast([]Event[TestMsg]{event})
	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	beforeMsg = TestMsg{
		Seq: 2,
		ID:  2,
	}
	event = Event[TestMsg]{Before: &beforeMsg}
	publisherTwo.Broadcast([]Event[TestMsg]{event})
	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	afterMsg = TestMsg{
		Seq: 3,
		ID:  3,
	}
	event = Event[TestMsg]{After: &afterMsg}
	publisherTwo.Broadcast([]Event[TestMsg]{event})

	require.Equal(t, 1, len(streamer.Msgs), "upsert message was not upserted")
	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
	require.True(t, ok, "message was not an upsert type")
	require.Equal(t, 3, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

	beforeMsg = TestMsg{
		Seq: 4,
		ID:  4,
	}
	event = Event[TestMsg]{Before: &beforeMsg}
	publisherTwo.Broadcast([]Event[TestMsg]{event})

	require.Equal(t, 2, len(streamer.Msgs), "upsert message was not upserted")
	deleteMsg, ok := streamer.Msgs[1].(DeleteMsg)
	require.True(t, ok, "message was not a delete type")
	require.Equal(t, "4", deleteMsg.Deleted, "Deleted number incorrect")

	// Msgs on publisher should not be sent
	afterMsg = TestMsg{
		Seq: 30,
		ID:  30,
	}
	event = Event[TestMsg]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsg]{event})

	require.Equal(t, 2, len(streamer.Msgs), "upsert message was not upserted")
}

func TestBroadcastWithPermissionFilters(t *testing.T) {
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsg]()
	publisherTwo := NewPublisher[TestMsg]()
	oneSub := NewSubscription[TestMsg](streamer, publisherTwo, trueAfterTwo, alwaysTrue)
	falseSub := NewSubscription[TestMsg](streamer, publisher, alwaysFalse, alwaysTrue)
	oneSub.Register()
	falseSub.Register()

	// Msgs sent on publisherTwo should be conditionally sent.
	afterMsg := TestMsg{
		Seq: 1,
		ID:  1,
	}
	event := Event[TestMsg]{After: &afterMsg}
	publisherTwo.Broadcast([]Event[TestMsg]{event})

	beforeMsg := TestMsg{
		Seq: 2,
		ID:  2,
	}
	event = Event[TestMsg]{Before: &beforeMsg}
	publisherTwo.Broadcast([]Event[TestMsg]{event})

	require.Equal(t, 0, len(streamer.Msgs), "picked up message we don't want")

	afterMsg = TestMsg{
		Seq: 3,
		ID:  3,
	}
	event = Event[TestMsg]{After: &afterMsg}
	publisherTwo.Broadcast([]Event[TestMsg]{event})

	require.Equal(t, 1, len(streamer.Msgs), "upsert message was not upserted")
	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
	require.True(t, ok, "message was not an upsert type")
	require.Equal(t, 3, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

	beforeMsg = TestMsg{
		Seq: 4,
		ID:  4,
	}
	event = Event[TestMsg]{Before: &beforeMsg}
	publisherTwo.Broadcast([]Event[TestMsg]{event})

	require.Equal(t, 2, len(streamer.Msgs), "upsert message was not upserted")
	deleteMsg, ok := streamer.Msgs[1].(DeleteMsg)
	require.True(t, ok, "message was not a delete type")
	require.Equal(t, "4", deleteMsg.Deleted, "Deleted number incorrect")

	// Msgs on publisher should not be sent.
	afterMsg = TestMsg{
		Seq: 3,
		ID:  3,
	}
	event = Event[TestMsg]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsg]{event})

	require.Equal(t, 2, len(streamer.Msgs), "upsert message was not upserted")
}

func TestBroadcastSeparateEvents(t *testing.T) {
	streamer := NewStreamer(prepareNothing)
	streamerTwo := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsg]()
	publisherTwo := NewPublisher[TestMsgJr]()
	publisherThree := NewPublisher[TestMsgJr]()
	trueSub := NewSubscription[TestMsg](streamer, publisher, alwaysTrue, alwaysTrue)
	separateSub := NewSubscription[TestMsgJr](streamerTwo, publisherTwo, alwaysTrueJr, alwaysTrueJr)
	togetherSub := NewSubscription[TestMsgJr](streamer, publisherThree, alwaysTrueJr, alwaysTrueJr)
	trueSub.Register()
	separateSub.Register()
	togetherSub.Register()

	// Msgs sent on publisher should be picked up.
	afterMsg := TestMsg{
		Seq: 0,
		ID:  0,
	}
	event := Event[TestMsg]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsg]{event})

	require.Equal(t, 1, len(streamer.Msgs), "picked up message we don't want")
	upsertMsg, ok := streamer.Msgs[0].(UpsertMsg)
	require.True(t, ok, "message was not an upsert type")
	require.Equal(t, 0, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

	beforeMsg := TestMsg{
		Seq: 1,
		ID:  1,
	}
	event = Event[TestMsg]{Before: &beforeMsg}
	publisher.Broadcast([]Event[TestMsg]{event})
	require.Equal(t, 2, len(streamer.Msgs), "picked up message we don't want")
	deleteMsg, ok := streamer.Msgs[1].(DeleteMsg)
	require.True(t, ok, "message was not a delete type")
	require.Equal(t, "1", deleteMsg.Deleted, "Deleted number incorrect")

	// Msgs sent on publisherTwo should not be picked up.
	afterMsgJr := TestMsgJr{
		Seq: 2,
		ID:  2,
	}
	eventJr := Event[TestMsgJr]{After: &afterMsgJr}
	publisherTwo.Broadcast([]Event[TestMsgJr]{eventJr})

	require.Equal(t, 2, len(streamer.Msgs), "picked up message we don't want")

	beforeMsgJr := TestMsgJr{
		Seq: 3,
		ID:  3,
	}
	eventJr = Event[TestMsgJr]{Before: &beforeMsgJr}
	publisherTwo.Broadcast([]Event[TestMsgJr]{eventJr})

	require.Equal(t, 2, len(streamer.Msgs), "picked up message we don't want")

	// Msgs sent onf publisherthree should be picked up.
	afterMsgJr = TestMsgJr{
		Seq: 4,
		ID:  4,
	}
	eventJr = Event[TestMsgJr]{After: &afterMsgJr}
	publisherThree.Broadcast([]Event[TestMsgJr]{eventJr})

	require.Equal(t, 3, len(streamer.Msgs), "upsert message was not upserted")
	upsertMsg, ok = streamer.Msgs[2].(UpsertMsg)
	require.True(t, ok, "message was not an upsert type")
	require.Equal(t, 4, int(upsertMsg.Msg.SeqNum()), "Sequence number incorrect")

	beforeMsgJr = TestMsgJr{
		Seq: 5,
		ID:  5,
	}
	eventJr = Event[TestMsgJr]{Before: &beforeMsgJr}
	publisherThree.Broadcast([]Event[TestMsgJr]{eventJr})

	require.Equal(t, 4, len(streamer.Msgs), "upsert message was not upserted")
	deleteMsg, ok = streamer.Msgs[3].(DeleteMsg)
	require.True(t, ok, "message was not a delete type")
	require.Equal(t, "5", deleteMsg.Deleted, "Deleted number incorrect")
}
