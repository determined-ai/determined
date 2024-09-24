package experiment

import (
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

// ExperimentRegistry is a registry of all experiments.
// It is meant to be used as a replacement for the actor registry.
// note: this can probably be a sync.map
var ExperimentRegistry = tasklist.NewRegistry[int, Experiment]()

// Experiment-specific interface types.
type (
	// RunReportProgress is a message sent to an experiment to indicate that a trial has
	// reported progress.
	RunReportProgress struct {
		Progress searcher.PartialUnits
		IsRaw    bool
	}

	// UserInitiatedEarlyRunExit is a user-injected message, provided through the early exit API. It
	// _should_ indicate the user is exiting, but in the event they don't, we will clean them up.
	UserInitiatedEarlyRunExit struct {
		RunID  int32
		Reason model.ExitedReason
	}

	// PatchRunState is a message sent to an experiment to indicate that a trial has
	// changed state.
	PatchRunState struct {
		RunID int32
		State model.StateWithReason
	}

	// RunSearcherState is a message sent to an search to indicate that a run has
	// changed searcher state.
	RunSearcherState struct {
		Create  searcher.Create
		RunID   *int32
		Stopped bool
		Closed  bool
	}
)

// Experiment is an interface that represents an experiment.
type Experiment interface {
	RunReportProgress(runID int32, msg RunReportProgress) error
	RunReportValidation(runID int32, metrics map[string]interface{}) error
	//TrialGetSearcherState(runID int32) (RunSearcherState, error)
	UserInitiatedEarlyRunExit(msg UserInitiatedEarlyRunExit) error
	PatchRunState(msg PatchRunState) error
	SetGroupMaxSlots(msg sproto.SetGroupMaxSlots)
	SetGroupWeight(weight float64) error
	SetGroupPriority(priority int) error
	ActivateExperiment() error
	PauseExperiment() error
	CancelExperiment() error
	KillExperiment() error
}
