package stream

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
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

func getStreamableScopes(accessMap map[model.AccessScopeID]bool) (bool, []model.AccessScopeID) {
	_, globalAccess := accessMap[model.GlobalAccessScopeID]
	var accessScopes []model.AccessScopeID
	// only populate accessScopes if user doesn't have global access
	if !globalAccess {
		for id, isPermitted := range accessMap {
			if isPermitted {
				accessScopes = append(accessScopes, id)
			}
		}
	}
	return globalAccess, accessScopes
}

func findExist(ctx context.Context, createFilteredIDQuery func() *bun.SelectQuery, since int64) ([]int64, error) {
	oldEventsQuery := createFilteredIDQuery()
	newEventsQuery := createFilteredIDQuery()
	// get events that happened prior to since that are relevant (appearance)
	oldEventsQuery.Where("seq <= ?", since)
	var exist []int64
	err := oldEventsQuery.Scan(ctx, &exist)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("error when scanning for old offline events: %v\n", err)
		return nil, err
	}
	// and events that happened since the last time this streamer checked
	newEventsQuery.Where("seq > ?", since)
	var newEntities []int64
	err = newEventsQuery.Scan(ctx, &newEntities)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("error when scanning for new offline events: %v\n", err)
		return nil, err
	}

	exist = append(exist, newEntities...)
	slices.Sort(exist)
	return exist, nil
}
