package searcher

import (
	"encoding/json"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type (
	manualSearchState struct {
		CreatedTrials    int              `json:"created_trials"`
		PendingTrials    int              `json:"pending_trials"`
		SearchMethodType SearchMethodType `json:"search_method_type"`
	}

	manualSearch struct {
		defaultSearchMethod
		expconf.ManualConfig
		manualSearchState
	}
)

func newManualSearch(config expconf.ManualConfig) SearchMethod {
	return &manualSearch{
		ManualConfig: config,
		manualSearchState: manualSearchState{
			SearchMethodType: ManualSearch,
		},
	}
}

func (s *manualSearch) initialOperations(ctx context) ([]Operation, error) {
	return nil, nil
}

func (s *manualSearch) progress(
	trialProgress map[model.RequestID]model.PartialUnits,
	trialsClosed map[model.RequestID]bool,
) float64 {
	return 0
}

// trialExitedEarly creates a new trial upon receiving an InvalidHP workload.
// Otherwise, it does nothing since actions are not taken based on search status.
func (s *manualSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason model.ExitedReason,
) ([]Operation, error) {
	return nil, nil
}

func (s *manualSearch) trialClosed(ctx context, requestID model.RequestID) ([]Operation, error) {
	return nil, nil
}
func (s *manualSearch) Snapshot() (json.RawMessage, error) {
	return json.Marshal(s.manualSearchState)
}

func (s *manualSearch) Restore(state json.RawMessage) error {
	if state == nil {
		return nil
	}
	return json.Unmarshal(state, &s.manualSearchState)
}
