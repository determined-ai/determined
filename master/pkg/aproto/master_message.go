package aproto

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"

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
	Failure   *ContainerFailure
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
	proxyIsIPv4 := net.ParseIP(proxy).To4() != nil

	info := c.ContainerInfo

	var addresses []cproto.Address
	switch info.HostConfig.NetworkMode {
	case "host":
		for port := range info.Config.ExposedPorts {
			addresses = append(addresses, cproto.Address{
				ContainerIP:   proxy,
				ContainerPort: port.Int(),
			})
		}
	default:
		if info.NetworkSettings == nil {
			return nil
		}
		networks := info.NetworkSettings.Networks
		ipAddresses := make([]string, 0, len(networks))
		for _, network := range networks {
			ipAddresses = append(ipAddresses, network.IPAddress)
		}

		for port, bindings := range info.NetworkSettings.Ports {
			// Unpublished port (possibly under direct connectivity mode)
			if len(bindings) == 0 {
				for _, ip := range ipAddresses {
					addresses = append(addresses, cproto.Address{
						ContainerIP:   ip,
						ContainerPort: port.Int(),
					})
				}
				continue
			}

			for _, binding := range bindings {
				for _, ip := range ipAddresses {
					hostIP := binding.HostIP
					switch {
					case hostIP == "0.0.0.0":
						// Just don't return the ipv4 binding for an ipv6 proxy
						if !proxyIsIPv4 {
							continue
						}
						hostIP = proxy
					case hostIP == "::":
						// And vice versa.
						if proxyIsIPv4 {
							continue
						}
						hostIP = proxy
					case hostIP == "":
						hostIP = proxy
					}

					hostPort, err := strconv.Atoi(binding.HostPort)
					if err != nil {
						panic(errors.Wrapf(err, "unexpected host port: %s", binding.HostPort))
					}

					addresses = append(addresses, cproto.Address{
						ContainerIP:   ip,
						ContainerPort: port.Int(),
						HostIP:        &hostIP,
						HostPort:      &hostPort,
					})
				}
			}
		}
	}
	return addresses
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
	ContainerID cproto.ID
	Timestamp   time.Time
	Level       *string
	PullMessage *string
	RunMessage  *RunMessage
	AuxMessage  *string
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
