package sproto

import (
	"fmt"
	"strings"
)

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

// TerminateDecision describes a terminating decision.
type TerminateDecision struct {
	InstanceIDs []string
	Reasons     map[string]string
}

// String returns a representative string.
func (t TerminateDecision) String() string {
	item := make([]string, len(t.Reasons))
	for id, reason := range t.Reasons {
		item = append(item, fmt.Sprintf("%s (reason: %s)", id, reason))
	}
	return strings.Join(item, ",")
}

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
