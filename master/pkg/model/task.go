package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/tasklog"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/logv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

// TaskID is the unique ID of a task among all tasks.
type TaskID string

// NewTaskID returns a random, globally unique task ID.
func NewTaskID() TaskID {
	return TaskID(uuid.New().String())
}

// TaskType is the type of a task.
type TaskType string

func (a TaskID) String() string {
	return string(a)
}

const (
	// TaskTypeTrial is the "TRIAL" job type for the enum public.job_type in Postgres.
	TaskTypeTrial TaskType = "TRIAL"
	// TaskTypeNotebook is the "NOTEBOOK" job type for the enum public.job_type in Postgres.
	TaskTypeNotebook TaskType = "NOTEBOOK"
	// TaskTypeShell is the "SHELL" job type for the enum public.job_type in Postgres.
	TaskTypeShell TaskType = "SHELL"
	// TaskTypeCommand is the "COMMAND" job type for the enum public.job_type in Postgres.
	TaskTypeCommand TaskType = "COMMAND"
	// TaskTypeTensorboard is the "TENSORBOARD" task type for the enum.task_type in Postgres.
	TaskTypeTensorboard TaskType = "TENSORBOARD"
	// TaskTypeCheckpointGC is the "CHECKPOINT_GC" job type for the enum public.job_type in Postgres.
	TaskTypeCheckpointGC TaskType = "CHECKPOINT_GC"
)

// TaskLogVersion is the version for our log-storing scheme. Useful because changing designs
// would involve either a really costly migration or versioning schemes and we pick the latter.
type TaskLogVersion int32

// CurrentTaskLogVersion describes the current scheme in which we store task
// logs. To avoid a migration that in some cases would be extremely
// costly, we record the log version so that we can just read old logs
// the old way and do the new however we please.
const (
	TaskLogVersion0       TaskLogVersion = 0
	TaskLogVersion1       TaskLogVersion = 1
	CurrentTaskLogVersion                = TaskLogVersion1
)

// Task is the model for a task in the database.
type Task struct {
	bun.BaseModel `bun:"table:tasks"`

	TaskID    TaskID     `db:"task_id" bun:"task_id,pk"`
	JobID     *JobID     `db:"job_id"`
	TaskType  TaskType   `db:"task_type"`
	StartTime time.Time  `db:"start_time"`
	EndTime   *time.Time `db:"end_time"`
	// LogVersion indicates how the logs were stored.
	LogVersion TaskLogVersion `db:"log_version"`

	// Relations.
	Job *Job `bun:"rel:belongs-to,join:job_id=job_id"`
}

// AllocationID is the ID of an allocation of a task. It is usually of the form
// TaskID.allocation_number, maybe with some other metadata if different types of
// allocations run.
type AllocationID string

// NewAllocationID casts string ptr to AllocationID ptr.
func NewAllocationID(in *string) *AllocationID {
	if in == nil {
		return nil
	}
	return ptrs.Ptr(AllocationID(*in))
}

func (a AllocationID) String() string {
	return string(a)
}

// ToTaskID converts an AllocationID to its taskID.
func (a AllocationID) ToTaskID() TaskID {
	return TaskID(a[:strings.LastIndex(string(a), ".")])
}

// Allocation is the model for an allocation in the database.
type Allocation struct {
	bun.BaseModel `bun:"table:allocations"`

	AllocationID AllocationID     `db:"allocation_id" bun:"allocation_id,pk"`
	TaskID       TaskID           `db:"task_id" bun:"task_id,notnull"`
	Slots        int              `db:"slots" bun:"slots,notnull"`
	ResourcePool string           `db:"resource_pool" bun:"resource_pool,notnull"`
	StartTime    *time.Time       `db:"start_time" bun:"start_time"`
	EndTime      *time.Time       `db:"end_time" bun:"end_time"`
	State        *AllocationState `db:"state" bun:"state"`
	IsReady      *bool            `db:"is_ready" bun:"is_ready"`
	Ports        map[string]int   `db:"ports" bun:"ports,notnull"`
	// ProxyAddress stores the explicitly provided task-provided proxy address for resource
	// managers that do not supply us with it. Comes from `determined.exec.prep_container --proxy`.
	ProxyAddress *string `db:"proxy_address" bun:"proxy_address"`
}

// AcceleratorData is the model for an allocation accelerator data in the database.
type AcceleratorData struct {
	bun.BaseModel `bun:"table:allocation_accelerators"`

	ContainerID      string       `db:"container_id" bun:"container_id,pk"`
	AllocationID     AllocationID `db:"allocation_id" bun:"allocation_id,notnull"`
	NodeName         string       `db:"node_name" bun:"node_name,notnull"`
	AcceleratorType  string       `db:"accelerator_type" bun:"accelerator_type,notnull"`
	AcceleratorUuids []string     `db:"accelerator_uuids" bun:"accelerator_uuids,array"`
}

// AllocationState represents the current state of the task. Value indicates a partial ordering.
type AllocationState string

// TaskStats is the model for task stats in the database.
type TaskStats struct {
	AllocationID AllocationID
	EventType    string
	StartTime    *time.Time
	EndTime      *time.Time
}

// ResourceAggregates is the model for resource_aggregates in the database.
type ResourceAggregates struct {
	Date            *time.Time
	AggregationType string
	AggregationKey  string
	Seconds         float32
}

const (
	// AllocationStatePending state denotes that the command is awaiting allocation.
	AllocationStatePending AllocationState = "PENDING"
	// AllocationStateWaiting state denotes that the command is waiting on data.
	AllocationStateWaiting AllocationState = "WAITING"
	// AllocationStateAssigned state denotes that the command has been assigned to an agent but has
	// not started yet.
	AllocationStateAssigned AllocationState = "ASSIGNED"
	// AllocationStatePulling state denotes that the command's base image is being pulled from the
	// Docker registry.
	AllocationStatePulling AllocationState = "PULLING"
	// AllocationStateStarting state denotes that the image has been pulled and the task is being
	// started, but the task is not ready yet.
	AllocationStateStarting AllocationState = "STARTING"
	// AllocationStateRunning state denotes that the service in the command is running.
	AllocationStateRunning AllocationState = "RUNNING"
	// AllocationStateTerminated state denotes that the command has exited or has been aborted.
	AllocationStateTerminated AllocationState = "TERMINATED"
	// AllocationStateTerminating state denotes that the command is terminating.
	AllocationStateTerminating AllocationState = "TERMINATING"
)

// MostProgressedAllocationState returns the further progressed state. E.G. a call
// with PENDING, PULLING and STARTING returns PULLING.
func MostProgressedAllocationState(states ...AllocationState) AllocationState {
	if len(states) == 0 {
		return AllocationStatePending
	}

	// Can't use taskv1.State_value[state] since in proto
	// "STATE_TERMINATING" > "STATE_TERMINATED"
	// while our model used to have
	// "STATE_TERMINATED" > "STATE_TERMINATING".
	statesToOrder := map[AllocationState]int{
		AllocationStatePending:     0,
		AllocationStateAssigned:    1,
		AllocationStatePulling:     2,
		AllocationStateStarting:    3,
		AllocationStateRunning:     4,
		AllocationStateWaiting:     5,
		AllocationStateTerminating: 6,
		AllocationStateTerminated:  7,
	}
	maxOrder, state := statesToOrder[states[0]], states[0]
	for _, s := range states {
		if order := statesToOrder[s]; order > maxOrder {
			maxOrder, state = order, s
		}
	}
	return state
}

// Proto returns the proto representation of the task state.
func (s AllocationState) Proto() taskv1.State {
	switch s {
	case AllocationStateWaiting:
		return taskv1.State_STATE_WAITING
	case AllocationStatePulling:
		return taskv1.State_STATE_PULLING
	case AllocationStateStarting:
		return taskv1.State_STATE_STARTING
	case AllocationStateRunning:
		return taskv1.State_STATE_RUNNING
	case AllocationStateTerminating:
		return taskv1.State_STATE_TERMINATING
	case AllocationStateTerminated:
		return taskv1.State_STATE_TERMINATED
	default:
		return taskv1.State_STATE_UNSPECIFIED
	}
}

const (
	defaultTaskLogContainer = "UNKNOWN CONTAINER"
	defaultTaskLogTime      = "UNKNOWN TIME"

	// LogLevelTrace is the trace task log level.
	LogLevelTrace = tasklog.LogLevelTrace
	// LogLevelDebug is the debug task log level.
	LogLevelDebug = tasklog.LogLevelDebug
	// LogLevelInfo is the info task log level.
	LogLevelInfo = tasklog.LogLevelInfo
	// LogLevelWarning is the warn task log level.
	LogLevelWarning = tasklog.LogLevelWarning
	// LogLevelError is the error task log level.
	LogLevelError = tasklog.LogLevelError
	// LogLevelCritical is the critical task log level.
	LogLevelCritical = tasklog.LogLevelCritical
	// LogLevelUnspecified is the unspecified task log level.
	LogLevelUnspecified = tasklog.LogLevelUnspecified
)

// TaskLogLevelFromProto returns a task log level from its protobuf repr.
func TaskLogLevelFromProto(l logv1.LogLevel) string {
	switch l {
	case logv1.LogLevel_LOG_LEVEL_UNSPECIFIED:
		return LogLevelUnspecified
	case logv1.LogLevel_LOG_LEVEL_TRACE:
		return LogLevelTrace
	case logv1.LogLevel_LOG_LEVEL_DEBUG:
		return LogLevelDebug
	case logv1.LogLevel_LOG_LEVEL_INFO:
		return LogLevelInfo
	case logv1.LogLevel_LOG_LEVEL_WARNING:
		return LogLevelWarning
	case logv1.LogLevel_LOG_LEVEL_ERROR:
		return LogLevelError
	case logv1.LogLevel_LOG_LEVEL_CRITICAL:
		return LogLevelCritical
	default:
		return LogLevelUnspecified
	}
}

// TaskLogLevelToProto returns a protobuf task log level from its string repr.
func TaskLogLevelToProto(l string) logv1.LogLevel {
	switch l {
	case LogLevelTrace:
		return logv1.LogLevel_LOG_LEVEL_TRACE
	case LogLevelDebug:
		return logv1.LogLevel_LOG_LEVEL_DEBUG
	case LogLevelInfo:
		return logv1.LogLevel_LOG_LEVEL_INFO
	case LogLevelWarning:
		return logv1.LogLevel_LOG_LEVEL_WARNING
	case LogLevelError:
		return logv1.LogLevel_LOG_LEVEL_ERROR
	case LogLevelCritical:
		return logv1.LogLevel_LOG_LEVEL_CRITICAL
	default:
		return logv1.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}

// TaskLog represents a structured log emitted by an allocation.
type TaskLog struct {
	// A task log should have one of these IDs after being persisted. All should be unique.
	ID *int `db:"id" json:"id,omitempty"`
	// The body of an Elasticsearch log response will look something like
	// { _id: ..., _source: { ... }} where _source is the rest of this struct.
	// StringID doesn't have serialization tags because it is not part of
	// _source and populated from _id.
	StringID     *string `json:"-"`
	TaskID       string  `db:"task_id" json:"task_id"`
	AllocationID *string `db:"allocation_id" json:"allocation_id"`
	AgentID      *string `db:"agent_id" json:"agent_id,omitempty"`
	// In the case of k8s, container_id is a pod name instead.
	ContainerID *string    `db:"container_id" json:"container_id,omitempty"`
	RankID      *int       `db:"rank_id" json:"rank_id,omitempty"`
	Timestamp   *time.Time `db:"timestamp" json:"timestamp"`
	Level       *string    `db:"level" json:"level"`
	Log         string     `db:"log" json:"log"`
	Source      *string    `db:"source" json:"source,omitempty"`
	StdType     *string    `db:"stdtype" json:"stdtype,omitempty"`
}

const (
	// RFC3339MicroTrailingZeroes unlike time.RFC3339Nano is a time format specifier that preserves
	// trailing zeroes.
	RFC3339MicroTrailingZeroes = "2006-01-02T15:04:05.000000Z07:00"
	// containerIDMaxLength is the max display length for a container ID in logs.
	containerIDMaxLength = 8
)

// Message resolves the flat version of the log that UIs have shown historically.
// TODO(task-unif): Should we just.. stop doing this? And send the log as is and let the
// UIs handle display (yes, IMO).
func (t *TaskLog) Message() string {
	var parts []string

	// e.g., "[2022-03-02T02:15:18.299569Z]"
	if t.Timestamp != nil {
		parts = append(parts, fmt.Sprintf("[%s]", t.Timestamp.Format(RFC3339MicroTrailingZeroes)))
	} else {
		parts = append(parts, fmt.Sprintf("[%s]", defaultTaskLogTime))
	}

	// e.g., " f6114bb3"
	if t.ContainerID != nil && *t.ContainerID != "" {
		containerID := *t.ContainerID
		if len(containerID) > containerIDMaxLength {
			containerID = containerID[:containerIDMaxLength]
		}
		parts = append(parts, containerID)
	} else {
		// Just so the logs visually line up.
		parts = append(parts, strings.Repeat(" ", containerIDMaxLength))
	}

	// e.g., " [rank=1]"
	if t.RankID != nil {
		parts = append(parts, fmt.Sprintf("[rank=%d]", *t.RankID))
	}

	parts = append(parts, ("||"))

	// e.g., " INFO"
	if t.Level != nil {
		parts = append(parts, fmt.Sprintf("%s:", *t.Level))
	}

	parts = append(parts, t.Log)

	return strings.Join(parts, " ")
}

// Proto converts a task log to its protobuf representation.
func (t TaskLog) Proto() (*apiv1.TaskLogsResponse, error) {
	var id string
	switch {
	case t.ID != nil:
		id = strconv.Itoa(*t.ID)
	case t.StringID != nil:
		id = *t.StringID
	default:
		panic("log had no valid ID")
	}

	var ts *timestamppb.Timestamp
	if t.Timestamp != nil {
		ts = timestamppb.New(*t.Timestamp)
	}

	var level logv1.LogLevel
	if t.Level == nil {
		level = logv1.LogLevel_LOG_LEVEL_UNSPECIFIED
	} else {
		level = TaskLogLevelToProto(*t.Level)
	}

	resp := &apiv1.TaskLogsResponse{
		Id:           id,
		TaskId:       t.TaskID,
		Timestamp:    ts,
		Level:        level,
		Message:      t.Message(),
		Log:          t.Log,
		AllocationId: t.AllocationID,
		AgentId:      t.AgentID,
		ContainerId:  t.ContainerID,
		Source:       t.Source,
		Stdtype:      t.StdType,
	}

	if t.RankID != nil {
		id := int32(*t.RankID)
		resp.RankId = &id
	}

	return resp, nil
}

// TaskLogBatch represents a batch of model.TaskLog.
type TaskLogBatch []*TaskLog

// Size implements logs.Batch.
func (t TaskLogBatch) Size() int {
	return len(t)
}

// ForEach implements logs.Batch.
func (t TaskLogBatch) ForEach(f func(interface{}) error) error {
	for _, tl := range t {
		if err := f(tl); err != nil {
			return err
		}
	}
	return nil
}

// TaskContextDirectory represents a row in database for a tasks context directory.
// This currently is only for notebooks, trials, tensorboards, and commands now.
// Trials aren't in it because they are stored on experiments.model_def.
// In addition trials can have many tasks but currently can only have one model_def.
// We would end up duplicating a lot of data migrating experiment's model_def over to this
// table. Also that migration would be pretty painful.
type TaskContextDirectory struct {
	bun.BaseModel `bun:"table:task_context_directory"`

	TaskID           TaskID `bun:"task_id"`
	ContextDirectory []byte `bun:"context_directory"`
}

// AccessScopeID is an identifier for an access scope.
type AccessScopeID int

// AccessScopeSet is a set of access scopes.
type AccessScopeSet = map[AccessScopeID]bool
