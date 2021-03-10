package grpcutil

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
	taskTokenHeader    = "x-task-token"
	userTokenHeader    = "x-user-token"
	cookieName         = "auth"
)

var unauthenticatedMethods = map[string]bool{
	"/determined.api.v1.Determined/Login":        true,
	"/determined.api.v1.Determined/GetMaster":    true,
	"/determined.api.v1.Determined/GetTelemetry": true,
}

var adminMethods = map[string]bool{
	"/determined.api.v1.Determined/DeleteExperiment": true,
}

var (
	// ErrInvalidCredentials notifies that the provided credentials are invalid or missing.
	ErrInvalidCredentials = status.Error(codes.Unauthenticated, "invalid credentials")
	// ErrTokenMissing notifies that the bearer token could not be found.
	ErrTokenMissing = status.Error(codes.InvalidArgument, "token missing")
	// ErrPermissionDenied notifies that the user does not have permission to access the method.
	ErrPermissionDenied = status.Error(codes.PermissionDenied, "user does not have permission")
)

// GetTaskSession returns the currently running task.
func GetTaskSession(ctx context.Context, d *db.PgDB) (*model.TaskSession, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, ErrTokenMissing
	}
	tokens := md[taskTokenHeader]
	if len(tokens) == 0 {
		return nil, ErrTokenMissing
	}

	token := tokens[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return nil, ErrInvalidCredentials
	}
	token = strings.TrimPrefix(token, "Bearer ")

	switch session, err := d.TaskSessionByToken(token); err {
	case nil:
		return session, nil
	case db.ErrNotFound:
		return nil, ErrInvalidCredentials
	default:
		return nil, err
	}
}

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
		return nil, nil, ErrInvalidCredentials
	default:
		return nil, nil, err
	}
}

func streamAuthInterceptor(db *db.PgDB) grpc.StreamServerInterceptor {
	return func(
		srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler,
	) error {
		if _, _, err := GetUser(ss.Context(), db); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func unaryAuthInterceptor(db *db.PgDB) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		if !unauthenticatedMethods[info.FullMethod] {
			if _, err = GetTaskSession(ctx, db); err == ErrTokenMissing {
				switch u, _, uErr := GetUser(ctx, db); {
				case uErr != nil:
					return nil, uErr
				case !u.Admin && adminMethods[info.FullMethod]:
					return nil, ErrPermissionDenied
				}
			} else if err != nil && err != ErrTokenMissing {
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
			Name:    cookieName,
			Value:   r.Token,
			Expires: time.Now().Add(db.SessionDuration),
			Path:    "/",
		})
	case *apiv1.LogoutResponse:
		http.SetCookie(w, &http.Cookie{
			Name:    cookieName,
			Value:   "",
			Expires: time.Unix(0, 0),
		})
	}
	return nil
}
