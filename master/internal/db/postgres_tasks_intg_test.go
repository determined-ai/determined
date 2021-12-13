//go:build integration
// +build integration

package db

import (
	"testing"
	"time"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/test/testutils"
)

func TestTaskLogInsert(t *testing.T) {
	testutils.ResolvePostgres()

	jobID := model.NewJobID()
	if err := db.AddJob(&model.Job{
		JobID:   jobID,
		JobType: model.JobTypeExperiment,
	}); err != nil {
		panic(err)
	}

	startTime := time.Now().UTC()
	var logs []*model.TaskLog
	for _, taskID := range []model.TaskID{model.NewTaskID(), model.NewTaskID()} {
		if err := db.AddTask(&model.Task{
			JobID:     jobID,
			TaskID:    model.NewTaskID(),
			TaskType:  model.TaskTypeTrial,
			StartTime: time.Now(),
		}); err != nil {
			panic(err)
		}

		for i := 0; i < 100; i++ {
			logs = append(logs, &model.TaskLog{
				TaskID:    string(taskID),
				Timestamp: ptrs.TimePtr(startTime.Add(time.Duration(i) * time.Second)),
				Log:       "some bs",
			})
		}
	}

	if err := db.AddTaskLogs(logs); err != nil {
		panic(err)
	}
}
