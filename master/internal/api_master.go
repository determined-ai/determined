package internal

import (
	"context"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const logCheckWaitTime = 100 * time.Millisecond

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

// effectiveOffset returns effective offset.
func effectiveOffset(reqOffset int, total int) (offset int) {
	switch {
	case reqOffset < -total:
		return 0
	case reqOffset < 0:
		return total + reqOffset
	default:
		return reqOffset
	}
}

// effectiveLimit returns effective limit.
// Input: non-negative offset and limit.
func effectiveLimit(limit int, offset int, total int) int {
	switch {
	case limit == 0:
		return -1
	case limit > total-offset:
		return total - offset
	default:
		return limit
	}
}

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
		logEnties := a.m.logs.Entries(offset, -1, limit)
		for _, log := range logEnties {
			offset++
			limit--
			if err := resp.Send(log.Proto()); err != nil {
				return err
			}
		}
		if len(logEnties) == 0 {
			time.Sleep(logCheckWaitTime)
		}
		if !req.Follow || limit == 0 {
			return nil
		} else if req.Follow {
			limit = -1
		}
		if err := resp.Context().Err(); err != nil {
			// context is closed
			return nil
		}
	}
}
