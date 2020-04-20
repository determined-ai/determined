package internal

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

type stepInfo struct {
	hasValidation bool
	hasCheckpoint bool
}

type trialWorkloadSequencer struct {
	// steps represents the operations that have been requested for this trial (whether by the
	// searcher, checkpoint-after-validation, or min validation/checkpoint period). Each stepInfo
	// implicitly represents a training operation and explicitly indicates the presence of validation
	// or checkpoint operations. The 0-th element must be a dummy stepInfo object with both members
	// set to true, corresponding to no real step; its presence makes indexing into the array match
	// step IDs and makes searching back for a validation/checkpoint a little nicer.
	steps []stepInfo
	// curStep and curStepDone represent the workloads that have been finished so far. The training
	// step for the step with ID curStep is implicitly done, and the state of the validation and
	// checkpoint are indicated by curStepDone.
	curStep     int
	curStepDone stepInfo

	curWorkload      searcher.Workload
	curWorkloadValid bool

	latestCheckpoint *model.Checkpoint
	experiment       *model.Experiment
	create           searcher.Create

	checkpointPolicy    string
	minValidationPeriod *int
	minCheckpointPeriod *int

	trialID      int
	trialIDValid bool
}

func newTrialWorkloadSequencer(
	exp *model.Experiment, create searcher.Create, firstCheckpoint *model.Checkpoint,
) *trialWorkloadSequencer {
	return &trialWorkloadSequencer{
		steps:               []stepInfo{{true, true}},
		curStepDone:         stepInfo{true, true},
		checkpointPolicy:    exp.Config.CheckpointPolicy,
		minValidationPeriod: exp.Config.MinValidationPeriod,
		minCheckpointPeriod: exp.Config.MinCheckpointPeriod,
		latestCheckpoint:    firstCheckpoint,
		create:              create,
		experiment:          exp,
	}
}

func (s *trialWorkloadSequencer) SetTrialID(trialID int) {
	s.trialID = trialID
	s.trialIDValid = true
}

func (s *trialWorkloadSequencer) LatestCheckpoint() *model.Checkpoint {
	return s.latestCheckpoint
}

func (s *trialWorkloadSequencer) WorkloadManagerType() model.WorkloadManagerType {
	return model.TrialWorkloadManagerType
}

func (s *trialWorkloadSequencer) OperationRequested(op searcher.WorkloadOperation) error {
	switch op.Kind {
	case searcher.RunStep:
		if op.StepID != len(s.steps) {
			return errors.New("illegal step requested")
		}
		s.steps = append(s.steps, stepInfo{})

	case searcher.CheckpointModel:
		if op.StepID < s.curStep || op.StepID >= len(s.steps) {
			return errors.New("illegal checkpoint requested")
		}
		s.steps[op.StepID].hasCheckpoint = true

	case searcher.ComputeValidationMetrics:
		if op.StepID < s.curStep || op.StepID >= len(s.steps) {
			return errors.New("illegal validation requested")
		}
		s.steps[op.StepID].hasValidation = true

	default:
		return errors.Errorf("illegal workload for trialWorkloadSequencer: %v", op.Kind)
	}
	s.curWorkloadValid = false
	return nil
}

func (s *trialWorkloadSequencer) WorkloadCompleted(
	msg searcher.CompletedMessage, experimentFuture actor.Response,
) error {
	// Checkpoints are allowed even if they were not specified by sequencer.workload(). This can
	// occur after a call to precloseCheckpointWorkload or during a replay.
	if s.UpToDate() {
		if msg.Workload.Kind != searcher.CheckpointModel {
			return errors.Errorf(
				"illegal non-checkpoint workload completed message received: %s", msg.Workload)
		}
	} else {
		w, err := s.Workload()
		if err != nil {
			return errors.Wrap(err, "error checking workload")
		}
		if msg.Workload != w {
			if msg.Workload.Kind != searcher.CheckpointModel {
				return errors.Errorf(
					"illegal completed message received: expected checkpoint or %s, got %s", w, msg.Workload)
			}
		}
	}

	switch msg.Workload.Kind {
	case searcher.RunStep:
		s.curStep++
		s.curStepDone = stepInfo{}
		if s.minValidationNeeded() {
			s.steps[msg.Workload.StepID].hasValidation = true
		}
		if s.minCheckpointNeeded() {
			s.steps[msg.Workload.StepID].hasCheckpoint = true
		}
	case searcher.CheckpointModel:
		// During replay, a checkpoint can show up for earlier than the current step ID if the
		// original trial was descheduled after a failure. Example: a trial runs steps 1 through 5,
		// then crashes, then reruns steps 1 through 3, and then gets descheduled and checkpoints
		// at step 3. The resulting event log (which does not know about crashes or save duplicate
		// events) will look like train1,train2,train3,train4,train5,checkpoint3.
		if msg.Workload.StepID > s.curStep {
			return errors.Errorf("invalid StepID in workload completed message: %s", msg.Workload)
		}
		s.steps[msg.Workload.StepID].hasCheckpoint = true
		if msg.Workload.StepID == s.curStep {
			s.curStepDone.hasCheckpoint = true
		}
		checkpoint := checkpointFromCheckpointMetrics(*msg.CheckpointMetrics)
		s.latestCheckpoint = &checkpoint
	case searcher.ComputeValidationMetrics:
		if s.curStep != msg.Workload.StepID {
			return errors.Errorf("invalid StepID in workload completed message: %s", msg.Workload)
		}
		s.curStepDone.hasValidation = true
		switch s.checkpointPolicy {
		case model.AllCheckpointPolicy:
			s.steps[msg.Workload.StepID].hasCheckpoint = true
		case model.BestCheckpointPolicy:
			if isBestValidation := experimentFuture.Get().(bool); isBestValidation {
				s.steps[msg.Workload.StepID].hasCheckpoint = true
			}
		}
	default:
		return errors.New("invalid operation for trialWorkloadSequencer")
	}
	s.curWorkloadValid = false
	return nil
}

func (s *trialWorkloadSequencer) Workload() (searcher.Workload, error) {
	if s.curWorkloadValid {
		return s.curWorkload, nil
	}

	if s.UpToDate() {
		return searcher.Workload{},
			errors.New("cannot call sequencer.Workload() with sequencer.UpToDate() == true")
	}
	if !s.trialIDValid {
		return searcher.Workload{},
			errors.New("cannot call sequencer.Workload() before sequencer.SetTrialID()")
	}

	step := s.steps[s.curStep]
	stepID := s.curStep
	var kind searcher.Kind
	switch {
	case step.hasValidation && !s.curStepDone.hasValidation:
		kind = searcher.ComputeValidationMetrics
	case step.hasCheckpoint && !s.curStepDone.hasCheckpoint:
		kind = searcher.CheckpointModel
	default:
		kind = searcher.RunStep
		stepID++
	}
	s.curWorkload = searcher.Workload{
		Kind:         kind,
		ExperimentID: s.experiment.ID,
		TrialID:      s.trialID,
		StepID:       stepID,
	}
	s.curWorkloadValid = true
	return s.curWorkload, nil
}

func (s *trialWorkloadSequencer) PrecloseCheckpointWorkload() *searcher.Workload {
	if s.curStepDone.hasCheckpoint {
		return nil
	}
	// Because no workloads can be issued without a trialID, having no trialID indicates we cannot
	// have finished any workloads at all.
	if !s.trialIDValid {
		return nil
	}
	return &searcher.Workload{
		Kind:         searcher.CheckpointModel,
		ExperimentID: s.experiment.ID,
		TrialID:      s.trialID,
		StepID:       s.curStep,
	}
}

func (s *trialWorkloadSequencer) TerminateWorkload() *searcher.Workload {
	return &searcher.Workload{
		Kind:         searcher.Terminate,
		ExperimentID: s.experiment.ID,
		TrialID:      s.trialID,
		StepID:       s.curStep,
	}
}

func (s *trialWorkloadSequencer) RollBackSequencer() int {
	// If any steps have been run but not checkpointed, find the last checkpointed step
	// and return the trialWorkloadSequencer's state to the completion of that step.
	if !s.curStepDone.hasCheckpoint {
		for s.curStep--; !s.steps[s.curStep].hasCheckpoint; {
			s.curStep--
		}
		s.curStepDone = s.steps[s.curStep]
	}
	s.curWorkloadValid = false
	return s.curStep
}

func (s *trialWorkloadSequencer) UpToDate() bool {
	// If all operations for the last asked-for step are done, then the trial has no more workloads
	// to run at the moment.
	return s.curStep == len(s.steps)-1 && s.curStepDone == s.steps[s.curStep]
}

func (s *trialWorkloadSequencer) minValidationNeeded() bool {
	validationPeriod := s.minValidationPeriod
	if validationPeriod == nil {
		return false
	}
	stepsSinceValidation := 0
	for i := s.curStep; !s.steps[i].hasValidation; i-- {
		stepsSinceValidation++
	}
	return stepsSinceValidation >= *validationPeriod
}

func (s *trialWorkloadSequencer) minCheckpointNeeded() bool {
	checkpointPeriod := s.minCheckpointPeriod
	if checkpointPeriod == nil {
		return false
	}
	stepsSinceCheckpoint := 0
	for i := s.curStep; !s.steps[i].hasCheckpoint; i-- {
		stepsSinceCheckpoint++
	}
	return stepsSinceCheckpoint >= *checkpointPeriod
}
