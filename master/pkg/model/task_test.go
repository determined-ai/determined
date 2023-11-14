package model

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

func TestTaskLogLevelFromLogrus(t *testing.T) {
	for _, l := range logrus.AllLevels {
		require.NotEqual(t, LogLevelUnspecified, TaskLogLevelFromLogrus(l), "unhandled log level")
	}
}

func TestAllocationProto(t *testing.T) {
	a := Allocation{
		AllocationID: "aid",
		TaskID:       "tid",
		Slots:        1,
		ResourcePool: "rp",
		StartTime:    nil,
		EndTime:      nil,
		State:        nil,
		IsReady:      nil,
		Ports:        nil,
		ProxyAddress: nil,
		ExitReason:   nil,
		ExitErr:      nil,
		StatusCode:   nil,
	}

	expected := &taskv1.Allocation{
		TaskId:       "tid",
		IsReady:      nil,
		StartTime:    nil,
		EndTime:      nil,
		AllocationId: "aid",
		State:        taskv1.State_STATE_UNSPECIFIED,
		Slots:        1,
		ExitReason:   nil,
		StatusCode:   nil,
	}
	require.Equal(t, expected, a.Proto())
}
