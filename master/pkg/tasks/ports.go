package tasks

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/determined-ai/determined/master/pkg/device"
)

// LocalRendezvousPort is the start of the range of ports used for rendezvous by tasks.
const LocalRendezvousPort = 1734

// LocalRendezvousPortOffset is the difference between the two rendezvous ports.
const LocalRendezvousPortOffset = 16

const (
	hostMode container.NetworkMode = "host"
)

func rendezvousPorts(devices []device.Device, networkMode container.NetworkMode) []nat.Port {
	ports := make([]nat.Port, 0)
	var min int
	if networkMode == hostMode {
		min = devices[0].ID
		for _, d := range devices {
			if d.ID < min {
				min = d.ID
			}
		}
	}
	ports = append(ports, nat.Port(fmt.Sprintf("%d/tcp", LocalRendezvousPort+min)))
	ports = append(
		ports, nat.Port(fmt.Sprintf("%d/tcp", LocalRendezvousPort+min+LocalRendezvousPortOffset)))
	return ports
}

func toPortSet(ports map[string]int) nat.PortSet {
	dockerPorts := make(nat.PortSet)
	for _, port := range ports {
		dockerPorts[nat.Port(fmt.Sprintf("%d/tcp", port))] = struct{}{}
	}
	return dockerPorts
}
