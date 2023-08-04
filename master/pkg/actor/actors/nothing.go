package actors

import "github.com/determined-ai/determined/master/pkg/actor"

// Nothing is a nothing placeholder actor, mainly for when actors are used as IDs and we haven't
// migrated away from them, yet.
var Nothing = nothing{}

type nothing struct{}

func (*nothing) Receive(ctx *actor.Context) error { return nil }
