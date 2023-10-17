//go:build integration
// +build integration

package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
	"github.com/determined-ai/determined/master/test/streamdata"
)

// simpleUpsert is for testing and just returns the preparable message that the streamer sends.
func simpleUpsert(i stream.PreparableMessage) interface{} {
	return i
}

// startupReadWriter implements WebsocketLike and stores all messages received from streaming.
type startupReadWriter struct {
	Data           []interface{}
	StartupMessage *StartupMsg
	Msg            interface{}
}

// ReadJSON sends the StartupMessage first, then send any Msg that is set.
func (s *startupReadWriter) ReadJSON(data interface{}) error {
	if s.StartupMessage != nil {
		targetMsg, ok := data.(*StartupMsg)
		if !ok {
			return fmt.Errorf("target message type is not a pointer to StartupMsg")
		}
		targetMsg.Known = s.StartupMessage.Known
		targetMsg.Subscribe = s.StartupMessage.Subscribe
		s.StartupMessage = nil
		return nil
	}
	if s.Msg != nil {
		_, ok := s.Msg.(*SubscriptionModMsg)
		if !ok {
			return fmt.Errorf("target message type is not a pointer to SubscriptionModMsg")
		}
		data = s.Msg
	}
	return nil
}

// Write
func (s *startupReadWriter) Write(data interface{}) error {
	s.Data = append(s.Data, data)
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
		StartupMessage: &startupMessage,
	}

	emptyMsg := StartupMsg{}
	err := trw.ReadJSON(&emptyMsg)
	require.NoError(t, err)
	require.Equal(t, emptyMsg.Known, startupMessage.Known)
	require.Equal(t, emptyMsg.Subscribe, startupMessage.Subscribe)
	require.True(t, trw.StartupMessage == nil)

	err = trw.Write("test")
	require.NoError(t, err)
	require.Equal(t, 1, len(trw.Data))
	dataStr, ok := trw.Data[0].(string)
	require.True(t, ok)
	require.Equal(t, "test", dataStr)
}

// setup sets up all the entities we need to test with and simplifies actual test fn code.
func setup(t *testing.T, startupMsg StartupMsg) (
	context.Context, model.User, *PublisherSet, startupReadWriter, errgroupx.Group, *db.PgDB, func(),
) {
	ctx := context.TODO()

	testUser := model.User{Username: uuid.New().String()}

	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	ps := NewPublisherSet()
	ps.DBAddress = pgDB.Url

	testReadWriter := startupReadWriter{
		StartupMessage: &startupMsg,
	}

	errgrp := errgroupx.WithContext(ctx)

	return ctx, testUser, ps, testReadWriter, errgrp, pgDB, cleanup
}

func TestStartup(t *testing.T) {
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

	ssupCtx := context.TODO()
	ctx, testUser, publisherSet, tester, _, pgDB, cleanup := setup(t, startupMessage)
	defer cleanup()

	trials := streamdata.GenerateStreamTrials()
	trials.MustMigrate(t, pgDB, "file://../../static/migrations")

	err := publisherSet.entrypoint(ssupCtx, ctx, testUser, &tester, simpleUpsert)
	require.NoError(t, err)

	deletions, trialMsgs, err := splitDeletionsAndTrials(tester.Data)
	require.NoError(t, err)
	require.Equal(t, 1, len(deletions), "did not receive 1 deletion message")
	require.Equal(t, "", deletions[0], "expected deleted trials to be empty, not %s", deletions[0])
	require.Equal(t, 0, len(trialMsgs), "received unexpected trial message")
	tester.Data = []interface{}{}

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
	tester.StartupMessage = &startupMessage
	publisherSet = NewPublisherSet()
	err = publisherSet.entrypoint(ssupCtx, ctx, testUser, &tester, simpleUpsert) // XXX: fix prepare func
	require.NoError(t, err)
	deletions, trialMsgs, err = splitDeletionsAndTrials(tester.Data)
	require.NoError(t, err)
	require.Equal(t, 1, len(deletions), "did not receive 1 deletion message")
	require.Equal(t, "4", deletions[0], "expected deleted trials to be 4, not %s", deletions[0])
	require.Equal(t, 1, len(trialMsgs), "received unexpected trial message")
	require.Equal(t, 3, trialMsgs[0].ID, "expected trialMsg with ID 3, received ID %d",
		trialMsgs[0].ID)
	tester.Data = []interface{}{}

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
	tester.StartupMessage = &startupMessage
	err = publisherSet.entrypoint(ssupCtx, ctx, testUser, &tester, simpleUpsert)
	require.NoError(t, err)
	deletions, trialMsgs, err = splitDeletionsAndTrials(tester.Data)
	require.NoError(t, err)
	require.Equal(t, 1, len(deletions), "did not receive 1 deletion message")
	require.Equal(t, "4", deletions[0], "expected deleted trials to be 4, not %s", deletions[0])
	require.Equal(t, 0, len(trialMsgs), "received unexpected trial message")

	// 3 is not known, and 4 does not exist
	startupMessage = StartupMsg{
		Known: KnownKeySet{
			Trials: "1,2,4",
		},
		Subscribe: SubscriptionSpecSet{
			Trials: &TrialSubscriptionSpec{
				TrialIds: []int{1, 2, 4},
				Since:    0,
			},
		},
	}
	tester.StartupMessage = &startupMessage
	err = publisherSet.entrypoint(ssupCtx, ctx, testUser, &tester, simpleUpsert)
	require.NoError(t, err)
	deletions, trialMsgs, err = splitDeletionsAndTrials(tester.Data)
	require.NoError(t, err)
	require.Equal(t, 1, len(deletions), "did not receive 1 deletion message")
	require.Equal(t, "4", deletions[0], "expected deleted trials to be 4, not %s", deletions[0])
	require.Equal(t, 1, len(trialMsgs), "received unexpected trial message")
	require.Equal(t, 3, trialMsgs[0].ID, "expected trialMsg with ID 3, received ID %d",
		trialMsgs[0].ID)

	// TODO: add test that tests for diverging known key sets and subscriptions
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

func preparableMsgToMap(message interface{}) (map[string]interface{}, error) {
	_, ok := message.(stream.PreparableMessage)

	if !ok {
		return nil, fmt.Errorf("provided message is not a preparable message")
	}

	bytes, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	output := map[string]interface{}{}
	err = json.Unmarshal(bytes, &output)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func TestTrialUpdate(t *testing.T) {
	startupMessage := StartupMsg{
		Known: KnownKeySet{
			Trials: "1,2,3",
		},
		Subscribe: SubscriptionSpecSet{
			Trials: &TrialSubscriptionSpec{
				ExperimentIds: []int{1},
				Since:         0,
			},
		},
	}

	_, testUser, ps, tester, errgrp, pgDB, cleanup := setup(t, startupMessage)
	defer func() {
		cleanup()
	}()

	trials := streamdata.GenerateStreamTrials()
	trials.MustMigrate(t, pgDB, "file://../../static/migrations")

	errgrp.Go(ps.Start)
	// run the entrypoint against our tester
	errgrp.Go(func(ctx context.Context) error {
		// rb is not sure if the test should care which ctx to use, since in testing
		// the publisher set is 1:1 with a single connection
		return ps.entrypoint(context.Background(), ctx, testUser, &tester, simpleUpsert)
	})

	// Now we can write our test as if we are the client talking to a websocket
	testBody := func(ctx context.Context) error {
		for len(tester.Data) == 0 {
			time.Sleep(time.Second) // perhaps a shorter time for a wait loop?
		}
		deletions, trialMsgs, err := splitDeletionsAndTrials(tester.Data)
		require.NoError(t, err)
		if len(deletions) != 1 || deletions[0] != "" {
			return fmt.Errorf("received unexpected deletion message")
		}
		if len(trialMsgs) != 0 {
			return fmt.Errorf("received unexpected trial message")
		}

		// send messages, check responses
		err = streamdata.ModTrial(ctx, 1, 1, false, false, "CANCELED")
		if err != nil {
			return err
		}

		for len(tester.Data) < 2 {
			time.Sleep(time.Second)
		}
		msgMap, err := preparableMsgToMap(tester.Data[1])
		if err != nil {
			return err
		}
		fmt.Println("message Map:")
		fmt.Println(msgMap)
		if msg, ok := msgMap["Msg"].(map[string]interface{}); !ok || msg["state"] != "CANCELED" {
			return fmt.Errorf("updated state should be canceled, not %s", msg["state"])
		}
		if msg, ok := msgMap["Msg"].(map[string]interface{}); !ok || msg["state"] != "ERROR" {
			return fmt.Errorf("test error")
		}

		// cancel the whole error group when the test succeeds
		errgrp.Cancel()
		return nil
	}
	errgrp.Go(testBody)
	require.NoError(t, errgrp.Wait())
}
