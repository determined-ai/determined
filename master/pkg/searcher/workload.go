package searcher

import "fmt"

// Kind defines the kind of workload that should be executed by trial runners.
type Kind string

const (
	// RunStep signals to a trial runner that it should run a training step.
	RunStep Kind = "RUN_STEP"
	// ComputeValidationMetrics signals to a trial runner it should compute validation metrics.
	ComputeValidationMetrics Kind = "COMPUTE_VALIDATION_METRICS"
	// CheckpointModel signals to the trial runner that the current model state should be
	// checkpointed.
	CheckpointModel Kind = "CHECKPOINT_MODEL"
	// Terminate signals to the trial runner that the current model state should be
	// terminated.
	Terminate Kind = "TERMINATE"
)

// Workload encompasses a single unit of work that a trial needs do before waiting for more work.
type Workload struct {
	Kind         Kind `json:"kind"`
	ExperimentID int  `json:"experiment_id"`
	TrialID      int  `json:"trial_id"`
	StepID       int  `json:"step_id"`
}

func (w Workload) String() string {
	return fmt.Sprintf("<%s: (%d,%d,%d)>", w.Kind, w.ExperimentID, w.TrialID, w.StepID)
}
