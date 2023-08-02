package grpcutil

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/authz"
)

// UnimplementedError is the error return by API endpoints that are not yet implemented.
var UnimplementedError = status.Error(codes.Unimplemented, "method not yet available")

var fallbackError = errorBody{
	Error: errorMessage{
		Code:    codes.Unknown,
		Message: "failed to marshal error message",
	},
}

type errorBody struct {
	Error errorMessage `json:"error"`
}

type errorMessage struct {
	Code    codes.Code `json:"code"`
	Reason  string     `json:"reason"`
	Message string     `json:"error"`
}

func errorHandler(
	_ context.Context, _ *runtime.ServeMux, m runtime.Marshaler,
	w http.ResponseWriter, _ *http.Request, e error,
) {
	w.Header().Set("Content-type", m.ContentType())

	s := status.Convert(e)
	if s.Code() == codes.Unknown {
		if authz.IsPermissionDenied(e) {
			s = status.New(codes.PermissionDenied, s.Message())
		} else {
			s = status.New(codes.Internal, s.Message())
		}
	}

	response := errorBody{
		Error: errorMessage{
			Code:    s.Code(),
			Reason:  s.Code().String(),
			Message: s.Message(),
		},
	}
	w.WriteHeader(runtime.HTTPStatusFromCode(s.Code()))
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		if err = encoder.Encode(fallbackError); err != nil {
			log.WithError(err).Error("error writing fallback error")
		}
	}
}

// ConnectionIsClosed returns whether the connection has been closed from the client's side.
func ConnectionIsClosed(stream grpc.ServerStream) bool {
	return stream.Context().Err() != nil
}
