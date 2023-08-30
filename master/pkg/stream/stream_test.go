package stream

import (
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type BroadcastVerification struct {
	Upserted []*websocket.PreparedMessage
	Deleted  []*websocket.PreparedMessage
}

type TestMsg struct {
	Seq          int64
	Verification *BroadcastVerification
}

func (tm TestMsg) SeqNum() int64 {
	return tm.Seq
}

func (tm TestMsg) UpsertMsg() *websocket.PreparedMessage {
	message, err := websocket.NewPreparedMessage(websocket.BinaryMessage, []byte("upserted"))
	if err != nil {
		return nil
	}
	tm.Verification.Upserted = append(tm.Verification.Upserted, message)
	return message
}

func (tm TestMsg) DeleteMsg() *websocket.PreparedMessage {
	message, err := websocket.NewPreparedMessage(websocket.BinaryMessage, []byte("deleted"))
	if err != nil {
		return nil
	}
	tm.Verification.Deleted = append(tm.Verification.Deleted, message)
	return message
}

func TestConfigureSubscription(t *testing.T) {
	streamer := NewStreamer()
	publisher := NewPublisher[TestMsg]()
	sub := NewSubscription[TestMsg](streamer, publisher)
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
		"publisher's subscriptions are are nil after configuration")
	require.True(t, publisher.Subscriptions[0].filter(TestMsg{}),
		"publisher's subscription has the wrong filter")

	sub2 := NewSubscription[TestMsg](streamer, publisher)
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
	streamer := NewStreamer()
	publisher := NewPublisher[TestMsg]()
	trueSub := NewSubscription[TestMsg](streamer, publisher)
	falseSub := NewSubscription[TestMsg](streamer, publisher)
	trueSub.Configure(func(msg TestMsg) bool {
		return true
	})
	falseSub.Configure(func(msg TestMsg) bool {
		return false
	})
	verifyBroadcast := BroadcastVerification{
		Upserted: nil,
		Deleted:  nil,
	}
	afterMsg := TestMsg{
		Seq:          0,
		Verification: &verifyBroadcast,
	}
	event := Event[TestMsg]{After: &afterMsg}
	publisher.Broadcast([]Event[TestMsg]{event})

	require.Equal(t, 1, len(verifyBroadcast.Upserted), "message was not upserted")
	require.Equal(t, 0, len(verifyBroadcast.Deleted), "deleted messages non-zero")

	beforeMsg := TestMsg{
		Seq:          1,
		Verification: &verifyBroadcast,
	}
	event = Event[TestMsg]{Before: &beforeMsg}
	publisher.Broadcast([]Event[TestMsg]{event})
	require.Equal(t, 1, len(verifyBroadcast.Upserted), "upserted message incorrectly added")
	require.Equal(t, 1, len(verifyBroadcast.Deleted), "message was not deleted")
}
