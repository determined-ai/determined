package model

import (
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/uptrace/bun"
)

// TaskAlert is the db representation of a task alert.
type TaskAlert struct {
	bun.BaseModel `bun:"table:task_alerts"`
	TaskID        string   `json:"task_id"`
	NodeID        string   `json:"node_id"`
	DeviceIDs     []string `json:"device_ids"`
	// Type?
	Action agentv1.Action `json:"action"`
}

func (t *TaskAlert) ToProto() agentv1.RunAlert {
	return agentv1.RunAlert{
		TaskId: t.TaskID,
		NodeId: t.NodeID,
		// DeviceIds: t.DeviceIDs,
		Action: t.Action,
	}
}

func (t *TaskAlert) fromProto(alert *agentv1.RunAlert) {
	t.TaskID = alert.TaskId
	t.NodeID = alert.NodeId
	// t.DeviceIDs = alert.DeviceIds
	t.Action = alert.Action
}
