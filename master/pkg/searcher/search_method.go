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

// SearchMethod is the interface for hyperparameter tuning methods. Implementations of this
// interface should use pointer receivers to ensure interface equality is calculated through pointer
// equality.
type SearchMethod interface {
	// initialTrials returns a set of initial trials the searcher would like to create.
	// This should be called only once after the searcher has been created.
	initialTrials(ctx context) ([]Action, error)
	// trialCreated informs the searcher that a trial has been created as a result of a Create
	// action and returns any additional Actions to perform.
	trialCreated(ctx context, trialID int32, action Create) ([]Action, error)
	// validationCompleted informs the searcher that a validation metric has been reported
	// and returns any resulting actions.
	validationCompleted(ctx context, trialID int32,
		metrics map[string]interface{}) ([]Action, error)
	// trialExited informs the searcher that the trial has exited.
	trialExited(ctx context, trialID int32) ([]Action, error)
	// progress returns search progress as a float between 0.0 and 1.0.
	progress(map[int32]float64, map[int32]bool) float64
	// trialExitedEarly informs the searcher that the trial has exited earlier than expected.
	trialExitedEarly(
		ctx context, trialID int32, exitedReason model.ExitedReason,
	) ([]Action, error)

	// TODO: refactor as model.Snapshotter interface or something
	model.Snapshotter
	Type() SearchMethodType
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
	// ASHASearch is the SearchMethodType for an ASHA searcher.
	ASHASearch SearchMethodType = "asha"
	// AdaptiveASHASearch is the SearchMethodType for an adaptive ASHA searcher.
	AdaptiveASHASearch SearchMethodType = "adaptive_asha"
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
		return newAsyncHalvingStoppingSearch(*c.RawAsyncHalvingConfig, c.SmallerIsBetter(), c.Metric())
	case c.RawAdaptiveASHAConfig != nil:
		return newAdaptiveASHASearch(*c.RawAdaptiveASHAConfig, c.SmallerIsBetter(), c.Metric())
	default:
		panic("no searcher type specified")
	}
}

type defaultSearchMethod struct{}

func (defaultSearchMethod) trialCreated(context, int32, Create) ([]Action, error) {
	return nil, nil
}

func (defaultSearchMethod) validationCompleted(context, int32, map[string]interface{}) ([]Action, error) {
	return nil, nil
}

// nolint:unused
func (defaultSearchMethod) trialExited(context, int32) ([]Action, error) {
	return nil, nil
}

// nolint:unused
func (defaultSearchMethod) trialExitedEarly(
	context, int32, model.ExitedReason,
) ([]Action, error) {
	return []Action{Shutdown{Failure: true}}, nil
}
