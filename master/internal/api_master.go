package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/cluster"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/license"
	"github.com/determined-ai/determined/master/internal/plugin/sso"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/version"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/logv1"
)

var masterLogsBatchMissWaitTime = time.Second

func (a *apiServer) GetMaster(
	ctx context.Context, _ *apiv1.GetMasterRequest,
) (*apiv1.GetMasterResponse, error) {
	product := apiv1.GetMasterResponse_PRODUCT_UNSPECIFIED
	if a.m.config.InternalConfig.ExternalSessions.Enabled() {
		product = apiv1.GetMasterResponse_PRODUCT_COMMUNITY
	}

	brand := "determined"
	if license.IsEE() {
		brand = "hpe"
	}
	masterResp := &apiv1.GetMasterResponse{
		Version:               version.Version,
		MasterId:              a.m.MasterID,
		ClusterId:             a.m.ClusterID,
		ClusterName:           a.m.config.ClusterName,
		TelemetryEnabled:      a.m.config.Telemetry.Enabled && a.m.config.Telemetry.SegmentWebUIKey != "",
		HasCustomLogo:         a.m.config.UICustomization.HasCustomLogo(),
		ExternalLoginUri:      a.m.config.InternalConfig.ExternalSessions.LoginURI,
		ExternalLogoutUri:     a.m.config.InternalConfig.ExternalSessions.LogoutURI,
		Branding:              brand,
		RbacEnabled:           config.GetAuthZConfig().IsRBACUIEnabled(),
		StrictJobQueueControl: config.GetAuthZConfig().StrictJobQueueControl,
		Product:               product,
		UserManagementEnabled: !a.m.config.InternalConfig.ExternalSessions.Enabled(),
		FeatureSwitches:       a.m.config.FeatureSwitches,
		ClusterMessage:        nil,
	}

	msg, err := db.GetActiveClusterMessage(ctx, db.Bun())
	if err == nil {
		masterResp.ClusterMessage = msg.ToProto()
	} else if err != db.ErrNotFound {
		logrus.WithError(err).Error("error fetching cluster-wide messages")
		return nil, status.Error(codes.Internal, "error fetching cluster-wide messages; check logs for details")
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
	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	permErr, err := cluster.AuthZProvider.Get().CanGetMasterConfig(ctx, u)
	if err != nil {
		return nil, err
	} else if permErr != nil {
		return nil, permErr
	}

	config, err := a.m.config.Printable()
	if err != nil {
		return nil, fmt.Errorf("error parsing master config: %w", err)
	}
	configStruct := &structpb.Struct{}
	err = protojson.Unmarshal(config, configStruct)
	return &apiv1.GetMasterConfigResponse{
		Config: configStruct,
	}, err
}

// This endpoint will only make ephermeral changes to the master config,
// that will be lost if the user restarts the cluster.
func (a *apiServer) PatchMasterConfig(
	ctx context.Context, req *apiv1.PatchMasterConfigRequest,
) (*apiv1.PatchMasterConfigResponse, error) {
	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	permErr, err := cluster.AuthZProvider.Get().CanUpdateMasterConfig(ctx, u)
	if err != nil {
		return nil, err
	} else if permErr != nil {
		return nil, permErr
	}

	paths := req.FieldMask.GetPaths()

	for _, path := range paths {
		switch path {
		case "log.level":
			logger.SetLogrus(a.m.config.Log)
		case "log.color":
			logger.SetLogrus(a.m.config.Log)
		default:
			panic(fmt.Sprintf("unsupported or invalid field: %s", path))
		}
	}
	return &apiv1.PatchMasterConfigResponse{}, err
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

	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}

	permErr, err := cluster.AuthZProvider.Get().CanGetMasterLogs(ctx, u)
	if err != nil {
		return err
	}
	if permErr != nil {
		return status.Error(codes.PermissionDenied, permErr.Error())
	}

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
	ctx context.Context,
	req *apiv1.ResourceAllocationRawRequest,
) (*apiv1.ResourceAllocationRawResponse, error) {
	resp := &apiv1.ResourceAllocationRawResponse{}

	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := a.m.canGetUsageDetails(ctx, u); err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("error fetching raw allocation data: %w", err)
	}

	return resp, nil
}

func (a *apiServer) ResourceAllocationAggregated(
	ctx context.Context,
	req *apiv1.ResourceAllocationAggregatedRequest,
) (*apiv1.ResourceAllocationAggregatedResponse, error) {
	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := a.m.canGetUsageDetails(ctx, u); err != nil {
		return nil, err
	}

	return a.m.fetchAggregatedResourceAllocation(req)
}

func (a *apiServer) GetClusterMessage(
	ctx context.Context,
	req *apiv1.GetClusterMessageRequest,
) (*apiv1.GetClusterMessageResponse, error) {
	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	permErr, err := cluster.AuthZProvider.Get().CanUpdateMasterConfig(ctx, u)
	if err != nil {
		return nil, err
	} else if permErr != nil {
		return nil, permErr
	}

	msg, err := db.GetClusterMessage(ctx, db.Bun())
	if err == db.ErrNotFound {
		return &apiv1.GetClusterMessageResponse{}, nil
	} else if err != nil {
		logrus.WithError(err).Error("error looking up cluster message")
		return nil, status.Error(codes.Internal, "error looking up cluster message; check logs for details")
	}

	return &apiv1.GetClusterMessageResponse{
		ClusterMessage: msg.ToProto(),
	}, nil
}

func (a *apiServer) SetClusterMessage(
	ctx context.Context,
	req *apiv1.SetClusterMessageRequest,
) (*apiv1.SetClusterMessageResponse, error) {
	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	permErr, err := cluster.AuthZProvider.Get().CanUpdateMasterConfig(ctx, u)
	if err != nil {
		return nil, err
	} else if permErr != nil {
		return nil, permErr
	}

	if req.EndTime != nil && req.Duration != nil {
		return nil,
			status.Errorf(codes.InvalidArgument, "EndTime and Duration are mutually exclusive")
	}

	mm := model.ClusterMessage{
		CreatedBy: int(u.ID),
		Message:   req.Message,
		StartTime: req.StartTime.AsTime(),
	}

	if req.EndTime != nil {
		mm.EndTime = sql.NullTime{
			Time:  req.EndTime.AsTime(),
			Valid: true,
		}
	}

	if req.Duration != nil {
		d, err := time.ParseDuration(*req.Duration)
		if err != nil || d < 0 {
			return nil, status.Error(codes.InvalidArgument,
				"Duration must be a Go-formatted duration string with a positive value")
		}

		mm.EndTime = sql.NullTime{
			Time:  req.StartTime.AsTime().Add(d),
			Valid: true,
		}
	}

	err = db.SetClusterMessage(ctx, db.Bun(), mm)
	if errors.Is(err, db.ErrInvalidInput) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	} else if err != nil {
		logrus.WithError(err).Error("error setting cluster message")
		return nil, status.Error(codes.Internal, "error setting cluster message; check logs for details")
	}

	return &apiv1.SetClusterMessageResponse{}, nil
}

func (a *apiServer) DeleteClusterMessage(
	ctx context.Context,
	req *apiv1.DeleteClusterMessageRequest,
) (*apiv1.DeleteClusterMessageResponse, error) {
	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	permErr, err := cluster.AuthZProvider.Get().CanUpdateMasterConfig(ctx, u)
	if err != nil {
		return nil, err
	} else if permErr != nil {
		return nil, permErr
	}

	err = db.ClearClusterMessage(ctx, db.Bun())
	if err != nil {
		logrus.WithError(err).Error("error clearing the cluster message")
		return nil, status.Error(codes.Internal, "error clearing the cluster message; check logs for details")
	}

	return &apiv1.DeleteClusterMessageResponse{}, nil
}
