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
	// initialRuns returns a set of initial runs the searcher would like to create.
	// This should be called only once after the searcher has been created.
	initialRuns(ctx context) ([]Action, error)
	// runCreated informs the searcher that a run has been created as a result of a Create
	// action and returns additional Actions to perform.
	runCreated(ctx context, runID int32, action Create) ([]Action, error)
	// validationCompleted informs the searcher that a validation metric has been reported.
	// xxx: reword comments
	validationCompleted(ctx context, runID int32,
		metrics map[string]interface{}) ([]Action, error)
	// runClosed informs the searcher that the trial has been closed as a result of a Close
	// operation.
	runClosed(ctx context, runID int32) ([]Action, error)
	// progress returns search progress as a float between 0.0 and 1.0.
	progress(map[int32]float64, map[int32]bool) float64
	// runExitedEarly informs the searcher that the run has exited earlier than expected.
	runExitedEarly(
		ctx context, runID int32, exitedReason model.ExitedReason,
	) ([]Action, error)

	// TODO: refactor as model.Snapshotter interface or something
	model.Snapshotter
	expconf.InUnits
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
	// AdaptiveSearch is the SearchMethodType for an adaptive searcher.
	AdaptiveSearch SearchMethodType = "adaptive"
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
	case c.RawAdaptiveASHAConfig != nil:
		return newAdaptiveASHASearch(*c.RawAdaptiveASHAConfig, c.SmallerIsBetter(), c.Metric())
	default:
		panic("no searcher type specified")
	}
}

type defaultSearchMethod struct{}

func (defaultSearchMethod) runCreated(context, int32, Create) ([]Action, error) {
	return nil, nil
}

func (defaultSearchMethod) validationCompleted(context, int32, map[string]interface{}) ([]Action, error) {
	return nil, nil
}

// nolint:unused
func (defaultSearchMethod) runClosed(context, int32) ([]Action, error) {
	return nil, nil
}

// nolint:unused
func (defaultSearchMethod) runExitedEarly(
	context, int32, model.ExitedReason,
) ([]Action, error) {
	return []Action{Shutdown{Failure: true}}, nil
}
