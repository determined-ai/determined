package cluster

import (
	"context"

	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
)

// MiscAuthZBasic is basic OSS controls.
type MiscAuthZBasic struct{}

// CanUpdateAgents checks if the user has access to update agents.
func (a *MiscAuthZBasic) CanUpdateAgents(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	if !curUser.Admin {
		return grpcutil.ErrPermissionDenied, nil
	}
	return nil, nil
}

// CanGetSensitiveAgentInfo returns nil and nil error.
func (a *MiscAuthZBasic) CanGetSensitiveAgentInfo(
	ctx context.Context, curUrser *model.User,
) (permErr error, err error) {
	return nil, nil
}

// CanGetMasterLogs returns nil and nil error.
func (a *MiscAuthZBasic) CanGetMasterLogs(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return nil, nil
}

// CanGetMasterConfig checks if user has access to master configs.
func (a *MiscAuthZBasic) CanGetMasterConfig(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	if !curUser.Admin {
		return grpcutil.ErrPermissionDenied, nil
	}
	return nil, nil
}

// CanUpdateMasterConfig checks if user has access to update master configs.
func (a *MiscAuthZBasic) CanUpdateMasterConfig(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	if !curUser.Admin {
		return grpcutil.ErrPermissionDenied, nil
	}
	return nil, nil
}

// CanGetUsageDetails returns nil and nil error.
func (a *MiscAuthZBasic) CanGetUsageDetails(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return nil, nil
}

// CanViewExternalJobs returns nil and nil error.
func (a *MiscAuthZBasic) CanViewExternalJobs(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	return nil, nil
}

func init() {
	AuthZProvider.Register("basic", &MiscAuthZBasic{})
}
