package agent

import (
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"

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
}

// AgentStarted notifies the master that the agent has started up.
type AgentStarted struct {
	Version string
	Label   string
	Devices []device.Device
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

// Addresses calculates the address of containers and hosts based on the container
// started information.
func (c ContainerStarted) Addresses() []container.Address {
	proxy := c.ProxyAddress

	proxyIP, _, err := net.ParseCIDR(proxy)
	if err != nil {
		panic(errors.Wrapf(err, "failed to parse ip %s", proxy))
	}
	proxyIsIPv4 := len(proxyIP) == net.IPv4len

	info := c.ContainerInfo

	var addresses []container.Address
	switch info.HostConfig.NetworkMode {
	case "host":
		for port := range info.Config.ExposedPorts {
			addresses = append(addresses, container.Address{
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
		networks := info.NetworkSettings.Networks
		ipAddresses := make([]string, 0, len(networks))
		for _, network := range networks {
			ipAddresses = append(ipAddresses, network.IPAddress)
		}

		for port, bindings := range info.NetworkSettings.Ports {
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

					addresses = append(addresses, container.Address{
						ContainerIP:   ip,
						ContainerPort: port.Int(),
						HostIP:        hostIP,
						HostPort:      hostPort,
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
