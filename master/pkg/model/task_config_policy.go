package model

import (
	"strings"

	"github.com/determined-ai/determined/proto/pkg/configpolicyv1"
)

// WorkloadType is the type of workload (experiment or NTSC) that the task config policy applies to.
type WorkloadType string

// Constants.

const (
	// UnknownType constant.
	UnknownType WorkloadType = "UNSPECIFIED"
	// ExperimentType constant.
	ExperimentType WorkloadType = "EXPERIMENT"
	// NTSCType constant.
	NTSCType WorkloadType = "NTSC"
)

// WorkloadTypeFromProto maps taskconfigpolicyv1.WorkloadType to WorkloadType.
func WorkloadTypeFromProto(workloadType configpolicyv1.WorkloadType) WorkloadType {
	str := workloadType.String()
	return WorkloadType(strings.TrimPrefix(str, "WORKLOAD_TYPE_"))
}

// WorkloadTypeToProto maps WorkloadType to taskconfigpolicyv1.WorkloadType.
func WorkloadTypeToProto(workloadType WorkloadType) configpolicyv1.WorkloadType {
	protoWorkloadType := configpolicyv1.WorkloadType_value["WORKLOAD_TYPE_"+string(workloadType)]
	return configpolicyv1.WorkloadType(protoWorkloadType)
}
