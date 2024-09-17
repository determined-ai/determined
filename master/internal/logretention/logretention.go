package logretention

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

const retainForever = -1

var (
	log = logrus.WithField("component", "log-retention")

	schedulerDefaultOpts = []gocron.SchedulerOption{
		gocron.WithLimitConcurrentJobs(1, gocron.LimitModeReschedule),
	}
	scheduler gocron.Scheduler

	// TestingOnlySynchronizationHelper is used for testing purposes to wait for the log retention scheduler to finish.
	TestingOnlySynchronizationHelper *sync.WaitGroup
)

func init() {
	SetupScheduler()
}

// SetupScheduler creates a new scheduler with the provided options.
// Should only be called by init() or test functions, and will panic on error.
func SetupScheduler(opts ...gocron.SchedulerOption) {
	schedulerOpts := append([]gocron.SchedulerOption{}, schedulerDefaultOpts...)
	schedulerOpts = append(schedulerOpts, opts...)

	newScheduler, err := gocron.NewScheduler(schedulerOpts...)
	if err != nil {
		panic(errors.Wrapf(err, "failed to create logretention scheduler"))
	}
	scheduler = newScheduler
}

// Schedule begins a log deletion schedule according to the provided LogRetentionPolicy.
func Schedule(config model.LogRetentionPolicy) error {
	// Create a task that deletes expired task logs.
	task := gocron.NewTask(func() {
		defer func() {
			if TestingOnlySynchronizationHelper != nil {
				TestingOnlySynchronizationHelper.Done()
			}
		}()
		count, err := DeleteExpiredTaskLogs(context.Background(), config.LogRetentionDays)
		if err != nil {
			log.WithError(err).Error("failed to delete expired task logs")
		} else if count > 0 {
			log.WithField("count", count).Info("deleted expired task logs")
		}
	})
	// If a cleanup schedule is set, schedule the cleanup task.
	if config.Schedule != nil {
		if d, err := time.ParseDuration(*config.Schedule); err == nil {
			// Try to parse out a duration.
			log.WithField("duration", d).Debug("running task log cleanup with duration")
			_, err := scheduler.NewJob(gocron.DurationJob(d), task)
			if err != nil {
				return errors.Wrapf(err, "failed to schedule duration task log cleanup")
			}
		} else {
			// Otherwise, use a cron.
			log.WithField("cron", *config.Schedule).Debug("running task log cleanup with cron")
			_, err := scheduler.NewJob(gocron.CronJob(*config.Schedule, false), task)
			if err != nil {
				return errors.Wrapf(err, "failed to schedule cron task log cleanup")
			}
		}
	}
	// Start the scheduler.
	scheduler.Start()
	return nil
}

// DeleteExpiredTaskLogs deletes task logs older than days time when defined and non-negative.
// Task configured values may override the default provided number of days for retention.
func DeleteExpiredTaskLogs(ctx context.Context, days *int16) (int64, error) {
	// If days is nil, use the default value of -1 to retain logs forever.
	var defaultLogRetentionDays int16 = retainForever
	if days != nil {
		defaultLogRetentionDays = *days
	}
	log.WithField("default-retention-days", defaultLogRetentionDays).Info("deleting expired task logs")
	r, err := db.Bun().NewRaw(fmt.Sprintf(`
		WITH log_retention_tasks AS (
			SELECT COALESCE(r.log_retention_days, %d) as log_retention_days, t.task_id, t.end_time
			FROM runs as r
			JOIN run_id_task_id as r_t ON r.id = r_t.run_id
			JOIN tasks as t ON r_t.task_id = t.task_id
			WHERE t.end_time IS NOT NULL
		)
		DELETE FROM task_logs
		WHERE task_id IN (
			SELECT task_id FROM log_retention_tasks
			WHERE log_retention_days >= 0
				AND end_time <= ( retention_timestamp() - make_interval(days => log_retention_days) )
		)
	`, defaultLogRetentionDays)).Exec(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "error deleting expired task logs")
	}
	rows, err := r.RowsAffected()
	log.WithFields(logrus.Fields{"rows": rows, "err": err}).Info("deleted expired task logs")
	return rows, err
}
