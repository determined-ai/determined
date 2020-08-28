package api

import (
	"reflect"

	"github.com/determined-ai/determined/master/pkg/actor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ActorRequest(system *actor.System, addr actor.Address, req actor.Message, v interface{}) error {
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
