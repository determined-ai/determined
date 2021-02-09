package tasks

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/determined-ai/determined/master/pkg/device"
)

// LocalRendezvousPort is the start of the range of ports used for rendezvous by tasks.
const LocalRendezvousPort = 1734

// LocalRendezvousPortOffset is the difference between the two rendezvous ports. It is chosen to be
// 16 since this is the maximum number of GPUs expected per agent.
const LocalRendezvousPortOffset = 16

const (
	hostMode container.NetworkMode = "host"
)

func rendezvousPorts(offset int) []int {
	base := LocalRendezvousPort + offset
	return []int{base, base + LocalRendezvousPortOffset}
}

// trialUniquePortOffset determines a deterministic, unique offset for ports that would otherwise
// collide when using host networking.
func trialUniquePortOffset(devices []device.Device) int {
	if len(devices) == 0 {
		return 0
	}
	min := devices[0].ID
	for _, d := range devices {
		if d.ID < min {
			min = d.ID
		}
	}
	return min
}

func toPortSet(ports map[string]int) nat.PortSet {
	dockerPorts := make(nat.PortSet)
	for _, port := range ports {
		dockerPorts[nat.Port(fmt.Sprintf("%d/tcp", port))] = struct{}{}
	}
	return dockerPorts
}
