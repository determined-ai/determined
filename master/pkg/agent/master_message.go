package agent

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
)

// TelemetryInfo contains the telemetry settings for the master.
type TelemetryInfo struct {
	Enabled    bool   `json:"enabled"`
	SegmentKey string `json:"segment_key,omitempty"`
}

// MasterInfo contains the master information that the agent has connected to.
type MasterInfo struct {
	Version   string        `json:"version"`
	MasterID  string        `json:"master_id"`
	ClusterID string        `json:"cluster_id"`
	Telemetry TelemetryInfo `json:"telemetry"`
}

// MasterMessage is a union type for all messages sent from agents.
type MasterMessage struct {
	AgentStarted          *AgentStarted
	ContainerStateChanged *ContainerStateChanged
	ContainerLog          *ContainerLog
}

// AgentStarted notifies the master that the agent has started up.
type AgentStarted struct {
	Version             string
	Label               string
	Devices             []device.Device
	RecoveredContainers []ContainerRecovered
}

// ContainerRecovered notifies the task that the agent recovered a container on startup.
type ContainerRecovered struct {
	Container        container.Container
	ContainerStarted ContainerStarted
}

// ContainerStateChanged notifies the master that the agent transitioned the container state.
type ContainerStateChanged struct {
	Container container.Container

	ContainerStarted *ContainerStarted
	ContainerStopped *ContainerStopped
}

// ContainerStarted notifies the master that the agent has started a container.
type ContainerStarted struct {
	ProxyAddress  string
	ContainerInfo types.ContainerJSON
}

// ContainerStopped notifies the master that a container was stopped on the agent.
type ContainerStopped struct {
	Failure *ContainerFailure
}

func (c ContainerStopped) String() string {
	if c.Failure == nil {
		return "container exited successfully with a zero exit code"
	}
	return c.Failure.Error()
}

// ContainerLog notifies the master that a new log message is available for the container.
type ContainerLog struct {
	Container container.Container
	Timestamp time.Time

	PullMessage *jsonmessage.JSONMessage
	RunMessage  *RunMessage
	AuxMessage  *string
}

// RunMessage holds the message sent by the container in the run phase.
type RunMessage struct {
	Value   string
	StdType stdcopy.StdType
}

func (c ContainerLog) String() string {
	msg := ""
	switch {
	case c.AuxMessage != nil:
		msg = *c.AuxMessage
	case c.RunMessage != nil:
		msg = strings.TrimSuffix(c.RunMessage.Value, "\n")
	case c.PullMessage != nil:
		buf := new(bytes.Buffer)
		if err := c.PullMessage.Display(buf, false); err != nil {
			msg = err.Error()
		} else {
			msg = buf.String()
			// Docker disables printing the progress bar in non-terminal mode.
			if msg == "" && c.PullMessage.Progress != nil {
				msg = c.PullMessage.Progress.String()
			}
			msg = strings.TrimSpace(msg)
		}
	default:
		panic("unknown log message received")
	}
	shortID := c.Container.ID[:8]
	timestamp := c.Timestamp.UTC().Format(time.RFC3339)
	return fmt.Sprintf("[%s] %s [%s] || %s", timestamp, shortID, c.Container.State, msg)
}
