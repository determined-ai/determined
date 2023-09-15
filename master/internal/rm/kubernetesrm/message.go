package kubernetesrm

import (
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// Incoming pods actor messages; pods actors must accept these messages.
type (
	// StartTaskPod notifies the pods actor to start a pod with the task spec.
	StartTaskPod struct {
		AllocationID model.AllocationID
		Spec         tasks.TaskSpec
		Slots        int
		Rank         int
		ResourcePool string
		Namespace    string

		LogContext logger.Context
	}
	// KillTaskPod notifies the pods actor to kill a pod.
	KillTaskPod struct {
		PodID cproto.ID
	}

	// PreemptTaskPod notifies the pods actor to preempt a pod.
	PreemptTaskPod struct {
		PodName string
	}

	// ChangePriority notifies the pods actor of a priority change and to resubmit the specified pod.
	ChangePriority struct {
		PodID cproto.ID
	}

	// ChangePosition notifies the pods actor of a position change and to resubmit the specified pod.
	ChangePosition struct {
		PodID cproto.ID
	}
)
