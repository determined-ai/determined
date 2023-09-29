package stream

import (
	"context"

	"github.com/lib/pq"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// StreamAuthZ is the interface for streaming authorization.
type StreamAuthZ interface {
	// GetTrialStreamableScopes returns a AccessScopeSet where the user has permission to view trials.
	GetTrialStreamableScopes(ctx context.Context, curUser model.User) (model.AccessScopeSet, error)

	// GetPermissionChangeListener returns a pointer listener
	// listening for permission change notifications if applicable.
	GetPermissionChangeListener() (*pq.Listener, error)
}

// AuthZProvider provides StreamAuthZ implementations.
var AuthZProvider authz.AuthZProviderType[StreamAuthZ]
