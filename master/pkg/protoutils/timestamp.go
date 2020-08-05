package protoutils

import (
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
)

// ToTimestamp converts a Go time struct to a protobuf message.
func ToTimestamp(t time.Time) *timestamp.Timestamp {
	return &timestamp.Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}
