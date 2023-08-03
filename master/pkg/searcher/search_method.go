package searcher

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type context struct {
	rand    *nprand.State
	hparams expconf.Hyperparameters
}

// SearchMethod is the interface for hyper-parameter tuning methods. Implementations of this
// interface should use pointer receivers to ensure interface equality is calculated through pointer
// equality.
type SearchMethod interface {
	// initialOperations returns a set of initial operations that the searcher would like to take.
	// This should be called only once after the searcher has been created.
	initialOperations(ctx context) ([]Operation, error)
	// trialCreated informs the searcher that a trial has been created as a result of a Create
	// operation.
	trialCreated(ctx context, requestID model.RequestID) ([]Operation, error)
	// validationCompleted informs the searcher that the validation workload initiated by the same
	// searcher has completed. It returns any new operations as a result of this workload
	// completing.
	validationCompleted(ctx context, requestID model.RequestID,
		metric interface{}, op ValidateAfter) ([]Operation, error)
	// trialClosed informs the searcher that the trial has been closed as a result of a Close
	// operation.
	trialClosed(ctx context, requestID model.RequestID) ([]Operation, error)
	// progress returns experiment progress as a float between 0.0 and 1.0.
	progress(map[model.RequestID]PartialUnits, map[model.RequestID]bool) float64
	// trialExitedEarly informs the searcher that the trial has exited earlier than expected.
	trialExitedEarly(
		ctx context, requestID model.RequestID, exitedReason model.ExitedReason,
	) ([]Operation, error)

	// TODO: refactor as model.Snapshotter interface or something
	model.Snapshotter
	expconf.InUnits
}

// CustomSearchMethod is the interface for the custom search method.
type CustomSearchMethod interface {
	getSearcherEventQueue() *SearcherEventQueue
	setCustomSearcherProgress(progress float64)
	trialProgress(ctx context, requestID model.RequestID, progress PartialUnits)
}

// SearchMethodType is the type of a SearchMethod. It is saved in snapshots to be used
// when shimming json blobs of searcher snapshots.
type SearchMethodType string

const (
	// SingleSearch is the SearchMethodType for a single searcher.
	SingleSearch SearchMethodType = "single"
	// RandomSearch is the SearchMethodType for a random searcher.
	RandomSearch SearchMethodType = "random"
	// GridSearch is the SearchMethodType for a grid searcher.
	GridSearch SearchMethodType = "grid"
	// AdaptiveSearch is the SearchMethodType for an adaptive searcher.
	AdaptiveSearch SearchMethodType = "adaptive"
	// ASHASearch is the SearchMethodType for an ASHA searcher.
	ASHASearch SearchMethodType = "asha"
	// AdaptiveASHASearch is the SearchMethodType for an adaptive ASHA searcher.
	AdaptiveASHASearch SearchMethodType = "adaptive_asha"
	// CustomSearch is the SearchMethodType for a custom searcher.
	CustomSearch SearchMethodType = "custom_search"
)

// NewSearchMethod returns a new search method for the provided searcher configuration.
func NewSearchMethod(c expconf.SearcherConfig) SearchMethod {
	switch {
	case c.RawSingleConfig != nil:
		return newSingleSearch(*c.RawSingleConfig)
	case c.RawRandomConfig != nil:
		return newRandomSearch(*c.RawRandomConfig)
	case c.RawGridConfig != nil:
		return newGridSearch(*c.RawGridConfig)
	case c.RawAsyncHalvingConfig != nil:
		if c.RawAsyncHalvingConfig.StopOnce() {
			return newAsyncHalvingStoppingSearch(*c.RawAsyncHalvingConfig, c.SmallerIsBetter())
		}
		return newAsyncHalvingSearch(*c.RawAsyncHalvingConfig, c.SmallerIsBetter())
	case c.RawAdaptiveASHAConfig != nil:
		return newAdaptiveASHASearch(*c.RawAdaptiveASHAConfig, c.SmallerIsBetter())
	case c.RawCustomConfig != nil:
		return newCustomSearch(*c.RawCustomConfig)
	default:
		panic("no searcher type specified")
	}
}

type defaultSearchMethod struct{}

func (defaultSearchMethod) trialCreated(context, model.RequestID) ([]Operation, error) {
	return nil, nil
}

func (defaultSearchMethod) validationCompleted(
	context, model.RequestID, interface{}, ValidateAfter,
) ([]Operation, error) {
	return nil, nil
}

// nolint:unused
func (defaultSearchMethod) trialClosed(context, model.RequestID) ([]Operation, error) {
	return nil, nil
}

// nolint:unused
func (defaultSearchMethod) trialExitedEarly(
	context, model.RequestID, model.ExitedReason,
) ([]Operation, error) {
	return []Operation{Shutdown{Failure: true}}, nil
}
