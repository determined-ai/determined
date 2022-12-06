package grpcutil

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	// nolint:gosec // These are not potential hardcoded credentials.
	gatewayTokenHeader = "grpcgateway-authorization"
	userTokenHeader    = "x-user-token"
	cookieName         = "auth"
)

type (
	userContextKey        struct{}
	userSessionContextKey struct{}
)

var unauthenticatedMethods = map[string]bool{
	"/determined.api.v1.Determined/Login":        true,
	"/determined.api.v1.Determined/GetMaster":    true,
	"/determined.api.v1.Determined/GetTelemetry": true,
}

var (
	// ErrInvalidCredentials notifies that the provided credentials are invalid or missing.
	ErrInvalidCredentials = status.Error(codes.Unauthenticated, "invalid credentials")
	// ErrTokenMissing notifies that the bearer token could not be found.
	ErrTokenMissing = status.Error(codes.Unauthenticated, "token missing")
	// ErrNotActive notifies that the user is not active.
	ErrNotActive = status.Error(codes.PermissionDenied, "user is not active")
	// ErrPermissionDenied notifies that the user does not have permission to access the method.
	ErrPermissionDenied = status.Error(codes.PermissionDenied, "user does not have permission")
)

// GetUser returns the currently logged in user.
func GetUser(ctx context.Context) (*model.User, *model.UserSession, error) {
	if user, ok := ctx.Value(userContextKey{}).(*model.User); ok {
		if session, ok := ctx.Value(userSessionContextKey{}).(*model.UserSession); ok {
			return user, session, nil // User token cache hit.
		}
		return user, nil, nil // Allocation token cache hit.
	}

	extConfig := config.GetMasterConfig().InternalConfig.ExternalSessions

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil, ErrTokenMissing
	}
	tokens := md[userTokenHeader]
	if len(tokens) == 0 {
		tokens = md[gatewayTokenHeader]
	}

	token := tokens[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return nil, nil, ErrInvalidCredentials
	}
	token = strings.TrimPrefix(token, "Bearer ")

	var userModel *model.User
	var session *model.UserSession
	var err error
	userModel, session, err = user.UserByToken(token, &extConfig)
	switch err {
	case nil:
		if !userModel.Active {
			return nil, nil, ErrPermissionDenied
		}
		return userModel, session, nil
	case sql.ErrNoRows, db.ErrNotFound:
		return nil, nil, ErrInvalidCredentials
	default:
		return nil, nil, err
	}
}

// Return error if user cannot be authenticated or lacks authorization.
func auth(ctx context.Context, db *db.PgDB, fullMethod string,
	extConfig *model.ExternalSessions,
) (*model.User, *model.UserSession, error) {
	if unauthenticatedMethods[fullMethod] {
		return nil, nil, nil
	}

	return GetUser(ctx)
}

func streamAuthInterceptor(db *db.PgDB,
	extConfig *model.ExternalSessions,
) grpc.StreamServerInterceptor {
	return func(
		srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler,
	) error {
		// Don't cache the result of the stream auth interceptor because
		// we can't easily modify ss's context and
		// we would have to worry about the user session expiring in the context.
		_, _, err := auth(ss.Context(), db, info.FullMethod, extConfig)
		fields := log.Fields{"endpoint": info.FullMethod}
		wrappedSS := grpc_middleware.WrappedServerStream{
			ServerStream:   ss,
			WrappedContext: context.WithValue(ss.Context(), audit.LogKey{}, fields),
		}
		if err != nil {
			return err
		}

		return handler(srv, &wrappedSS)
	}
}

func unaryAuthInterceptor(db *db.PgDB,
	extConfig *model.ExternalSessions,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		user, session, err := auth(ctx, db, info.FullMethod, extConfig)
		if err != nil {
			return nil, err
		}
		if user != nil {
			ctx = context.WithValue(ctx, userContextKey{}, user)
		}
		if session != nil {
			ctx = context.WithValue(ctx, userSessionContextKey{}, session)
		}

		return handler(ctx, req)
	}
}

func authZInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		fields := log.Fields{"endpoint": info.FullMethod}
		ctx = context.WithValue(ctx, audit.LogKey{}, fields)

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
