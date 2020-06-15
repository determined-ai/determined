package internal

import (
	"reflect"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

type apiServer struct {
	m *Master
}

func (a *apiServer) pagination(values interface{}, offset, limit int32) (*apiv1.Pagination, error) {
	total := int32(reflect.ValueOf(values).Len())
	startIndex := offset
	if offset < 0 {
		startIndex = total + offset
	}
	endIndex := startIndex + limit
	if limit == 0 || endIndex > total {
		endIndex = total
	}
	err := grpc.ValidateRequest(
		func() (bool, string) { return 0 <= startIndex && startIndex <= total, "offset out of bounds" },
	)
	return &apiv1.Pagination{
		Offset:     offset,
		Limit:      limit,
		StartIndex: startIndex,
		EndIndex:   endIndex,
		Total:      total,
	}, err
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
