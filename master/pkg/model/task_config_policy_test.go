package model

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/proto/pkg/configpolicyv1"
)

func TestWorkloadTypeFromProto(t *testing.T) {
	tests := []struct {
		name         string
		workloadType configpolicyv1.WorkloadType
		expected     WorkloadType
	}{
		{"unknown type", configpolicyv1.WorkloadType_WORKLOAD_TYPE_UNSPECIFIED, UnknownType},
		{"experiment type", configpolicyv1.WorkloadType_WORKLOAD_TYPE_EXPERIMENT, ExperimentType},
		{"NTSC type", configpolicyv1.WorkloadType_WORKLOAD_TYPE_NTSC, NTSCType},
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
		expected     configpolicyv1.WorkloadType
	}{
		{"unknown type", UnknownType, configpolicyv1.WorkloadType_WORKLOAD_TYPE_UNSPECIFIED},
		{"experiment type", ExperimentType, configpolicyv1.WorkloadType_WORKLOAD_TYPE_EXPERIMENT},
		{"NTSC type", NTSCType, configpolicyv1.WorkloadType_WORKLOAD_TYPE_NTSC},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, WorkloadTypeToProto(test.workloadType))
		})
	}
}
