package model

import (
	"time"

	"github.com/determined-ai/determined/master/internal/logs"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/version"
)

// State is the run state of an experiment / trial / step / etc.
type State string

// WorkloadSequencerType is the type of sequencer that a trial actor should use.
type WorkloadSequencerType string

// WorkloadManagerType indicates which type of workloads the harness should prepare to receive.
type WorkloadManagerType string

// Constants.

const (
	// ActiveState constant.
	ActiveState State = "ACTIVE"
	// CanceledState constant.
	CanceledState State = "CANCELED"
	// CompletedState constant.
	CompletedState State = "COMPLETED"
	// DeletedState constant.
	DeletedState State = "DELETED"
	// ErrorState constant.
	ErrorState State = "ERROR"
	// PausedState constant.
	PausedState State = "PAUSED"
	// StoppingCanceledState constant.
	StoppingCanceledState State = "STOPPING_CANCELED"
	// StoppingCompletedState constant.
	StoppingCompletedState State = "STOPPING_COMPLETED"
	// StoppingErrorState constant.
	StoppingErrorState State = "STOPPING_ERROR"

	// TrialWorkloadSequencerType constant.
	TrialWorkloadSequencerType WorkloadSequencerType = "TRIAL_WORKLOAD_SEQUENCER"

	// TrialWorkloadManagerType handles model training loads.
	TrialWorkloadManagerType WorkloadManagerType = "TRIAL_WORKLOAD_MANAGER"
)

// States and transitions

// reverseTransitions computes the reverse transition table.
func reverseTransitions(
	transitions map[State]map[State]bool) map[State]map[State]bool {
	ret := make(map[State]map[State]bool)
	for state := range transitions {
		ret[state] = make(map[State]bool)
	}
	for start, ends := range transitions {
		for end := range ends {
			ret[end][start] = true
		}
	}
	return ret
}

// RunningStates are the valid running states.
var RunningStates = map[State]bool{
	ActiveState: true,
	PausedState: true,
}

// StoppingStates are the valid stopping states.
var StoppingStates = map[State]bool{
	StoppingCanceledState:  true,
	StoppingCompletedState: true,
	StoppingErrorState:     true,
}

// TerminalStates are the valid terminal states.
var TerminalStates = map[State]bool{
	CanceledState:  true,
	CompletedState: true,
	ErrorState:     true,
}

// ManualStates are the states the user can set an experiment to.
var ManualStates = map[State]bool{
	ActiveState:           true,
	PausedState:           true,
	StoppingCanceledState: true,
}

// StoppingToTerminalStates maps from stopping states to the corresponding terminal states.
var StoppingToTerminalStates = map[State]State{
	StoppingCanceledState:  CanceledState,
	StoppingCompletedState: CompletedState,
	StoppingErrorState:     ErrorState,
}

// ExperimentTransitions maps experiment states to their possible transitions.
var ExperimentTransitions = map[State]map[State]bool{
	ActiveState: {
		PausedState:            true,
		StoppingCanceledState:  true,
		StoppingCompletedState: true,
		StoppingErrorState:     true,
	},
	CanceledState:  {},
	CompletedState: {},
	ErrorState:     {},
	PausedState: {
		ActiveState:            true,
		StoppingCanceledState:  true,
		StoppingCompletedState: true,
		StoppingErrorState:     true,
	},
	StoppingCanceledState: {
		CanceledState:      true,
		StoppingErrorState: true,
	},
	StoppingCompletedState: {
		CompletedState:     true,
		StoppingErrorState: true,
	},
	StoppingErrorState: {
		ActiveState: true,
		ErrorState:  true,
	},
}

// ExperimentReverseTransitions lists possible ancestor states.
var ExperimentReverseTransitions = reverseTransitions(ExperimentTransitions)

// TrialTransitions maps trial states to their possible transitions.
var TrialTransitions = map[State]map[State]bool{
	ActiveState: {
		CanceledState:  true,
		CompletedState: true,
		ErrorState:     true,
	},
	CanceledState:  {},
	CompletedState: {},
	ErrorState:     {},
}

// TrialReverseTransitions list possible ancestor states.
var TrialReverseTransitions = reverseTransitions(TrialTransitions)

// StepTransitions maps step and validation states to their possible transitions.
var StepTransitions = map[State]map[State]bool{
	ActiveState: {
		CompletedState: true,
		ErrorState:     true,
	},
	CompletedState: {},
	ErrorState:     {},
}

// StepReverseTransitions list possible ancestor states.
var StepReverseTransitions = reverseTransitions(StepTransitions)

// CheckpointTransitions maps checkpoint states to their possible transitions.
var CheckpointTransitions = map[State]map[State]bool{
	ActiveState: {
		CompletedState: true,
		ErrorState:     true,
	},
	CompletedState: {
		DeletedState: true,
	},
	DeletedState: {},
	ErrorState:   {},
}

// CheckpointReverseTransitions list possible ancestor states.
var CheckpointReverseTransitions = reverseTransitions(CheckpointTransitions)

// Database row types.

// Experiment represents a row from the `experiments` table.
type Experiment struct {
	ID     int              `db:"id"`
	State  State            `db:"state"`
	Config ExperimentConfig `db:"config"`
	// The model definition is stored as a .tar.gz file (raw bytes).
	ModelDefinitionBytes []byte     `db:"model_definition"`
	StartTime            time.Time  `db:"start_time"`
	EndTime              *time.Time `db:"end_time"`
	ParentID             *int       `db:"parent_id"`
	Archived             bool       `db:"archived"`
	GitRemote            *string    `db:"git_remote"`
	GitCommit            *string    `db:"git_commit"`
	GitCommitter         *string    `db:"git_committer"`
	GitCommitDate        *time.Time `db:"git_commit_date"`
	OwnerID              *UserID    `db:"owner_id"`
}

// ExperimentDescriptor is a minimal description of an experiment.
type ExperimentDescriptor struct {
	ID       int              `json:"id"`
	Archived bool             `json:"archived"`
	Config   ExperimentConfig `json:"config"`
	Labels   []string         `json:"labels"`
}

// NewExperiment creates a new experiment struct in the paused state.  Note
// that the experiment ID will not be set.
func NewExperiment(
	config ExperimentConfig,
	modelDefinitionBytes []byte,
	parentID *int,
	archived bool,
	gitRemote, gitCommit, gitCommitter *string,
	gitCommitDate *time.Time,
) (*Experiment, error) {
	if len(modelDefinitionBytes) == 0 {
		return nil, errors.New("empty model definition")
	}
	if !(gitRemote == nil && gitCommit == nil && gitCommitter == nil && gitCommitDate == nil) &&
		!(gitRemote != nil && gitCommit != nil && gitCommitter != nil && gitCommitDate != nil) {
		return nil, errors.New(
			"all of git_remote, git_commit, git_committer and git_commit_date must be nil or non-nil")
	}
	return &Experiment{
		State:                PausedState,
		Config:               config,
		ModelDefinitionBytes: modelDefinitionBytes,
		StartTime:            time.Now().UTC(),
		ParentID:             parentID,
		Archived:             archived,
		GitRemote:            gitRemote,
		GitCommit:            gitCommit,
		GitCommitter:         gitCommitter,
		GitCommitDate:        gitCommitDate,
	}, nil
}

// Transition changes the state of the experiment to the new state. If the state was not modified
// the first return value returns false. If the state transition is illegal, an error is returned.
func (e *Experiment) Transition(state State) (bool, error) {
	if e.State == state {
		return false, nil
	}
	if !ExperimentTransitions[e.State][state] {
		return false, errors.Errorf("illegal transition %v -> %v for experiment %v",
			e.State, state, e.ID)
	}
	e.State = state
	if TerminalStates[state] {
		now := time.Now().UTC()
		e.EndTime = &now
	}
	return true, nil
}

// Trial represents a row from the `trials` table.
type Trial struct {
	ID                    int        `db:"id"`
	ExperimentID          int        `db:"experiment_id"`
	State                 State      `db:"state"`
	StartTime             time.Time  `db:"start_time"`
	EndTime               *time.Time `db:"end_time"`
	HParams               JSONObj    `db:"hparams"`
	WarmStartCheckpointID *int       `db:"warm_start_checkpoint_id"`
	Seed                  int64      `db:"seed"`
}

// NewTrial creates a new trial in the active state.  Note that the trial ID
// will not be set.
func NewTrial(
	experimentID int,
	hparams JSONObj,
	warmStartCheckpointID *int,
	trialSeed int64) *Trial {
	return &Trial{
		ExperimentID:          experimentID,
		State:                 ActiveState,
		StartTime:             time.Now().UTC(),
		HParams:               hparams,
		WarmStartCheckpointID: warmStartCheckpointID,
		Seed:                  trialSeed,
	}
}

// Step represents a row from the `steps` table.
type Step struct {
	TrialID               int        `db:"trial_id"`
	ID                    int        `db:"id"`
	State                 State      `db:"state"`
	StartTime             time.Time  `db:"start_time"`
	EndTime               *time.Time `db:"end_time"`
	NumBatches            int        `db:"num_batches"`
	PriorBatchesProcessed int        `db:"prior_batches_processed"`
	Metrics               JSONObj    `db:"metrics"`
}

// NewStep creates a new step in the active state.
func NewStep(trialID, stepID, numBatches, priorBatchesProcessed int) *Step {
	return &Step{
		TrialID:               trialID,
		ID:                    stepID,
		State:                 ActiveState,
		StartTime:             time.Now().UTC(),
		NumBatches:            numBatches,
		PriorBatchesProcessed: priorBatchesProcessed,
	}
}

// NewNoOpStep creates a new step in the completed state.
func NewNoOpStep(trialID, stepID int) *Step {
	now := time.Now().UTC()
	return &Step{
		TrialID:               trialID,
		ID:                    stepID,
		State:                 CompletedState,
		StartTime:             now,
		EndTime:               &now,
		NumBatches:            0,
		PriorBatchesProcessed: 0,
	}
}

// IsNew checks whether this step describes a new, in-progress step.
func (s *Step) IsNew() bool {
	return s.State == ActiveState && s.EndTime == nil && len(s.Metrics) == 0
}

// Validation represents a row from the `validations` table.
type Validation struct {
	ID        int        `db:"id" json:"id"`
	TrialID   int        `db:"trial_id" json:"trial_id"`
	StepID    int        `db:"step_id" json:"step_id"`
	State     State      `db:"state" json:"state"`
	StartTime time.Time  `db:"start_time" json:"start_time"`
	EndTime   *time.Time `db:"end_time" json:"end_time"`
	Metrics   JSONObj    `db:"metrics" json:"metrics"`
}

// NewValidation creates a new validation in the active state.
func NewValidation(trialID, stepID int) *Validation {
	return &Validation{
		TrialID:   trialID,
		StepID:    stepID,
		State:     ActiveState,
		StartTime: time.Now().UTC(),
	}
}

// IsNew checks whether this validation describes a new, in-progress validation operation.
func (v *Validation) IsNew() bool {
	return v.State == ActiveState && v.ID == 0 && v.EndTime == nil && len(v.Metrics) == 0
}

// Checkpoint represents a row from the `checkpoints` table.
type Checkpoint struct {
	ID                int        `db:"id" json:"id"`
	TrialID           int        `db:"trial_id" json:"trial_id"`
	StepID            int        `db:"step_id" json:"step_id"`
	State             State      `db:"state" json:"state"`
	StartTime         time.Time  `db:"start_time" json:"start_time"`
	EndTime           *time.Time `db:"end_time" json:"end_time"`
	UUID              *string    `db:"uuid" json:"uuid"`
	Resources         JSONObj    `db:"resources" json:"resources"`
	Metadata          JSONObj    `db:"metadata" json:"metadata"`
	Framework         string     `db:"framework" json:"framework"`
	Format            string     `db:"format" json:"format"`
	DeterminedVersion string     `db:"determined_version" json:"determined_version"`
}

// NewCheckpoint creates a new checkpoint in the active state.
func NewCheckpoint(trialID, stepID int) *Checkpoint {
	return &Checkpoint{
		TrialID:           trialID,
		StepID:            stepID,
		State:             ActiveState,
		StartTime:         time.Now().UTC(),
		Metadata:          JSONObj{},
		DeterminedVersion: version.Version,
	}
}

// IsNew checks whether this checkpoint describes a new, in-progress checkpoint operation.
func (c *Checkpoint) IsNew() bool {
	return c.State == ActiveState && c.ID == 0 && c.EndTime == nil &&
		c.UUID == nil && len(c.Resources) == 0 && len(c.Metadata) == 0
}

// TrialLog represents a row from the `trial_logs` table.
type TrialLog struct {
	ID      int    `db:"id"`
	TrialID int    `db:"trial_id"`
	Message string `db:"message"`

	AgentID *string `db:"agent_id"`
	// In the case of k8s, container_id is a pod name instead.
	ContainerID *string    `db:"container_id"`
	RankID      *int       `db:"rank_id"`
	Timestamp   *time.Time `db:"timestamp"`
	Level       *string    `db:"level"`
	StdType     *int       `db:"std_type"`
	Source      *string    `db:"source"`
}

// TrialLogBatch represents a batch of model.TrialLog.
type TrialLogBatch []*TrialLog

// Size implements logs.Batch.
func (t TrialLogBatch) Size() int {
	return len(t)
}

// ForEach implements logs.Batch.
func (t TrialLogBatch) ForEach(f func(logs.Record) error) error {
	for _, tl := range t {
		err := f(tl)
		if err != nil {
			return err
		}
	}
	return nil
}

// SearcherEvent represents a row from the `searcher_events` table.
type SearcherEvent struct {
	ID           int     `db:"id"`
	ExperimentID int     `db:"experiment_id"`
	EventType    string  `db:"event_type"`
	Content      JSONObj `db:"content"`
}
