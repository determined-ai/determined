package model

import (
	"fmt"
	"strings"
	"time"
)

// Model represents a row from the `models` table.
type Model struct {
	ID              int       `db:"id" json:"id"`
	Name            string    `db:"name" json:"name"`
	Description     string    `db:"description" json:"description"`
	Metadata        JSONObj   `db:"metadata" json:"metadata"`
	CreationTime    time.Time `db:"creation_time" json:"creation_time"`
	LastUpdatedTime time.Time `db:"last_updated_time" json:"last_updated_time"`
	Labels          []string  `db:"labels" json:"labels"`
	Username        string    `db:"username" json:"username"`
	Archived        bool      `db:"archived" json:"archived"`
	NumVersions     int       `db:"num_versions" json:"num_versions"`
}

// ModelVersion represents a row from the `model_versions` table.
type ModelVersion struct {
	ID              int       `db:"id" json:"id"`
	Version         int       `db:"version" json:"version"`
	CheckpointID    int       `db:"checkpoint_id" json:"checkpoint_id"`
	CreationTime    time.Time `db:"creation_time" json:"creation_time"`
	ModelID         int       `db:"model_id" json:"model_id"`
	Metadata        JSONObj   `db:"metadata" json:"metadata"`
	Name            string    `db:"name" json:"name"`
	LastUpdatedTime time.Time `db:"last_updated_time" json:"last_updated_time"`
	Comment         string    `db:"comment" json:"comment"`
	Notes           string    `db:"readme" json:"notes"`
	Username        string    `db:"username" json:"username"`
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
	// Terminating is when the instance is in the process of being terminated.
	Terminating InstanceState = "Terminating"
	// SpotRequestPendingAWS indicates that the instance is actually a pending AWS spot request.
	SpotRequestPendingAWS InstanceState = "SpotRequestPendingAWS"
)

// Instance connects a provider's name for a compute resource to the Determined agent name.
type Instance struct {
	ID                  string
	LaunchTime          time.Time
	LastStateChangeTime time.Time
	AgentName           string
	State               InstanceState
}

// InstanceType describes an instance type.
type InstanceType interface {
	Name() string
	Slots() int
}

func (inst Instance) String() string {
	if inst.State == "" {
		return inst.ID
	}
	return fmt.Sprintf("%s (%s)", inst.ID, inst.State)
}

// Equals checks if this instance is the same resource as instance `other`.
func (inst Instance) Equals(other Instance) bool {
	return inst.ID == other.ID && inst.LaunchTime.Equal(other.LaunchTime) &&
		inst.AgentName == other.AgentName && inst.State == other.State
}

// FmtInstances formats instance ids and states to print.
func FmtInstances(instances []*Instance) string {
	instanceIDs := make([]string, 0, len(instances))
	for _, inst := range instances {
		instanceIDs = append(instanceIDs, inst.String())
	}
	return strings.Join(instanceIDs, ", ")
}
