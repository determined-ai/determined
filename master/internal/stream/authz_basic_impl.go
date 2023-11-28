package stream

import (
	"context"

	"github.com/lib/pq"

	"github.com/determined-ai/determined/master/pkg/model"
)

// StreamAuthZBasic is classic OSS Determined authentication for streaming clients.
type StreamAuthZBasic struct{}

// GetTrialStreamableScopes always returns an AccessScopeSet with global permission and a nil error.
func (a *StreamAuthZBasic) GetTrialStreamableScopes(
	ctx context.Context,
	curUser model.User,
) (model.AccessScopeSet, error) {
	return model.AccessScopeSet{model.GlobalAccessScopeID: true}, nil
}

// GetMetricStreamableScopes always returns an AccessScopeSet
// with global permission and a nil error.
func (a *StreamAuthZBasic) GetMetricStreamableScopes(
	ctx context.Context,
	curUser model.User,
) (model.AccessScopeSet, error) {
	return model.AccessScopeSet{model.GlobalAccessScopeID: true}, nil
}

// GetExperimentStreamableScopes always returns an AccessScopeSet with global permission and a nil error.
func (a *StreamAuthZBasic) GetExperimentStreamableScopes(
	ctx context.Context,
	curUser model.User,
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
