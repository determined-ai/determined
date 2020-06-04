package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) Login(
	_ context.Context, req *apiv1.LoginRequest) (*apiv1.LoginResponse, error) {
	user, err := a.m.db.UserByUsername(req.Username)
	switch err {
	case nil:
	case db.ErrNotFound:
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	default:
		return nil, err
	}

	if !user.ValidatePassword(req.Password) {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	if !user.Active {
		return nil, status.Error(codes.PermissionDenied, "caller does not have permission")
	}

	token, err := a.m.db.StartUserSession(user)
	return &apiv1.LoginResponse{Token: token}, err
}

func (a *apiServer) CurrentUser(
	ctx context.Context, _ *apiv1.CurrentUserRequest) (*apiv1.CurrentUserResponse, error) {
	_, _, err := grpc.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, err
	}
	return &apiv1.CurrentUserResponse{}, nil
}

func (a *apiServer) Logout(
	ctx context.Context, _ *apiv1.LogoutRequest) (*apiv1.LogoutResponse, error) {
	_, userSession, err := grpc.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, err
	}
	err = a.m.db.DeleteSessionByID(userSession.ID)
	return &apiv1.LogoutResponse{}, err
}
