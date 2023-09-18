package stream

import (
	"context"

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

// WaitForPermissionChange always returns nil.
func (a *StreamAuthZBasic) WaitForPermissionChange() error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &StreamAuthZBasic{})
}
