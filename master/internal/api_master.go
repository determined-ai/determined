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

// effectiveOffset Returns effective offset.
func effectiveOffset(reqOffset int, totalItems int) (offset int) {
	offset = reqOffset
	if reqOffset < 0 {
		offset = totalItems + offset
		if offset < 0 {
			offset = 0
		}
	}
	return offset
}

// effectiveLimit Returns effective limit.
// Input: Limit 0 is treated as no limit
// Output: Limit -1 is treated as no limit
func effectiveLimit(reqLimit int, offset int, totalItems int) (limit int) {
	limit = -1
	if reqLimit != 0 {
		limit = reqLimit
		if limit > totalItems-offset {
			limit = totalItems - offset
		}
	}
	return limit
}

// TODO could we have this work with generic requests that have offset and limit?
func effectiveOffsetNLimit(reqOffset int, reqLimit int, totalItems int) (offset int, limit int) {
	offset = effectiveOffset(reqOffset, totalItems)
	limit = effectiveLimit(reqLimit, offset, totalItems)
	return offset, limit
}

func (a *apiServer) MasterLogs(
	req *apiv1.MasterLogsRequest, resp apiv1.Determined_MasterLogsServer) error {
	if err := grpc.ValidateRequest(
		grpc.ValidateLimit(req.Limit),
	); err != nil {
		return err
	}
	total := a.m.logs.Len()
	offset, limit := effectiveOffsetNLimit(int(req.Offset), int(req.Limit), total)

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
