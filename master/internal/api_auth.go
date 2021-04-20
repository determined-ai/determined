package internal

import (
	"context"
	"crypto/sha512"
	"fmt"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const clientSidePasswordSalt = "GubPEmmotfiK9TMD6Zdw" // #nosec G101

// replicateClientSideSaltAndHash replicates the password salt and hash done on the client side.
// We need this because we hash passwords on the client side, but when SCIM posts a user with
// a password to password sync, it doesn't - so when we try to log in later, we get a weird,
// unrecognizable sha512 hash from the frontend.
func replicateClientSideSaltAndHash(password string) string {
	if password == "" {
		return password
	}
	sum := sha512.Sum512([]byte(clientSidePasswordSalt + password))
	return fmt.Sprintf("%x", sum)
}

func (a *apiServer) Login(
	_ context.Context, req *apiv1.LoginRequest) (*apiv1.LoginResponse, error) {
	user, err := a.m.db.UserByUsername(req.Username)
	switch err {
	case nil:
	case db.ErrNotFound:
		return nil, grpcutil.ErrInvalidCredentials
	default:
		return nil, err
	}

	var hashedPassword string
	if req.IsHashed {
		hashedPassword = req.Password
	} else {
		hashedPassword = replicateClientSideSaltAndHash(req.Password)
	}
	if !user.ValidatePassword(hashedPassword) {
		return nil, grpcutil.ErrInvalidCredentials
	}

	if !user.Active {
		return nil, grpcutil.ErrPermissionDenied
	}

	token, err := a.m.db.StartUserSession(user)
	if err != nil {
		return nil, err
	}
	fullUser, err := getUser(a.m.db, user.Username)
	return &apiv1.LoginResponse{Token: token, User: fullUser}, err
}

func (a *apiServer) CurrentUser(
	ctx context.Context, _ *apiv1.CurrentUserRequest) (*apiv1.CurrentUserResponse, error) {
	user, _, err := grpcutil.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, err
	}
	fullUser, err := getUser(a.m.db, user.Username)
	return &apiv1.CurrentUserResponse{User: fullUser}, err
}

func (a *apiServer) Logout(
	ctx context.Context, _ *apiv1.LogoutRequest) (*apiv1.LogoutResponse, error) {
	_, userSession, err := grpcutil.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, err
	}
	err = a.m.db.DeleteUserSessionByID(userSession.ID)
	return &apiv1.LogoutResponse{}, err
}
