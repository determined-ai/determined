package searcher

import (
	"encoding/json"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
)

type (
	customSearchState struct {
		SearchMethodType     SearchMethodType `json:"search_method_type"`
		SearcherEventQueue   *SearcherEventQueue
		CustomSearchProgress float64
	}

	customSearch struct {
		expconf.CustomConfig
		customSearchState
	}
)

func newCustomSearch(config expconf.CustomConfig) SearchMethod {
	return &customSearch{
		CustomConfig: config,
		customSearchState: customSearchState{
			SearchMethodType:   CustomSearch,
			SearcherEventQueue: newSearcherEventQueue(),
		},
	}
}

func (s *customSearch) initialOperations(ctx context) ([]Operation, error) {
	// For this method and all the other methods in customSearch, the ID will be set in Enqueue.
	s.SearcherEventQueue.Enqueue(&experimentv1.SearcherEvent{
		Event: &experimentv1.SearcherEvent_InitialOperations{
			InitialOperations: &experimentv1.InitialOperations{},
		},
	})

	return nil, nil
}

func (s *customSearch) getSearcherEventQueue() *SearcherEventQueue {
	return s.SearcherEventQueue
}

func (s *customSearch) setCustomSearcherProgress(progress float64) {
	s.customSearchState.CustomSearchProgress = progress
}

func (s *customSearch) trialProgress(
	ctx context,
	requestID model.RequestID,
	progress PartialUnits,
) {
	s.SearcherEventQueue.Enqueue(&experimentv1.SearcherEvent{
		Event: &experimentv1.SearcherEvent_TrialProgress{
			TrialProgress: &experimentv1.TrialProgress{
				RequestId:    requestID.String(),
				PartialUnits: float64(progress),
			},
		},
	})
}

func (s *customSearch) trialCreated(ctx context, requestID model.RequestID) ([]Operation, error) {
	s.SearcherEventQueue.Enqueue(&experimentv1.SearcherEvent{
		Event: &experimentv1.SearcherEvent_TrialCreated{
			TrialCreated: &experimentv1.TrialCreated{
				RequestId: requestID.String(),
			},
		},
	})
	return nil, nil
}

func (s *customSearch) progress(
	trialProgress map[model.RequestID]PartialUnits,
	trialsClosed map[model.RequestID]bool,
) float64 {
	return s.customSearchState.CustomSearchProgress
}

func (s *customSearch) validationCompleted(
	ctx context, requestID model.RequestID, metric interface{}, op ValidateAfter,
) ([]Operation, error) {
	protoMetric, err := structpb.NewValue(metric)
	if err != nil {
		return nil, errors.Wrapf(err, "illegal type for metric=%v", metric)
	}
	s.SearcherEventQueue.Enqueue(&experimentv1.SearcherEvent{
		Event: &experimentv1.SearcherEvent_ValidationCompleted{
			ValidationCompleted: &experimentv1.ValidationCompleted{
				RequestId:           requestID.String(),
				ValidateAfterLength: op.ToProto().Length,
				Metric:              protoMetric,
			},
		},
	})
	return nil, nil
}

func (s *customSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason model.ExitedReason,
) ([]Operation, error) {
	s.SearcherEventQueue.Enqueue(&experimentv1.SearcherEvent{
		Event: &experimentv1.SearcherEvent_TrialExitedEarly{
			TrialExitedEarly: &experimentv1.TrialExitedEarly{
				RequestId:    requestID.String(),
				ExitedReason: exitedReason.ToSearcherProto(),
			},
		},
	})
	return nil, nil
}

func (s *customSearch) trialClosed(ctx context, requestID model.RequestID) ([]Operation, error) {
	s.SearcherEventQueue.Enqueue(&experimentv1.SearcherEvent{
		Event: &experimentv1.SearcherEvent_TrialClosed{
			TrialClosed: &experimentv1.TrialClosed{
				RequestId: requestID.String(),
			},
		},
	})
	return nil, nil
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
	// TODO: Does unit make sense for custom search?
	return expconf.Batches
}
