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

func prepareNothing(message PreparableMessage) interface{} {
	return message
}

func TestConfigureSubscription(t *testing.T) {
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsg]()
	sub := NewSubscription[TestMsg](streamer, publisher, alwaysTrue)
	require.True(t, sub.filter == nil, "subscription filter is non nil after instantiation")
	require.True(t, len(publisher.Subscriptions) == 0,
		"publisher's subscriptions are non-nil after instantiation")
	dummyFilter := func(msg TestMsg) bool {
		return true
	}
	sub.Configure(dummyFilter)
	require.True(t, sub.filter != nil, "subscription filter is nil after configuration")
	require.True(t, sub.filter(TestMsg{}), "set filter does not work")
	require.True(t, len(publisher.Subscriptions) == 1,
		"publisher's subscriptions are nil after configuration")
	require.True(t, publisher.Subscriptions[0].filter(TestMsg{}),
		"publisher's subscription has the wrong filter")

	sub2 := NewSubscription[TestMsg](streamer, publisher, alwaysTrue)
	require.True(t, sub2.filter == nil, "subscription filter is non nil after instantiation")
	sub2.Configure(func(msg TestMsg) bool {
		return false
	})
	require.True(t, sub2.filter != nil, "subscription filter is nil after configuration")
	require.False(t, sub2.filter(TestMsg{}), "set filter does not work")
	require.True(t, len(publisher.Subscriptions) == 2,
		"publisher's subscriptions are nil after configuration")
	require.False(t, publisher.Subscriptions[1].filter(TestMsg{}),
		"publisher's subscription has the wrong filter")

	sub.Configure(nil)
	require.True(t, sub.filter == nil, "subscription filter is not nil after deletion")
	require.True(t, len(publisher.Subscriptions) == 1,
		"publisher's still has subscriptions after deletion")
	require.False(t, publisher.Subscriptions[0].filter(TestMsg{}),
		"publisher removed the wrong subscription")

	sub2.Configure(func(msg TestMsg) bool {
		return true
	})
	require.True(t, len(publisher.Subscriptions) == 1,
		"publisher should have replaced the subscription filter")
	require.True(t, publisher.Subscriptions[0].filter(TestMsg{}),
		"filter still set to the old filter")
}

func TestBroadcast(t *testing.T) {
	streamer := NewStreamer(prepareNothing)
	publisher := NewPublisher[TestMsg]()
	trueSub := NewSubscription[TestMsg](streamer, publisher, alwaysTrue)
	falseSub := NewSubscription[TestMsg](streamer, publisher, alwaysTrue)
	trueSub.Configure(func(msg TestMsg) bool {
		return true
	})
	falseSub.Configure(func(msg TestMsg) bool {
		return false
	})
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
