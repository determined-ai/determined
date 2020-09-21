package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"runtime/debug"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpclogrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	proto "github.com/determined-ai/determined/proto/pkg/apiv1"
)

const jsonPretty = "application/json+pretty"

// NewGRPCServer creates a Determined gRPC service.
func NewGRPCServer(db *db.PgDB, srv proto.DeterminedServer) *grpc.Server {
	logger := logrus.NewEntry(logrus.StandardLogger())
	opts := []grpclogrus.Option{
		grpclogrus.WithLevels(grpcCodeToLogrusLevel),
	}
	grpclogrus.ReplaceGrpcLogger(logger)
	grpcS := grpc.NewServer(
		grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(
			grpclogrus.StreamServerInterceptor(logger, opts...),
			grpcrecovery.StreamServerInterceptor(),
			streamAuthInterceptor(db),
		)),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			grpclogrus.UnaryServerInterceptor(logger, opts...),
			grpcrecovery.UnaryServerInterceptor(grpcrecovery.WithRecoveryHandler(
				func(p interface{}) (err error) {
					logger.Error(string(debug.Stack()))
					return status.Errorf(codes.Internal, "%s", p)
				},
			)),
			unaryAuthInterceptor(db),
		)),
	)
	proto.RegisterDeterminedServer(grpcS, srv)
	return grpcS
}

// NewGRPCMux creates a new gRPC server mux.
func NewGRPCMux() *runtime.ServeMux {
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

func echoToMux(c echo.Context) (*http.Request, *echo.Response) {
	request := c.Request()
	if c.Request().Header.Get("Authorization") == "" {
		if cookie, err := c.Cookie(cookieName); err == nil {
			request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cookie.Value))
		}
	}
	if _, ok := request.URL.Query()["pretty"]; ok {
		request.Header.Set("Accept", jsonPretty)
	}
	return request, c.Response()
}

// PassToGRPCProxy passes an echo server request to gRPC-gateway servermux.
func PassToGRPCProxy(c echo.Context, mux http.Handler) {
	request, response := echoToMux(c)
	mux.ServeHTTP(response, request)
}

// RegisterHTTPProxy registers grpc-gateway with the master echo server.
func RegisterHTTPProxy(
	e *echo.Echo,
	mux *runtime.ServeMux,
	port int,
	enableCORS bool,
	cert *tls.Certificate,
) error {
	addr := fmt.Sprintf(":%d", port)
	var opts []grpc.DialOption
	if cert == nil {
		opts = append(opts, grpc.WithInsecure())
	} else {
		// Since this connection is coming directly back to this process, we can skip verification.
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		})))
	}
	err := proto.RegisterDeterminedHandlerFromEndpoint(context.Background(), mux, addr, opts)
	if err != nil {
		return err
	}
	apiV1 := e.Group("/api/v1")
	apiV1.Any("/*", func(c echo.Context) error {
		if enableCORS {
			api.AddCORSHeader(c)
		}
		PassToGRPCProxy(c, mux)
		return nil
	}, middleware.RemoveTrailingSlash())
	return nil
}
