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
	syslog               = logrus.WithField("component", "log-retention")
	schedulerDefaultOpts = []gocron.SchedulerOption{gocron.WithLimitConcurrentJobs(1, gocron.LimitModeReschedule)}
)

// Scheduler is a thin wrapper around gocron.Scheduler adds some functionality for testing.
type Scheduler struct {
	sched gocron.Scheduler
	// TestingOnlySynchronizationHelper is used for testing purposes to wait for the log retention scheduler to finish.
	TestingOnlySynchronizationHelper *sync.WaitGroup
}

// NewScheduler creates a new scheduler with the provided options.
func NewScheduler(opts ...gocron.SchedulerOption) (*Scheduler, error) {
	opts = append(schedulerDefaultOpts, opts...)
	s, err := gocron.NewScheduler(opts...)
	if err != nil {
		return nil, err
	}
	return &Scheduler{sched: s}, nil
}

// Schedule begins a log deletion schedule according to the provided LogRetentionPolicy.
func (s *Scheduler) Schedule(config model.LogRetentionPolicy) error {
	// Create a task that deletes expired task logs.
	task := gocron.NewTask(func() {
		defer func() {
			if s.TestingOnlySynchronizationHelper != nil {
				s.TestingOnlySynchronizationHelper.Done()
			}
		}()
		count, err := DeleteExpiredTaskLogs(context.Background(), config.LogRetentionDays)
		if err != nil {
			syslog.WithError(err).Error("failed to delete expired task logs")
		} else if count > 0 {
			syslog.WithField("count", count).Info("deleted expired task logs")
		}
	})
	// If a cleanup schedule is set, schedule the cleanup task.
	if config.Schedule != nil {
		if d, err := time.ParseDuration(*config.Schedule); err == nil {
			// Try to parse out a duration.
			syslog.WithField("duration", d).Debug("running task log cleanup with duration")
			_, err := s.sched.NewJob(gocron.DurationJob(d), task)
			if err != nil {
				return errors.Wrapf(err, "failed to schedule duration task log cleanup")
			}
		} else {
			// Otherwise, use a cron.
			syslog.WithField("cron", *config.Schedule).Debug("running task log cleanup with cron")
			_, err := s.sched.NewJob(gocron.CronJob(*config.Schedule, false), task)
			if err != nil {
				return errors.Wrapf(err, "failed to schedule cron task log cleanup")
			}
		}
	}
	// Start the scheduler.
	s.sched.Start()
	return nil
}

// Shutdown stops the internal gocron.Scheduler.
func (s *Scheduler) Shutdown() error {
	return s.sched.Shutdown()
}

// DeleteExpiredTaskLogs deletes task logs older than days time when defined and non-negative.
// Task configured values may override the default provided number of days for retention.
func DeleteExpiredTaskLogs(ctx context.Context, days *int16) (int64, error) {
	// If days is nil, use the default value of -1 to retain logs forever.
	var defaultLogRetentionDays int16 = retainForever
	if days != nil {
		defaultLogRetentionDays = *days
	}
	syslog.WithField("default-retention-days", defaultLogRetentionDays).Info("deleting expired task logs")
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
	syslog.WithFields(logrus.Fields{"rows": rows, "err": err}).Info("deleted expired task logs")
	return rows, err
}
