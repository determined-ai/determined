package searcher

import (
	"encoding/json"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

type (
	customSearchState struct {
		// store the operations
		// store the events
		SearchMethodType   SearchMethodType `json:"search_method_type"`
		SearcherEventQueue *SearcherEventQueue
	}

	customSearch struct {
		defaultSearchMethod
		expconf.CustomConfig
		customSearchState
		eventCount int32
	}
)

func newCustomSearch(config expconf.CustomConfig) SearchMethod {
	return &customSearch{
		CustomConfig: config,
		customSearchState: customSearchState{
			SearchMethodType:   CustomSearch,
			SearcherEventQueue: createSearcherEventQueue(),
		},
	}
}

func createSearcherEventQueue() *SearcherEventQueue {
	return newSearcherEventQueue()
}
func (s *customSearch) initialOperations(ctx context) ([]Operation, error) {
	s.eventCount++
	event := experimentv1.SearcherEvent_InitialOpsEvent{
		InitialOpsEvent: &experimentv1.InitialOpsEvent{},
	}
	searcherEvent := experimentv1.SearcherEvent{
		Id:    s.eventCount,
		Event: &event,
	}

	err := s.SearcherEventQueue.Enqueue(&searcherEvent)
	return nil, err
}

func (s *customSearch) GetSearcherEventQueue(context) *SearcherEventQueue {
	return s.SearcherEventQueue
}

func (s *customSearch) trialCreated(context, model.RequestID) ([]Operation, error) {
	s.eventCount++
	event := experimentv1.SearcherEvent_TrialCreated{
		TrialCreated: &experimentv1.TrialCreated{},
	}
	searcherEvent := experimentv1.SearcherEvent{
		Id:    s.eventCount,
		Event: &event,
	}

	err := s.SearcherEventQueue.Enqueue(&searcherEvent)
	return nil, err
}

func (s *customSearch) progress(
	trialProgress map[model.RequestID]PartialUnits,
	trialsClosed map[model.RequestID]bool) float64 {
	// TODO we need progress event
	return 0.99
}

func (s *customSearch) validationCompleted(
	context, model.RequestID, float64,
) ([]Operation, error) {
	s.eventCount++
	event := experimentv1.SearcherEvent_ValidationCompleted{
		ValidationCompleted: &experimentv1.ValidationCompleted{},
	}
	searcherEvent := experimentv1.SearcherEvent{
		Id:    s.eventCount,
		Event: &event,
	}

	err := s.SearcherEventQueue.Enqueue(&searcherEvent)
	return nil, err
}

func (s *customSearch) trialExitedEarly(context, model.RequestID,
	model.ExitedReason) ([]Operation, error) {
	s.eventCount++
	event := experimentv1.SearcherEvent_TrialExitedEarly{
		TrialExitedEarly: &experimentv1.TrialExitedEarly{},
	}
	searcherEvent := experimentv1.SearcherEvent{
		Id:    s.eventCount,
		Event: &event,
	}

	err := s.SearcherEventQueue.Enqueue(&searcherEvent)
	return nil, err
}

func (s *customSearch) trialClosed(ctx context, requestID model.RequestID) ([]Operation, error) {
	s.eventCount++
	event := experimentv1.SearcherEvent_TrialClosed{
		TrialClosed: &experimentv1.TrialClosed{},
	}
	searcherEvent := experimentv1.SearcherEvent{
		Id:    s.eventCount,
		Event: &event,
	}

	err := s.SearcherEventQueue.Enqueue(&searcherEvent)
	return nil, err
}

func (s *customSearch) Snapshot() (json.RawMessage, error) {
	return json.Marshal(s.customSearchState)
}

func (s *customSearch) Restore(state json.RawMessage) error {
	if state == nil {
		return nil
	}
	return json.Unmarshal(state, &s.customSearchState)
}

func (s *customSearch) Unit() expconf.Unit {
	// TODO does unit make sense for custom search?
	return expconf.Batches
}
