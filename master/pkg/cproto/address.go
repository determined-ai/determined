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
	HostIP *string `json:"host_ip,omitempty"`
	// HostPort is the IP port from outside the container. This can be different
	// than the ContainerPort because of network forwarding on the host machine.
	HostPort *int `json:"host_port,omitempty"`
}

func (a Address) String() string {
	addrString := fmt.Sprintf("%s:%d", a.ContainerIP, a.ContainerPort)
	if a.HostIP != nil {
		addrString = fmt.Sprintf("%s:%d:%s", *a.HostIP, *a.HostPort, addrString)
	}
	return addrString
}

func (a Address) TargetIP() string {
	if a.HostIP != nil {
		return *a.HostIP
	}
	return a.ContainerIP
}

func (a Address) TargetPort() int {
	if a.HostPort != nil {
		return *a.HostPort
	}
	return a.ContainerPort
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
