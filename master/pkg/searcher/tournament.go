package searcher

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/workload"
)

// tournamentSearch runs multiple search methods in tandem. Callbacks for completed operations
// are sent to the originating search method that created the corresponding operation.
type tournamentSearch struct {
	subSearches             []SearchMethod
	subSearchUnitsCompleted map[SearchMethod]float64
	trialTable              map[RequestID]SearchMethod
}

func newTournamentSearch(subSearches ...SearchMethod) *tournamentSearch {
	return &tournamentSearch{
		subSearches:             subSearches,
		subSearchUnitsCompleted: make(map[SearchMethod]float64),
		trialTable:              make(map[RequestID]SearchMethod),
	}
}

func (s *tournamentSearch) initialOperations(ctx context) ([]Operation, error) {
	var operations []Operation
	for _, subSearch := range s.subSearches {
		ops, err := subSearch.initialOperations(ctx)
		if err != nil {
			return nil, err
		}
		s.markCreates(subSearch, ops)
		operations = append(operations, ops...)
	}
	return operations, nil
}

func (s *tournamentSearch) trialCreated(ctx context, requestID RequestID) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	ops, err := subSearch.trialCreated(ctx, requestID)
	return s.markCreates(subSearch, ops), err
}

func (s *tournamentSearch) trainCompleted(
	ctx context, requestID RequestID, train Train,
) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	s.subSearchUnitsCompleted[subSearch] += float64(train.Length.Units)
	ops, err := subSearch.trainCompleted(ctx, requestID, train)
	return s.markCreates(subSearch, ops), err
}

func (s *tournamentSearch) checkpointCompleted(
	ctx context, requestID RequestID, checkpoint Checkpoint, metrics workload.CheckpointMetrics,
) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	ops, err := subSearch.checkpointCompleted(ctx, requestID, checkpoint, metrics)
	return s.markCreates(subSearch, ops), err
}

func (s *tournamentSearch) validationCompleted(
	ctx context, requestID RequestID, validate Validate, metrics workload.ValidationMetrics,
) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	ops, err := subSearch.validationCompleted(ctx, requestID, validate, metrics)
	return s.markCreates(subSearch, ops), err
}

// trialClosed informs the searcher that the trial has been closed as a result of a Close operation.
func (s *tournamentSearch) trialClosed(ctx context, requestID RequestID) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	ops, err := subSearch.trialClosed(ctx, requestID)
	return s.markCreates(subSearch, ops), err
}

func (s *tournamentSearch) trialExitedEarly(ctx context, requestID RequestID) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	ops, err := subSearch.trialExitedEarly(ctx, requestID)
	return s.markCreates(subSearch, ops), err
}

// progress returns experiment progress as a float between 0.0 and 1.0.
func (s *tournamentSearch) progress(float64) float64 {
	sum := 0.0
	for _, subSearch := range s.subSearches {
		sum += subSearch.progress(s.subSearchUnitsCompleted[subSearch])
	}
	return sum / float64(len(s.subSearches))
}

func (s *tournamentSearch) Unit() model.Unit {
	return s.subSearches[0].Unit()
}

func (s *tournamentSearch) markCreates(subSearch SearchMethod, operations []Operation) []Operation {
	for _, operation := range operations {
		switch operation := operation.(type) {
		case Create:
			s.trialTable[operation.RequestID] = subSearch
		}
	}
	return operations
}
