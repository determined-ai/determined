package aproto

import (
	"fmt"
	"strconv"
	"time"

	"golang.org/x/exp/maps"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TelemetryInfo contains the telemetry settings for the master.
type TelemetryInfo struct {
	Enabled                  bool   `json:"enabled"`
	SegmentKey               string `json:"segment_key,omitempty"`
	OtelEnabled              bool   `json:"otel_enabled"`
	OtelExportedOtlpEndpoint string `json:"otel_endpoint"`
}

// MasterInfo contains the master information that the agent has connected to.
type MasterInfo struct {
	Version     string        `json:"version"`
	MasterID    string        `json:"master_id"`
	ClusterID   string        `json:"cluster_id"`
	ClusterName string        `json:"cluster_name"`
	Telemetry   TelemetryInfo `json:"telemetry"`
}

// MasterMessage is a union type for all messages sent from agents.
type MasterMessage struct {
	AgentStarted          *AgentStarted
	ContainerStateChanged *ContainerStateChanged
	ContainerLog          *ContainerLog
	ContainerStatsRecord  *ContainerStatsRecord
}

// ContainerReattach is a struct describing containers that can be reattached.
type ContainerReattach struct {
	Container cproto.Container
}

// ContainerReattachAck is a struct describing containers reattachment success.
type ContainerReattachAck struct {
	Container cproto.Container
	Failure   *ContainerFailureError
}

// FromContainerStateChanged forms a container reattach ack from a state change message.
func FromContainerStateChanged(csc *ContainerStateChanged) *ContainerReattachAck {
	ack := ContainerReattachAck{Container: csc.Container}
	if csc.ContainerStopped != nil {
		ack.Failure = csc.ContainerStopped.Failure
	}
	return &ack
}

// ID is an identifier for an agent.
type ID string

// AgentStarted notifies the master that the agent has started up.
type AgentStarted struct {
	Version              string
	Devices              []device.Device
	ContainersReattached []ContainerReattachAck
}

// ContainerStateChanged notifies the master that the agent transitioned the container state.
type ContainerStateChanged struct {
	Container cproto.Container

	ContainerStarted *ContainerStarted
	ContainerStopped *ContainerStopped
}

// ContainerStarted notifies the master that the agent has started a container.
type ContainerStarted struct {
	ProxyAddress  string
	ContainerInfo types.ContainerJSON
}

func (c ContainerStarted) String() string {
	return fmt.Sprintf("docker container %s running", c.ContainerInfo.ID)
}

// Addresses calculates the address of containers and hosts based on the container
// started information.
func (c ContainerStarted) Addresses() []cproto.Address {
	proxy := c.ProxyAddress
	info := c.ContainerInfo

	var addresses []cproto.Address
	switch networkMode := info.HostConfig.NetworkMode; networkMode {
	case "host":
		for port := range info.Config.ExposedPorts {
			addresses = append(addresses, cproto.Address{
				ContainerIP:   proxy,
				ContainerPort: port.Int(),
				HostIP:        proxy,
				HostPort:      port.Int(),
			})
		}
	default:
		if info.NetworkSettings == nil {
			return nil
		}

		// We used to return a cproto.Address for each network in info.NetworkSettings.Networks that existed, but the
		// harness only wants the tuple (host_ip, host_port) and disregards container_ip, and (!!) if it does receive
		// two cproto.Address mappings for the rendezvous port of a container, it would just explode anyway because it
		// expects a single address per container and doesn't know which one to use.
		containerIPOnNetwork := "missing"
		if containerNetwork := info.NetworkSettings.Networks[networkMode.NetworkName()]; containerNetwork != nil {
			containerIPOnNetwork = containerNetwork.IPAddress
		}

		for port, bindings := range info.NetworkSettings.Ports {
			for _, binding := range bindings {
				hostPort, err := strconv.Atoi(binding.HostPort)
				if err != nil {
					panic(fmt.Errorf("unexpected host port %s: %w", binding.HostPort, err))
				}

				addresses = append(addresses, cproto.Address{
					ContainerIP:   containerIPOnNetwork,
					ContainerPort: port.Int(),
					HostIP:        proxy,
					HostPort:      hostPort,
				})
			}
		}

		// Remove duplicates, just incase. In theory, I can't think of a case where we would have them, but
		// in practice somehow we had them before and I'd rather be careful.
		dedup := map[cproto.Address]bool{}
		for _, addr := range addresses {
			dedup[addr] = true
		}
		addresses = maps.Keys(dedup)
	}
	return addresses
}

// ContainerStopped notifies the master that a container was stopped on the agent.
type ContainerStopped struct {
	Failure *ContainerFailureError
}

func (c ContainerStopped) String() string {
	if c.Failure == nil {
		return "container exited successfully with a zero exit code"
	}
	return c.Failure.Error()
}

// ContainerLog notifies the master that a new log message is available for the container.
type ContainerLog struct {
	ContainerID cproto.ID
	Timestamp   time.Time
	Level       *string
	PullMessage *string
	RunMessage  *RunMessage
	AuxMessage  *string
	Source      *string
	AgentID     *string
}

// RunMessage holds the message sent by the container in the run phase.
type RunMessage struct {
	Value   string
	StdType stdcopy.StdType
}

// ContainerStatsRecord notifies the master that about the container stats of docker.
// For now this carries stats of docker image pull.
type ContainerStatsRecord struct {
	EndStats bool
	Stats    *model.TaskStats
	TaskType model.TaskType
}
