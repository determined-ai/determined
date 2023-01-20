//nolint:exhaustivestruct
package searcher

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

type idMaker struct {
	num int32
}

func (m *idMaker) next() int32 {
	m.num++
	return m.num
}

// Test a few methods (not all because they are similar) from CustomSearchMethod and the queue.
func TestCustomSearchMethod(t *testing.T) {
	config := expconf.SearcherConfig{
		RawCustomConfig: &expconf.CustomConfig{},
	}

	customSearchMethod := NewSearchMethod(config)
	rand := nprand.New(0)
	ctx := context{rand: rand}

	queue := customSearchMethod.(CustomSearchMethod).getSearcherEventQueue()
	require.Equal(t, 0, len(queue.events))

	var expEvents []*experimentv1.SearcherEvent
	var ids idMaker

	// Add initialOperations.
	_, err := customSearchMethod.initialOperations(ctx)
	require.NoError(t, err)

	expEvents = append(expEvents, &experimentv1.SearcherEvent{
		Event: &experimentv1.SearcherEvent_InitialOperations{
			InitialOperations: &experimentv1.InitialOperations{},
		},
		Id: ids.next(),
	})
	require.Equal(t, expEvents, queue.GetEvents())

	// Add trialExitedEarly.
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
	expEvents = append(expEvents, &experimentv1.SearcherEvent{
		Event: &trialExitedEarlyEvent,
		Id:    ids.next(),
	})
	require.Equal(t, expEvents, queue.GetEvents())

	// Add ValidationCompleted.
	validateAfterOp := ValidateAfter{requestID, uint64(200)}
	metric := float64(10.3)
	_, err = customSearchMethod.validationCompleted(
		ctx,
		requestID,
		metric,
		validateAfterOp,
	)
	require.NoError(t, err)

	protoMetric, err := structpb.NewValue(metric)
	require.NoError(t, err)

	validationCompletedEvent := experimentv1.SearcherEvent_ValidationCompleted{
		ValidationCompleted: &experimentv1.ValidationCompleted{
			RequestId:           requestID.String(),
			Metrics:             protoMetric,
			ValidateAfterLength: validateAfterOp.ToProto().Length,
		},
	}
	expEvents = append(expEvents, &experimentv1.SearcherEvent{
		Event: &validationCompletedEvent,
		Id:    ids.next(),
	})
	require.Equal(t, expEvents, queue.GetEvents())

	// Add ValidationCompleted with a dictionary of all metrics.
	validateAfterOp2 := ValidateAfter{requestID, uint64(300)}
	allMetrics := map[string]interface{}{
		"themetric": float64(10.3),
	}
	_, err = customSearchMethod.validationCompleted(
		ctx,
		requestID,
		allMetrics,
		validateAfterOp2,
	)
	require.NoError(t, err)

	protoAllMetrics, err := structpb.NewValue(allMetrics)
	require.NoError(t, err)
	validationCompletedEvent2 := experimentv1.SearcherEvent_ValidationCompleted{
		ValidationCompleted: &experimentv1.ValidationCompleted{
			RequestId:           requestID.String(),
			Metrics:             protoAllMetrics,
			ValidateAfterLength: validateAfterOp2.ToProto().Length,
		},
	}
	expEvents = append(expEvents, &experimentv1.SearcherEvent{
		Event: &validationCompletedEvent2,
		Id:    ids.next(),
	})
	require.Equal(t, expEvents, queue.GetEvents())

	// Add trialProgress.
	trialProgress := 0.02
	customSearchMethod.(CustomSearchMethod).trialProgress(ctx, requestID, PartialUnits(trialProgress))
	require.NoError(t, err)

	expEvents = append(expEvents, &experimentv1.SearcherEvent{
		Event: &experimentv1.SearcherEvent_TrialProgress{
			TrialProgress: &experimentv1.TrialProgress{
				RequestId:    requestID.String(),
				PartialUnits: trialProgress,
			},
		},
		Id: ids.next(),
	})
	require.Equal(t, expEvents, queue.GetEvents())

	// Set customSearcherProgress.
	searcherProgress := 0.4
	customSearchMethod.(CustomSearchMethod).setCustomSearcherProgress(searcherProgress)
	require.Equal(t, searcherProgress, customSearchMethod.progress(nil, nil))

	// Check removeUpto.
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

	queue := customSearchMethod.(CustomSearchMethod).getSearcherEventQueue()
	w, err := queue.Watch()
	require.NoError(t, err)

	// Should immediately receive initial status.
	select {
	case <-w.C:
		t.Fatal("received a non-empty channel but should not have")
	default:
	}

	var expEvents []*experimentv1.SearcherEvent
	var ids idMaker

	// Add initialOperations.
	_, err = customSearchMethod.initialOperations(ctx)
	require.NoError(t, err)

	expEvents = append(expEvents, &experimentv1.SearcherEvent{
		Event: &experimentv1.SearcherEvent_InitialOperations{
			InitialOperations: &experimentv1.InitialOperations{},
		},
		Id: ids.next(),
	})
	require.Equal(t, expEvents, queue.GetEvents())

	// Receive events in the watcher channel after it's added.
	select {
	case eventsInWatcher := <-w.C:
		require.Equal(t, queue.GetEvents(), eventsInWatcher)
	default:
		t.Fatal("did not receive events")
	}

	// Unwatching should work.
	queue.Unwatch(w.ID)

	// Receive events when you create a new watcher after events exist.
	w2, err := queue.Watch()
	require.NoError(t, err)
	select {
	case eventsInWatcher2 := <-w2.C:
		require.Equal(t, queue.GetEvents(), eventsInWatcher2)
	default:
		t.Fatal("did not receive events")
	}

	// Unwatching should work.
	queue.Unwatch(w2.ID)
}
