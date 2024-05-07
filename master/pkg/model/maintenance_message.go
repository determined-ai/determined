package model

import (
	"database/sql"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// ClusterMessage represents a server status from the `cluster_messages` table.
type ClusterMessage struct {
	CreatedBy   int
	Message     string
	StartTime   time.Time
	EndTime     sql.NullTime
	CreatedTime sql.NullTime
}

// ToProto converts m to a type suitable for gRPC protobuf response.
func (m *ClusterMessage) ToProto() *apiv1.ClusterMessage {
	if m == nil {
		return nil
	}

	ret := &apiv1.ClusterMessage{
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
