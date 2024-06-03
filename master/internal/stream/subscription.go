package stream

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
)

// SubscriptionSet is a set of all subscribers for this PublisherSet.
//
// There is one SubscriptionSet for each websocket connection.  It has one SubscriptionManager per
// streamable type.
type SubscriptionSet struct {
	Projects      *subscriptionState[*ProjectMsg, ProjectSubscriptionSpec]
	Models        *subscriptionState[*ModelMsg, ModelSubscriptionSpec]
	ModelVersions *subscriptionState[*ModelVersionMsg, ModelVersionSubscriptionSpec]
	Experiments   *subscriptionState[*ExperimentMsg, ExperimentSubscriptionSpec]
}

// subscriptionState contains per-type subscription state.
type subscriptionState[T stream.Msg, S any] struct {
	Subscription       stream.Subscription[T]
	CollectStartupMsgs CollectStartupMsgsFunc[S]
}

// SubscriptionSpecSet is the set of subscription specs that can be sent in startup message.
type SubscriptionSpecSet struct {
	Projects     *ProjectSubscriptionSpec      `json:"projects"`
	Models       *ModelSubscriptionSpec        `json:"models"`
	ModelVersion *ModelVersionSubscriptionSpec `json:"modelversions"`
	Experiments  *ExperimentSubscriptionSpec   `json:"experiments"`
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
	var modelSubscriptionState *subscriptionState[*ModelMsg, ModelSubscriptionSpec]
	var modelVersionSubscriptionState *subscriptionState[*ModelVersionMsg, ModelVersionSubscriptionSpec]
	var experimentSubscriptionState *subscriptionState[*ExperimentMsg, ExperimentSubscriptionSpec]

	if spec.Projects != nil {
		projectSubscriptionState = &subscriptionState[*ProjectMsg, ProjectSubscriptionSpec]{
			stream.NewSubscription(
				streamer,
				ps.Projects,
				newPermFilter(ctx, user, ProjectMakePermissionFilter, &err),
				newFilter(spec.Projects, ProjectMakeFilter, &err),
				ProjectMakeHydrator(),
			),
			ProjectCollectStartupMsgs,
		}
	}
	if spec.Models != nil {
		modelSubscriptionState = &subscriptionState[*ModelMsg, ModelSubscriptionSpec]{
			stream.NewSubscription(
				streamer,
				ps.Models,
				newPermFilter(ctx, user, ModelMakePermissionFilter, &err),
				newFilter(spec.Models, ModelMakeFilter, &err),
				ModelMakeHydrator(),
			),
			ModelCollectStartupMsgs,
		}
	}
	if spec.ModelVersion != nil {
		modelVersionSubscriptionState = &subscriptionState[*ModelVersionMsg, ModelVersionSubscriptionSpec]{
			stream.NewSubscription(
				streamer,
				ps.ModelVersions,
				newPermFilter(ctx, user, ModelVersionMakePermissionFilter, &err),
				newFilter(spec.ModelVersion, ModelVersionMakeFilter, &err),
				ModelVersionMakeHydrator(),
			),
			ModelVersionCollectStartupMsgs,
		}
	}
	if spec.Experiments != nil {
		experimentSubscriptionState = &subscriptionState[*ExperimentMsg, ExperimentSubscriptionSpec]{
			stream.NewSubscription(
				streamer,
				ps.Experiments,
				newPermFilter(ctx, user, ExperimentMakePermissionFilter, &err),
				newFilter(spec.Experiments, ExperimentMakeFilter, &err),
				ExperimentMakeHydrator(),
			),
			ExperimentCollectStartupMsgs,
		}
	}

	return SubscriptionSet{
		Projects:      projectSubscriptionState,
		Models:        modelSubscriptionState,
		ModelVersions: modelVersionSubscriptionState,
		Experiments:   experimentSubscriptionState,
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
	if ss.Models != nil {
		err = startup(
			ctx, user, &msgs, err,
			ss.Models, known.Models,
			sub.Models, ss.Models.Subscription.Streamer.PrepareFn,
		)
	}
	if ss.ModelVersions != nil {
		err = startup(
			ctx, user, &msgs, err,
			ss.ModelVersions, known.ModelVersions,
			sub.ModelVersion, ss.ModelVersions.Subscription.Streamer.PrepareFn,
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
		return err
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
