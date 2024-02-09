package stream

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
)

// SubscriptionSet is a set of all subscribers for this PublisherSet.
//
// There is one SubscriptionSet for each websocket connection.  It has one SubscriptionManager per
// streamable type.
type SubscriptionSet struct {
	Projects *subscriptionState[*ProjectMsg, ProjectSubscriptionSpec]
}

// subscriptionState contains per-type subscription state.
type subscriptionState[T stream.Msg, S any] struct {
	Subscription       stream.Subscription[T]
	CollectStartupMsgs CollectStartupMsgsFunc[S]
}

// SubscriptionSpecSet is the set of subscription specs that can be sent in startup message.
type SubscriptionSpecSet struct {
	Projects *ProjectSubscriptionSpec `json:"projects"`
}

// CollectStartupMsgsFunc collects messages that were missed prior to startup.
type CollectStartupMsgsFunc[S any] func(
	ctx context.Context,
	user model.User,
	known string,
	spec S,
) (
	[]stream.MarshallableMsg, error,
)

// NewSubscriptionSet constructor for SubscriptionSet.
func NewSubscriptionSet(
	ctx context.Context,
	streamer *stream.Streamer,
	ps *PublisherSet,
	user model.User,
	spec SubscriptionSpecSet,
) (SubscriptionSet, error) {
	var err error
	var projectSubscriptionState *subscriptionState[*ProjectMsg, ProjectSubscriptionSpec]

	if spec.Projects != nil {
		projectSubscriptionState = &subscriptionState[*ProjectMsg, ProjectSubscriptionSpec]{
			stream.NewSubscription(
				streamer,
				ps.Projects,
				newPermFilter(ctx, user, ProjectMakePermissionFilter, &err),
				newFilter(spec.Projects, ProjectMakeFilter, &err),
			),
			ProjectCollectStartupMsgs,
		}
	}

	return SubscriptionSet{
		Projects: projectSubscriptionState,
	}, err
}

// Startup handles starting up the Subscription objects in the SubscriptionSet.
func (ss *SubscriptionSet) Startup(ctx context.Context, user model.User, startupMsg StartupMsg) (
	[]interface{}, error,
) {
	known := startupMsg.Known
	sub := startupMsg.Subscribe

	var msgs []interface{}
	var err error
	if ss.Projects != nil {
		err = startup(
			ctx, user, &msgs, err,
			ss.Projects, known.Projects,
			sub.Projects, ss.Projects.Subscription.Streamer.PrepareFn,
		)
	}
	return msgs, err
}

// startup performs the streamer startup process,
// - registering a new subscription with it's type-specific publisher
// - collecting historical msgs that occurred prior to the streamers connection.
func startup[T stream.Msg, S any](
	ctx context.Context,
	user model.User,
	msgs *[]interface{},
	err error,
	state *subscriptionState[T, S],
	known string,
	spec *S,
	prepare func(message stream.MarshallableMsg) interface{},
) error {
	if err != nil {
		return err
	}
	if spec == nil {
		// no change
		return nil
	}
	// Sync subscription with publishers.  Do this before initial scan so that we don't
	// miss any events.
	state.Subscription.Register()

	// Scan for historical msgs matching newly-added subscriptions.
	newmsgs, err := state.CollectStartupMsgs(ctx, user, known, *spec)
	if err != nil {
		return echo.ErrCookieNotFound
	}
	for _, msg := range newmsgs {
		*msgs = append(*msgs, prepare(msg))
	}
	return nil
}

// UnregisterAll unregisters all Subscription's in the SubscriptionSet.
func (ss *SubscriptionSet) UnregisterAll() {
	if ss.Projects != nil {
		ss.Projects.Subscription.Unregister()
	}
}
