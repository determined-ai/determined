package provisioner

import (
	"fmt"
	"strings"
	"time"
)

// instanceType describes an instance type.
type instanceType interface {
	name() string
	slots() int
}

// InstanceState is an enum type that describes an instance state.
type InstanceState string

const (
	// Unknown describes the instance state cannot be recognized.
	Unknown InstanceState = "Unknown"
	// Starting describes the instance is starting up.
	Starting InstanceState = "Starting"
	// Running describes the instance is running.
	Running InstanceState = "Running"
	// Stopping describes the instance is stopping.
	Stopping InstanceState = "Stopping"
	// Stopped describes the instance is stopped.
	Stopped InstanceState = "Stopped"
)

// Instance connects an instance provider's name for a compute resource to the Determined agent name
type Instance struct {
	ID         string
	LaunchTime time.Time
	AgentName  string
	State      InstanceState
}

func (inst Instance) String() string {
	if inst.State == "" {
		return inst.ID
	}
	return fmt.Sprintf("%s (%s)", inst.ID, inst.State)
}

func (inst Instance) equals(other Instance) bool {
	return inst.ID == other.ID && inst.AgentName == other.AgentName && inst.State == other.State
}

func fmtInstances(instances []*Instance) string {
	instanceIDs := make([]string, 0, len(instances))
	for _, inst := range instances {
		instanceIDs = append(instanceIDs, inst.String())
	}
	return strings.Join(instanceIDs, ", ")
}
