package stream

import (
	"context"
	"time"

	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
)

const (
	minReconn = 1 * time.Second
	maxReconn = 10 * time.Second
)

// permFilterQuery adds a filter to the provided bun query to filter for workspaces the user has
// access to.
func permFilterQuery(
	q *bun.SelectQuery, accessScopes []model.AccessScopeID,
) *bun.SelectQuery {
	return q.Where("workspace_id in (?)", bun.In(accessScopes))
}

// newDBListener creates a new default pq.Listener for streaming updates.
func newDBListener(dbAddress, channel string) (*pq.Listener, error) {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Errorf("listener on (%s) reported problem: %s", dbAddress, err.Error())
		}
	}
	listener := pq.NewListener(
		dbAddress,
		minReconn,
		maxReconn,
		reportProblem,
	)
	err := listener.Listen(channel)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

// newPermFilter creates a new permission filter based on the provided user and filter function
// for populating the streamer filter field.
func newPermFilter[T stream.Msg](
	ctx context.Context,
	user model.User,
	permFilterFn func(context.Context, model.User) (func(T) bool, error),
	err *error,
) func(T) bool {
	if *err != nil {
		return nil
	}
	out, tempErr := permFilterFn(ctx, user)
	if tempErr != nil {
		*err = tempErr
		return nil
	}
	return out
}

// newFilter creates a filter function based on the provided SubscriptionSpec
// and the streamable type-specific filter constructor.
func newFilter[S any, T stream.Msg](
	spec S,
	filterConstructor func(S) (func(T) bool, error),
	err *error,
) func(T) bool {
	if *err != nil {
		return nil
	}
	out, tempErr := filterConstructor(spec)
	if tempErr != nil {
		*err = tempErr
		return nil
	}
	return out
}
