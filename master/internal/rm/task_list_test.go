package rm

import (
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestAllocateRequestComparator(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40
	newTime := time.Now()
	oldTime := newTime.Add(-time.Minute * 15)

	agents := []*mockAgent{
		{id: "agent1", slots: 1, maxZeroSlotContainers: 1},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority, weight: 0.5},
		{id: "group2", priority: &higherPriority, weight: 1},
	}
	tasks := []*mockTask{
		{id: "task1", jobID: "job1", group: groups[0], jobSubmissionTime: oldTime},
		{id: "task2", jobID: "job2", group: groups[1], jobSubmissionTime: newTime},
	}

	system := actor.NewSystem(t.Name())
	taskList, _, _ := setupSchedulerStates(t, system, tasks, groups, agents)
	assert.Equal(t, aReqComparator(taskList.taskByID["task1"], taskList.taskByID["task2"]), -1)

	tasks = []*mockTask{
		{id: "task1", jobID: "job1", group: groups[0], jobSubmissionTime: newTime},
		{id: "task2", jobID: "job2", group: groups[1], jobSubmissionTime: oldTime},
	}
	system = actor.NewSystem(t.Name())
	taskList, _, _ = setupSchedulerStates(t, system, tasks, groups, agents)
	assert.Equal(t, aReqComparator(taskList.taskByID["task1"], taskList.taskByID["task2"]), 1)

	tasks = []*mockTask{
		{id: "task1", jobID: "job1", group: groups[0], jobSubmissionTime: newTime},
		{id: "task2", jobID: "job2", group: groups[1], jobSubmissionTime: newTime},
	}
	system = actor.NewSystem(t.Name())
	taskList, _, _ = setupSchedulerStates(t, system, tasks, groups, agents)
	assert.Equal(t, aReqComparator(taskList.taskByID["task1"], taskList.taskByID["task2"]), -1)
}
