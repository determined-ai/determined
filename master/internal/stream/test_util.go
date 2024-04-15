//go:build integration
// +build integration

package stream

import (
	"fmt"
	"regexp"
	"sort"
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
// otherwise, returns the MarshallableMsg that the streamer sends.
func testPrepareFunc(i stream.MarshallableMsg) interface{} {
	// fmt.Printf("testPrepareFunc: %#v, type:%+v\n", i, reflect.TypeOf(i))
	switch msg := i.(type) {
	case stream.UpsertMsg:
		// fmt.Println("is upsertmsg")
		// fmt.Printf("typedMsg: %#v, type:%+v\n", msg.Msg, reflect.TypeOf(msg.Msg))
		switch typedMsg := msg.Msg.(type) {
		case *ProjectMsg:
			return fmt.Sprintf(
				"key: %s, project_id: %d, state: %s, workspace_id: %d",
				ProjectsUpsertKey,
				typedMsg.ID,
				typedMsg.State,
				typedMsg.WorkspaceID,
			)
		case *ModelMsg:
			return fmt.Sprintf(
				"key: %s, model_id: %d, workspace_id: %d",
				ModelsUpsertKey,
				typedMsg.ID,
				typedMsg.WorkspaceID,
			)
		case *ModelVersionMsg:
			return fmt.Sprintf(
				"key: %s, model_version_id: %d, model_id: %d, workspace_id: %v",
				ModelVersionsUpsertKey,
				typedMsg.ID,
				typedMsg.ModelID,
				typedMsg.WorkspaceID,
			)
		}
	case stream.DeleteMsg:
		return fmt.Sprintf("key: %s, deleted: %s", msg.Key, msg.Deleted)
	case stream.SyncMsg:
		return fmt.Sprintf("key: %s, sync_id: %s, complete: %t", syncKey, msg.SyncID, msg.Complete)
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
		// fmt.Printf("stringMsg: %#v\n", msg)
		if !ok {
			t.Errorf("read unexpected message, likely due to type not being added to testPrepareFunc: %#v", msg)
		}
		*data = stringMsg
	}
}

func (s *mockSocket) ReadUntilFound(
	t *testing.T,
	data *[]string,
	expected []string,
) {
	var msg string
	checklist := map[string]struct{}{}
	for _, s := range expected {
		checklist[s] = struct{}{}
	}

	for len(checklist) > 0 {
		s.ReadIncoming(t, &msg)
		*data = append(*data, msg)
		delete(checklist, msg)
		t.Logf("ReadUntilFound()\n\tcurrently read:\t%#v\n\tlooking for:\t%q", *data, checklist)
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
		ProjectsUpsertKey,
		ModelsUpsertKey,
		ModelVersionsUpsertKey,
	}
	deleteKeys := []string{
		ProjectsDeleteKey,
		ModelsDeleteKey,
		ModelVersionsDeleteKey,
	}

	for i := range upsertKeys {
		upsertKeys[i] = "^key: " + upsertKeys[i]
	}

	for i := range deleteKeys {
		deleteKeys[i] = "^key: " + deleteKeys[i]
	}

	upsertPattern := regexp.MustCompile(
		strings.Join(upsertKeys, "|"),
	)
	deletePattern := regexp.MustCompile(
		strings.Join(deleteKeys, "|"),
	)
	syncPattern := regexp.MustCompile("^key: " + syncKey)

	for _, msg := range messages {
		switch {
		case deletePattern.MatchString(msg):
			deletions = append(deletions, msg)
		case upsertPattern.MatchString(msg):
			upserts = append(upserts, msg)
		case syncPattern.MatchString(msg):
			syncs = append(syncs, msg)
		default:
			t.Errorf("unknown message type: %q", msg)
		}
	}
	return deletions, upserts, syncs
}

func validateMsgs(
	t *testing.T,
	syncs []string,
	expectedSyncs []string,
	upserts []string,
	expectedUpserts []string,
	deletions []string,
	expectedDeletions []string,
) {
	// sort expected & actual upsert/deletion messages
	// we expect the ordering of the sync messages to be consistent
	sort.Strings(upserts)
	sort.Strings(expectedUpserts)
	sort.Strings(deletions)
	sort.Strings(expectedDeletions)

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
	case len(syncs) != len(expectedSyncs):
		t.Errorf(
			"did not receive expected number of sync message:\n\texpected: %#v\n\tactual: %v",
			len(expectedSyncs),
			len(syncs),
		)
	// check if content of messages is correct
	default:
		for i := range syncs {
			if syncs[i] != expectedSyncs[i] {
				t.Errorf(
					"did not receive expected sync message:\n\texpected: %#v\n\tactual: %q",
					expectedSyncs[i],
					syncs[i],
				)
			}
		}
		for i := range upserts {
			if upserts[i] != expectedUpserts[i] {
				t.Errorf(
					"did not receive expected upsert message:\n\texpected: %#v\n\tactual: %q",
					expectedUpserts[i],
					upserts[i],
				)
			}
		}
		for i := range deletions {
			if deletions[i] != expectedDeletions[i] {
				t.Errorf(
					"did not receive expected deletion message:\n\texpected: %#v\n\tactual: %q",
					expectedDeletions[i],
					deletions[i],
				)
			}
		}
	}
}
