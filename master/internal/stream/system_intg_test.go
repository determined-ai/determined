//go:build integration
// +build integration

package stream

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
	"github.com/determined-ai/determined/master/test/streamdata"
)

func simpleUpsert(i stream.PreparableMessage) interface{} {
	return i
}

func recordDeletion(s1, s2 string) interface{} {
	fmt.Println("recording deletion:", s1+" and "+s2)
	return s2
}

type startupReadWriter struct {
	data           []interface{}
	startupMessage *StartupMsg
}

func (s *startupReadWriter) ReadJSON(data interface{}) error {
	if s.startupMessage == nil {
		return fmt.Errorf("startup message has been sent")
	}
	targetMsg, ok := data.(*StartupMsg)
	if !ok {
		return fmt.Errorf("target message type is not a pointer to StartupMsg")
	}
	targetMsg.Known = s.startupMessage.Known
	targetMsg.Subscribe = s.startupMessage.Subscribe
	s.startupMessage = nil
	return nil
}

func (s *startupReadWriter) Write(data interface{}) error {
	s.data = append(s.data, data)
	return nil
}

func (s *startupReadWriter) Close() error {
	return nil
}

func TestStartupReadWriter(t *testing.T) {
	startupMessage := StartupMsg{
		Known: KnownKeySet{Trials: "1,2,3"},
		Subscribe: SubscriptionSpecSet{
			Trials: &TrialSubscriptionSpec{
				ExperimentIds: []int{1},
				Since:         0,
			},
		},
	}

	trw := startupReadWriter{
		startupMessage: &startupMessage,
	}

	emptyMsg := StartupMsg{}
	err := trw.ReadJSON(&emptyMsg)
	require.NoError(t, err)
	require.Equal(t, emptyMsg.Known, startupMessage.Known)
	require.Equal(t, emptyMsg.Subscribe, startupMessage.Subscribe)
	require.True(t, trw.startupMessage == nil)

	err = trw.Write("test")
	require.NoError(t, err)
	require.Equal(t, 1, len(trw.data))
	dataStr, ok := trw.data[0].(string)
	require.True(t, ok)
	require.Equal(t, "test", dataStr)
}

func TestStartup(t *testing.T) {
	ssupCtx := context.TODO()
	ctx := context.TODO()
	testUser := model.User{Username: uuid.New().String()}
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer func() {
		fmt.Println("cleaning up?")
		cleanup()
	}()

	trials := streamdata.GenerateStreamTrials()
	trials.MustMigrate(t, pgDB, "file://../../static/migrations")

	startupMessage := StartupMsg{
		Known: KnownKeySet{
			Trials: "1,2,3",
		},
		Subscribe: SubscriptionSpecSet{
			Trials: &TrialSubscriptionSpec{
				ExperimentIds: []int{1}, // trials 1,2,3 exist
				Since:         0,
			},
		},
	}

	tester := startupReadWriter{
		startupMessage: &startupMessage,
	}
	publisherSet := NewPublisherSet()
	err := publisherSet.entrypoint(ssupCtx, ctx, testUser, &tester, simpleUpsert)
	require.NoError(t, err)

	deletions, trialMsgs, err := splitDeletionsAndTrials(tester.data)
	require.NoError(t, err)
	require.Equal(t, 1, len(deletions), "did not receive 1 deletion message")
	require.Equal(t, "0", deletions[0], "expected deleted trials to be 0, not %s", deletions[0])
	require.Equal(t, 0, len(trialMsgs), "received unexpected trial message")
	tester.data = []interface{}{}

	// don't know about trial 3, and trial 4 doesn't exist
	startupMessage = StartupMsg{
		Known: KnownKeySet{
			Trials: "1,2,4",
		},
		Subscribe: SubscriptionSpecSet{
			Trials: &TrialSubscriptionSpec{
				ExperimentIds: []int{1},
				Since:         0,
			},
		},
	}
	tester.startupMessage = &startupMessage
	publisherSet = NewPublisherSet()
	err = publisherSet.entrypoint(ssupCtx, ctx, testUser, &tester, simpleUpsert) // XXX: fix prepare func
	require.NoError(t, err)
	deletions, trialMsgs, err = splitDeletionsAndTrials(tester.data)
	require.NoError(t, err)
	require.Equal(t, 1, len(deletions), "did not receive 1 deletion message")
	require.Equal(t, "4", deletions[0], "expected deleted trials to be 4, not %s", deletions[0])
	require.Equal(t, 1, len(trialMsgs), "received unexpected trial message")
	require.Equal(t, 3, trialMsgs[0].ID, "expected trialMsg with ID 3, received ID %d",
		trialMsgs[0].ID)
	tester.data = []interface{}{}

	// Subscribe to all known trials, but 4 doesn't exist
	startupMessage = StartupMsg{
		Known: KnownKeySet{
			Trials: "1,2,3,4",
		},
		Subscribe: SubscriptionSpecSet{
			Trials: &TrialSubscriptionSpec{
				TrialIds: []int{1, 2, 3, 4},
				Since:    0,
			},
		},
	}
	tester.startupMessage = &startupMessage
	err = publisherSet.entrypoint(ssupCtx, ctx, testUser, &tester, simpleUpsert)
	require.NoError(t, err)
	deletions, trialMsgs, err = splitDeletionsAndTrials(tester.data)
	require.NoError(t, err)
	require.Equal(t, 1, len(deletions), "did not receive 1 deletion message")
	require.Equal(t, "4", deletions[0], "expected deleted trials to be 4, not %s", deletions[0])
	require.Equal(t, 0, len(trialMsgs), "received unexpected trial message")
}

func splitDeletionsAndTrials(messages []interface{}) ([]string, []*TrialMsg, error) {
	var deletions []string
	var trialMsgs []*TrialMsg
	for _, msg := range messages {
		if deletion, ok := msg.(stream.DeleteMsg); ok {
			deletions = append(deletions, deletion.Deleted)
		} else if upsert, ok := msg.(stream.UpsertMsg); ok {
			trialMsg, ok := upsert.Msg.(*TrialMsg)
			if !ok {
				return nil, nil, fmt.Errorf("expected a trial message, but received %t",
					reflect.TypeOf(upsert.Msg))
			}
			trialMsgs = append(trialMsgs, trialMsg)
		} else {
			return nil, nil, fmt.Errorf("expected a string or *TrialMsg, but received %t",
				reflect.TypeOf(msg))
		}
	}
	return deletions, trialMsgs, nil
}
