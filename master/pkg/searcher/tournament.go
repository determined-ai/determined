package searcher

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/workload"
)

// tournamentSearch runs multiple search methods in tandem. Callbacks for completed operations
// are sent to the originating search method that created the corresponding operation.
type (
	tournamentSearchState struct {
		SubSearchUnitsCompleted []float64               `json:"sub_search_units_completed"`
		TrialTable              map[model.RequestID]int `json:"trial_table"`
		SubSearchStates         []json.RawMessage       `json:"sub_search_states"`
	}
	tournamentSearch struct {
		subSearches []SearchMethod
		tournamentSearchState
	}
)

func newTournamentSearch(subSearches ...SearchMethod) *tournamentSearch {
	return &tournamentSearch{
		subSearches: subSearches,
		tournamentSearchState: tournamentSearchState{
			SubSearchUnitsCompleted: make([]float64, len(subSearches)),
			TrialTable:              make(map[model.RequestID]int),
			SubSearchStates:         make([]json.RawMessage, len(subSearches)),
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

func (s *tournamentSearch) initialOperations(ctx context) ([]Operation, error) {
	var operations []Operation
	for i, subSearch := range s.subSearches {
		ops, err := subSearch.initialOperations(ctx)
		if err != nil {
			return nil, err
		}
		s.markCreates(i, ops)
		operations = append(operations, ops...)
	}
	return operations, nil
}

func (s *tournamentSearch) trialCreated(
	ctx context, requestID model.RequestID,
) ([]Operation, error) {
	subSearchID := s.TrialTable[requestID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.trialCreated(ctx, requestID)
	return s.markCreates(subSearchID, ops), err
}

func (s *tournamentSearch) trainCompleted(
	ctx context, requestID model.RequestID, train Train,
) ([]Operation, error) {
	subSearchID := s.TrialTable[requestID]
	subSearch := s.subSearches[subSearchID]
	s.SubSearchUnitsCompleted[subSearchID] += float64(train.Length.Units)
	ops, err := subSearch.trainCompleted(ctx, requestID, train)
	return s.markCreates(subSearchID, ops), err
}

func (s *tournamentSearch) checkpointCompleted(
	ctx context, requestID model.RequestID, checkpoint Checkpoint, metrics workload.CheckpointMetrics,
) ([]Operation, error) {
	subSearchID := s.TrialTable[requestID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.checkpointCompleted(ctx, requestID, checkpoint, metrics)
	return s.markCreates(subSearchID, ops), err
}

func (s *tournamentSearch) validationCompleted(
	ctx context, requestID model.RequestID, validate Validate, metrics workload.ValidationMetrics,
) ([]Operation, error) {
	subSearchID := s.TrialTable[requestID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.validationCompleted(ctx, requestID, validate, metrics)
	return s.markCreates(subSearchID, ops), err
}

// trialClosed informs the searcher that the trial has been closed as a result of a Close operation.
func (s *tournamentSearch) trialClosed(
	ctx context, requestID model.RequestID,
) ([]Operation, error) {
	subSearchID := s.TrialTable[requestID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.trialClosed(ctx, requestID)
	return s.markCreates(subSearchID, ops), err
}

func (s *tournamentSearch) trialExitedEarly(
	ctx context, requestID model.RequestID, exitedReason workload.ExitedReason,
) ([]Operation, error) {
	subSearchID := s.TrialTable[requestID]
	subSearch := s.subSearches[subSearchID]
	ops, err := subSearch.trialExitedEarly(ctx, requestID, exitedReason)
	return s.markCreates(subSearchID, ops), err
}

// progress returns experiment progress as a float between 0.0 and 1.0.
func (s *tournamentSearch) progress(float64) float64 {
	sum := 0.0
	for id, subSearch := range s.subSearches {
		sum += subSearch.progress(s.SubSearchUnitsCompleted[id])
	}
	return sum / float64(len(s.subSearches))
}

func (s *tournamentSearch) Unit() model.Unit {
	return s.subSearches[0].Unit()
}

func (s *tournamentSearch) markCreates(subSearchID int, operations []Operation) []Operation {
	for _, operation := range operations {
		switch operation := operation.(type) {
		case Create:
			s.TrialTable[operation.RequestID] = subSearchID
		}
	}
	return operations
}
