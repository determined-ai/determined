package stream

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	TestMsgUpsertKey = "testmsg"
	TestMsgDeleteKey = "testmsg_deleted"
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

func alwaysTrue(msg TestMsg) bool {
	return true
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
	require.True(t, publisher.Subscriptions[0].filter(TestMsg{}),
		"publisher's subscription has the wrong filter")

	sub2 := NewSubscription[TestMsg](streamer, publisher, alwaysTrue, alwaysFalse)
	require.True(t, sub2.filter != nil, "subscription filter is nil after instantiation")

	sub2.Register()
	require.True(t, sub2.filter != nil, "subscription filter is nil after configuration")
	require.False(t, sub2.filter(TestMsg{}), "set filter does not work")
	require.True(t, len(publisher.Subscriptions) == 2,
		"publisher's subscriptions are nil after configuration")
	require.False(t, publisher.Subscriptions[1].filter(TestMsg{}),
		"publisher's subscription has the wrong filter")

	sub.Unregister()
	require.True(t, len(publisher.Subscriptions) == 1,
		"publisher's still has subscriptions after deletion")
	require.False(t, publisher.Subscriptions[0].filter(TestMsg{}),
		"publisher removed the wrong subscription")
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
