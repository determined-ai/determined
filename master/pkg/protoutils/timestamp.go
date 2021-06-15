package protoutils

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/golang/protobuf/ptypes/timestamp"
)

// ToTimestamp converts a Go time struct to a protobuf message.
func ToTimestamp(t time.Time) *timestamp.Timestamp {
	return &timestamp.Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

// TimeSliceFromProto converts a slice of *timestamppb.Timestamp to a slice of time.Time.
func TimeSliceFromProto(pTimes []*timestamppb.Timestamp) ([]time.Time, error) {
	ts := make([]time.Time, len(pTimes))
	for i, pt := range pTimes {
		if err := pt.CheckValid(); err != nil {
			return nil, err
		}
		ts[i] = pt.AsTime()
	}
	return ts, nil
}

// TimeProtoSliceFromTimes converts a slice of strings to a slice of *timestamppb.Timestamp.
func TimeProtoSliceFromTimes(times []time.Time) ([]*timestamppb.Timestamp, error) {
	ts := make([]*timestamppb.Timestamp, 0, len(times))
	for _, t := range times {
		ts = append(ts, timestamppb.New(t))
	}
	return ts, nil
}
