package searcher

import (
	"testing"

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
	searcherEvent := experimentv1.SearcherEvent{
		Event: &initOpsEvent,
		Id:    1,
	}
	expEvents = append(expEvents, &searcherEvent)
	expEventCount += 1
	require.Equal(t, expEvents, queue.events)
	require.Equal(t, expEventCount, queue.eventCount)

	// Add trialClosed

	//requestID := model.NewRequestID(rand)
	//validateAfterOp := ValidateAfter{requestID, uint64(200)}

}
