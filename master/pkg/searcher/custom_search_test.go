package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/stretchr/testify/require"
)

func TestCustomSearchMethod(t *testing.T) {
	config := expconf.SearcherConfig{
		RawCustomConfig: &expconf.CustomConfig{},
	}

	customSearchMethod := NewSearchMethod(config)
	ctx := context{rand: nprand.New(0)}
	queue := customSearchMethod.getSearcherEventQueue()
	expEvents := make([]*experimentv1.SearcherEvent, 0)
	require.Equal(t, 0, len(queue.events))
	customSearchMethod.initialOperations(ctx)
	expEventCount := 0
	initOpsEvent := experimentv1.SearcherEvent_InitialOperations{
		InitialOperations: &experimentv1.InitialOperations{},
	}
	searcherEvent := experimentv1.SearcherEvent{
		Event: &initOpsEvent,
	}
	expEvents = append(expEvents, &searcherEvent)
	expEventCount += 1
	require.Equal(t, expEvents, queue.events)
	require.Equal(t, expEventCount, queue.eventCount)

}
