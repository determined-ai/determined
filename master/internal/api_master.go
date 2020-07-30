package internal

import (
	"context"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetMaster(
	_ context.Context, _ *apiv1.GetMasterRequest) (*apiv1.GetMasterResponse, error) {
	config, err := a.m.config.Printable()
	if err != nil {
		return nil, errors.Wrap(err, "error parsing master config")
	}
	configStruct := &structpb.Struct{}
	err = protojson.Unmarshal(config, configStruct)
	return &apiv1.GetMasterResponse{
		Version:   a.m.Version,
		MasterId:  a.m.MasterID,
		ClusterId: a.m.ClusterID,
		Config:    configStruct,
	}, err
}

func (a *apiServer) MasterLogs(
	req *apiv1.MasterLogsRequest, resp apiv1.Determined_MasterLogsServer) error {
	if err := grpc.ValidateRequest(
		grpc.ValidateLimit(req.Limit),
	); err != nil {
		return err
	}
	total := a.m.logs.Len()
	offset := int(req.Offset)
	if req.Offset < 0 {
		offset = total + offset
		if offset < 0 {
			offset = 0
		}
	}

	limit := -1
	if req.Limit != 0 {
		limit = int(req.Limit)
		if limit > total-offset {
			limit = total - offset
		}
	}

	for {
		for _, log := range a.m.logs.Entries(offset, -1, limit) {
			offset++
			limit--
			if err := resp.Send(log.Proto()); err != nil {
				return err
			}
		}
		if !req.Follow || limit == 0 {
			return nil
		}
	}
}
