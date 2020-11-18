package resourcemanagers

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestCalculatingDesiredInstanceNum(t *testing.T) {
	system := actor.NewSystem(t.Name())
	taskList := newTaskList()

	// Test basic
	forceAddTask(t, system, taskList, "task1", 1, 1)
	forceAddTask(t, system, taskList, "task2", 0, 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 0), 0)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 1), 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 2), 1)

	// Test increased desired instance number.
	forceAddTask(t, system, taskList, "task3", 0, 1)
	forceAddTask(t, system, taskList, "task4", 1, 1)
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
	forceAddTask(t, system, taskList, "task5", 0, 0)
	forceAddTask(t, system, taskList, "task6", 1, 0)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 0), 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 1), 2)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 2), 1)

	// Test distributed training tasks.
	forceAddTask(t, system, taskList, "task7", 0, 4)
	forceAddTask(t, system, taskList, "task8", 1, 4)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 0), 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 1), 6)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 2), 3)

	// Test unschedulable distributed training tasks.
	forceAddTask(t, system, taskList, "task9", 0, 3)
	forceAddTask(t, system, taskList, "task10", 1, 3)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 0), 1)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 1), 9)
	assert.Equal(t, calculateDesiredNewInstanceNum(taskList, 2), 3)
}
