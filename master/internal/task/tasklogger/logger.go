package tasklogger

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/model"
)

var (
	// FlushInterval is the longest time that the logger will buffer logs in memory before
	// flushing them to the database. This is set low to ensure a good user experience.
	FlushInterval = 20 * time.Millisecond
	// BufferSize is the largest number of logs lines that can be buffered before flushing them to
	// the database. For the strategy of many-rows-per-insert, performance was significantly worse
	// below 500, and no improvements after 1000.
	BufferSize = 1000
)

// Logger is an abstraction for inserting master-side inserted task logs, such as
// scheduling and provisioning information, or container exit statuses.
// TODO(DET-9537): Add graceful shutdown for the tasklogger, so that when we
// intentionally blip the master for something like a configuration update
// we do not lose logs.
type Logger struct {
	backend Writer
	inbox   chan *model.TaskLog
}

// New creates an logger which can buffer up task logs and flush them periodically.
// There should only be one logger shared across the entire system.
func New(backend Writer) *Logger {
	l := Logger{
		backend: backend,
		inbox:   make(chan *model.TaskLog, BufferSize),
	}

	go l.run()
	return &l
}

// Insert a log into the buffer to be flush within some interval.
func (l *Logger) Insert(tl *model.TaskLog) {
	l.inbox <- tl
}

func (l *Logger) run() {
	pending := make([]*model.TaskLog, 0, BufferSize)
	defer l.flush(pending)

	t := time.NewTicker(FlushInterval)
	defer t.Stop()
	for {
		var flush bool
		select {
		case <-t.C:
			flush = len(pending) > 0
		case tl := <-l.inbox:
			pending = append(pending, tl)
			flush = len(pending) >= BufferSize
		}
		if !flush {
			continue
		}

		l.flush(pending)
		pending = make([]*model.TaskLog, 0, BufferSize)
	}
}

func (l *Logger) flush(pending []*model.TaskLog) {
	err := l.backend.AddTaskLogs(pending)
	if err != nil {
		syslog.WithError(err).Errorf("failed to save task logs")
	}
}
