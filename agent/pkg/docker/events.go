package docker

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

type (
	// Event describes some Docker-layer event.
	Event struct {
		Log   *LogEvent
		Stats *StatsEvent
	}
	// LogEvent describes a log emitted from the Docker layer.
	LogEvent struct {
		Level     string
		Timestamp time.Time
		Message   string
	}
	// StatsEvent describes some stats about a Docker operation, such as IMAGEPULL.
	StatsEvent struct {
		Kind      string
		StartTime *time.Time
		EndTime   *time.Time
	}
)

// NewLogEvent initializes a new Event that is of kind 'LogEvent'.
func NewLogEvent(level, message string) Event {
	return Event{Log: &LogEvent{Level: level, Timestamp: time.Now().UTC(), Message: message}}
}

// NewBeginStatsEvent initializes a new beginning Event that is of kind 'StatsEvent' for the kind.
func NewBeginStatsEvent(kind string) Event {
	return Event{Stats: &StatsEvent{Kind: kind, StartTime: ptrs.Ptr(time.Now().UTC())}}
}

// NewEndStatsEvent initializes a new ending Event that is of kind 'StatsEvent' for the kind.
func NewEndStatsEvent(kind string) Event {
	return Event{Stats: &StatsEvent{Kind: kind, EndTime: ptrs.Ptr(time.Now().UTC())}}
}
