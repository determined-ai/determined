package model

import (
	"fmt"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/logv1"

	"github.com/google/uuid"

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

const (
	// TaskTypeTrial is the "TRIAL" job type for the enum public.job_type in Postgres.
	TaskTypeTrial = "TRIAL"
	// TaskTypeNotebook is the "NOTEBOOK" job type for the enum public.job_type in Postgres.
	TaskTypeNotebook = "NOTEBOOK"
	// TaskTypeShell is the "SHELL" job type for the enum public.job_type in Postgres.
	TaskTypeShell = "SHELL"
	// TaskTypeCommand is the "COMMAND" job type for the enum public.job_type in Postgres.
	TaskTypeCommand = "COMMAND"
	// TaskTypeTensorboard is the "TENSORBOARD" task type for the enum.task_type in Postgres.
	TaskTypeTensorboard = "TENSORBOARD"
	// TaskTypeCheckpointGC is the "CHECKPOINT_GC" job type for the enum public.job_type in Postgres.
	TaskTypeCheckpointGC = "CHECKPOINT_GC"
)

// TaskLogVersion is the version for our log-storing scheme. Useful because changing designs
// would involve either a really costly migration or versioning schemes and we pick the latter.
type TaskLogVersion int32

// CurrentTaskLogVersion describes the current scheme in which we store task
// logs. To avoid a migration that in some cases would be extremely
// costly, we record the log version so that we can just read old logs
// the old way and do the new however we please.
const CurrentTaskLogVersion TaskLogVersion = 1

// Task is the model for a task in the database.
type Task struct {
	TaskID    TaskID     `db:"task_id"`
	JobID     JobID      `db:"job_id"`
	TaskType  TaskType   `db:"task_type"`
	StartTime time.Time  `db:"start_time"`
	EndTime   *time.Time `db:"end_time"`
	// LogVersion indicates how the logs were stored.
	LogVersion TaskLogVersion `db:"log_version"`
}

// AllocationID is the ID of an allocation of a task. It is usually of the form
// TaskID.allocation_number, maybe with some other metadata if different types of
// allocations run.
type AllocationID string

// NewAllocationID returns a new unique task id.
func NewAllocationID(name string) AllocationID {
	return AllocationID(name)
}

func (a AllocationID) String() string {
	return string(a)
}

// Allocation is the model for an allocation in the database.
type Allocation struct {
	AllocationID AllocationID `db:"allocation_id"`
	TaskID       TaskID       `db:"task_id"`
	Slots        int          `db:"slots"`
	AgentLabel   string       `db:"agent_label"`
	ResourcePool string       `db:"resource_pool"`
	StartTime    time.Time    `db:"start_time"`
	EndTime      *time.Time   `db:"end_time"`
}

// AllocationState represents the current state of the task. Value indicates a partial ordering.
type AllocationState int

const (
	// AllocationStatePending state denotes that the command is awaiting allocation.
	AllocationStatePending AllocationState = 0
	// AllocationStateAssigned state denotes that the command has been assigned to an agent but has
	// not started yet.
	AllocationStateAssigned AllocationState = 1
	// AllocationStatePulling state denotes that the command's base image is being pulled from the
	// Docker registry.
	AllocationStatePulling AllocationState = 2
	// AllocationStateStarting state denotes that the image has been pulled and the task is being
	// started, but the task is not ready yet.
	AllocationStateStarting AllocationState = 3
	// AllocationStateRunning state denotes that the service in the command is running.
	AllocationStateRunning AllocationState = 4
	// AllocationStateTerminating state denotes that the command is terminating.
	AllocationStateTerminating AllocationState = 5
	// AllocationStateTerminated state denotes that the command has exited or has been aborted
	AllocationStateTerminated AllocationState = 6
)

// MostProgressedAllocationState returns the further progressed state. E.G. a call
// with PENDING, PULLING and STARTING returns PULLING.
func MostProgressedAllocationState(states ...AllocationState) AllocationState {
	if len(states) == 0 {
		return AllocationStatePending
	}

	max := states[0]
	for _, state := range states {
		if state > max {
			max = state
		}
	}
	return max
}

// String returns the string representation of the task state.
func (s AllocationState) String() string {
	switch s {
	case AllocationStatePending:
		return "PENDING"
	case AllocationStateAssigned:
		return "ASSIGNED"
	case AllocationStatePulling:
		return "PULLING"
	case AllocationStateStarting:
		return "STARTING"
	case AllocationStateRunning:
		return "RUNNING"
	case AllocationStateTerminating:
		return "TERMINATING"
	case AllocationStateTerminated:
		return "TERMINATED"
	default:
		return "UNSPECIFIED"
	}
}

// Proto returns the proto representation of the task state.
func (s AllocationState) Proto() taskv1.State {
	switch s {
	case AllocationStatePending:
		return taskv1.State_STATE_PENDING
	case AllocationStateAssigned:
		return taskv1.State_STATE_ASSIGNED
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
	LogLevelTrace = "TRACE"
	// LogLevelDebug is the debug task log level.
	LogLevelDebug = "DEBUG"
	// LogLevelInfo is the info task log level.
	LogLevelInfo = "INFO"
	// LogLevelWarn is the warn task log level.
	LogLevelWarn = "WARN"
	// LogLevelError is the error task log level.
	LogLevelError = "ERROR"
	// LogLevelCritical is the critical task log level.
	LogLevelCritical = "CRITICAL"
	// LogLevelUnspecified is the unspecified task log level.
	LogLevelUnspecified = "UNSPECIFIED"
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
		return LogLevelWarn
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
	case LogLevelWarn:
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
	// _source and populated from from _id.
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
	FlatLog     string     `db:"message" json:"message,omitempty"`
	Source      *string    `db:"source" json:"source,omitempty"`
	StdType     *string    `db:"stdtype" json:"stdtype,omitempty"`
}

// Message resolves the flat version of the log that UIs have shown historically.
// TODO(task-unif): Should we just.. stop doing this? And send the log as is and let the
// UIs handle display (yes, IMO).
func (t *TaskLog) Message() string {
	if t.FlatLog != "" {
		return t.FlatLog
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
	if t.ContainerID != nil && *t.ContainerID != "" {
		containerID = *t.ContainerID
		if len(containerID) > containerIDMaxLength {
			containerID = containerID[:containerIDMaxLength]
		}
		containerID = fmt.Sprintf("%s ", containerID)
	}

	var rankID string
	if t.RankID != nil {
		rankID = fmt.Sprintf("[rank=%d] ", *t.RankID)
	}

	var level string
	if t.Level != nil {
		level = fmt.Sprintf("%s: ", *t.Level)
	}

	return fmt.Sprintf("[%s] %s%s|| %s %s", timestamp, containerID, rankID, level, t.Log)
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

	return &apiv1.TaskLogsResponse{
		Id:        id,
		Timestamp: ts,
		Level:     level,
		Message:   t.FlatLog,
	}, nil
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
