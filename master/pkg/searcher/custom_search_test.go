//nolint:exhaustivestruct
package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Testing a few methods (not all because they are similar)
// from customSearcherMethod to ensure that the methods
// and the queue is working well.
func TestCustomSearchMethod(t *testing.T) {
	config := expconf.SearcherConfig{
		RawCustomConfig: &expconf.CustomConfig{},
	}

	customSearchMethod := NewSearchMethod(config)
	rand := nprand.New(0)
	ctx := context{rand: rand}

	var queue *SearcherEventQueue
	queue = customSearchMethod.(CustomSearchMethod).getSearcherEventQueue()

	expEvents := make([]*experimentv1.SearcherEvent, 0)
	require.Equal(t, 0, len(queue.events))
	// Add initialOperations
	_, err := customSearchMethod.initialOperations(ctx)
	require.NoError(t, err)
	var expEventCount int32
	initOpsEvent := experimentv1.SearcherEvent_InitialOperations{
		InitialOperations: &experimentv1.InitialOperations{},
	}
	expEventCount++
	searcherEvent := experimentv1.SearcherEvent{
		Event: &initOpsEvent,
		Id:    expEventCount,
	}
	expEvents = append(expEvents, &searcherEvent)
	require.Equal(t, expEvents, queue.GetEvents())
	// Add trialExitedEarly
	requestID := model.NewRequestID(rand)
	exitedReason := model.Errored
	_, err = customSearchMethod.trialExitedEarly(ctx, requestID, exitedReason)
	require.NoError(t, err)
	trialExitedEarlyEvent := experimentv1.SearcherEvent_TrialExitedEarly{
		TrialExitedEarly: &experimentv1.TrialExitedEarly{
			RequestId:    requestID.String(),
			ExitedReason: experimentv1.TrialExitedEarly_EXITED_REASON_UNSPECIFIED,
		},
	}
	expEventCount++
	searcherEvent2 := experimentv1.SearcherEvent{
		Event: &trialExitedEarlyEvent,
		Id:    expEventCount,
	}
	expEvents = append(expEvents, &searcherEvent2)
	require.Equal(t, expEvents, queue.GetEvents())

	// Add validationAfter.
	validateAfterOp := ValidateAfter{requestID, uint64(200)}
	metric := float64(10.3)
	_, err = customSearchMethod.validationCompleted(ctx, requestID, metric, validateAfterOp)
	require.NoError(t, err)
	validationCompletedEvent := experimentv1.SearcherEvent_ValidationCompleted{
		ValidationCompleted: &experimentv1.ValidationCompleted{
			RequestId:           requestID.String(),
			Metric:              metric,
			ValidateAfterLength: validateAfterOp.ToProto().Length,
		},
	}
	expEventCount++
	searcherEvent3 := experimentv1.SearcherEvent{
		Event: &validationCompletedEvent,
		Id:    expEventCount,
	}
	expEvents = append(expEvents, &searcherEvent3)
	require.Equal(t, expEvents, queue.GetEvents())

	// Add trialProgress.
	trialProgress := 0.02
	customSearchMethod.(CustomSearchMethod).trialProgress(ctx, requestID, PartialUnits(trialProgress))
	require.NoError(t, err)
	trialProgressEvent := experimentv1.SearcherEvent_TrialProgress{
		TrialProgress: &experimentv1.TrialProgress{
			RequestId:    requestID.String(),
			PartialUnits: trialProgress,
		},
	}
	expEventCount++
	searcherEvent4 := experimentv1.SearcherEvent{
		Event: &trialProgressEvent,
		Id:    expEventCount,
	}
	expEvents = append(expEvents, &searcherEvent4)
	require.Equal(t, expEvents, queue.GetEvents())

	// Set customSearcherProgress.
	searcherProgress := 0.4
	customSearchMethod.(CustomSearchMethod).setCustomSearcherProgress(searcherProgress)
	require.Equal(t, searcherProgress, customSearchMethod.progress(nil, nil))

	// Check removeUpto
	err = queue.RemoveUpTo(2)
	require.NoError(t, err)
	require.Equal(t, expEvents[2:], queue.events)
}

func TestCustomSearchWatcher(t *testing.T) {
	config := expconf.SearcherConfig{
		RawCustomConfig: &expconf.CustomConfig{},
	}

	customSearchMethod := NewSearchMethod(config)
	rand := nprand.New(0)
	ctx := context{rand: rand}

	var queue *SearcherEventQueue
	queue = customSearchMethod.(CustomSearchMethod).getSearcherEventQueue()
	id := uuid.New()
	w, err := queue.Watch(id)
	require.NoError(t, err)

	// should immediately receive initial status.
	select {
	case <-w.C:
		t.Fatal("received a non-empty channel but should not have")
	default:
	}

	expEvents := make([]*experimentv1.SearcherEvent, 0)
	// Add initialOperations
	_, err = customSearchMethod.initialOperations(ctx)
	require.NoError(t, err)
	var expEventCount int32
	initOpsEvent := experimentv1.SearcherEvent_InitialOperations{
		InitialOperations: &experimentv1.InitialOperations{},
	}
	expEventCount++
	searcherEvent := experimentv1.SearcherEvent{
		Event: &initOpsEvent,
		Id:    expEventCount,
	}
	expEvents = append(expEvents, &searcherEvent)
	require.Equal(t, expEvents, queue.GetEvents())

	// add events and then you should recieve events in the watcher channel.
	eventsInWatcher := <-w.C
	select {
	case <-w.C:
		require.Equal(t, queue.GetEvents(), eventsInWatcher)
	default:
		t.Fatal("did not receive events")
	}

	// unwatching should work.
	queue.Unwatch(id)

}
