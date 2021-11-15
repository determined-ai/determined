package task

import (
	"time"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	// logFlushInterval is the longest time that the logger will buffer logs in memory before
	// flushing them to the database. This is set low to ensure a good user experience.
	logFlushInterval = 20 * time.Millisecond
	// logBuffer is the largest number of logs lines that can be buffered before flushing them to
	// the database. For the strategy of many-rows-per-insert, performance was significantly worse
	// below 500, and no improvements after 1000.
	logBuffer = 1000
)

var loggerAddr = actor.Addr("taskLogger")

type (
	// flushLogs is a message that the trial actor sends to itself via
	// NotifyAfter(), which is used to guarantee that logs are not held too
	// long without flushing.
	flushLogs struct{}
)

// LogBackend is an interface task log backends, such as elastic or postgres,
// must support to provide the features surfaced in our API.
type LogBackend interface {
	TaskLogs(
		taskID model.TaskID, limit int, filters []api.Filter, order apiv1.OrderBy, state interface{},
	) ([]*model.TaskLog, interface{}, error)
	AddTaskLogs([]*model.TaskLog) error
	TaskLogsCount(taskID model.TaskID, filters []api.Filter) (int, error)
	TaskLogsFields(taskID model.TaskID) (*apiv1.TaskLogsFieldsResponse, error)
	DeleteTaskLogs(taskIDs []model.TaskID) error
}

type (
	logger struct {
		backend      LogBackend
		pending      []*model.TaskLog
		lastLogFlush time.Time
	}

	// Logger is an abstraction for inserting master-side inserted task logs, such as
	// scheduling and provisioning information, or container exit statuses.
	Logger struct {
		inner *actor.Ref
	}
)

// NewLogger creates an logger which can buffer up task logs and flush them periodically.
// There should only be one logger shared across the entire system.
func NewLogger(system *actor.System, backend LogBackend) *Logger {
	return &Logger{
		inner: system.MustActorOf(loggerAddr, &logger{
			backend:      backend,
			lastLogFlush: time.Now(),
			pending:      make([]*model.TaskLog, 0, logBuffer),
		}),
	}
}

// NewCustomLogger returns a logger backend by a custom actor, for tests only.
func NewCustomLogger(logger *actor.Ref) *Logger {
	return &Logger{
		inner: logger,
	}
}

// Insert always asynchronously inserts a task log. Though, this means we do not have
// any mechanism for backpressure, so use with caution (a few, important logs should go here).
func (l *Logger) Insert(ctx *actor.Context, tl model.TaskLog) {
	ctx.Tell(l.inner, tl)
}

func (l *logger) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, logFlushInterval, flushLogs{})

	case flushLogs:
		l.tryFlushLogs(ctx, true)
		actors.NotifyAfter(ctx, logFlushInterval, flushLogs{})

	case model.TaskLog:
		l.pending = append(l.pending, &msg)
		l.tryFlushLogs(ctx, false)

	case actor.PostStop:
		// Flush any final logs.
		l.tryFlushLogs(ctx, true)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (l *logger) tryFlushLogs(ctx *actor.Context, forceFlush bool) {
	if forceFlush || len(l.pending) >= logBuffer {
		if err := l.backend.AddTaskLogs(l.pending); err != nil {
			ctx.Log().WithError(err).Errorf("failed to save task logs")
		}
		l.pending = l.pending[:0]
	}
}
