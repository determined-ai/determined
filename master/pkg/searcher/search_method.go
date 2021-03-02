package searcher

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/workload"
)

type context struct {
	rand    *nprand.State
	hparams model.Hyperparameters
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
	// trainCompleted informs the searcher that the training workload initiated by the same searcher
	// has completed. It returns any new operations as a result of this workload completing.
	trainCompleted(ctx context, requestID model.RequestID, train Train) ([]Operation, error)
	// validationCompleted informs the searcher that the validation workload initiated by the same
	// searcher has completed. It returns any new operations as a result of this workload
	// completing.
	validationCompleted(
		ctx context, requestID model.RequestID, validate Validate, metrics workload.ValidationMetrics,
	) ([]Operation, error)
	// trialClosed informs the searcher that the trial has been closed as a result of a Close
	// operation.
	trialClosed(ctx context, requestID model.RequestID) ([]Operation, error)
	// progress returns experiment progress as a float between 0.0 and 1.0. As search methods
	// receive completed workloads, they should internally track progress.
	progress(totalUnitsCompleted float64) float64
	// trialExitedEarly informs the searcher that the trial has exited earlier than expected.
	trialExitedEarly(
		ctx context, requestID model.RequestID, exitedReason workload.ExitedReason,
	) ([]Operation, error)
	// TODO: refactor as model.Snapshotter interface or something
	model.Snapshotter
	model.InUnits
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
	// SHASearch is the SearchMethodType for a SHA searcher.
	SHASearch SearchMethodType = "sha"
	// AdaptiveSearch is the SearchMethodType for a adaptive searcher.
	AdaptiveSearch SearchMethodType = "adaptive"
	// AdaptiveSimpleSearch is the SearchMethodType for a adaptive simple searcher.
	AdaptiveSimpleSearch SearchMethodType = "adaptive_simple"
	// ASHASearch is the SearchMethodType for a ASHA searcher.
	ASHASearch SearchMethodType = "asha"
	// AdaptiveASHASearch is the SearchMethodType for a adaptive ASHA searcher.
	AdaptiveASHASearch SearchMethodType = "adaptive_asha"
	// PBTSearch is the SearchMethodType for a PBT searcher.
	PBTSearch SearchMethodType = "pbt"
)

// NewSearchMethod returns a new search method for the provided searcher configuration.
func NewSearchMethod(c model.SearcherConfig) SearchMethod {
	switch {
	case c.SingleConfig != nil:
		return newSingleSearch(*c.SingleConfig)
	case c.RandomConfig != nil:
		return newRandomSearch(*c.RandomConfig)
	case c.GridConfig != nil:
		return newGridSearch(*c.GridConfig)
	case c.SyncHalvingConfig != nil:
		return newSyncHalvingSearch(*c.SyncHalvingConfig)
	case c.AdaptiveConfig != nil:
		return newAdaptiveSearch(*c.AdaptiveConfig)
	case c.AdaptiveSimpleConfig != nil:
		return newAdaptiveSimpleSearch(*c.AdaptiveSimpleConfig)
	case c.AsyncHalvingConfig != nil:
		return newAsyncHalvingSearch(*c.AsyncHalvingConfig)
	case c.AdaptiveASHAConfig != nil:
		return newAdaptiveASHASearch(*c.AdaptiveASHAConfig)
	case c.PBTConfig != nil:
		return newPBTSearch(*c.PBTConfig)
	default:
		panic("no searcher type specified")
	}
}

type defaultSearchMethod struct{}

func (defaultSearchMethod) trialCreated(context, model.RequestID) ([]Operation, error) {
	return nil, nil
}
func (defaultSearchMethod) trainCompleted(context, model.RequestID, Train) ([]Operation, error) {
	return nil, nil
}

func (defaultSearchMethod) validationCompleted(
	context, model.RequestID, Validate, workload.ValidationMetrics,
) ([]Operation, error) {
	return nil, nil
}

func (defaultSearchMethod) trialClosed(context, model.RequestID) ([]Operation, error) {
	return nil, nil
}

func (defaultSearchMethod) trialExitedEarly( //nolint: unused
	context, model.RequestID, workload.ExitedReason) ([]Operation, error) {
	return []Operation{Shutdown{Failure: true}}, nil
}
