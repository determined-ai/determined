package protoutil

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TimeSliceFromProto converts a slice of *timestamppb.Timestamp to a slice of time.Time.
func TimeSliceFromProto(pTimes []*timestamppb.Timestamp) ([]time.Time, error) {
	ts := make([]time.Time, len(pTimes))
	for i, pt := range pTimes {
		t, err := ptypes.Timestamp(pt)
		if err != nil {
			return nil, fmt.Errorf("failed to convert proto timestamp: %w", err)
		}
		ts[i] = t
	}
	return ts, nil
}
