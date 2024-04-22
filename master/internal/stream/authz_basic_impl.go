package stream

import (
	"context"

	"github.com/lib/pq"

	"github.com/determined-ai/determined/master/pkg/model"
)

// StreamAuthZBasic is classic OSS Determined authentication for streaming clients.
type StreamAuthZBasic struct{}

// GetProjectStreamableScopes always returns an AccessScopeSet with global permissions and a nil error.
func (a *StreamAuthZBasic) GetProjectStreamableScopes(
	_ context.Context,
	_ model.User,
) (model.AccessScopeSet, error) {
	return model.AccessScopeSet{model.GlobalAccessScopeID: true}, nil
}

// GetModelStreamableScopes always returns an AccessScopeSet with global permissions and a nil error.
func (a *StreamAuthZBasic) GetModelStreamableScopes(
	_ context.Context,
	_ model.User,
) (model.AccessScopeSet, error) {
	return model.AccessScopeSet{model.GlobalAccessScopeID: true}, nil
}

// GetModelVersionStreamableScopes always returns an AccessScopeSet with global permissions and a nil error.
func (a *StreamAuthZBasic) GetModelVersionStreamableScopes(
	_ context.Context,
	_ model.User,
) (model.AccessScopeSet, error) {
	return model.AccessScopeSet{model.GlobalAccessScopeID: true}, nil
}

// GetPermissionChangeListener always returns a nil pointer and a nil error.
func (a *StreamAuthZBasic) GetPermissionChangeListener() (*pq.Listener, error) {
	return nil, nil
}

func init() {
	AuthZProvider.Register("basic", &StreamAuthZBasic{})
}
