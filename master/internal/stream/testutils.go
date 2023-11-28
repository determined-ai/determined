package stream

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/determined-ai/determined/master/pkg/stream"
)

const (
	channelBufferSize = 10
)

// simpleUpsert is for testing and just returns the preparable message that the streamer sends.
func simpleUpsert(i stream.PreparableMessage) interface{} {
	return i
}

// mockSocket implements WebsocketLike and stores all messages received from streaming.
type mockSocket struct {
	inbound  chan interface{}
	outbound chan *StartupMsg
	closed   chan struct{}
}

// newMockSocket creates a new instance mockSocket and initialize it's conditional variables.
func newMockSocket() *mockSocket {
	return &mockSocket{
		inbound:  make(chan interface{}, channelBufferSize),
		outbound: make(chan *StartupMsg, channelBufferSize),
		closed:   make(chan struct{}),
	}
}

// WriteOut synchrounously appends the StartupMsg to outbound messages.
func (s *mockSocket) WriteOutbound(startup *StartupMsg) error {
	select {
	case <-s.closed:
		return &websocket.CloseError{Code: websocket.CloseAbnormalClosure}
	case s.outbound <- startup:
		return nil
	}
}

// ReadJSON implements WebsocketLike's ReadJSON(), blocks until able to read outbound messages off the mockSocket
// or the mockSocket is closed.
func (s *mockSocket) ReadJSON(data interface{}) error {
	select {
	case <-s.closed:
		return &websocket.CloseError{Code: websocket.CloseAbnormalClosure}
	case msg := <-s.outbound:
		targetMsg, ok := data.(*StartupMsg)
		if !ok {
			return fmt.Errorf("target message type is not a pointer to StartupMsg")
		}
		*targetMsg = *msg
		return nil
	}
}

// Write implements WebsocketLike's Write(), appends the data to mockSocket's inbound messages.
func (s *mockSocket) Write(data interface{}) error {
	select {
	case <-s.closed:
		return &websocket.CloseError{Code: websocket.CloseAbnormalClosure}
	case s.inbound <- data:
		return nil
	}
}

// ReadData synchrounously reads an inbound message off the mockSocket.
func (s *mockSocket) ReadIncoming(data *interface{}) error {
	select {
	case <-s.closed:
		return &websocket.CloseError{Code: websocket.CloseAbnormalClosure}
	case msg := <-s.inbound:
		*data = msg
		return nil
	}
}

// ReadUntil reads until the terminationMsg has been read.
func (s *mockSocket) ReadUntil(
	t *testing.T,
	testCaseDescription string, // XXX (corban): we should use subtests instead, this won't be necessary
	data *[]interface{},
	terminationMsg interface{},
) {
	var msg interface{}
ReadLoop:
	for {
		if reflect.TypeOf(msg) == reflect.TypeOf(terminationMsg) {
			switch typedMsg := msg.(type) {
			case stream.UpsertMsg:
				if err := validateUpsertMsg(typedMsg.Msg, terminationMsg.(stream.UpsertMsg).Msg); err == nil {
					break ReadLoop
				}
			case string:
				if typedMsg == terminationMsg.(string) {
					break ReadLoop
				}
			case SyncMsg:
				if typedMsg.SyncID == terminationMsg.(SyncMsg).SyncID {
					break ReadLoop
				}
			}
		}
		if err := s.ReadIncoming(&msg); err != nil {
			t.Errorf("%s: %s", testCaseDescription, err)
		}
		*data = append(*data, msg)
	}
}

// Close closes the mockSocket.
func (s *mockSocket) Close() {
	close(s.closed)
}

func splitMsgs[M stream.Msg](
	t *testing.T,
	testCaseDescription string,
	messages []interface{},
) (
	deletions []string,
	upserts []stream.Msg,
	syncs []SyncMsg,
) {
	typeHolder := new(M)
	for _, msg := range messages {
		switch typedMsg := msg.(type) {
		case stream.DeleteMsg:
			deletions = append(deletions, typedMsg.Deleted)
		case stream.UpsertMsg:
			upsertM, ok := typedMsg.Msg.(M)
			if !ok {
				t.Errorf("%s: expected %T, but received %T", testCaseDescription, typeHolder, typedMsg.Msg)
			}
			upserts = append(upserts, upsertM)
		case SyncMsg:
			syncs = append(syncs, typedMsg)
		default:
			t.Errorf("%s: expected a string or %T, but received %T",
				testCaseDescription,
				typeHolder,
				reflect.TypeOf(msg).Name(),
			)
		}
	}
	return deletions, upserts, syncs
}

func validateMsgs[M stream.Msg](
	t *testing.T,
	testCaseDescription string,
	sync SyncMsg,
	expectedSync SyncMsg,
	upserts []stream.Msg,
	expectedUpserts []M,
	deletions []string,
	expectedDeletions []string,
) {
	switch {
	// check if we received the correct number of trial messages
	case len(upserts) != len(expectedUpserts):
		t.Errorf(
			"%s: did not receive expected number of upsert messages: expected %d, actual: %d",
			testCaseDescription,
			len(expectedUpserts),
			len(upserts),
		)
	// check if we received the correct number of deletion messages
	case len(deletions) != len(expectedDeletions):
		t.Errorf(
			"%s: did not receive expected number of deletion messages: expected %v, actual: %v",
			testCaseDescription,
			len(expectedDeletions),
			len(deletions),
		)
	// check if we receieved the correct SyncMsg
	case sync.SyncID != expectedSync.SyncID:
		t.Errorf(
			"%s: did not receive expected sync message: expected: %v, actual: %v",
			testCaseDescription,
			expectedSync,
			sync,
		)
	// check if content of messages is correct
	default:
		// XXX: this expects messages to be sent in a deterministic order, is this actually enforced?
		// should msgs be sorted then?
		for i := range upserts {
			if err := validateUpsertMsg(upserts[i], expectedUpserts[i]); err != nil {
				t.Errorf("%s: %s", testCaseDescription, err.Error())
			}
		}
		// XXX: this expects messages to be sent in a deterministic order, is this actually enforced?
		// should deletions be sorted then?
		for i := range deletions {
			if deletions[i] != expectedDeletions[i] {
				t.Errorf(
					"%s: did not receive expected deletion messages: expected: %v, actual: %v",
					testCaseDescription,
					expectedDeletions,
					deletions,
				)
			}
		}
	}
}

func validateUpsertMsg(upsert stream.Msg, expectedUpsert stream.Msg) error {
	switch msg := upsert.(type) {
	case *TrialMsg:
		expectedMsg := expectedUpsert.(*TrialMsg)
		// XXX (corban): improve the completeness of this validation.
		// creating a `testString()` for each of for these upsert messages would be a good idea
		if msg.ID != expectedMsg.ID || msg.ExperimentID != expectedMsg.ExperimentID || msg.State != expectedMsg.State {
			return fmt.Errorf(
				"did not receive expected trial message: expected: %v, actual: %v",
				expectedMsg,
				msg,
			)
		}
		return nil
	case *MetricMsg:
		expectedMsg := expectedUpsert.(*MetricMsg)
		// XXX (corban): improve the completeness of this validation.
		if msg.ID != expectedMsg.ID || msg.ExperimentID != expectedMsg.ExperimentID {
			return fmt.Errorf(
				"did not receive expected metric message: expected: %v, actual: %v",
				expectedMsg,
				msg,
			)
		}
		return nil
	// case *ExperimentMsg:
	// 	msg := upsert.(*ExperimentMsg)
	// 	expectedMsg := expectedUpsert.(*ExperimentMsg)
	// 	if msg.ID != expectedMsg.ID || msg.WorkspaceID != expectedMsg.WorkspaceID {
	// 		return fmt.Errorf(
	//			"did not receive expected metric message: expected %v, actual: %v",
	//			expectedMsg,
	//			msg,
	// 		)
	// 	}
	// 	return nil
	default:
		return fmt.Errorf("upsert msg is not a valid type: %v", upsert)
	}
}
