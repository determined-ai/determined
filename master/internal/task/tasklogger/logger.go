package tasklogger

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
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

// Logger is an abstraction for inserting master-side inserted task logs, such as
// scheduling and provisioning information, or container exit statuses.
type Logger struct {
	backend Writer
	inbox   chan *model.TaskLog
}

// New creates an logger which can buffer up task logs and flush them periodically.
// There should only be one logger shared across the entire system.
func New(backend Writer) *Logger {
	l := &Logger{
		backend: backend,
		inbox:   make(chan *model.TaskLog, logBuffer),
	}

	go l.run()

	return l
}

// Insert always asynchronously inserts a task log. As a consequence, this means it does not have
// any mechanism for backpressure, so use with caution (a few, important logs should go here).
func (l *Logger) Insert(tl *model.TaskLog) {
	l.inbox <- tl
}

func (l *Logger) run() {
	pending := make([]*model.TaskLog, 0, logBuffer)
	defer l.flush(pending)

	t := time.NewTicker(logFlushInterval)
	for {
		var flush bool
		select {
		case <-t.C:
			flush = true
		case tl := <-l.inbox:
			pending = append(pending, tl)
			flush = len(pending) >= logBuffer
		}
		if !flush {
			continue
		}

		l.flush(pending)
		pending = make([]*model.TaskLog, 0, logBuffer)
	}
}

func (l *Logger) flush(pending []*model.TaskLog) {
	// TODO(mar): maybe dont hold the lock while shipping, but then you can get backed up.
	err := l.backend.AddTaskLogs(pending)
	if err != nil {
		logrus.WithField("component", "tasklogger").
			WithError(err).
			Errorf("failed to save task logs")
	}
}
