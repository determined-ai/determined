package internal

import (
	"context"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/proto/pkg/logv1"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetMaster(
	_ context.Context, _ *apiv1.GetMasterRequest) (*apiv1.GetMasterResponse, error) {
	return &apiv1.GetMasterResponse{
		Version:          a.m.Version,
		MasterId:         a.m.MasterID,
		ClusterId:        a.m.ClusterID,
		ClusterName:      a.m.config.ClusterName,
		TelemetryEnabled: a.m.config.Telemetry.Enabled && a.m.config.Telemetry.SegmentWebUIKey != "",
	}, nil
}

func (a *apiServer) GetTelemetry(
	_ context.Context, _ *apiv1.GetTelemetryRequest) (*apiv1.GetTelemetryResponse, error) {
	resp := apiv1.GetTelemetryResponse{}
	if a.m.config.Telemetry.Enabled && a.m.config.Telemetry.SegmentWebUIKey != "" {
		resp.Enabled = true
		resp.SegmentKey = a.m.config.Telemetry.SegmentWebUIKey
	}
	return &resp, nil
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
	if err := grpcutil.ValidateRequest(
		grpcutil.ValidateLimit(req.Limit),
		grpcutil.ValidateFollow(req.Limit, req.Follow),
	); err != nil {
		return err
	}

	onBatch := func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			lr := r.(*logger.Entry)
			return resp.Send(&apiv1.MasterLogsResponse{
				LogEntry: &logv1.LogEntry{Id: int32(lr.ID), Message: lr.Message},
			})
		})
	}

	fetch := func(lr api.BatchRequest) (api.Batch, error) {
		if lr.Follow {
			lr.Limit = -1
		}
		return logger.EntriesBatch(a.m.logs.Entries(lr.Offset, -1, lr.Limit)), nil
	}

	total := a.m.logs.Len()
	offset, limit := api.EffectiveOffsetNLimit(int(req.Offset), int(req.Limit), total)
	lReq := api.BatchRequest{Offset: offset, Limit: limit, Follow: req.Follow}

	return api.NewBatchStreamProcessor(
		lReq,
		fetch,
		onBatch,
		nil,
		masterLogsBatchWaitTime,
	).Run(resp.Context())
}

func (a *apiServer) ResourceAllocationRaw(
	_ context.Context,
	req *apiv1.ResourceAllocationRawRequest,
) (*apiv1.ResourceAllocationRawResponse, error) {
	resp := &apiv1.ResourceAllocationRawResponse{}

	if req.TimestampAfter == nil {
		return nil, errors.New("no start time provided")
	}
	if req.TimestampBefore == nil {
		return nil, errors.New("no end time provided")
	}
	start := time.Unix(req.TimestampAfter.Seconds, int64(req.TimestampAfter.Nanos)).UTC()
	end := time.Unix(req.TimestampBefore.Seconds, int64(req.TimestampBefore.Nanos)).UTC()
	if start.After(end) {
		return nil, errors.New("start time cannot be after end time")
	}

	if err := a.m.db.QueryProto(
		"get_raw_allocation", &resp.ResourceEntries, start.UTC(), end.UTC(),
	); err != nil {
		return nil, errors.Wrap(err, "error fetching raw allocation data")
	}

	return resp, nil
}

func (a *apiServer) ResourceAllocationAggregated(
	_ context.Context,
	req *apiv1.ResourceAllocationAggregatedRequest,
) (*apiv1.ResourceAllocationAggregatedResponse, error) {
	resp := &apiv1.ResourceAllocationAggregatedResponse{}

	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, errors.Wrap(err, "invalid start date")
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, errors.Wrap(err, "invalid end date")
	}
	if start.After(end) {
		return nil, errors.New("start date cannot be after end date")
	}

	if err := a.m.db.QueryProto(
		"get_aggregated_allocation", &resp.ResourceEntries, start.UTC(), end.UTC(),
	); err != nil {
		return nil, errors.Wrap(err, "error fetching aggregated allocation data")
	}

	return resp, nil
}
