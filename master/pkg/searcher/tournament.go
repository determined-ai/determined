package searcher

import (
	"encoding/json"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// tournamentSearch runs multiple search methods in tandem. Callbacks for completed operations
// are sent to the originating search method that created the corresponding operation.
type (
	tournamentSearchState struct {
		RunTable         map[int32]int     `json:"run_table"`
		SubSearchStates  []json.RawMessage `json:"sub_search_states"`
		SearchMethodType SearchMethodType  `json:"search_method_type"`
	}
	tournamentSearch struct {
		subSearches []SearchMethod
		tournamentSearchState
	}
)

func newTournamentSearch(mt SearchMethodType, subSearches ...SearchMethod) *tournamentSearch {
	return &tournamentSearch{
		subSearches: subSearches,
		tournamentSearchState: tournamentSearchState{
			RunTable:         make(map[int32]int),
			SubSearchStates:  make([]json.RawMessage, len(subSearches)),
			SearchMethodType: mt,
		},
	}
}

func (s *tournamentSearch) Snapshot() (json.RawMessage, error) {
	for i := range s.subSearches {
		b, err := s.subSearches[i].Snapshot()
		if err != nil {
			return nil, errors.Wrap(err, "failed to save subsearch")
		}
		s.SubSearchStates[i] = b
	}
	return json.Marshal(s.tournamentSearchState)
}

func (s *tournamentSearch) Restore(state json.RawMessage) error {
	err := json.Unmarshal(state, &s.tournamentSearchState)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal tournament state")
	}
	for i := range s.subSearches {
		if err := s.subSearches[i].Restore(s.SubSearchStates[i]); err != nil {
			return errors.Wrap(err, "failed to load subsearch")
		}
	}
	return nil
}

func (s *tournamentSearch) initialRuns(ctx context) ([]Action, error) {
	var actions []Action
	for i, subSearch := range s.subSearches {
		creates, err := subSearch.initialRuns(ctx)
		if err != nil {
			return nil, err
		}
		// Set SubSearchID on the create actions.
		for _, create := range creates {
			action := create.(Create)
			action.SubSearchID = i
			actions = append(actions, action)
		}
	}
	return actions, nil
}

func (s *tournamentSearch) runCreated(
	ctx context, runID int32, action Create,
) ([]Action, error) {
	s.RunTable[runID] = action.SubSearchID
	subSearch := s.subSearches[action.SubSearchID]
	ops, err := subSearch.runCreated(ctx, runID, action)
	return s.markCreates(action.SubSearchID, runID, ops), err
}

func (s *tournamentSearch) validationCompleted(
	ctx context, runID int32, metrics map[string]interface{},
) ([]Action, error) {
	subSearchID := s.RunTable[runID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.validationCompleted(ctx, runID, metrics)
	return s.markCreates(subSearchID, runID, ops), err
}

// trialClosed informs the searcher that the trial has been closed as a result of a Close operation.
func (s *tournamentSearch) runClosed(
	ctx context, runID int32,
) ([]Action, error) {
	subSearchID := s.RunTable[runID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.runClosed(ctx, runID)
	return s.markCreates(subSearchID, runID, ops), err
}

func (s *tournamentSearch) runExitedEarly(
	ctx context, runID int32, exitedReason model.ExitedReason,
) ([]Action, error) {
	subSearchID := s.RunTable[runID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.runExitedEarly(ctx, runID, exitedReason)
	return s.markCreates(subSearchID, runID, ops), err
}

// progress returns experiment progress as a float between 0.0 and 1.0.
func (s *tournamentSearch) progress(
	trialProgress map[int32]float64,
	trialsClosed map[int32]bool,
) float64 {
	sum := 0.0
	for subSearchID, subSearch := range s.subSearches {
		subSearchTrialProgress := map[int32]float64{}
		for rID, p := range trialProgress {
			if subSearchID == s.RunTable[rID] {
				subSearchTrialProgress[rID] = p
			}
		}
		subSearchTrialsClosed := map[int32]bool{}
		for rID, closed := range trialsClosed {
			if subSearchID == s.RunTable[rID] {
				subSearchTrialsClosed[rID] = closed
			}
		}
		sum += subSearch.progress(subSearchTrialProgress, subSearchTrialsClosed)
	}
	return sum / float64(len(s.subSearches))
}

func (s *tournamentSearch) Unit() expconf.Unit {
	return s.subSearches[0].Unit()
}

func (s *tournamentSearch) markCreates(subSearchID int, runID int32, actions []Action) []Action {
	for _, action := range actions {
		if _, ok := action.(Create); ok {
			s.RunTable[runID] = subSearchID
		}
	}
	return actions
}

func (s *tournamentSearch) Type() SearchMethodType {
	return s.SearchMethodType
}
