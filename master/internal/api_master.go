package internal

import (
	"context"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/plugin/sso"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/version"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/logv1"
)

var masterLogsBatchMissWaitTime = time.Second

func (a *apiServer) GetMaster(
	_ context.Context, _ *apiv1.GetMasterRequest,
) (*apiv1.GetMasterResponse, error) {
	product := apiv1.GetMasterResponse_PRODUCT_UNSPECIFIED
	if a.m.config.InternalConfig.ExternalSessions.Enabled() {
		product = apiv1.GetMasterResponse_PRODUCT_COMMUNITY
	}
	masterResp := &apiv1.GetMasterResponse{
		Version:               version.Version,
		MasterId:              a.m.MasterID,
		ClusterId:             a.m.ClusterID,
		ClusterName:           a.m.config.ClusterName,
		TelemetryEnabled:      a.m.config.Telemetry.Enabled && a.m.config.Telemetry.SegmentWebUIKey != "",
		ExternalLoginUri:      a.m.config.InternalConfig.ExternalSessions.LoginURI,
		ExternalLogoutUri:     a.m.config.InternalConfig.ExternalSessions.LogoutURI,
		Branding:              "determined",
		RbacEnabled:           config.GetAuthZConfig().IsRBACUIEnabled(),
		Product:               product,
		UserManagementEnabled: !a.m.config.InternalConfig.ExternalSessions.Enabled(),
		FeatureSwitches:       a.m.config.FeatureSwitches,
	}
	sso.AddProviderInfoToMasterResponse(a.m.config, masterResp)

	return masterResp, nil
}

func (a *apiServer) GetTelemetry(
	_ context.Context, _ *apiv1.GetTelemetryRequest,
) (*apiv1.GetTelemetryResponse, error) {
	resp := apiv1.GetTelemetryResponse{}
	if a.m.config.Telemetry.Enabled && a.m.config.Telemetry.SegmentWebUIKey != "" {
		resp.Enabled = true
		resp.SegmentKey = a.m.config.Telemetry.SegmentWebUIKey
	}
	return &resp, nil
}

func (a *apiServer) GetMasterConfig(
	ctx context.Context, _ *apiv1.GetMasterConfigRequest,
) (*apiv1.GetMasterConfigResponse, error) {
	if err := userShouldBeAdmin(ctx, a); err != nil {
		return nil, err
	}

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
	req *apiv1.MasterLogsRequest, resp apiv1.Determined_MasterLogsServer,
) error {
	if err := grpcutil.ValidateRequest(
		grpcutil.ValidateLimit(req.Limit),
		grpcutil.ValidateFollow(req.Limit, req.Follow),
	); err != nil {
		return err
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

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	res := make(chan api.BatchResult, 1)
	go api.NewBatchStreamProcessor(
		lReq,
		fetch,
		nil,
		false,
		nil,
		&masterLogsBatchMissWaitTime,
	).Run(ctx, res)

	return processBatches(res, func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			lr := r.(*logger.Entry)
			return resp.Send(&apiv1.MasterLogsResponse{
				LogEntry: &logv1.LogEntry{
					Id:        int32(lr.ID),
					Message:   lr.Message,
					Timestamp: timestamppb.New(lr.Time),
					Level:     logger.LogrusLevelToProto(lr.Level),
				},
			})
		})
	})
}

func (a *apiServer) ResourceAllocationRaw(
	_ context.Context,
	req *apiv1.ResourceAllocationRawRequest,
) (*apiv1.ResourceAllocationRawResponse, error) {
	resp := &apiv1.ResourceAllocationRawResponse{}

	if req.TimestampAfter == nil {
		return nil, status.Error(codes.InvalidArgument, "no start time provided")
	}
	if req.TimestampBefore == nil {
		return nil, status.Error(codes.InvalidArgument, "no end time provided")
	}
	start := time.Unix(req.TimestampAfter.Seconds, int64(req.TimestampAfter.Nanos)).UTC()
	end := time.Unix(req.TimestampBefore.Seconds, int64(req.TimestampBefore.Nanos)).UTC()
	if start.After(end) {
		return nil, status.Error(codes.InvalidArgument, "start time cannot be after end time")
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
	return a.m.fetchAggregatedResourceAllocation(req)
}
