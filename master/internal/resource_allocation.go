package internal

import (
	"time"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
)

type aggregateTick struct{}

func nextAllocationTime(now time.Time) time.Time {
	target := time.Date(now.Year(), now.Month(), now.Day(), 0, 1, 0, 0, time.UTC)
	if target.Before(now) {
		target = target.AddDate(0, 0, 1)
	}
	return target
}

type allocationAggregator struct {
	db *db.PgDB
}

func (a *allocationAggregator) schedule(ctx *actor.Context) {
	now := time.Now().UTC()
	target := nextAllocationTime(now)
	dt := target.Sub(now)
	ctx.Log().Infof(
		"scheduling next resource allocation aggregation in %s at %s",
		dt.Round(time.Second),
		target,
	)
	actors.NotifyAfter(ctx, dt, aggregateTick{})
}

func (a *allocationAggregator) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart, aggregateTick:
		// Don't return the error, since we want to keep this actor alive and try again next time.
		if err := a.db.UpdateResourceAllocationAggregation(); err != nil {
			ctx.Log().Errorf("failed to aggregate resource allocation: %s", err)
		}
		a.schedule(ctx)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
