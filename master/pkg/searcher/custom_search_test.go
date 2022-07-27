//nolint:exhaustivestruct
package searcher

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

// Testing a few methods (not all because they are similar)
// from customSearcherMethod to ensure that the methods
// and the queue is working well.
func TestCustomSearchMethod(t *testing.T) {
	config := expconf.SearcherConfig{
		RawCustomConfig: &expconf.CustomConfig{RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(500))},
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
		}}
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

	// Check removeUpto
	err = queue.RemoveUpTo(2)
	require.NoError(t, err)
	require.Equal(t, expEvents[2:], queue.events)

	// Add trialProgress.
	progress := 0.02
	customSearchMethod.(CustomSearchMethod).trialProgress(ctx, requestID, PartialUnits(progress))
	require.NoError(t, err)
	trialProgressEvent := experimentv1.SearcherEvent_TrialProgress{
		TrialProgress: &experimentv1.TrialProgress{},
	}
	expEventCount++
	searcherEvent4 := experimentv1.SearcherEvent{
		Event: &trialProgressEvent,
		Id:    expEventCount,
	}
	expEvents = append(expEvents, &searcherEvent4)
	require.Equal(t, expEvents, queue.GetEvents())

	// Set customSearcherProgress.
	customSearchMethod.(CustomSearchMethod).setCustomSearcherProgress(progress)
	require.Equal(t, progress, customSearchMethod.progress(nil, nil))
}
