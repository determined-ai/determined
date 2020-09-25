package api

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// ProcessActorResponseError checks actor resposne for errors.
func ProcessActorResponseError(resp *actor.Response) error {
	if (*resp).Empty() {
		src := (*resp).Source()
		msg := "actor not found"
		if src != nil {
			msg = fmt.Sprintf("/api/v1%s not found", src.Address().String())
		}
		return status.Error(codes.NotFound, msg)
	}
	if err := (*resp).Error(); err != nil {
		return err
	}
	return nil
}
