package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
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
	queue := customSearchMethod.getSearcherEventQueue()
	expEvents := make([]*experimentv1.SearcherEvent, 0)
	require.Equal(t, 0, len(queue.events))
	// Add initialOperations
	customSearchMethod.initialOperations(ctx)
	var expEventCount int32
	expEventCount = 0
	initOpsEvent := experimentv1.SearcherEvent_InitialOperations{
		InitialOperations: &experimentv1.InitialOperations{},
	}
	expEventCount += 1
	searcherEvent := experimentv1.SearcherEvent{
		Event: &initOpsEvent,
		Id:    expEventCount,
	}
	expEvents = append(expEvents, &searcherEvent)
	require.Equal(t, expEvents, queue.GetEvents())
	// Add trialExitedEarly
	requestID := model.NewRequestID(rand)
	exitedReason := model.Errored
	customSearchMethod.trialExitedEarly(ctx, requestID, exitedReason)
	trialExitedEarlyEvent := experimentv1.SearcherEvent_TrialExitedEarly{
		TrialExitedEarly: &experimentv1.TrialExitedEarly{
			RequestId:    requestID.String(),
			ExitedReason: experimentv1.TrialExitedEarly_EXITED_REASON_UNSPECIFIED,
		}}
	expEventCount += 1
	searcherEvent2 := experimentv1.SearcherEvent{
		Event: &trialExitedEarlyEvent,
		Id:    expEventCount,
	}
	expEvents = append(expEvents, &searcherEvent2)
	require.Equal(t, expEvents, queue.GetEvents())

	// Add validationAfter operation.
	validateAfterOp := ValidateAfter{requestID, uint64(200)}
	metric := float64(10.3)
	customSearchMethod.validationCompleted(ctx, requestID, metric, validateAfterOp)
	validationCompletedEvent := experimentv1.SearcherEvent_ValidationCompleted{
		ValidationCompleted: &experimentv1.ValidationCompleted{
			RequestId: requestID.String(),
			Metric:    metric,
			Op:        validateAfterOp.ToProto(),
		},
	}
	expEventCount += 1
	searcherEvent3 := experimentv1.SearcherEvent{
		Event: &validationCompletedEvent,
		Id:    expEventCount,
	}
	expEvents = append(expEvents, &searcherEvent3)
	require.Equal(t, expEvents, queue.GetEvents())
}
