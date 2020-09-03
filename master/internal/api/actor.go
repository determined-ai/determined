package api

import (
	"reflect"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// ActorRequest is a helper to ask an actor and populate the response in v.
func ActorRequest(
	system *actor.System,
	addr actor.Address,
	req actor.Message,
	v interface{},
) error {
	resp := system.AskAt(addr, req)
	if resp.Empty() {
		return status.Errorf(codes.NotFound, "/api/v1%s not found", addr)
	}
	if err := resp.Error(); err != nil {
		return err
	}
	reflect.ValueOf(v).Elem().Set(reflect.ValueOf(resp.Get()))
	return resp.Error()
}
