package internal

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"unicode/utf8"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/cluster"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/plugin/sso"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/version"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/logv1"
)

// MaintenanceMessageMaxLength caps the length of a server maintenance message.
const MaintenanceMessageMaxLength = 250

var masterLogsBatchMissWaitTime = time.Second

func (a *apiServer) GetMaster(
	ctx context.Context, _ *apiv1.GetMasterRequest,
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
		StrictJobQueueControl: config.GetAuthZConfig().StrictJobQueueControl,
		Product:               product,
		UserManagementEnabled: !a.m.config.InternalConfig.ExternalSessions.Enabled(),
		FeatureSwitches:       a.m.config.FeatureSwitches,
		MaintenanceMessage:    &apiv1.MaintenanceMessage{},
	}

	query := db.Bun().NewSelect().
		Model(masterResp.MaintenanceMessage).
		Column("id").
		Column("message").
		ColumnExpr("proto_time(start_time) AS start_time").
		ColumnExpr("proto_time(end_time) AS end_time").
		Where("start_time <= NOW()").
		Where("end_time IS NULL OR end_time >= NOW()").
		OrderExpr("maintenance_message.start_time DESC").
		Limit(1)
	err := query.Scan(ctx)
	if err == sql.ErrNoRows {
		masterResp.MaintenanceMessage = nil
	} else if err != nil {
		return nil, errors.Wrap(err, "error fetching server maintenance messages")
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
		return nil, errors.Wrap(err, "error parsing master config")
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
		return nil, errors.Wrap(err, "error fetching raw allocation data")
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

func (a *apiServer) SetMaintenanceMessage(
	ctx context.Context,
	req *apiv1.SetMaintenanceMessageRequest,
) (*apiv1.SetMaintenanceMessageResponse, error) {
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

	if msgLen := utf8.RuneCountInString(req.Message); msgLen > MaintenanceMessageMaxLength {
		return nil, status.Errorf(codes.InvalidArgument,
			"message must be at most %d characters; got %d", MaintenanceMessageMaxLength, msgLen)
	}

	startTime := req.StartTime.AsTime()
	mm := model.MaintenanceMessage{
		CreatorID: int(u.ID),
		Message:   req.Message,
		StartTime: startTime,
	}

	var endTime time.Time
	if req.EndTime != nil {
		endTime = req.EndTime.AsTime()
		if endTime.Before(startTime) {
			return nil, status.Error(codes.InvalidArgument, "end time must be after start time")
		}
		if endTime.Before(time.Now()) {
			return nil, status.Error(codes.InvalidArgument, "end time must be after current time")
		}
		mm.EndTime = &endTime
	}

	err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err = tx.NewUpdate().
			Table("maintenance_messages").
			Set("end_time = NOW()").
			Where("end_time >= NOW() OR end_time IS NULL").
			Exec(ctx)
		if err != nil {
			return errors.Wrap(err, "error clearing previous server maintenance messages")
		}

		_, err = tx.NewInsert().Model(&mm).Exec(ctx)
		if err != nil {
			return errors.Wrap(err, "error setting the server maintenance message")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &apiv1.SetMaintenanceMessageResponse{}, nil
}

func (a *apiServer) DeleteMaintenanceMessage(
	ctx context.Context,
	req *apiv1.DeleteMaintenanceMessageRequest,
) (*apiv1.DeleteMaintenanceMessageResponse, error) {
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

	_, err = db.Bun().NewUpdate().
		Table("maintenance_messages").
		Set("end_time = NOW()").
		Where("end_time >= NOW() OR end_time IS NULL").
		Exec(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error clearing the server maintenance message")
	}

	return &apiv1.DeleteMaintenanceMessageResponse{}, nil
}

func (a *apiServer) DeleteMaintenanceMessage(
	ctx context.Context,
	req *apiv1.DeleteMaintenanceMessageRequest,
) (*apiv1.DeleteMaintenanceMessageResponse, error) {
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

	holder := &apiv1.MaintenanceMessage{}
	if err := a.m.db.QueryProto("delete_maintenance_message", holder, req.Id); err != nil {
		return nil, errors.Wrap(err, "error deleting a server maintenance message")
	}

	return &apiv1.DeleteMaintenanceMessageResponse{}, nil
}
