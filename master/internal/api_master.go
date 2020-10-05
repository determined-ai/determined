package internal

import (
	"context"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/actor"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/logs"
	"github.com/determined-ai/determined/master/internal/logs/fetchers"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/proto/pkg/logv1"

	"github.com/determined-ai/determined/master/internal/api"
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

	onBatch := func(b logs.Batch) error {
		return b.ForEach(func(r logs.Record) error {
			return resp.Send(&apiv1.MasterLogsResponse{
				LogEntry: logEntryToProtoLogEntry(r.(*logger.Entry)),
			})
		})
	}

	total := a.m.logs.Len()
	offset, limit := api.EffectiveOffsetAndLimit(int(req.Offset), int(req.Limit), total)

	return a.m.system.MustActorOf(
		actor.Addr("logStore-"+uuid.New().String()),
		logs.NewStoreBatchProcessor(
			resp.Context(),
			limit,
			req.Follow,
			fetchers.NewMasterLogsFetcher(a.m.logs, offset),
			onBatch,
			nil, // The only condition that would terminate master logs would terminate us as well, heh.
			nil,
		),
	).AwaitTermination()
}

// logEntryToProtoLogEntry turns a logger.LogEntry into logv1.LogEntry.
func logEntryToProtoLogEntry(logEntry *logger.Entry) *logv1.LogEntry {
	return &logv1.LogEntry{Id: int32(logEntry.ID), Message: logEntry.Message}
}
