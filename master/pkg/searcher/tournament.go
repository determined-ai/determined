package searcher

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// tournamentSearch trial multiple search methods in tandem. Callbacks for completed actions
// are sent to the originating search method that initiated the corresponding action.
type (
	tournamentSearchState struct {
		TrialTable       map[int32]int     `json:"trial_table"`
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
			TrialTable:       make(map[int32]int),
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

func (s *tournamentSearch) initialTrials(ctx context) ([]Action, error) {
	var actions []Action
	for i, subSearch := range s.subSearches {
		creates, err := subSearch.initialTrials(ctx)
		if err != nil {
			return nil, err
		}
		s.markCreates(i, creates)
		actions = append(actions, creates...)
	}
	return actions, nil
}

func (s *tournamentSearch) trialCreated(
	ctx context, trialID int32, action Create,
) ([]Action, error) {
	s.TrialTable[trialID] = action.SubSearchID
	subSearch := s.subSearches[action.SubSearchID]
	ops, err := subSearch.trialCreated(ctx, trialID, action)
	return s.markCreates(action.SubSearchID, ops), err
}

func (s *tournamentSearch) validationCompleted(
	ctx context, trialID int32, metrics map[string]interface{},
) ([]Action, error) {
	subSearchID := s.TrialTable[trialID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.validationCompleted(ctx, trialID, metrics)
	return s.markCreates(subSearchID, ops), err
}

// runExited informs the searcher that the run has exited.
func (s *tournamentSearch) trialExited(
	ctx context, trialID int32,
) ([]Action, error) {
	subSearchID := s.TrialTable[trialID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.trialExited(ctx, trialID)
	return s.markCreates(subSearchID, ops), err
}

func (s *tournamentSearch) trialExitedEarly(
	ctx context, trialID int32, exitedReason model.ExitedReason,
) ([]Action, error) {
	subSearchID := s.TrialTable[trialID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.trialExitedEarly(ctx, trialID, exitedReason)
	return s.markCreates(subSearchID, ops), err
}

// progress returns experiment progress as a float between 0.0 and 1.0.
func (s *tournamentSearch) progress(
	trialProgress map[int32]float64,
	trialsClosed map[int32]bool,
) float64 {
	sum := 0.0
	for subSearchID, subSearch := range s.subSearches {
		subSearchTrialProgress := map[int32]float64{}
		for tID, p := range trialProgress {
			if subSearchID == s.TrialTable[tID] {
				subSearchTrialProgress[tID] = p
			}
		}
		subSearchTrialsClosed := map[int32]bool{}
		for tID, closed := range trialsClosed {
			if subSearchID == s.TrialTable[tID] {
				subSearchTrialsClosed[tID] = closed
			}
		}
		sum += subSearch.progress(subSearchTrialProgress, subSearchTrialsClosed)
	}
	return sum / float64(len(s.subSearches))
}

func (s *tournamentSearch) markCreates(subSearchID int, actions []Action) []Action {
	for i, action := range actions {
		if _, ok := action.(Create); ok {
			create := action.(Create)
			create.SubSearchID = subSearchID
			actions[i] = create
		}
	}
	return actions
}

func (s *tournamentSearch) Type() SearchMethodType {
	return s.SearchMethodType
}
