package internal

import (
	"context"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/pkg/logger"
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

// TODO update the response format to wrap logv1.LogEntry.
func logToProtoMasterLog(log *logger.Entry) *apiv1.MasterLogsResponse {
	return &apiv1.MasterLogsResponse{Id: int32(log.ID), Message: log.Message}
}

func fetchMasterLogs(logBuffer *logger.LogBuffer) api.LogFetcher {
	return func(req api.LogsRequest) ([]*logger.Entry, error) {
		return logBuffer.Entries(req.Offset, -1, req.Limit), nil
	}
}

func (a *apiServer) MasterLogs(
	req *apiv1.MasterLogsRequest, resp apiv1.Determined_MasterLogsServer) error {
	if err := grpc.ValidateRequest(
		grpc.ValidateLimit(req.Limit),
	); err != nil {
		return err
	}
	total := a.m.logs.Len()
	offset, limit := api.EffectiveOffsetNLimit(int(req.Offset), int(req.Limit), total)

	logRequest := api.LogsRequest{Offset: offset, Limit: limit, Follow: req.Follow}

	onLogEntry := func(log *logger.Entry) error {
		return resp.Send(logToProtoMasterLog(log))
	}

	return api.ProcessLogs(
		resp.Context(),
		logRequest,
		fetchMasterLogs(a.m.logs),
		onLogEntry,
		nil,
	)
}
