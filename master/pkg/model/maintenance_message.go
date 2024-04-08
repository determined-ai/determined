package model

import (
	"database/sql"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// MaintenanceMessage represents a server status from the `maintenance_messages` table.
type MaintenanceMessage struct {
	CreatedBy   int
	Message     string
	StartTime   time.Time
	EndTime     sql.NullTime
	CreatedTime sql.NullTime
}

// ToProto converts m to a type suitable for gRPC protobuf response.
func (m *MaintenanceMessage) ToProto() *apiv1.MaintenanceMessage {
	if m == nil {
		return nil
	}

	ret := &apiv1.MaintenanceMessage{
		Message:   m.Message,
		StartTime: timestamppb.New(m.StartTime),
	}
	if m.EndTime.Valid {
		ret.EndTime = timestamppb.New(m.EndTime.Time)
	}
	if m.CreatedTime.Valid {
		ret.CreatedTime = timestamppb.New(m.CreatedTime.Time)
	}
	return ret
}
