package internal

import (
	"context"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/actor"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/proto/pkg/logv1"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetMaster(
	_ context.Context, _ *apiv1.GetMasterRequest) (*apiv1.GetMasterResponse, error) {
	return &apiv1.GetMasterResponse{
		Version:     a.m.Version,
		MasterId:    a.m.MasterID,
		ClusterId:   a.m.ClusterID,
		ClusterName: a.m.config.ClusterName,
	}, nil
}

func (a *apiServer) GetMasterConfig(
	_ context.Context, _ *apiv1.GetMasterConfigRequest) (*apiv1.GetMasterConfigResponse, error) {
	config, err := a.m.config.Printable()
	if err != nil {
		return nil, errors.Wrap(err, "error parsing master config")
	}
	configStruct := &structpb.Struct{}
	err = protojson.Unmarshal(config, configStruct)
	return &apiv1.GetMasterConfigResponse{
		Config: configStruct,
	}, err
}

func (a *apiServer) MasterLogs(
	req *apiv1.MasterLogsRequest, resp apiv1.Determined_MasterLogsServer) error {
	if err := grpc.ValidateRequest(
		grpc.ValidateLimit(req.Limit),
		grpc.ValidateFollow(req.Limit, req.Follow),
	); err != nil {
		return err
	}

	onBatch := func(b api.LogBatch) error {
		return b.ForEach(func(r interface{}) error {
			lr := r.(*logger.Entry)
			return resp.Send(&apiv1.MasterLogsResponse{
				LogEntry: &logv1.LogEntry{Id: int32(lr.ID), Message: lr.Message},
			})
		})
	}

	fetch := func(lr api.LogsRequest) (api.LogBatch, error) {
		if lr.Follow {
			lr.Limit = -1
		}
		return logger.EntriesBatch(a.m.logs.Entries(lr.Offset, -1, lr.Limit)), nil
	}

	total := a.m.logs.Len()
	offset, limit := api.EffectiveOffsetNLimit(int(req.Offset), int(req.Limit), total)
	lReq := api.LogsRequest{Offset: offset, Limit: limit, Follow: req.Follow}

	return a.m.system.MustActorOf(
		actor.Addr("logStore-"+uuid.New().String()),
		api.NewLogStoreProcessor(resp.Context(), lReq, fetch, onBatch, nil, nil),
	).AwaitTermination()
}
