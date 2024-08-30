package model

import (
	"testing"

	"github.com/determined-ai/determined/proto/pkg/taskconfigpolicyv1"
	"github.com/stretchr/testify/require"
)

func TestWorkloadTypeFromProto(t *testing.T) {
	tests := []struct {
		name         string
		workloadType taskconfigpolicyv1.WorkloadType
		expected     WorkloadType
	}{
		{"unknown type", taskconfigpolicyv1.WorkloadType_TYPE_UNSPECIFIED, UnknownType},
		{"experiment type", taskconfigpolicyv1.WorkloadType_TYPE_EXPERIMENT, ExperimentType},
		{"NTSC type", taskconfigpolicyv1.WorkloadType_TYPE_NTSC, NTSCType},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, WorkloadTypeFromProto(test.workloadType))
		})
	}
}

func TestWorkloadTypeToProto(t *testing.T) {
	tests := []struct {
		name         string
		workloadType WorkloadType
		expected     taskconfigpolicyv1.WorkloadType
	}{
		{"unknown type", UnknownType, taskconfigpolicyv1.WorkloadType_TYPE_UNSPECIFIED},
		{"experiment type", ExperimentType, taskconfigpolicyv1.WorkloadType_TYPE_EXPERIMENT},
		{"NTSC type", NTSCType, taskconfigpolicyv1.WorkloadType_TYPE_NTSC},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, WorkloadTypeToProto(test.workloadType))
		})
	}
}
