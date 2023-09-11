package stream

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// StreamAuthZ is the interface for streaming authorization.
type StreamAuthZ interface {
	// GetTrialStreamableScopes returns a AccessScopeSet where the user has permission to view trials.
	GetTrialStreamableScopes(ctx context.Context, curUser model.User) (model.AccessScopeSet, error)
}

// AuthZProvider provides StreamAuthZ implementations.
var AuthZProvider authz.AuthZProviderType[StreamAuthZ]
