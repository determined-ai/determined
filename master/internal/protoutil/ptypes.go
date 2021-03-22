package protoutil

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

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
