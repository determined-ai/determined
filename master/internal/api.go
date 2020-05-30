package internal

import (
	"reflect"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type apiServer struct {
	m *Master
}

func (a *apiServer) actorRequest(addr string, req actor.Message, v interface{}) error {
	actorAddr := actor.Address{}
	if err := actorAddr.UnmarshalText([]byte(addr)); err != nil {
		return status.Errorf(codes.InvalidArgument, "/api/v1%s is not a valid path", addr)
	}
	resp := a.m.system.AskAt(actorAddr, req)
	if resp.Empty() {
		return status.Errorf(codes.NotFound, "/api/v1%s not found", addr)
	}
	if err := resp.Error(); err != nil {
		return err
	}
	reflect.ValueOf(v).Elem().Set(reflect.ValueOf(resp.Get()))
	return resp.Error()
}
