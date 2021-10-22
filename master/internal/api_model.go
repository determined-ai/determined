package internal

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

func (a *apiServer) GetModel(
	_ context.Context, req *apiv1.GetModelRequest) (*apiv1.GetModelResponse, error) {
	m := &modelv1.Model{}
	switch err := a.m.db.QueryProto("get_model", m, req.ModelId); err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %d not found", req.ModelId)
	default:
		return &apiv1.GetModelResponse{Model: m},
			errors.Wrapf(err, "error fetching model %d from database", req.ModelId)
	}
}

func (a *apiServer) GetModels(
	_ context.Context, req *apiv1.GetModelsRequest) (*apiv1.GetModelsResponse, error) {
	resp := &apiv1.GetModelsResponse{}
	nameFilterExpr := strings.ToLower(req.Name)
	descFilterExpr := strings.ToLower(req.Description)
	archFilterExpr := ""
	if req.Archived != nil {
		archFilterExpr = strconv.FormatBool(req.Archived.Value)
	}
	userFilterExpr := strings.Join(req.Users, ",")
	labelFilterExpr := strings.Join(req.Labels, ",")
	// Construct the ordering expression.
	sortColMap := map[apiv1.GetModelsRequest_SortBy]string{
		apiv1.GetModelsRequest_SORT_BY_UNSPECIFIED:       "id",
		apiv1.GetModelsRequest_SORT_BY_NAME:              "name",
		apiv1.GetModelsRequest_SORT_BY_DESCRIPTION:       "description",
		apiv1.GetModelsRequest_SORT_BY_CREATION_TIME:     "creation_time",
		apiv1.GetModelsRequest_SORT_BY_LAST_UPDATED_TIME: "last_updated_time",
		apiv1.GetModelsRequest_SORT_BY_NUM_VERSIONS:      "num_versions",
	}
	orderByMap := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC",
	}
	orderExpr := ""
	switch _, ok := sortColMap[req.SortBy]; {
	case !ok:
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	case sortColMap[req.SortBy] != "id": //nolint:goconst // Not actually the same constant.
		orderExpr = fmt.Sprintf(
			"%s %s, id %s",
			sortColMap[req.SortBy], orderByMap[req.OrderBy], orderByMap[req.OrderBy],
		)
	default:
		orderExpr = fmt.Sprintf("id %s", orderByMap[req.OrderBy])
	}
	err := a.m.db.QueryProtof(
		"get_models",
		[]interface{}{orderExpr},
		&resp.Models,
		archFilterExpr,
		userFilterExpr,
		labelFilterExpr,
		nameFilterExpr,
		descFilterExpr,
	)
	if err != nil {
		return nil, err
	}
	return resp, a.paginate(&resp.Pagination, &resp.Models, req.Offset, req.Limit)
}

func (a *apiServer) PostModel(
	ctx context.Context, req *apiv1.PostModelRequest) (*apiv1.PostModelResponse, error) {
	b, err := protojson.Marshal(req.Model.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling model.Metadata")
	}

	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	m := &modelv1.Model{}
	reqLabels := strings.Join(req.Model.Labels, ",")
	err = a.m.db.QueryProto(
		"insert_model", m, req.Model.Name, req.Model.Description, b,
		reqLabels, user.User.Id, time.Now(), time.Now(),
	)

	return &apiv1.PostModelResponse{Model: m},
		errors.Wrapf(err, "error creating model %s in database", req.Model.Name)
}

func (a *apiServer) PatchModel(
	ctx context.Context, req *apiv1.PatchModelRequest) (*apiv1.PatchModelResponse, error) {
	getResp, err := a.GetModel(ctx, &apiv1.GetModelRequest{ModelId: req.Model.Id})
	if err != nil {
		return nil, err
	}

	currModel := getResp.Model
	madeChanges := false

	if req.Model.Description != nil && req.Model.Description.Value != currModel.Description {
		log.Infof("model (%d) description changing from \"%s\" to \"%s\"",
			req.Model.Id, currModel.Description, req.Model.Description)
		madeChanges = true
		currModel.Description = req.Model.Description.Value
	}

	currMeta, err := protojson.Marshal(currModel.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling database model metadata")
	}
	if req.Model.Metadata != nil {
		newMeta, err := protojson.Marshal(req.Model.Metadata)
		if err != nil {
			return nil, errors.Wrap(err, "error marshaling request model metadata")
		}

		if !bytes.Equal(currMeta, newMeta) {
			log.Infof("model (%d) metadata changing from %s to %s",
				req.Model.Id, currMeta, newMeta)
			madeChanges = true
			currMeta = newMeta
		}
	}

	currLabels := strings.Join(currModel.Labels, ",")
	if req.Model.Labels != nil {
		var reqLabelList []string
		for i := 0; i < len(req.Model.Labels); i++ {
			reqLabelList = append(reqLabelList, req.Model.Labels[i].Value)
		}
		reqLabels := strings.Join(reqLabelList, ",")
		if currLabels != reqLabels {
			log.Infof("model (%d) labels changing from %s to %s",
				req.Model.Id, currModel.Labels, req.Model.Labels)
			madeChanges = true
		}
		currLabels = reqLabels
	}

	if !madeChanges {
		return &apiv1.PatchModelResponse{Model: currModel}, nil
	}

	finalModel := &modelv1.Model{}
	err = a.m.db.QueryProto(
		"update_model", finalModel, req.Model.Id, currModel.Description, currMeta, currLabels, time.Now())

	return &apiv1.PatchModelResponse{Model: finalModel},
		errors.Wrapf(err, "error updating model %d in database", req.Model.Id)
}

func (a *apiServer) GetModelVersion(
	_ context.Context, req *apiv1.GetModelVersionRequest) (*apiv1.GetModelVersionResponse, error) {
	resp := &apiv1.GetModelVersionResponse{}
	resp.ModelVersion = &modelv1.ModelVersion{}

	switch err := a.m.db.QueryProto(
		"get_model_version", resp.ModelVersion, req.ModelId, req.ModelVersion); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %s version %d not found", req.ModelId, req.ModelVersion)
	default:
		return resp, err
	}
}

func (a *apiServer) GetModelVersions(
	ctx context.Context, req *apiv1.GetModelVersionsRequest) (*apiv1.GetModelVersionsResponse, error) {
	getResp, err := a.GetModel(ctx, &apiv1.GetModelRequest{ModelId: req.ModelId})
	if err != nil {
		return nil, err
	}

	resp := &apiv1.GetModelVersionsResponse{Model: getResp.Model}
	if err := a.m.db.QueryProto("get_model_versions", &resp.ModelVersions, req.ModelId); err != nil {
		return nil, err
	}

	a.sort(resp.ModelVersions, req.OrderBy, req.SortBy, apiv1.GetModelVersionsRequest_SORT_BY_VERSION)
	return resp, a.paginate(&resp.Pagination, &resp.ModelVersions, req.Offset, req.Limit)
}

func (a *apiServer) PostModelVersion(
	ctx context.Context, req *apiv1.PostModelVersionRequest) (*apiv1.PostModelVersionResponse, error) {
	// make sure that the model exists before adding a version
	getResp, err := a.GetModel(ctx, &apiv1.GetModelRequest{ModelId: req.ModelId})
	if err != nil {
		return nil, err
	}

	// make sure the checkpoint exists
	c := &checkpointv1.Checkpoint{}

	switch getCheckpointErr := a.m.db.QueryProto("get_checkpoint", c, req.CheckpointUuid); {
	case getCheckpointErr == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "checkpoint %s not found", req.CheckpointUuid)
	case getCheckpointErr != nil:
		return nil, getCheckpointErr
	}

	if c.State != checkpointv1.State_STATE_COMPLETED {
		return nil, errors.Errorf(
			"checkpoint %s is in %s state. checkpoints for model versions must be in a COMPLETED state",
			c.Uuid, c.State,
		)
	}

	respModelVersion := &apiv1.PostModelVersionResponse{}
	respModelVersion.ModelVersion = &modelv1.ModelVersion{}

	err = a.m.db.QueryProto(
		"insert_model_version",
		respModelVersion.ModelVersion,
		req.ModelId,
		req.CheckpointUuid,
	)

	respModelVersion.ModelVersion.Model = getResp.Model
	respModelVersion.ModelVersion.Checkpoint = c

	return respModelVersion, errors.Wrapf(err, "error adding model version to model %d", req.ModelId)
}
