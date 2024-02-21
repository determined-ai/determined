package sproto

import (
	"fmt"
	"strings"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
)

// Message protocol from the default resource manager to an agent actor.
type (
	// StartTaskContainer notifies the agent to start the task with the provided task spec.
	StartTaskContainer struct {
		AllocationID model.AllocationID
		TaskID       model.TaskID
		JobID        model.JobID
		aproto.StartContainer

		LogContext logger.Context
	}
	// KillTaskContainer notifies the agent to kill a task container.
	KillTaskContainer struct {
		ContainerID cproto.ID

		LogContext logger.Context
	}
)

// AgentSummary contains information about an agent for external display.
type AgentSummary struct {
	Name   string
	IsIdle bool
}

// ScalingInfo describes the information that is needed for scaling.
type ScalingInfo struct {
	DesiredNewInstances int
	Agents              map[string]AgentSummary
}

// Update updates its desired new instance number and the agent summaries.
func (s *ScalingInfo) Update(desiredNewInstanceNum int, agents map[string]AgentSummary) bool {
	updated := false

	if desiredNewInstanceNum != s.DesiredNewInstances {
		updated = true
	}

	if len(s.Agents) != len(agents) {
		updated = true
	} else {
		for name, agent := range agents {
			previousAgent, ok := s.Agents[name]
			if !ok || previousAgent != agent {
				updated = true
			}
		}
	}

	if updated {
		s.DesiredNewInstances = desiredNewInstanceNum
		s.Agents = agents
	}

	return updated
}

// Constant protocol for the reasons of terminating an instance.
const (
	// TerminateStoppedInstances represents the reason for terminating stopped instances.
	TerminateStoppedInstances = "stopped"
	// TerminateLongDisconnectedInstances represents the reason for terminating long
	// disconnected instances.
	TerminateLongDisconnectedInstances = "long disconnected"
	// TerminateLongIdleInstances represents the reason for terminating long idle instances.
	TerminateLongIdleInstances = "long idle"
	// InstanceNumberExceedsMaximum represents the reason for terminating instances because
	// the instance number exceeding the maximum.
	InstanceNumberExceedsMaximum = "instance number exceeding maximum"
)

// TerminateDecision describes a terminating decision.
type TerminateDecision struct {
	InstanceIDs []string
	Reasons     map[string]string
}

// String returns a representative string.
func (t TerminateDecision) String() string {
	var item []string
	for id, reason := range t.Reasons {
		item = append(item, fmt.Sprintf("%s (reason: %s)", id, reason))
	}
	return strings.Join(item, ",")
}
