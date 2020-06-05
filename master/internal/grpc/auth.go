package grpc

import (
	"context"
	"net/http"
	"strings"
	"time"

	// nolint:staticcheck // This is needed until grpc-gateway fully transitions
	// to the new protobuf API.
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	// nolint:gosec // These are not potential hardcoded credentials.
	gatewayTokenHeader = "grpcgateway-authorization"
	userTokenHeader    = "x-user-token"
)

var (
	// ErrInvalidCredentials notifies that the provided credentials are invalid or missing.
	ErrInvalidCredentials = status.Error(codes.Unauthenticated, "invalid credentials")
	// ErrTokenMissing notifies that the bearer token could not be found.
	ErrTokenMissing = status.Error(codes.InvalidArgument, "token missing")
	// ErrPermissionDenied notifies that the user does not have permission to access the method.
	ErrPermissionDenied = status.Error(codes.PermissionDenied, "user does not have permission")
)

// GetUser returns the currently logged in user.
func GetUser(ctx context.Context, d *db.PgDB) (*model.User, *model.UserSession, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil, ErrTokenMissing
	}
	tokens := md[userTokenHeader]
	if len(tokens) == 0 {
		tokens = md[gatewayTokenHeader]
		if len(tokens) == 0 {
			return nil, nil, ErrTokenMissing
		}
	}

	token := tokens[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return nil, nil, ErrInvalidCredentials
	}
	token = strings.TrimPrefix(token, "Bearer ")

	switch user, session, err := d.UserByToken(token); err {
	case nil:
		if !user.Active {
			return nil, nil, ErrPermissionDenied
		}
		return user, session, nil
	case db.ErrNotFound:
		return nil, nil, ErrPermissionDenied
	default:
		return nil, nil, err
	}
}

func authInterceptor(db *db.PgDB) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		if info.FullMethod != "/determined.api.v1.Determined/Login" {
			if _, _, err := GetUser(ctx, db); err != nil {
				return nil, err
			}
		}
		return handler(ctx, req)
	}
}

func userTokenResponse(_ context.Context, w http.ResponseWriter, resp proto.Message) error {
	switch r := resp.(type) {
	case *apiv1.LoginResponse:
		http.SetCookie(w, &http.Cookie{
			Name:    "auth",
			Value:   r.Token,
			Expires: time.Now().Add(db.SessionDuration),
		})
	case *apiv1.LogoutResponse:
		http.SetCookie(w, &http.Cookie{
			Name:    "auth",
			Value:   "",
			Expires: time.Unix(0, 0),
		})
	}
	return nil
}
