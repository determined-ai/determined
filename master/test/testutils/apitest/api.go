package apitest

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc/metadata"

	"github.com/determined-ai/determined/master/internal/user"
)

// WithCredentials returns a logged in context.Context to call gRPC APIs directly with.
func WithCredentials(ctx context.Context) context.Context {
	detUser, err := user.ByUsername(ctx, "determined")
	if err != nil {
		log.Panicln(err)
	}

	token, err := user.StartSession(ctx, detUser)
	if err != nil {
		log.Panicln(err)
	}

	return metadata.NewIncomingContext(
		ctx,
		metadata.Pairs("x-user-token", fmt.Sprintf("Bearer %s", token)),
	)
}
