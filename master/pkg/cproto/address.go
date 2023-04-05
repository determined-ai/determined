package cproto

import (
	"fmt"

	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

// Address represents an exposed port on a container.
type Address struct {
	// ContainerIP is the IP address from inside the container.
	ContainerIP string `json:"container_ip"`
	// ContainerPort is the port from inside the container.
	ContainerPort int `json:"container_port"`

	// HostIP is the IP address from outside the container. This can be
	// different than the ContainerIP because of network forwarding on the host
	// machine.
	HostIP string `json:"host_ip"`
	// HostPort is the IP port from outside the container. This can be different
	// than the ContainerPort because of network forwarding on the host machine.
	HostPort int `json:"host_port"`
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d:%s:%d", a.HostIP, a.HostPort, a.ContainerIP, a.ContainerPort)
}

// Proto returns the proto representation of address.
func (a *Address) Proto() *taskv1.Address {
	if a == nil {
		return nil
	}
	return &taskv1.Address{
		ContainerIp:   a.ContainerIP,
		ContainerPort: int32(a.ContainerPort),
		HostIp:        a.HostIP,
		HostPort:      int32(a.HostPort),
	}
}
