package searcher

// tournamentSearch runs multiple search methods in tandem. Callbacks for completed operations
// are sent to the originating search method that created the corresponding operation.
type tournamentSearch struct {
	subSearches        []SearchMethod
	trialTable         map[RequestID]SearchMethod
	workloadsCompleted map[SearchMethod]int
}

func newTournamentSearch(subSearches ...SearchMethod) SearchMethod {
	return &tournamentSearch{
		subSearches:        subSearches,
		trialTable:         make(map[RequestID]SearchMethod),
		workloadsCompleted: make(map[SearchMethod]int),
	}
}

func (s *tournamentSearch) initialOperations(ctx Context) {
	for _, subSearch := range s.subSearches {
		subSearch.initialOperations(ctx)
		s.markCreates(ctx, subSearch)
	}
}

func (s *tournamentSearch) trainCompleted(ctx Context, requestID RequestID, message Workload) {
	subSearch := s.trialTable[requestID]
	s.workloadsCompleted[subSearch]++
	subSearch.trainCompleted(ctx, requestID, message)
	s.markCreates(ctx, subSearch)
}

func (s *tournamentSearch) validationCompleted(
	ctx Context, requestID RequestID, message Workload, metrics ValidationMetrics,
) error {
	subSearch := s.trialTable[requestID]
	s.workloadsCompleted[subSearch]++
	err := subSearch.validationCompleted(ctx, requestID, message, metrics)
	s.markCreates(ctx, subSearch)
	return err
}

// progress returns experiment progress as a float between 0.0 and 1.0.
func (s *tournamentSearch) progress(int) float64 {
	sum := 0.0
	for _, subSearch := range s.subSearches {
		sum += subSearch.progress(s.workloadsCompleted[subSearch])
	}
	return sum / float64(len(s.subSearches))
}

func (s *tournamentSearch) markCreates(ctx Context, subSearch SearchMethod) {
	for _, operation := range ctx.(*context).pendingOperations() {
		switch operation := operation.(type) {
		case Create:
			if _, ok := s.trialTable[operation.RequestID]; !ok {
				s.trialTable[operation.RequestID] = subSearch
			}
		}
	}
}
