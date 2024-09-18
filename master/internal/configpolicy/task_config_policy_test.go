package configpolicy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPriorityWithinLimit(t *testing.T) {
	testCases := []struct {
		name            string
		userPriority    int
		adminLimit      int
		smallerIsHigher bool
		ok              bool
	}{
		{"smaller is higher - ok", 10, 1, true, true},
		{"smaller is higher - not ok", 10, 20, true, false},
		{"smaller is higher - equal", 20, 20, true, true},
		{"smaller is lower - ok", 11, 13, false, true},
		{"smaller is lower - not ok", 13, 11, false, false},
		{"smaller is lower - equal", 11, 11, false, true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ok := priorityWithinLimit(tt.userPriority, tt.adminLimit, tt.smallerIsHigher)
			require.Equal(t, tt.ok, ok)
		})
	}
}
