//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/logretention"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

const pgTimeFormat = "2006-01-02T15:04:05.888738 -07:00:00"

func setRetentionTime(timestamp string) error {
	_, err := db.Bun().NewRaw(fmt.Sprintf(`
	CREATE or REPLACE FUNCTION retention_timestamp() RETURNS TIMESTAMPTZ AS $$ 
    BEGIN
        RETURN %s;
    END 
    $$ LANGUAGE PLPGSQL;
	`, timestamp)).Exec(context.Background())
	return err
}

func quoteSetRetentionTime(timestamp time.Time) error {
	return setRetentionTime(fmt.Sprintf("'%s'", timestamp.Format(pgTimeFormat)))
}

func resetRetentionTime() error {
	return setRetentionTime("transaction_timestamp()")
}

// nolint: exhaustruct
func createTestRetentionExperiment(
	ctx context.Context, t *testing.T, api *apiServer, config string, numTrials int,
) (*experimentv1.Experiment, []int, []model.TaskID) {
	conf := fmt.Sprintf(`
entrypoint: test
checkpoint_storage:
  type: shared_fs
  host_path: /tmp
hyperparameters:
  n_filters1:
    count: 100
    maxval: 100
    minval: 1
    type: int
searcher:
  name: grid
  metric: none
  max_length: %d
  max_concurrent_trials: %d
%s
`, numTrials, numTrials, config)
	createReq := &apiv1.CreateExperimentRequest{
		ModelDefinition: []*utilv1.File{{Content: []byte{1}}},
		Config:          conf,
		ParentId:        0,
		Activate:        true,
		ProjectId:       1,
	}

	// No checkpoint specified anywhere.
	resp, err := api.CreateExperiment(ctx, createReq)
	require.NoError(t, err)
	require.Empty(t, resp.Warnings)
	trialIDs, taskIDs, err := db.ExperimentsTrialAndTaskIDs(ctx, db.Bun(), []int{int(resp.Experiment.Id)})
	require.NoError(t, err)
	return resp.Experiment, trialIDs, taskIDs
}

func TestDeleteExpiredTaskLogs(t *testing.T) {
	// Reset retention time to transaction time on exit.
	defer func() {
		require.NoError(t, resetRetentionTime())
	}()

	api, _, ctx := setupAPITest(t, nil)

	// Clear all logs.
	_, err := db.Bun().NewDelete().Model(&model.TaskLog{}).Where("TRUE").Exec(context.Background())
	require.NoError(t, err)

	// Create an experiment1 with 5 trials and no special config.
	experiment1, trialIDs1, taskIDs1 := createTestRetentionExperiment(ctx, t, api, "", 5)
	require.Nil(t, experiment1.EndTime)
	require.Len(t, trialIDs1, 5)
	require.Len(t, taskIDs1, 5)

	// Create an experiment1 with 5 trials and a config to expire in 1000 days.
	experiment2, trialIDs2, taskIDs2 := createTestRetentionExperiment(ctx, t, api, "log_retention_days: 1000", 5)
	require.Nil(t, experiment2.EndTime)
	require.Len(t, trialIDs2, 5)
	require.Len(t, taskIDs2, 5)

	// Create an experiment1 with 5 trials and config to never expire.
	experiment3, trialIDs3, taskIDs3 := createTestRetentionExperiment(ctx, t, api, "log_retention_days: -1", 5)
	require.Nil(t, experiment3.EndTime)
	require.Len(t, trialIDs3, 5)
	require.Len(t, taskIDs3, 5)

	taskIDs := []model.TaskID{}
	taskIDs = append(taskIDs, taskIDs1...)
	taskIDs = append(taskIDs, taskIDs2...)
	taskIDs = append(taskIDs, taskIDs3...)

	// Add logs for each task.
	for _, taskID := range taskIDs {
		task, err := db.TaskByID(ctx, taskID)
		require.NoError(t, err)
		require.Nil(t, task.EndTime)
		require.NoError(t, api.m.db.AddTaskLogs(
			[]*model.TaskLog{{TaskID: string(taskID), Log: "log1\n"}}))
		require.NoError(t, api.m.db.AddTaskLogs(
			[]*model.TaskLog{{TaskID: string(taskID), Log: "log2\n"}}))
	}

	// Check that the logs are there.
	for _, taskID := range taskIDs {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, logCount, 2)
	}

	// Move time database time 30 days in the future.
	require.NoError(t, quoteSetRetentionTime(time.Now().AddDate(0, 0, 30)))

	// Verify that the logs are still there if we delete with 0 day expiration.
	count, err := logretention.DeleteExpiredTaskLogs(ptrs.Ptr(int16(0)))
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	// Add an end time to the task logs.
	for _, taskID := range taskIDs {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 2, logCount)
		task, err := db.TaskByID(context.Background(), taskID)
		require.NoError(t, err)
		task.EndTime = ptrs.Ptr(time.Now())
		res, err := db.Bun().NewUpdate().Model(task).Where("task_id = ?", taskID).Exec(context.Background())
		require.NoError(t, err)
		rows, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), rows)
	}
	// Verify that the logs are still there if we delete without an expirary.
	count, err = logretention.DeleteExpiredTaskLogs(nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	// Move time database time 100 days in the future.
	require.NoError(t, quoteSetRetentionTime(time.Now().AddDate(0, 0, 100).Add(time.Second)))
	// Verify that the logs are deleted with a 100 day expiration.
	count, err = logretention.DeleteExpiredTaskLogs(ptrs.Ptr(int16(100)))
	require.NoError(t, err)
	require.Equal(t, int64(10), count)

	// Ensure that experiment1 logs are deleted.
	for _, taskID := range taskIDs1 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 0, logCount)
	}
	// Ensure that experiment2 logs are not deleted.
	for _, taskID := range taskIDs2 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 2, logCount)
	}
	// Ensure that experiment3 logs are not deleted.
	for _, taskID := range taskIDs3 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 2, logCount)
	}

	// Move time database time 999 days in the future.
	require.NoError(t, quoteSetRetentionTime(time.Now().AddDate(0, 0, 999).Add(time.Second)))
	// Verify that the logs are not deleted with a 999 day expiration.
	count, err = logretention.DeleteExpiredTaskLogs(ptrs.Ptr(int16(999)))
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	// Move time database time 1000 days in the future.
	require.NoError(t, quoteSetRetentionTime(time.Now().AddDate(0, 0, 1000).Add(time.Second)))
	// Verify that the logs are deleted with a 1000 day expiration.
	count, err = logretention.DeleteExpiredTaskLogs(ptrs.Ptr(int16(1000)))
	require.NoError(t, err)
	require.Equal(t, int64(10), count)

	// Ensure that experiment2 logs are deleted.
	for _, taskID := range taskIDs2 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 0, logCount)
	}
	// Ensure that experiment3 logs are not deleted.
	for _, taskID := range taskIDs3 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 2, logCount)
	}

	// Move time database time 100 years in the future.
	require.NoError(t, quoteSetRetentionTime(time.Now().AddDate(100, 0, 0).Add(time.Second)))
	// Verify that the logs are not deleted with a 0 day expiration.
	count, err = logretention.DeleteExpiredTaskLogs(ptrs.Ptr(int16(0)))
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	// Ensure that experiment3 logs are not deleted.
	for _, taskID := range taskIDs3 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 2, logCount)
	}
}

func countTaskLogs(db *db.PgDB, taskIDs []model.TaskID) (int, error) {
	count := 0
	for _, taskID := range taskIDs {
		logCount, err := db.TaskLogsCount(taskID, nil)
		if err != nil {
			return 0, err
		}
		count += logCount
	}
	return count, nil
}

func incrementScheduler(
	t *testing.T,
	timestamp time.Time,
	fakeClock clockwork.FakeClock,
	days int,
) (time.Time, clockwork.FakeClock) {
	for i := 0; i < days; i++ {
		fakeClock.BlockUntil(1)
		logretention.WaitGroup.Add(1)
		timestamp = timestamp.AddDate(0, 0, 1)
		require.NoError(t, quoteSetRetentionTime(timestamp))
		fakeClock.Advance(timestamp.Sub(fakeClock.Now()))
		logretention.WaitGroup.Wait()
	}
	return timestamp, fakeClock
}

func TestScheduleRetention(t *testing.T) {
	// Reset retention time to transaction time on exit.
	defer func() {
		require.NoError(t, resetRetentionTime())
	}()

	fakeClock := clockwork.NewFakeClock()
	logretention.SetupScheduler(gocron.WithClock(fakeClock))
	logretention.WaitGroup = &sync.WaitGroup{}

	api, _, ctx := setupAPITest(t, nil)

	err := logretention.Schedule(model.LogRetentionPolicy{
		Days:     ptrs.Ptr(int16(100)),
		Schedule: ptrs.Ptr("0 0 * * *"),
	})
	require.NoError(t, err)

	// Clear all logs.
	_, err = db.Bun().NewDelete().Model(&model.TaskLog{}).Where("TRUE").Exec(context.Background())
	require.NoError(t, err)

	// Create an experiment1 with 5 trials and no special config.
	experiment1, trialIDs1, taskIDs1 := createTestRetentionExperiment(ctx, t, api, "", 5)
	require.Nil(t, experiment1.EndTime)
	require.Len(t, trialIDs1, 5)
	require.Len(t, taskIDs1, 5)

	// Create an experiment1 with 5 trials and a config to expire in 1000 days.
	experiment2, trialIDs2, taskIDs2 := createTestRetentionExperiment(ctx, t, api, "log_retention_days: 1000", 5)
	require.Nil(t, experiment2.EndTime)
	require.Len(t, trialIDs2, 5)
	require.Len(t, taskIDs2, 5)

	// Create an experiment1 with 5 trials and config to never expire.
	experiment3, trialIDs3, taskIDs3 := createTestRetentionExperiment(ctx, t, api, "log_retention_days: -1", 5)
	require.Nil(t, experiment3.EndTime)
	require.Len(t, trialIDs3, 5)
	require.Len(t, taskIDs3, 5)

	taskIDs := []model.TaskID{}
	taskIDs = append(taskIDs, taskIDs1...)
	taskIDs = append(taskIDs, taskIDs2...)
	taskIDs = append(taskIDs, taskIDs3...)

	// Add logs for each task.
	for _, taskID := range taskIDs {
		task, err := db.TaskByID(ctx, taskID)
		require.NoError(t, err)
		require.Nil(t, task.EndTime)
		require.NoError(t, api.m.db.AddTaskLogs(
			[]*model.TaskLog{{TaskID: string(taskID), Log: "log1\n"}}))
		require.NoError(t, api.m.db.AddTaskLogs(
			[]*model.TaskLog{{TaskID: string(taskID), Log: "log2\n"}}))
	}

	// Check that the logs are there.
	for _, taskID := range taskIDs {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, logCount, 2)
	}

	// Advance time to midnight.
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	midnight, fakeClock = incrementScheduler(t, midnight, fakeClock, 1)

	// Verify that the logs are still there.
	count, err := countTaskLogs(api.m.db, taskIDs)
	require.NoError(t, err)
	require.Equal(t, 30, count)

	// Add an end time to the task logs.
	for _, taskID := range taskIDs {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 2, logCount)
		task, err := db.TaskByID(context.Background(), taskID)
		require.NoError(t, err)
		task.EndTime = ptrs.Ptr(time.Now())
		res, err := db.Bun().NewUpdate().Model(task).Where("task_id = ?", taskID).Exec(context.Background())
		require.NoError(t, err)
		rows, err := res.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), rows)
	}
	// Advance time by 1 day.
	midnight, fakeClock = incrementScheduler(t, midnight, fakeClock, 1)
	// Verify that the logs are still there.
	count, err = countTaskLogs(api.m.db, taskIDs)
	require.NoError(t, err)
	require.Equal(t, 30, count)

	// Advance time by 98 days.
	midnight, fakeClock = incrementScheduler(t, midnight, fakeClock, 98)
	// Verify that some logs are deleted.
	count, err = countTaskLogs(api.m.db, taskIDs)
	require.NoError(t, err)
	require.Equal(t, 20, count)

	// Ensure that experiment1 logs are deleted.
	for _, taskID := range taskIDs1 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 0, logCount)
	}
	// Ensure that experiment2 logs are not deleted.
	for _, taskID := range taskIDs2 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 2, logCount)
	}
	// Ensure that experiment3 logs are not deleted.
	for _, taskID := range taskIDs3 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 2, logCount)
	}

	// Move time 899 days in the future.
	midnight, fakeClock = incrementScheduler(t, midnight, fakeClock, 899)
	// Verify that no logs are deleted.
	count, err = countTaskLogs(api.m.db, taskIDs)
	require.NoError(t, err)
	require.Equal(t, 20, count)

	// Move time 1 day in the future.
	midnight, fakeClock = incrementScheduler(t, midnight, fakeClock, 1)
	// Verify that no logs are deleted.
	count, err = countTaskLogs(api.m.db, taskIDs)
	require.NoError(t, err)
	require.Equal(t, 10, count)

	// Ensure that experiment2 logs are deleted.
	for _, taskID := range taskIDs2 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 0, logCount)
	}
	// Ensure that experiment3 logs are not deleted.
	for _, taskID := range taskIDs3 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 2, logCount)
	}

	// Move time 10 years in the future.
	_, _ = incrementScheduler(t, midnight, fakeClock, 365*10)
	// Verify that no logs are deleted.
	count, err = countTaskLogs(api.m.db, taskIDs)
	require.NoError(t, err)
	require.Equal(t, 10, count)

	// Ensure that experiment3 logs are not deleted.
	for _, taskID := range taskIDs3 {
		logCount, err := api.m.db.TaskLogsCount(taskID, nil)
		require.NoError(t, err)
		require.Equal(t, 2, logCount)
	}
}
