package model

import (
	"strings"

	"github.com/determined-ai/determined/proto/pkg/configpolicyv1"
)

type WorkloadType string

// Constants.

const (
	// Unspecified constant.
	UnknownType WorkloadType = "UNSPECIFIED"
	// ExperimentType constant.
	ExperimentType WorkloadType = "EXPERIMENT"
	// NTSCType constant.
	NTSCType WorkloadType = "NTSC"
)

// WorkloadTypeFromProto maps taskconfigpolicyv1.WorkloadType to WorkloadType.
func WorkloadTypeFromProto(workloadType configpolicyv1.WorkloadType) WorkloadType {
	str := workloadType.String()
	return WorkloadType(strings.TrimPrefix(str, "TYPE_"))
}

// WorkloadTypeToProto maps WorkloadType to taskconfigpolicyv1.WorkloadType.
func WorkloadTypeToProto(workloadType WorkloadType) configpolicyv1.WorkloadType {
	protoWorkloadType := configpolicyv1.WorkloadType_value["TYPE_"+string(workloadType)]
	return configpolicyv1.WorkloadType(protoWorkloadType)
}
