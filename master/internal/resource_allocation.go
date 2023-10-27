package internal

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
)

func periodicallyAggregateResourceAllocation(db *db.PgDB) {
	for {
		// Don't return the error, since we want to keep this actor alive and try again next time.
		if err := db.UpdateResourceAllocationAggregation(); err != nil {
			logrus.Errorf("failed to aggregate resource allocation: %s", err)
		}
		sleepUntilNextAggregationTime()
	}
}

func sleepUntilNextAggregationTime() {
	now := time.Now().UTC()
	target := nextAggregationTime(now)
	dt := target.Sub(now)
	logrus.Infof(
		"scheduling next resource allocation aggregation in %s at %s",
		dt.Round(time.Second),
		target,
	)
	time.Sleep(dt)
}

func nextAggregationTime(now time.Time) time.Time {
	target := time.Date(now.Year(), now.Month(), now.Day(), 0, 1, 0, 0, time.UTC)
	if target.Before(now) {
		target = target.AddDate(0, 0, 1)
	}
	return target
}
