package model

import (
	"strings"

	"github.com/determined-ai/determined/proto/pkg/taskconfigpolicyv1"
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
func WorkloadTypeFromProto(workloadType taskconfigpolicyv1.WorkloadType) WorkloadType {
	str := workloadType.String()
	return WorkloadType(strings.TrimPrefix(str, "TYPE_"))
}

// WorkloadTypeToProto maps WorkloadType to taskconfigpolicyv1.WorkloadType.
func WorkloadTypeToProto(workloadType WorkloadType) taskconfigpolicyv1.WorkloadType {
	protoWorkloadType := taskconfigpolicyv1.WorkloadType_value["TYPE_"+string(workloadType)]
	return taskconfigpolicyv1.WorkloadType(protoWorkloadType)
}
