package tasklogger

import (
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
)

var syslog = logrus.WithField("component", "tasklogger")

var defaultLogger = nullLogger

// SetDefaultLogger sets the task.Logger singleton used by package-level functions.
func SetDefaultLogger(l *Logger) {
	defaultLogger = l
}

// Insert a log with the default task logger.
func Insert(l *model.TaskLog) {
	if defaultLogger == nullLogger {
		// TODO(DET-9538): With the old behavior (in the actor system), using ctx.Tell to send a
		// log to an uninitialized logger resulted in a dropped log. For now, keep this behavior
		// with a big scary error.
		syslog.Error("use of uninitialized tasklogger")
	}
	defaultLogger.insert(l)
}
