package searcher

import (
	"encoding/json"
	"fmt"
	"github.com/determined-ai/determined/master/pkg/workload"
	"io"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// ValidationFunction calculates the validation metric for the validation step.
type ValidationFunction func(random *rand.Rand, trialID, idx int) float64

// ConstantValidation returns the same validation metric for all validation steps.
func ConstantValidation(_ *rand.Rand, _, _ int) float64 { return 1 }

// RandomValidation returns a random validation metric for each validation step.
func RandomValidation(rand *rand.Rand, _, _ int) float64 { return rand.Float64() }

// SimulationResults holds all created trials and all executed workloads for each trial.
type SimulationResults map[RequestID][]Runnable

// MarshalJSON implements the json.Marshaler interface.
func (s SimulationResults) MarshalJSON() ([]byte, error) {
	summary := make(map[string]int)

	for _, ops := range s {
		var keyParts []string
		for _, op := range ops {
			switch op := op.(type) {
			case Train:
				switch op.Length.Unit {
				case model.Records:
					keyParts = append(keyParts, fmt.Sprintf("%dR", op.Length.Units))
				case model.Batches:
					keyParts = append(keyParts, fmt.Sprintf("%dB", op.Length.Units))
				case model.Epochs:
					keyParts = append(keyParts, fmt.Sprintf("%dE", op.Length.Units))
				}
			case Validate:
				keyParts = append(keyParts, "V")
			case Checkpoint:
				keyParts = append(keyParts, "C")
			default:
				return nil, errors.Errorf("unexpected operation: %v", op)
			}
		}
		summary[strings.Join(keyParts, " ")]++
	}

	return json.Marshal(summary)
}

// Simulation holds the configuration and results of simulated run of a searcher.
type Simulation struct {
	Results SimulationResults `json:"results"`
	Seed    int64             `json:"seed"`
}

// Simulate simulates the searcher.
func Simulate(
	s *Searcher, seed *int64, valFunc ValidationFunction, randomOrder bool, metricName string,
) (Simulation, error) {
	simulation := Simulation{
		Results: make(SimulationResults),
		Seed:    time.Now().Unix(),
	}
	random := rand.New(rand.NewSource(simulation.Seed))
	if seed != nil {
		simulation.Seed = *seed
		random = rand.New(rand.NewSource(*seed))
	}

	pending := make(map[RequestID][]Operation)
	trialIDs := make(map[RequestID]int)
	var requestIDs []RequestID
	ops, err := s.InitialOperations()
	if err != nil {
		return simulation, err
	}

	lastProgress := s.Progress()
	if lastProgress != 0.0 {
		return simulation, errors.Errorf("Initial searcher progress started at %f", lastProgress)
	}

	shutdown, err := handleOperations(pending, &requestIDs, ops)
	if err != nil {
		return simulation, err
	}

	nextTrialID := 1
	trialOpIdxs := map[RequestID]int{}
	for !shutdown {
		requestID, err := pickTrial(random, pending, requestIDs, randomOrder)
		if err != nil {
			return simulation, err
		}
		operation := pending[requestID][0]
		pending[requestID] = pending[requestID][1:]

		switch operation := operation.(type) {
		case Create:
			simulation.Results[requestID] = []Runnable{}
			trialIDs[requestID] = nextTrialID
			ops, err := s.TrialCreated(operation, nextTrialID)
			if err != nil {
				return simulation, err
			}
			trialOpIdxs[requestID] = 0
			shutdown, err = handleOperations(pending, &requestIDs, ops)
			if err != nil {
				return simulation, err
			}
			nextTrialID++
		case Runnable:
			metrics, err := generateMetrics(
				random, trialIDs[requestID], trialOpIdxs[requestID], operation, valFunc, metricName)
			if err != nil {
				return simulation, err
			}
			simulation.Results[requestID] = append(simulation.Results[requestID], operation)
			if train, ok := operation.(Train); ok {
				// If it's a train simulate we ran it to the event log as one huge workload,
				// so that progress is correctly updated.
				s.WorkloadCompleted(workload.CompletedMessage{Workload: workload.Workload{
					Kind:    workload.RunStep,
					TrialID: trialIDs[requestID],
					StepID:  trialOpIdxs[requestID],
				}}, float64(train.Length.Units))
			}
			ops, err := s.OperationCompleted(trialIDs[requestID], operation, metrics)
			if err != nil {
				return simulation, err
			}
			trialOpIdxs[requestID]++
			shutdown, err = handleOperations(pending, &requestIDs, ops)
			if err != nil {
				return simulation, err
			}
		case Close:
			delete(pending, requestID)
			ops, err := s.TrialClosed(requestID)
			if err != nil {
				return simulation, err
			}
			shutdown, err = handleOperations(pending, &requestIDs, ops)
			if err != nil {
				return simulation, err
			}
		default:
			return simulation, errors.Errorf("unexpected searcher operation: %T", operation)
		}
		if shutdown {
			if len(pending) != 0 {
				return simulation, errors.New("searcher shutdown prematurely")
			}
			break
		}

		progress := s.Progress()
		if progress < lastProgress {
			return simulation, errors.Errorf(
				"searcher progress dropped from %f%% to %f%%", lastProgress*100, progress*100)
		}
		lastProgress = progress
	}

	lastProgress = s.Progress()
	if lastProgress != 1.0 {
		return simulation, errors.Errorf(
			"searcher progress did not end at 100%%: %f%%", lastProgress*100)
	}
	if len(simulation.Results) != len(requestIDs) {
		return simulation, errors.New("more trials created than completed")
	}
	return simulation, nil
}

func handleOperations(
	pending map[RequestID][]Operation, requestIDs *[]RequestID, operations []Operation,
) (bool, error) {
	for _, operation := range operations {
		switch op := operation.(type) {
		case Create:
			*requestIDs = append(*requestIDs, op.RequestID)
			pending[op.RequestID] = []Operation{op}
		case Requested:
			pending[op.GetRequestID()] = append(pending[op.GetRequestID()], op)
		case Shutdown:
			return true, nil
		default:
			return false, errors.Errorf("unexpected operation: %T", operation)
		}
	}
	return false, nil
}

func pickTrial(
	random *rand.Rand, pending map[RequestID][]Operation, requestIDs []RequestID, randomOrder bool,
) (RequestID, error) {
	// If randomOrder is false, then return the first id from requestIDs that has any operations
	// pending.
	if !randomOrder {
		for _, requestID := range requestIDs {
			operations := pending[requestID]
			if len(operations) > 0 {
				return requestID, nil
			}
		}
		return RequestID{}, errors.New("tried to pick a trial when no trial had pending operations")
	}

	// If randomOrder is true, pseudo-randomly select a trial that has pending operations.
	var candidates []RequestID
	for requestID, operations := range pending {
		if len(operations) > 0 {
			candidates = append(candidates, requestID)
		}
	}
	if len(candidates) == 0 {
		return RequestID{}, errors.New("tried to pick a trial when no trial had pending operations")
	}

	// Map iteration order is nondeterministic, even for identical maps in the same run, so sort the
	// candidates before selecting one.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Before(candidates[j])
	})

	choice := random.Intn(len(candidates))
	return candidates[choice], nil
}

func generateMetrics(
	random *rand.Rand, trialID, opIdx int, operation Runnable, valFunc ValidationFunction,
	metric string,
) (interface{}, error) {
	switch operation.(type) {
	case Train:
		return make(map[string]interface{}), nil
	case Checkpoint:
		var requestID uuid.UUID
		_, err := io.ReadFull(random, requestID[:])
		if err != nil {
			return nil, err
		}
		return &workload.CheckpointMetrics{
			UUID:      requestID,
			Resources: map[string]int{},
		}, nil
	case Validate:
		return &workload.ValidationMetrics{
			NumInputs: 1,
			Metrics: map[string]interface{}{
				metric: valFunc(random, trialID, opIdx),
			},
		}, nil
	default:
		return nil, errors.Errorf("unexpected workload: %v", operation)
	}
}
