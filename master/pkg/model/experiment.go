package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/protoutils"

	"github.com/jackc/pgtype"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/logv1"
)

// StateWithReason is the run state of an experiment with
// an informational reason used for logging purposes.
type StateWithReason struct {
	State               State
	InformationalReason string
}

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
	// ErrorState constant.
	ErrorState State = "ERROR"
	// PausedState constant.
	PausedState State = "PAUSED"
	// StoppingKilledState constant.
	StoppingKilledState State = "STOPPING_KILLED"
	// StoppingCanceledState constant.
	StoppingCanceledState State = "STOPPING_CANCELED"
	// StoppingCompletedState constant.
	StoppingCompletedState State = "STOPPING_COMPLETED"
	// StoppingErrorState constant.
	StoppingErrorState State = "STOPPING_ERROR"
	// DeletingState constant.
	DeletingState State = "DELETING"
	// DeleteFailedState constant.
	DeleteFailedState State = "DELETE_FAILED"
	// DeletedState constant.
	DeletedState State = "DELETED"
	// PartiallyDeletedState constant.
	PartiallyDeletedState State = "PARTIALLY_DELETED"

	// TrialWorkloadSequencerType constant.
	TrialWorkloadSequencerType WorkloadSequencerType = "TRIAL_WORKLOAD_SEQUENCER"

	// Legacy json path for validation metrics.
	legacyValidationMetricsPath = "validation_metrics"
	// Legacy json path for training metrics.
	legacyTrainingMetricsPath = "avg_metrics"
)

// StateFromProto maps experimentv1.State to State.
func StateFromProto(state experimentv1.State) State {
	str := state.String()
	return State(strings.TrimPrefix(str, "STATE_"))
}

// StateToProto maps State to experimentv1.State.
func StateToProto(state State) experimentv1.State {
	stateEnum := experimentv1.State_value["STATE_"+string(state)]
	return experimentv1.State(stateEnum)
}

// States and transitions

// reverseTransitions computes the reverse transition table.
func reverseTransitions(
	transitions map[State]map[State]bool,
) map[State]map[State]bool {
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

// DeletingStates are the valid deleting states.
var DeletingStates = map[State]bool{
	DeletedState:      true,
	DeleteFailedState: true,
	DeletingState:     true,
}

// RunningStates are the valid running states.
var RunningStates = map[State]bool{
	ActiveState: true,
	PausedState: true,
}

// StoppingStates are the valid stopping states.
var StoppingStates = map[State]bool{
	StoppingCanceledState:  true,
	StoppingKilledState:    true,
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
	StoppingKilledState:   true,
}

// StoppingToTerminalStates maps from stopping states to the corresponding terminal states.
var StoppingToTerminalStates = map[State]State{
	StoppingKilledState:    CanceledState,
	StoppingCanceledState:  CanceledState,
	StoppingCompletedState: CompletedState,
	StoppingErrorState:     ErrorState,
}

// ExperimentTransitions maps experiment states to their possible transitions.
var ExperimentTransitions = map[State]map[State]bool{
	ActiveState: {
		PausedState:            true,
		StoppingKilledState:    true,
		StoppingCanceledState:  true,
		StoppingCompletedState: true,
		StoppingErrorState:     true,
	},
	PausedState: {
		ActiveState:            true,
		StoppingKilledState:    true,
		StoppingCanceledState:  true,
		StoppingCompletedState: true,
		StoppingErrorState:     true,
	},
	StoppingCanceledState: {
		CanceledState:       true,
		StoppingKilledState: true,
		StoppingErrorState:  true,
	},
	StoppingKilledState: {
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
	CanceledState: {
		DeletingState: true,
	},
	CompletedState: {
		DeletingState: true,
	},
	ErrorState: {
		DeletingState: true,
	},
	DeletingState: {
		DeletedState:      true,
		DeleteFailedState: true,
	},
	DeleteFailedState: {
		DeletingState: true,
	},
	DeletedState: {},
}

// StatesToStrings converts a State map to a list of strings for db queries.
func StatesToStrings(inStates map[State]bool) []string {
	states := make([]string, 0, len(inStates))
	for state := range inStates {
		states = append(states, string(state))
	}
	return states
}

// NonTerminalStates where an experiment can be canceled or killed.
var NonTerminalStates = func() []State {
	var states []State
	for s := range ExperimentTransitions {
		if !TerminalStates[s] && !DeletingStates[s] {
			states = append(states, s)
		}
	}
	return states
}()

// ExperimentReverseTransitions lists possible ancestor states.
var ExperimentReverseTransitions = reverseTransitions(ExperimentTransitions)

// TrialTransitions maps trial states to their possible transitions.
// Trials are mostly the same as experiments, but when immediate exits through
// ErrorState allowed since can die immediately and let the RM clean us up.
var TrialTransitions = map[State]map[State]bool{
	ActiveState: {
		PausedState:            true,
		StoppingKilledState:    true,
		StoppingCanceledState:  true,
		StoppingCompletedState: true,
		StoppingErrorState:     true,
		ErrorState:             true,
		// User-canceled trials go directly from ACTIVE to COMPLETED with no in between.
		CompletedState: true,
	},
	CanceledState:  {},
	CompletedState: {},
	ErrorState:     {},
	PausedState: {
		ActiveState:            true,
		StoppingKilledState:    true,
		StoppingCanceledState:  true,
		StoppingCompletedState: true,
		StoppingErrorState:     true,
		ErrorState:             true,
	},
	// The pattern of the transitory states here is that they
	// can always degrade into a more severe state, but never
	// the other way.
	StoppingCompletedState: {
		StoppingCanceledState: true,
		StoppingKilledState:   true,
		StoppingErrorState:    true,
		CompletedState:        true,
		ErrorState:            true,
	},
	StoppingCanceledState: {
		StoppingKilledState: true,
		StoppingErrorState:  true,
		CanceledState:       true,
		ErrorState:          true,
	},
	StoppingKilledState: {
		StoppingErrorState: true,
		CanceledState:      true,
		ErrorState:         true,
	},
	StoppingErrorState: {
		ActiveState: true,
		ErrorState:  true,
	},
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
	ID    int    `db:"id"`
	JobID JobID  `db:"job_id"`
	State State  `db:"state"`
	Notes string `db:"notes"`

	// Offer a LegacyConfig rather than ExperimentConfig since most of the system is about querying
	// experiments which ran some time in the past, which is exactly what LegacyConfig is for.
	Config         expconf.LegacyConfig `db:"config"`
	OriginalConfig string               `db:"original_config"`

	// The model definition is stored as a .tar.gz file (raw bytes).
	ModelDefinitionBytes []byte     `db:"model_definition" bun:"model_definition"`
	StartTime            time.Time  `db:"start_time"`
	EndTime              *time.Time `db:"end_time"`
	ParentID             *int       `db:"parent_id"`
	Archived             bool       `db:"archived"`
	GitRemote            *string    `db:"git_remote"`
	GitCommit            *string    `db:"git_commit"`
	GitCommitter         *string    `db:"git_committer"`
	GitCommitDate        *time.Time `db:"git_commit_date"`
	OwnerID              *UserID    `db:"owner_id"`
	Username             string     `db:"username"`
	ProjectID            int        `db:"project_id"`
	Unmanaged            bool       `db:"unmanaged"`
}

// ExperimentFromProto converts a experimentv1.Experiment to a model.Experiment.
func ExperimentFromProto(e *experimentv1.Experiment) (*Experiment, error) {
	var uid *UserID
	if e.UserId != 0 {
		uid = ptrs.Ptr(UserID(e.UserId))
	}
	var endTime *time.Time
	if e.EndTime != nil {
		endTime = ptrs.Ptr(e.EndTime.AsTime())
	}
	var parentID *int
	if e.ForkedFrom != nil {
		parentID = ptrs.Ptr(int(e.ForkedFrom.Value))
	}

	byts, err := json.Marshal(e.Config)
	if err != nil {
		return nil, err
	}

	config, err := expconf.ParseLegacyConfigJSON(byts)
	if err != nil {
		return nil, err
	}

	return &Experiment{
		ID:             int(e.Id),
		JobID:          JobID(e.JobId),
		State:          StateFromProto(e.State),
		Notes:          e.Notes,
		Config:         config,
		OriginalConfig: e.OriginalConfig,
		// ModelDefinitionBytes:
		StartTime: e.StartTime.AsTime(),
		EndTime:   endTime,
		ParentID:  parentID,
		Archived:  e.Archived,
		// GitRemote:
		// GitCommitDate
		OwnerID:   uid,
		Username:  e.Username,
		ProjectID: int(e.ProjectId),
		Unmanaged: e.Unmanaged,
	}, nil
}

// NewExperiment creates a new experiment struct in the paused state.  Note
// that the experiment ID will not be set.
func NewExperiment(
	config expconf.ExperimentConfig,
	originalConfig string,
	modelDefinitionBytes []byte,
	parentID *int,
	archived bool,
	gitRemote, gitCommit, gitCommitter *string,
	gitCommitDate *time.Time,
	projectID int,
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
		JobID:                NewJobID(),
		Config:               config.AsLegacy(),
		OriginalConfig:       originalConfig,
		ModelDefinitionBytes: modelDefinitionBytes,
		StartTime:            time.Now().UTC(),
		ParentID:             parentID,
		Archived:             archived,
		GitRemote:            gitRemote,
		GitCommit:            gitCommit,
		GitCommitter:         gitCommitter,
		GitCommitDate:        gitCommitDate,
		ProjectID:            projectID,
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
	bun.BaseModel `bun:"table:trials"`

	ID                    int            `db:"id" bun:",pk,autoincrement"`
	RequestID             *RequestID     `db:"request_id"`
	ExperimentID          int            `db:"experiment_id"`
	State                 State          `db:"state"`
	StartTime             time.Time      `db:"start_time"`
	EndTime               *time.Time     `db:"end_time"`
	HParams               map[string]any `db:"hparams" bun:"hparams"`
	WarmStartCheckpointID *int           `db:"warm_start_checkpoint_id"`
	Seed                  int64          `db:"seed"`
	TotalBatches          int            `db:"total_batches"`
}

// TrialTaskID represents a row from the `trial_id_task_id` table.
type TrialTaskID struct {
	bun.BaseModel `bun:"table:trial_id_task_id"`

	TrialID int
	TaskID  TaskID
}

// NewTrial creates a new trial in the specified state.  Note that the trial ID
// will not be set.
func NewTrial(
	state State,
	requestID RequestID,
	experimentID int,
	hparams JSONObj,
	warmStartCheckpoint *Checkpoint,
	trialSeed int64,
) *Trial {
	var warmStartCheckpointID *int
	if warmStartCheckpoint != nil {
		warmStartCheckpointID = &warmStartCheckpoint.ID
	}
	return &Trial{
		RequestID:             &requestID,
		ExperimentID:          experimentID,
		State:                 state,
		StartTime:             time.Now().UTC(),
		HParams:               hparams,
		WarmStartCheckpointID: warmStartCheckpointID,
		Seed:                  trialSeed,
	}
}

// TrialMetrics represents a row from the `steps` or `validations` table.
type TrialMetrics struct {
	ID           int        `db:"id" json:"id"`
	TrialID      int        `db:"trial_id" json:"trial_id"`
	TrialRunID   int        `db:"trial_run_id" json:"-"`
	TotalBatches int        `db:"total_batches" json:"total_batches"`
	EndTime      *time.Time `db:"end_time" json:"end_time"`
	Metrics      JSONObj    `db:"metrics" json:"metrics"`
}

// TrialMetricsJSONPath returns the legacy JSON path to the metrics field in the metrics table.
func TrialMetricsJSONPath(isValidation bool) string {
	if isValidation {
		return legacyValidationMetricsPath
	}
	return legacyTrainingMetricsPath
}

// TrialSummaryMetricsJSONPath returns the JSON path to the trials metric summary.
func TrialSummaryMetricsJSONPath(metricGroup MetricGroup) string {
	switch metricGroup {
	case ValidationMetricGroup:
		return legacyValidationMetricsPath
	case TrainingMetricGroup:
		return legacyTrainingMetricsPath
	default:
		return metricGroup.ToString()
	}
}

// TrialSummaryMetricGroup returns the metric group for the given summary JSON path.
func TrialSummaryMetricGroup(jsonPath string) MetricGroup {
	var mGroup MetricGroup
	switch jsonPath {
	case TrialSummaryMetricsJSONPath(TrainingMetricGroup):
		mGroup = TrainingMetricGroup
	case TrialSummaryMetricsJSONPath(ValidationMetricGroup):
		mGroup = ValidationMetricGroup
	default:
		mGroup = MetricGroup(jsonPath)
	}
	return mGroup
}

// Represent order of active states (Queued -> Pulling -> Starting -> Running).
var experimentStateIndex = map[experimentv1.State]int{
	experimentv1.State_STATE_UNSPECIFIED:        0,
	experimentv1.State_STATE_ACTIVE:             1,
	experimentv1.State_STATE_QUEUED:             2,
	experimentv1.State_STATE_PULLING:            3,
	experimentv1.State_STATE_STARTING:           4,
	experimentv1.State_STATE_RUNNING:            5,
	experimentv1.State_STATE_PAUSED:             6,
	experimentv1.State_STATE_COMPLETED:          7,
	experimentv1.State_STATE_CANCELED:           8,
	experimentv1.State_STATE_ERROR:              9,
	experimentv1.State_STATE_STOPPING_COMPLETED: 10,
	experimentv1.State_STATE_STOPPING_KILLED:    12,
	experimentv1.State_STATE_STOPPING_CANCELED:  13,
	experimentv1.State_STATE_STOPPING_ERROR:     14,
	experimentv1.State_STATE_DELETED:            15,
	experimentv1.State_STATE_DELETING:           16,
	experimentv1.State_STATE_DELETE_FAILED:      17,
}

// MostProgressedExperimentState returns the more advanced active state
// based on experimentStateIndex (Queued -> Pulling -> Starting -> Running).
func MostProgressedExperimentState(
	state1 experimentv1.State, state2 experimentv1.State,
) experimentv1.State {
	stateValue1 := experimentStateIndex[state1]
	stateValue2 := experimentStateIndex[state2]
	if stateValue1 > stateValue2 {
		return state1
	}
	return state2
}

const (
	// StepsCompletedMetadataKey is the key within metadata to find steps completed now, if it exists.
	StepsCompletedMetadataKey = "steps_completed"
)

// CheckpointV2 represents a row from the `checkpoints_v2` table.
type CheckpointV2 struct {
	bun.BaseModel `bun:"table:checkpoints_v2"`
	ID            int                    `db:"id" bun:"id,pk,autoincrement"`
	UUID          uuid.UUID              `db:"uuid"`
	TaskID        TaskID                 `db:"task_id"`
	AllocationID  *AllocationID          `db:"allocation_id"`
	ReportTime    time.Time              `db:"report_time"`
	State         State                  `db:"state"`
	Resources     map[string]int64       `db:"resources"`
	Metadata      map[string]interface{} `db:"metadata"`
	Size          int64                  `db:"size"`
}

// CheckpointTrainingMetadata is a substruct of checkpoints encapsulating training specific
// information.
type CheckpointTrainingMetadata struct {
	TrialID           int      `db:"trial_id"`
	ExperimentID      int      `db:"experiment_id"`
	ExperimentConfig  JSONObj  `db:"experiment_config"`
	HParams           JSONObj  `db:"hparams"`
	TrainingMetrics   JSONObj  `db:"training_metrics"`
	ValidationMetrics JSONObj  `db:"validation_metrics"`
	SearcherMetric    *float64 `db:"searcher_metric"`
	StepsCompleted    int      `db:"steps_completed"`
}

// Checkpoint represents a row from the `checkpoints_view` view.
type Checkpoint struct {
	ID int `db:"id"`

	UUID         *uuid.UUID    `db:"uuid"`
	TaskID       *TaskID       `db:"task_id"`
	AllocationID *AllocationID `db:"allocation_id"`
	ReportTime   time.Time     `db:"report_time"`
	State        State         `db:"state"`
	Resources    JSONObj       `db:"resources"`
	Metadata     JSONObj       `db:"metadata"`
	Size         int64         `db:"size"`

	CheckpointTrainingMetadata
}

// TrialLog represents a row from the `trial_logs` table.
type TrialLog struct {
	// A trial log should have one of these IDs. All should be unique.
	// TODO(Brad): This must be int64.
	ID *int `db:"id" json:"id,omitempty"`
	// The body of an Elasticsearch log response will look something like
	// { _id: ..., _source: { ... }} where _source is the rest of this struct.
	// StringID doesn't have serialization tags because it is not part of
	// _source and populated from _id.
	StringID *string `json:"-"`

	TrialID int    `db:"trial_id" json:"trial_id"`
	Message string `db:"message" json:"message,omitempty"`

	AgentID *string `db:"agent_id" json:"agent_id,omitempty"`
	// In the case of k8s, container_id is a pod name instead.
	ContainerID *string    `db:"container_id" json:"container_id,omitempty"`
	RankID      *int       `db:"rank_id" json:"rank_id,omitempty"`
	Timestamp   *time.Time `db:"timestamp" json:"timestamp"`
	Level       *string    `db:"level" json:"level"`
	Log         *string    `db:"log" json:"log"`
	Source      *string    `db:"source" json:"source,omitempty"`
	StdType     *string    `db:"stdtype" json:"stdtype,omitempty"`
}

// Proto converts a trial log to its protobuf representation.
func (t TrialLog) Proto() (*apiv1.TrialLogsResponse, error) {
	resp := &apiv1.TrialLogsResponse{
		TrialId:     int32(t.TrialID),
		Message:     t.Message,
		AgentId:     t.AgentID,
		ContainerId: t.ContainerID,
		Log:         t.Log,
		Source:      t.Source,
		Stdtype:     t.StdType,
	}

	switch {
	case t.ID != nil:
		resp.Id = strconv.Itoa(*t.ID)
	case t.StringID != nil:
		resp.Id = *t.StringID
	default:
		panic("log had no valid ID")
	}

	if t.Timestamp != nil {
		resp.Timestamp = timestamppb.New(*t.Timestamp)
	}

	if t.Level == nil {
		resp.Level = logv1.LogLevel_LOG_LEVEL_UNSPECIFIED
	} else {
		switch *t.Level {
		case LogLevelTrace:
			resp.Level = logv1.LogLevel_LOG_LEVEL_TRACE
		case LogLevelDebug:
			resp.Level = logv1.LogLevel_LOG_LEVEL_DEBUG
		case LogLevelInfo:
			resp.Level = logv1.LogLevel_LOG_LEVEL_INFO
		case LogLevelWarning:
			resp.Level = logv1.LogLevel_LOG_LEVEL_WARNING
		case LogLevelError:
			resp.Level = logv1.LogLevel_LOG_LEVEL_ERROR
		case LogLevelCritical:
			resp.Level = logv1.LogLevel_LOG_LEVEL_CRITICAL
		default:
			resp.Level = logv1.LogLevel_LOG_LEVEL_UNSPECIFIED
		}
	}

	if t.RankID != nil {
		id := int32(*t.RankID)
		resp.RankId = &id
	}

	return resp, nil
}

// Resolve resolves the legacy Message field from the others provided.
func (t *TrialLog) Resolve() {
	if t.Message != "" {
		return
	}

	var timestamp string
	if t.Timestamp != nil {
		timestamp = t.Timestamp.Format(time.RFC3339Nano)
	} else {
		timestamp = defaultTaskLogTime
	}

	// This is just to match postgres.
	const containerIDMaxLength = 8
	var containerID string
	if t.ContainerID != nil {
		containerID = *t.ContainerID
		if len(containerID) > containerIDMaxLength {
			containerID = containerID[:containerIDMaxLength]
		}
	} else {
		containerID = defaultTaskLogContainer
	}

	var rankID string
	if t.RankID != nil {
		rankID = fmt.Sprintf("[rank=%d] ", *t.RankID)
	}

	var level string
	if t.Level != nil {
		level = fmt.Sprintf("%s: ", *t.Level)
	}

	t.Message = fmt.Sprintf("[%s] [%s] %s|| %s %s",
		timestamp, containerID, rankID, level, *t.Log)
}

// TrialProfilerMetricsBatch represents a row from the `trial_profiler_metrics` table.
type TrialProfilerMetricsBatch struct {
	Values     pgtype.Float4Array      `db:"values"`
	Batches    pgtype.Int4Array        `db:"batches"`
	Timestamps pgtype.TimestamptzArray `db:"timestamps"`
	Labels     []byte                  `db:"labels"`
}

// ToProto converts a TrialProfilerMetricsBatch to its protobuf representation.
func (t *TrialProfilerMetricsBatch) ToProto() (*trialv1.TrialProfilerMetricsBatch, error) {
	var pBatch trialv1.TrialProfilerMetricsBatch
	var err error

	var pLabels trialv1.TrialProfilerMetricLabels
	if err = protojson.Unmarshal(t.Labels, &pLabels); err != nil {
		return nil, errors.Wrap(err, "unmarshaling labels")
	}
	pBatch.Labels = &pLabels

	var protoValues []float32
	if err = t.Values.AssignTo(&protoValues); err != nil {
		return nil, errors.Wrap(err, "setting values")
	}
	pBatch.Values = protoValues

	var protoBatches []int32
	if err = t.Batches.AssignTo(&protoBatches); err != nil {
		return nil, errors.Wrap(err, "setting batches")
	}
	pBatch.Batches = protoBatches

	var protoTimes []time.Time
	if err = t.Timestamps.AssignTo(&protoTimes); err != nil {
		return nil, errors.Wrap(err, "setting timestamps")
	}
	if pBatch.Timestamps, err = protoutils.TimeProtoSliceFromTimes(protoTimes); err != nil {
		return nil, errors.Wrap(err, "parsing times to proto")
	}

	return &pBatch, nil
}

// TrialLogBatch represents a batch of model.TrialLog.
type TrialLogBatch []*TrialLog

// Size implements logs.Batch.
func (t TrialLogBatch) Size() int {
	return len(t)
}

// ForEach implements logs.Batch.
func (t TrialLogBatch) ForEach(f func(interface{}) error) error {
	for _, tl := range t {
		if err := f(tl); err != nil {
			return err
		}
	}
	return nil
}

// TrialProfilerMetricsBatchBatch represents a batch of trialv1.TrialProfilerMetricsBatch.
type TrialProfilerMetricsBatchBatch []*trialv1.TrialProfilerMetricsBatch

// Size implements logs.Batch.
func (t TrialProfilerMetricsBatchBatch) Size() int {
	return len(t)
}

// ForEach implements logs.Batch.
func (t TrialProfilerMetricsBatchBatch) ForEach(f func(interface{}) error) error {
	for _, tl := range t {
		if err := f(tl); err != nil {
			return err
		}
	}
	return nil
}

// ExitedReason defines why a workload exited early.
type ExitedReason string

const (
	// Errored signals the searcher that the workload errored out.
	Errored ExitedReason = "ERRORED"
	// UserRequestedStop signals the searcher that the user requested a cancelation, from code.
	UserRequestedStop ExitedReason = "USER_REQUESTED_STOP"
	// UserCanceled signals the searcher that the user requested a cancelation, from the CLI or UI.
	UserCanceled ExitedReason = "USER_CANCELED"
	// InvalidHP signals the searcher that the user raised an InvalidHP exception.
	InvalidHP ExitedReason = "INVALID_HP"
	// InitInvalidHP signals the searcher that the user raised an InvalidHP exception
	// in the trial init.
	InitInvalidHP ExitedReason = "INIT_INVALID_HP"
)

// ExitedReasonFromProto returns an ExitedReason from its protobuf representation.
func ExitedReasonFromProto(r trialv1.TrialEarlyExit_ExitedReason) ExitedReason {
	switch r {
	case trialv1.TrialEarlyExit_EXITED_REASON_UNSPECIFIED:
		return Errored
	case trialv1.TrialEarlyExit_EXITED_REASON_INVALID_HP:
		return InvalidHP
	case trialv1.TrialEarlyExit_EXITED_REASON_INIT_INVALID_HP:
		return InitInvalidHP
	default:
		panic(fmt.Errorf("unexpected exited reason: %v", r))
	}
}

// ToSearcherProto converts an ExitedReason to its protobuf representation for searcher purposes.
func (r ExitedReason) ToSearcherProto() experimentv1.TrialExitedEarly_ExitedReason {
	switch r {
	case Errored:
		return experimentv1.TrialExitedEarly_EXITED_REASON_UNSPECIFIED
	case InvalidHP:
		return experimentv1.TrialExitedEarly_EXITED_REASON_INVALID_HP
	case UserRequestedStop:
		return experimentv1.TrialExitedEarly_EXITED_REASON_USER_REQUESTED_STOP
	case UserCanceled:
		return experimentv1.TrialExitedEarly_EXITED_REASON_USER_CANCELED
	default:
		panic(fmt.Errorf("unexpected exited reason: %v", r))
	}
}
