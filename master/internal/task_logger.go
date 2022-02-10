package internal

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	// logFlushInterval is the longest time that the trialLogger will buffer logs in memory before
	// flushing them to the database. This is set low to ensure a good user experience.
	logFlushInterval = 20 * time.Millisecond
	// logBuffer is the largest number of logs lines that can be buffered before flushing them to
	// the database. For the strategy of many-rows-per-insert, performance was significantly worse
	// below 500, and no improvements after 1000.
	logBuffer = 1000
)

type (
	// flushLogs is a message that the trial actor sends to itself via
	// NotifyAfter(), which is used to guarantee that logs are not held too
	// long without flushing.
	flushLogs struct{}
)

type taskLogger struct {
	backend      TaskLogBackend
	pending      []*model.TaskLog
	lastLogFlush time.Time
}

// newTaskLogger creates an actor which can buffer up task logs and flush them periodically.
// There should only be one taskLogger shared across the entire system.
func newTaskLogger(backend TaskLogBackend) actor.Actor {
	return &taskLogger{
		backend:      backend,
		lastLogFlush: time.Now(),
		pending:      make([]*model.TaskLog, 0, logBuffer),
	}
}

func (l *taskLogger) Receive(ctx *actor.Context) error {
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

func (l *taskLogger) tryFlushLogs(ctx *actor.Context, forceFlush bool) {
	if forceFlush || len(l.pending) >= logBuffer {
		if err := l.backend.AddTaskLogs(l.pending); err != nil {
			ctx.Log().WithError(err).Errorf("failed to save task logs")
		}
		l.pending = l.pending[:0]
	}
}
