package tasks

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

const (
	hostMode container.NetworkMode = "host"
)

func toPortSet(ports map[string]int) nat.PortSet {
	dockerPorts := make(nat.PortSet)
	for _, port := range ports {
		dockerPorts[nat.Port(fmt.Sprintf("%d/tcp", port))] = struct{}{}
	}
	return dockerPorts
}
