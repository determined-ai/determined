package grpc

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	userTokenHeader = "grpcgateway-authorization"
)

var errInvalidCredentials = status.Error(codes.Unauthenticated, "invalid credentials")
var errTokenMissing = status.Error(codes.InvalidArgument, "token missing")

// GetUser returns the currently logged in user.
func GetUser(ctx context.Context, d *db.PgDB) (*model.User, *model.UserSession, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil, errTokenMissing
	}
	tokens := md[userTokenHeader]
	if len(tokens) == 0 {
		return nil, nil, errTokenMissing
	}

	token := tokens[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return nil, nil, errInvalidCredentials
	}
	token = strings.TrimPrefix(token, "Bearer ")

	switch user, session, err := d.UserByToken(token); err {
	case nil:
		if !user.Active {
			return nil, nil, errInvalidCredentials
		}
		return user, session, nil
	case db.ErrNotFound:
		return nil, nil, errInvalidCredentials
	default:
		return nil, nil, err
	}
}

func authInterceptor(db *db.PgDB) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		if info.FullMethod == "/determined.api.v1.Determined/Login" {
			resp, err := handler(ctx, req)
			if err != nil {
				return nil, err
			}
			login := resp.(*apiv1.LoginResponse)
			err = grpc.SendHeader(ctx, metadata.Pairs(userTokenHeader, login.Token))
			return resp, err
		}
		if _, _, err := GetUser(ctx, db); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func userTokenMatcher(key string) (string, bool) {
	switch key {
	case userTokenHeader:
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
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
