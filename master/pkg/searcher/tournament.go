package searcher

// tournamentSearch runs multiple search methods in tandem. Callbacks for completed operations
// are sent to the originating search method that created the corresponding operation.
type tournamentSearch struct {
	subSearches        []SearchMethod
	trialTable         map[RequestID]SearchMethod
	workloadsCompleted map[SearchMethod]int
}

func newTournamentSearch(subSearches ...SearchMethod) *tournamentSearch {
	return &tournamentSearch{
		subSearches:        subSearches,
		trialTable:         make(map[RequestID]SearchMethod),
		workloadsCompleted: make(map[SearchMethod]int),
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
	ctx context, requestID RequestID, message Workload,
) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	s.workloadsCompleted[subSearch]++
	ops, err := subSearch.trainCompleted(ctx, requestID, message)
	return s.markCreates(subSearch, ops), err
}

func (s *tournamentSearch) checkpointCompleted(
	ctx context, requestID RequestID, message Workload, metrics CheckpointMetrics,
) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	s.workloadsCompleted[subSearch]++
	ops, err := subSearch.checkpointCompleted(ctx, requestID, message, metrics)
	return s.markCreates(subSearch, ops), err
}

func (s *tournamentSearch) validationCompleted(
	ctx context, requestID RequestID, message Workload, metrics ValidationMetrics,
) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	s.workloadsCompleted[subSearch]++
	ops, err := subSearch.validationCompleted(ctx, requestID, message, metrics)
	return s.markCreates(subSearch, ops), err
}

// trialClosed informs the searcher that the trial has been closed as a result of a Close operation.
func (s *tournamentSearch) trialClosed(ctx context, requestID RequestID) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	ops, err := subSearch.trialClosed(ctx, requestID)
	return s.markCreates(subSearch, ops), err
}

func (s *tournamentSearch) trialExitedEarly(
	ctx context, requestID RequestID, message Workload,
) ([]Operation, error) {
	subSearch := s.trialTable[requestID]
	ops, err := subSearch.trialExitedEarly(ctx, requestID, message)
	return s.markCreates(subSearch, ops), err
}

// progress returns experiment progress as a float between 0.0 and 1.0.
func (s *tournamentSearch) progress(int) float64 {
	sum := 0.0
	for _, subSearch := range s.subSearches {
		sum += subSearch.progress(s.workloadsCompleted[subSearch])
	}
	return sum / float64(len(s.subSearches))
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
