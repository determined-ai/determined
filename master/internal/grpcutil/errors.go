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
)

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
	w.WriteHeader(runtime.HTTPStatusFromCode(status.Code(e)))

	s := status.Convert(e)
	if s.Code() == codes.Unknown {
		s = status.New(codes.Internal, s.Message())
	}

	response := errorBody{
		Error: errorMessage{
			Code:    s.Code(),
			Reason:  s.Code().String(),
			Message: s.Message(),
		},
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(response); err != nil {
		if err = encoder.Encode(fallbackError); err != nil {
			log.WithError(err).Error("error writing fallback error")
		}
	}
}

func ConnectionIsClosed(stream grpc.ServerStream) bool {
	return stream.Context().Err() != nil
}
