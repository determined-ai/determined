package telemetry

// debugLogger is an implementation of Segment's logger type that prints all messages at the debug
// level in order to reduce noise from failed messages.
type debugLogger struct{}

// Logf implements the analytics.Logger interface.
func (debugLogger) Logf(s string, a ...interface{}) {
	syslog.Debugf("segment log message: "+s, a...)
}

// Errorf implements the analytics.Logger interface.
func (debugLogger) Errorf(s string, a ...interface{}) {
	syslog.Debugf("segment error message: "+s, a...)
}
