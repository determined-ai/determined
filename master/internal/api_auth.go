package internal

import (
	"context"
	"crypto/sha512"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/user"
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
	ctx context.Context, req *apiv1.LoginRequest,
) (*apiv1.LoginResponse, error) {
	fmt.Println("In login")
	fmt.Println(req.Username)
	if a.m.config.InternalConfig.ExternalSessions.JwtKey != "" {
		return nil, status.Error(codes.FailedPrecondition, "authentication is configured to be external")
	}

	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "missing argument: username")
	}

	userModel, err := user.UserByUsername(req.Username)
	switch err {
	case nil:
	case db.ErrNotFound:
		fmt.Println(req.Username)
		fmt.Println("DB ERR not found")
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

	if !userModel.ValidatePassword(hashedPassword) {
		fmt.Println(req.Username)
		fmt.Println("Password is not right")
		return nil, grpcutil.ErrInvalidCredentials
	}

	if !userModel.Active {
		fmt.Println(userModel.Username)
		fmt.Println("user is not active")
		return nil, grpcutil.ErrNotActive
	} else {
		fmt.Println(userModel.Username)
		fmt.Println("user is active")
	}
	token, err := a.m.db.StartUserSession(userModel)
	if err != nil {
		return nil, err
	}
	fullUser, err := getUser(a.m.db, userModel.ID)
	return &apiv1.LoginResponse{Token: token, User: fullUser}, err
}

func (a *apiServer) CurrentUser(
	ctx context.Context, _ *apiv1.CurrentUserRequest,
) (*apiv1.CurrentUserResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	fullUser, err := getUser(a.m.db, user.ID)
	return &apiv1.CurrentUserResponse{User: fullUser}, err
}

func (a *apiServer) Logout(
	ctx context.Context, _ *apiv1.LogoutRequest,
) (*apiv1.LogoutResponse, error) {
	_, userSession, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if userSession == nil {
		return nil, status.Error(codes.InvalidArgument,
			"cannot manually logout of an allocation session")
	}

	err = a.m.db.DeleteUserSessionByID(userSession.ID)
	return &apiv1.LogoutResponse{}, err
}
