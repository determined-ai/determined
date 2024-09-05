package model

import (
	"strings"
	"time"

	"github.com/determined-ai/determined/master/pkg/schemas/configpolicy"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/configpolicyv1"
	"github.com/uptrace/bun"
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

// ExperimentTaskConfigPolicy is the bun model of a task config policy.
type ExperimentTaskConfigPolicy struct {
	bun.BaseModel   `bun:"table:task_config_policies"`
	WorkspaceID     int                      `bun:"workspace_id,unique"`
	LastUpdatedBy   int                      `bun:"last_updated_by,notnull"`
	LastUpdatedTime time.Time                `bun:"last_updated_time,notnull"`
	WorkloadType    WorkloadType             `bun:"workload_type,notnull"`
	InvariantConfig expconf.ExperimentConfig `bun:"invariant_config"`
	Constraints     configpolicy.Constraints `bun:"constraints"`
}

// NTSCTaskConfigPolicy is the bun model of a task config policy.
type NTSCTaskConfigPolicy struct {
	bun.BaseModel   `bun:"table:task_config_policies"`
	WorkspaceID     int                      `bun:"workspace_id,unique"`
	LastUpdatedBy   int                      `bun:"last_updated_by,notnull"`
	LastUpdatedTime time.Time                `bun:"last_updated_time,notnull"`
	WorkloadType    WorkloadType             `bun:"workload_type,notnull"`
	InvariantConfig CommandConfig            `bun:"invariant_config"`
	Constraints     configpolicy.Constraints `bun:"constraints"`
}

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

func (w WorkloadType) String() string {
	return string(w)
}
