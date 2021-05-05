package grpcutil

import (
	"context"
	"crypto/tls"
	"fmt"
	"runtime/debug"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpclogrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	proto "github.com/determined-ai/determined/proto/pkg/apiv1"
)

const jsonPretty = "application/json+pretty"

// NewGRPCServer creates a Determined gRPC service.
func NewGRPCServer(db *db.PgDB, srv proto.DeterminedServer) *grpc.Server {
	// In go-grpc, the INFO log level is used primarily for debugging
	// purposes, so omit INFO messages from the master log.
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	logEntry := logrus.NewEntry(logger)
	grpclogrus.ReplaceGrpcLogger(logEntry)

	opts := []grpclogrus.Option{
		grpclogrus.WithLevels(grpcCodeToLogrusLevel),
	}
	grpcS := grpc.NewServer(
		grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(
			grpclogrus.StreamServerInterceptor(logEntry, opts...),
			grpcrecovery.StreamServerInterceptor(),
			streamAuthInterceptor(db),
		)),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			grpclogrus.UnaryServerInterceptor(logEntry, opts...),
			grpcrecovery.UnaryServerInterceptor(grpcrecovery.WithRecoveryHandler(
				func(p interface{}) (err error) {
					logEntry.Error(string(debug.Stack()))
					return status.Errorf(codes.Internal, "%s", p)
				},
			)),
			unaryAuthInterceptor(db),
		)),
	)
	proto.RegisterDeterminedServer(grpcS, srv)
	return grpcS
}

// newGRPCGatewayMux creates a new gRPC server mux.
func newGRPCGatewayMux() *runtime.ServeMux {
	serverOpts := []runtime.ServeMuxOption{
		runtime.WithMarshalerOption(jsonPretty,
			&runtime.JSONPb{EmitDefaults: true, Indent: "    "}),
		runtime.WithMarshalerOption(runtime.MIMEWildcard,
			&runtime.JSONPb{EmitDefaults: true}),
		runtime.WithProtoErrorHandler(errorHandler),
		runtime.WithForwardResponseOption(userTokenResponse),
	}
	return runtime.NewServeMux(serverOpts...)
}

// RegisterHTTPProxy registers grpc-gateway with the master echo server.
func RegisterHTTPProxy(ctx context.Context, e *echo.Echo, port int, cert *tls.Certificate) error {
	addr := fmt.Sprintf(":%d", port)
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1 << 26)),
	}
	if cert == nil {
		opts = append(opts, grpc.WithInsecure())
	} else {
		// Since this connection is coming directly back to this process, we can skip verification.
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		})))
	}
	mux := newGRPCGatewayMux()
	err := proto.RegisterDeterminedHandlerFromEndpoint(ctx, mux, addr, opts)
	if err != nil {
		return err
	}
	handler := func(c echo.Context) error {
		request := c.Request()
		if c.Request().Header.Get("Authorization") == "" {
			if cookie, err := c.Cookie(cookieName); err == nil {
				request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cookie.Value))
			}
		}
		if _, ok := request.URL.Query()["pretty"]; ok {
			request.Header.Set("Accept", jsonPretty)
		}
		mux.ServeHTTP(c.Response(), request)
		return nil
	}
	apiV1 := e.Group("/api/v1")
	apiV1.Any("/*", handler, middleware.RemoveTrailingSlash())
	return nil
}
