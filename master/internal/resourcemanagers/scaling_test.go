package resourcemanagers

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestCalculatingDesiredInstanceNum(t *testing.T) {
	system := actor.NewSystem(t.Name())
	taskList := newTaskList()

	// Test one-slot allocated and pending tasks.
	forceAddTask(t, system, taskList, "task1", 1, 1)
	forceAddTask(t, system, taskList, "task2", 0, 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 100), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 100), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 100), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 0), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 0), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 0), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 1), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 1), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 1), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 2), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 2), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 2), 1)

	// Test more one-slot allocated and pending tasks.
	forceAddTask(t, system, taskList, "task3", 0, 1)
	forceAddTask(t, system, taskList, "task4", 1, 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 100), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 100), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 100), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 0), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 0), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 0), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 1), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 1), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 1), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 2), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 2), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 2), 1)

	// Test existing task got allocated/preempted.
	forceSetTaskAllocations(t, taskList, "task3", 1)
	forceSetTaskAllocations(t, taskList, "task4", 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 100), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 100), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 100), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 0), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 0), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 0), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 1), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 1), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 1), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 2), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 2), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 2), 1)

	// Test zero slot tasks.
	forceAddTask(t, system, taskList, "task5", 0, 0)
	forceAddTask(t, system, taskList, "task6", 1, 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 100), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 100), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 100), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 0), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 0), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 0), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 1), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 1), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 1), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 2), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 2), 2)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 2), 1)

	// Test distributed training tasks.
	forceAddTask(t, system, taskList, "task7", 0, 4)
	forceAddTask(t, system, taskList, "task8", 1, 4)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 100), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 100), 6)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 100), 3)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 0), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 0), 6)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 0), 3)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 1), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 1), 6)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 1), 3)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 2), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 2), 6)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 2), 3)

	// Test unschedulable distributed training tasks.
	forceAddTask(t, system, taskList, "task9", 0, 3)
	forceAddTask(t, system, taskList, "task10", 1, 3)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 100), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 100), 9)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 100), 3)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 0), 0)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 0), 9)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 0), 3)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 1), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 1), 9)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 1), 3)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 0, 2), 1)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 1, 2), 9)
	assert.Equal(t, calculateDesiredNewAgentNum(taskList, 2, 2), 3)
}
