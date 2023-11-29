package stream

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/determined-ai/determined/master/pkg/stream"
)

const (
	channelBufferSize = 10
	syncKey           = "sync_msg"
)

// testPrepareFunc returns a string representation of known messages;
// otherwise, returns the preparable message that the streamer sends.
func testPrepareFunc(i stream.PreparableMessage) interface{} {
	switch msg := i.(type) {
	case stream.UpsertMsg:
		switch typedMsg := msg.Msg.(type) {
		case *TrialMsg:
			return fmt.Sprintf(
				"%s (%d): %s %d %d",
				TrialsUpsertKey,
				typedMsg.ID,
				typedMsg.State,
				typedMsg.ExperimentID,
				typedMsg.WorkspaceID,
			)
		case *MetricMsg:
			return fmt.Sprintf(
				"%s (%d): %t %d %d",
				MetricsUpsertKey,
				typedMsg.ID,
				typedMsg.Archived,
				typedMsg.ExperimentID,
				typedMsg.WorkspaceID,
			)
			// case *ExperimentMsg:
			// 	return fmt.Sprintf(
			// 		"%d: %s %d %d",
			// 		typedMsg.ID,
			// 		typedMsg.State,
			// 		typedMsg.ProjectID,
			// 		typedMsg.WorkspaceID,
			// 	)
		}
	case stream.DeleteMsg:
		return fmt.Sprintf("%s: %s", msg.Key, msg.Deleted)
	case SyncMsg:
		return fmt.Sprintf("%s: %s", syncKey, msg.SyncID)
	}
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
func (s *mockSocket) WriteOutbound(t *testing.T, startup *StartupMsg) {
	select {
	case <-s.closed:
		t.Error(&websocket.CloseError{Code: websocket.CloseAbnormalClosure})
	case s.outbound <- startup:
		break
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
func (s *mockSocket) ReadIncoming(t *testing.T, data *string) {
	select {
	case <-s.closed:
		t.Error(&websocket.CloseError{Code: websocket.CloseAbnormalClosure})
	case msg := <-s.inbound:
		stringMsg, ok := msg.(string)
		if !ok {
			t.Errorf("read unexpected message, likely due to type not being added to testPrepareFunc: %v", msg)
		}
		*data = stringMsg
	}
}

// ReadUntil reads until the terminationMsg has been read.
func (s *mockSocket) ReadUntil(
	t *testing.T,
	data *[]string,
	terminationMsg string,
) {
	msg := ""
	for {
		s.ReadIncoming(t, &msg)
		*data = append(*data, msg)
		if msg == terminationMsg {
			break
		}
	}
}

// Close closes the mockSocket.
func (s *mockSocket) Close() {
	close(s.closed)
}

func splitMsgs(
	t *testing.T,
	messages []string,
) (
	deletions []string,
	upserts []string,
	syncs []string,
) {
	upsertKeys := []string{
		TrialsUpsertKey,
		MetricsUpsertKey,
		// ExperimentUpsertKey,
	}
	deleteKeys := []string{
		TrialsDeleteKey,
		MetricsDeleteKey,
		// ExperimentDeleteKey,
	}

	for i := range upsertKeys {
		upsertKeys[i] = "^" + upsertKeys[i]
	}

	for i := range deleteKeys {
		deleteKeys[i] = "^" + deleteKeys[i]
	}

	upsertPattern := regexp.MustCompile(
		strings.Join(upsertKeys, "|"),
	)
	deletePattern := regexp.MustCompile(
		strings.Join(deleteKeys, "|"),
	)
	syncPattern := regexp.MustCompile("^" + syncKey)

	for _, msg := range messages {
		switch {
		case deletePattern.MatchString(msg):
			deletions = append(deletions, msg)
		case upsertPattern.MatchString(msg):
			upserts = append(upserts, msg)
		case syncPattern.MatchString(msg):
			syncs = append(syncs, msg)
		default:
			t.Errorf("unknown message type: %s", msg)
		}
	}
	return deletions, upserts, syncs
}

func listToSet(l []string) map[string]struct{} {
	set := make(map[string]struct{}, len(l))
	for _, val := range l {
		set[val] = struct{}{}
	}
	return set
}

func validateMsgs(
	t *testing.T,
	sync string,
	expectedSync string,
	upserts []string,
	expectedUpserts []string,
	deletions []string,
	expectedDeletions []string,
) {
	expectedUpsertSet := listToSet(expectedUpserts)
	expectedDeletionSet := listToSet(expectedDeletions)

	switch {
	// check if we received the correct number of trial messages
	case len(upserts) != len(expectedUpserts):
		t.Errorf(
			"did not receive expected number of upsert messages:\n\texpected %d\n\tactual: %d",
			len(expectedUpserts),
			len(upserts),
		)
	// check if we received the correct number of deletion messages
	case len(deletions) != len(expectedDeletions):
		t.Errorf(
			"did not receive expected number of deletion messages:\n\texpected %v\n\tactual: %v",
			len(expectedDeletions),
			len(deletions),
		)
	// check if we receieved the correct SyncMsg
	case sync != expectedSync:
		t.Errorf(
			"did not receive expected sync message:\n\texpected: %v\n\tactual: %v",
			expectedSync,
			sync,
		)
	// check if content of messages is correct
	default:
		for i := range upserts {
			if _, ok := expectedUpsertSet[upserts[i]]; !ok {
				t.Errorf(
					"did not received unxpected upsert message:\n\texpected: %v\n\tactual: %v",
					expectedUpserts,
					upserts[i],
				)
			}
		}
		for i := range deletions {
			if _, ok := expectedDeletionSet[deletions[i]]; !ok {
				t.Errorf(
					"did not received unxpected deletion message:\n\texpected: %s\n\tactual: %s",
					expectedDeletions,
					deletions[i],
				)
			}
		}
	}
}
