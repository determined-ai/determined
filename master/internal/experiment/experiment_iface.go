package experiment

import (
	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// ExperimentRegistry is a registry of all experiments.
// It is meant to be used as a replacement for the actor registry.
// note: this can probably be a sync.map
var ExperimentRegistry = tasklist.NewRegistry[int, Experiment]()

// Experiment-specific interface types.
type (
	// TrialCompleteOperation is a message sent to an experiment to indicate that a trial has
	// completed an operation.
	TrialCompleteOperation struct {
		RequestID model.RequestID
		Op        searcher.ValidateAfter
		Metric    interface{}
	}

	// TrialReportProgress is a message sent to an experiment to indicate that a trial has
	// reported progress.
	TrialReportProgress struct {
		RequestID model.RequestID
		Progress  searcher.PartialUnits
	}

	// UserInitiatedEarlyTrialExit is a user-injected message, provided through the early exit API. It
	// _should_ indicate the user is exiting, but in the event they don't, we will clean them up.
	UserInitiatedEarlyTrialExit struct {
		RequestID model.RequestID
		Reason    model.ExitedReason
	}

	// PatchTrialState is a message sent to an experiment to indicate that a trial has
	// changed state.
	PatchTrialState struct {
		RequestID model.RequestID
		State     model.StateWithReason
	}

	// TrialSearcherState is a message sent to an experiment to indicate that a trial has
	// changed searcher state.
	TrialSearcherState struct {
		Create   searcher.Create
		Op       searcher.ValidateAfter
		Complete bool
		Closed   bool
	}
)

// Experiment is an interface that represents an experiment.
type Experiment interface {
	TrialCompleteOperation(msg TrialCompleteOperation) error
	TrialReportProgress(msg TrialReportProgress) error
	TrialGetSearcherState(requestID model.RequestID) (TrialSearcherState, error)
	UserInitiatedEarlyTrialExit(msg UserInitiatedEarlyTrialExit) error
	PatchTrialState(msg PatchTrialState) error
	SetGroupMaxSlots(msg sproto.SetGroupMaxSlots)
	SetGroupWeight(weight float64) error
	SetGroupPriority(priority int) error
	PerformSearcherOperations(msg *apiv1.PostSearcherOperationsRequest) error
	GetSearcherEventsWatcher() (*searcher.EventsWatcher, error)
	UnwatchEvents(id uuid.UUID) error
	ActivateExperiment() error
	PauseExperiment() error
	CancelExperiment() error
	KillExperiment() error
}
