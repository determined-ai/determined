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
	// TrialReportProgress is a message sent to an experiment to indicate that a trial has
	// reported progress.
	TrialReportProgress struct {
		Progress searcher.PartialUnits
		IsRaw    bool
	}

	// UserInitiatedEarlyTrialExit is a user-injected message, provided through the early exit API. It
	// _should_ indicate the user is exiting, but in the event they don't, we will clean them up.
	UserInitiatedEarlyTrialExit struct {
		TrialID int32
		Reason  model.ExitedReason
	}

	// PatchTrialState is a message sent to an experiment to indicate that a trial has
	// changed state.
	PatchTrialState struct {
		TrialID int32
		State   model.StateWithReason
	}

	// TrialSearcherState is a message sent to an search to indicate that a run has
	// changed searcher state.
	TrialSearcherState struct {
		Create  searcher.Create
		TrialID *int32
		Stopped bool
		Closed  bool
	}
)

// Experiment is an interface that represents an experiment.
type Experiment interface {
	TrialReportProgress(trialID int32, msg TrialReportProgress) error
	TrialReportValidation(trialID int32, metrics map[string]interface{}) error
	UserInitiatedEarlyTrialExit(msg UserInitiatedEarlyTrialExit) error
	PatchTrialState(msg PatchTrialState) error
	SetGroupMaxSlots(msg sproto.SetGroupMaxSlots)
	SetGroupWeight(weight float64) error
	SetGroupPriority(priority int) error
	ActivateExperiment() error
	PauseExperiment() error
	CancelExperiment() error
	KillExperiment() error
}
