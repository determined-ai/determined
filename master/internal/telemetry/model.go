package telemetry

import "time"

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
	// Terminating is when the instance is in the process of being terminated.
	Terminating InstanceState = "Terminating"
	// SpotRequestPendingAWS indicates that the instance is actually a pending AWS spot request.
	SpotRequestPendingAWS InstanceState = "SpotRequestPendingAWS"
)

// Instance connects a provider's name for a compute resource to the Determined agent name.
// This struct is identical to provisioner.Instance but is used for telemetry to avoid import cycle.
type Instance struct {
	ID                  string
	LaunchTime          time.Time
	LastStateChangeTime time.Time
	AgentName           string
	State               InstanceState
}
