//go:build integration
// +build integration

package stream

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

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
	switch msg := i.(type) {
	case stream.UpsertMsg:
		switch typedMsg := msg.Msg.(type) {
		case *ProjectMsg:
			return fmt.Sprintf(
				"type: %s, project_id: %d, state: %s, workspace_id: %d",
				ProjectsUpsertKey,
				typedMsg.ID,
				typedMsg.State,
				typedMsg.WorkspaceID,
			)
		}
	case stream.DeleteMsg:
		return fmt.Sprintf("type: %s, deleted: %s", msg.Key, msg.Deleted)
	case stream.SyncMsg:
		return fmt.Sprintf("type: %s, sync_id: %s, complete: %t", syncKey, msg.SyncID, msg.Complete)
	}
	return i
}

// mockSocket implements WebsocketLike and stores all messages received from streaming.
type mockSocket struct {
	fromServer chan interface{}
	toServer   chan *StartupMsg
	closed     bool
}

// newMockSocket creates a new instance mockSocket and initialize it's conditional variables.
func newMockSocket() *mockSocket {
	return &mockSocket{
		fromServer:  make(chan interface{}, channelBufferSize),
		toServer: make(chan *StartupMsg, channelBufferSize),
	}
}

// WriteToServer synchronously appends the StartupMsg to toServer messages.
func (s *mockSocket) WriteToServer(t *testing.T, startup *StartupMsg) {
	select {
	case s.toServer <- startup:
		break
	}
}

// ReadJSON implements WebsocketLike's ReadJSON(), blocks until able to read toServer messages off
// the mockSocket or the mockSocket is closed.
func (s *mockSocket) ReadJSON(data interface{}) error {
	targetMsg, ok := data.(*StartupMsg)
	if !ok {
		return fmt.Errorf("target message type is not a pointer to StartupMsg")
	}
	select {
	case msg, ok := <-s.toServer:
		if !ok {
			return &websocket.CloseError{Code: websocket.CloseAbnormalClosure}
		}
		*targetMsg = *msg
		return nil
	}
}

// Write implements WebsocketLike's Write(), appends the data to mockSocket's fromServer messages.
func (s *mockSocket) Write(data interface{}) error {
	select {
	case s.fromServer <- data:
		return nil
	}
}


// ReadFromServer synchronously reads an fromServer message off the mockSocket.
func (s *mockSocket) ReadFromServer(t *testing.T) string {
	select {
	case msg, ok := <-s.fromServer:
		if !ok {
			t.Fatal(&websocket.CloseError{Code: websocket.CloseAbnormalClosure})
		}
		stringMsg, ok := msg.(string)
		if !ok {
			t.Fatalf("read unexpected message, likely due to type not being added to testPrepareFunc: %#v", msg)
		}
		return stringMsg
	}
}

// AssertEOF makes sure that the server closes the websocket before sending any more messages
func (s *mockSocket) AssertEOF(t *testing.T) {
	select {
	case msg, ok := <-s.fromServer:
		if !ok {
			// this is the EOF we were looking for
			return
		}
		t.Fatalf("expected EOF from server but got msg instead: %v", msg)
	case <- time.After(3 * time.Second):
		t.Fatal("expected EOF from server but timed out instead")
	}
}

func (s *mockSocket) ReadUntilFound(
	t *testing.T,
	expected... string,
) []string {
	var recvd []string
	checklist := map[string]bool{}
	for _, s := range expected {
		checklist[s] = false
	}

	for len(checklist) > 0 {
		t.Logf("ReadUntilFound()\n")
		t.Logf("    have recvd:\n")
		for _, r := range recvd {
			t.Logf("      - %v\n", r)
		}
		t.Logf("    still awaiting:\n")
		for c := range checklist {
			t.Logf("      - %v\n", c)
		}
		msg := s.ReadFromServer(t)
		recvd = append(recvd, msg)
		delete(checklist, msg)
	}

	t.Logf("ReadUntilFound() complete")

	return recvd
}

// Close closes the mockSocket.
func (s *mockSocket) Close() {
	if s.closed {
		return
	}
	close(s.fromServer)
	close(s.toServer)
	s.closed = true
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
	}
	deleteKeys := []string{
		ProjectsDeleteKey,
	}

	for i := range upsertKeys {
		upsertKeys[i] = "^type: " + upsertKeys[i]
	}

	for i := range deleteKeys {
		deleteKeys[i] = "^type: " + deleteKeys[i]
	}

	upsertPattern := regexp.MustCompile(
		strings.Join(upsertKeys, "|"),
	)
	deletePattern := regexp.MustCompile(
		strings.Join(deleteKeys, "|"),
	)
	syncPattern := regexp.MustCompile("^type: " + syncKey)

	for _, msg := range messages {
		switch {
		case deletePattern.MatchString(msg):
			deletions = append(deletions, msg)
		case upsertPattern.MatchString(msg):
			upserts = append(upserts, msg)
		case syncPattern.MatchString(msg):
			syncs = append(syncs, msg)
		default:
			t.Fatalf("unknown message type: %q", msg)
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
		t.Fatalf(
			"did not receive expected number of upsert messages:\n\texpected %d\n\tactual: %d",
			len(expectedUpserts),
			len(upserts),
		)
	// check if we received the correct number of deletion messages
	case len(deletions) != len(expectedDeletions):
		t.Fatalf(
			"did not receive expected number of deletion messages:\n\texpected %v\n\tactual: %v",
			len(expectedDeletions),
			len(deletions),
		)
	// check if we receieved the correct SyncMsg
	case len(syncs) != len(expectedSyncs):
		t.Fatalf(
			"did not receive expected number of sync message:\n\texpected: %#v\n\tactual: %v",
			len(expectedSyncs),
			len(syncs),
		)
	// check if content of messages is correct
	default:
		for i := range syncs {
			if syncs[i] != expectedSyncs[i] {
				t.Fatalf(
					"did not receive expected sync message:\n\texpected: %#v\n\tactual: %q",
					expectedSyncs[i],
					syncs[i],
				)
			}
		}
		for i := range upserts {
			if upserts[i] != expectedUpserts[i] {
				t.Fatalf(
					"did not receive expected upsert message:\n\texpected: %#v\n\tactual: %q",
					expectedUpserts[i],
					upserts[i],
				)
			}
		}
		for i := range deletions {
			if deletions[i] != expectedDeletions[i] {
				t.Fatalf(
					"did not receive expected deletion message:\n\texpected: %#v\n\tactual: %q",
					expectedDeletions[i],
					deletions[i],
				)
			}
		}
	}
}
