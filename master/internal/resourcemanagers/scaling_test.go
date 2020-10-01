package resourcemanagers

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func addTask(
	t *testing.T,
	system *actor.System,
	taskList *taskList,
	taskID string,
	numAllocated int,
	slotsNeeded int,
) {
	ref, created := system.ActorOf(actor.Addr(taskID), &mockTask{system: system})
	assert.Assert(t, created)

	req := &AllocateRequest{
		ID:          TaskID(taskID),
		TaskActor:   ref,
		Group:       ref,
		SlotsNeeded: slotsNeeded,
	}
	taskList.AddTask(req)
	setTaskAllocations(t, taskList, taskID, numAllocated)
}

func setTaskAllocations(
	t *testing.T,
	taskList *taskList,
	taskID string,
	numAllocated int,
) {
	req, ok := taskList.GetTaskByID(TaskID(taskID))
	assert.Check(t, ok)
	if numAllocated > 0 {
		allocated := &ResourcesAllocated{ID: TaskID(taskID), Allocations: []Allocation{}}
		for i := 0; i < numAllocated; i++ {
			allocated.Allocations = append(allocated.Allocations, containerAllocation{})
		}
		taskList.SetAllocations(req.TaskActor, allocated)
	} else {
		taskList.SetAllocations(req.TaskActor, nil)
	}
}

func TestCalculatingDesiredInstanceNum(t *testing.T) {
	system := actor.NewSystem(t.Name())
	taskList := newTaskList()

	// Test basic
	addTask(t, system, taskList, "task1", 1, 1)
	addTask(t, system, taskList, "task2", 0, 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 0), 0)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 1), 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 2), 1)

	// Test increased desired instance number.
	addTask(t, system, taskList, "task3", 0, 1)
	addTask(t, system, taskList, "task4", 1, 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 0), 0)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 1), 2)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 2), 1)

	// Test existing task got allocated/preempted.
	setTaskAllocations(t, taskList, "task3", 0)
	setTaskAllocations(t, taskList, "task4", 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 0), 0)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 1), 2)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 2), 1)

	// Test zero slot tasks.
	addTask(t, system, taskList, "task5", 0, 0)
	addTask(t, system, taskList, "task6", 1, 0)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 0), 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 1), 2)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 2), 1)

	// Test distributed training tasks.
	addTask(t, system, taskList, "task7", 0, 4)
	addTask(t, system, taskList, "task8", 1, 4)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 0), 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 1), 6)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 2), 3)

	// Test unschedulable distributed training tasks.
	addTask(t, system, taskList, "task9", 0, 3)
	addTask(t, system, taskList, "task10", 1, 3)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 0), 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 1), 9)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 2), 3)
}
