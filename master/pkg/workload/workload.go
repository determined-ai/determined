package workload

import (
	"fmt"
)

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
	Kind                  Kind `json:"kind"`
	ExperimentID          int  `json:"experiment_id"`
	TrialID               int  `json:"trial_id"`
	StepID                int  `json:"step_id"`
	NumBatches            int  `json:"num_batches"`
	TotalBatchesProcessed int  `json:"total_batches_processed"`
}

func (w Workload) String() string {
	var extra string
	if w.Kind == RunStep {
		extra += fmt.Sprintf(" (%d Batches)", w.NumBatches)
	}
	extra += fmt.Sprintf(" (%d Prior Batches)", w.TotalBatchesProcessed)
	return fmt.Sprintf("<%s%s: (%d,%d,%d)>", w.Kind, extra, w.ExperimentID, w.TrialID, w.StepID)
}
