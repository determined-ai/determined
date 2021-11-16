package internal

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

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

func (a *apiServer) GetModelLabels(
	_ context.Context, req *apiv1.GetModelLabelsRequest) (*apiv1.GetModelLabelsResponse, error) {
	resp := &apiv1.GetModelLabelsResponse{}
	err := a.m.db.QueryProto("get_model_labels", resp)
	if err != nil {
		return nil, err
	}

	return resp, errors.Wrapf(err, "error getting model labels")
}

func (a *apiServer) PostModel(
	ctx context.Context, req *apiv1.PostModelRequest) (*apiv1.PostModelResponse, error) {
	b, err := protojson.Marshal(req.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling model.Metadata")
	}

	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	m := &modelv1.Model{}
	reqLabels := strings.Join(req.Labels, ",")
	err = a.m.db.QueryProto(
		"insert_model", m, req.Name, req.Description, b,
		reqLabels, user.User.Id,
	)

	return &apiv1.PostModelResponse{Model: m},
		errors.Wrapf(err, "error creating model %s in database", req.Name)
}

func (a *apiServer) PatchModel(
	ctx context.Context, req *apiv1.PatchModelRequest) (*apiv1.PatchModelResponse, error) {
	getResp, err := a.GetModel(ctx, &apiv1.GetModelRequest{ModelId: req.ModelId})
	if err != nil {
		return nil, err
	}

	currModel := getResp.Model
	madeChanges := false

	if req.Model.Description != nil && req.Model.Description.Value != currModel.Description {
		log.Infof("model (%d) description changing from \"%s\" to \"%s\"",
			req.ModelId, currModel.Description, req.Model.Description.Value)
		madeChanges = true
		currModel.Description = req.Model.Description.Value
	}

	currMeta, err := protojson.Marshal(currModel.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling database model metadata")
	}
	if req.Model.Metadata != nil {
		newMeta, err2 := protojson.Marshal(req.Model.Metadata)
		if err2 != nil {
			return nil, errors.Wrap(err2, "error marshaling request model metadata")
		}

		if !bytes.Equal(currMeta, newMeta) {
			log.Infof("model (%d) metadata changing from %s to %s",
				req.ModelId, currMeta, newMeta)
			madeChanges = true
			currMeta = newMeta
		}
	}

	currLabels := strings.Join(currModel.Labels, ",")
	if req.Model.Labels != nil {
		var reqLabelList []string
		for _, el := range req.Model.Labels.Values {
			sval := el.String()
			reqLabelList = append(reqLabelList, sval[14:len(sval)-1])
		}
		reqLabels := strings.Join(reqLabelList, ",")
		if currLabels != reqLabels {
			log.Infof("model (%d) labels changing from %s to %s",
				req.ModelId, currModel.Labels, reqLabels)
			madeChanges = true
		}
		currLabels = reqLabels
	}

	if !madeChanges {
		return &apiv1.PatchModelResponse{Model: currModel}, nil
	}

	finalModel := &modelv1.Model{}
	err = a.m.db.QueryProto(
		"update_model", finalModel, req.ModelId, currModel.Description, currMeta, currLabels)

	return &apiv1.PatchModelResponse{Model: finalModel},
		errors.Wrapf(err, "error updating model %d in database", req.ModelId)
}

func (a *apiServer) ArchiveModel(
	ctx context.Context, req *apiv1.ArchiveModelRequest) (*apiv1.ArchiveModelResponse, error) {
	holder := &modelv1.Model{}
	err := a.m.db.QueryProto("archive_model", holder, req.ModelId)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "model %d was not found and cannot be archived",
			req.ModelId)
	}

	return &apiv1.ArchiveModelResponse{},
		errors.Wrapf(err, "error archiving model %d", req.ModelId)
}

func (a *apiServer) UnarchiveModel(
	ctx context.Context, req *apiv1.UnarchiveModelRequest) (*apiv1.UnarchiveModelResponse, error) {
	holder := &modelv1.Model{}
	err := a.m.db.QueryProto("unarchive_model", holder, req.ModelId)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "model %d was not found and cannot be un-archived",
			req.ModelId)
	}

	return &apiv1.UnarchiveModelResponse{},
		errors.Wrapf(err, "error unarchiving model %d", req.ModelId)
}

func (a *apiServer) DeleteModel(
	ctx context.Context, req *apiv1.DeleteModelRequest) (*apiv1.DeleteModelResponse,
	error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	holder := &modelv1.Model{}
	err = a.m.db.QueryProto("delete_model", holder, req.ModelId, user.User.Id,
		user.User.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "model %d does not exist or not delete-able by this user",
			req.ModelId)
	}

	return &apiv1.DeleteModelResponse{},
		errors.Wrapf(err, "error deleting model %d", req.ModelId)
}

func (a *apiServer) GetModelVersion(
	_ context.Context, req *apiv1.GetModelVersionRequest) (*apiv1.GetModelVersionResponse, error) {
	resp := &apiv1.GetModelVersionResponse{}
	resp.ModelVersion = &modelv1.ModelVersion{}

	switch err := a.m.db.QueryProto(
		"get_model_version", resp.ModelVersion, req.ModelId, req.ModelVersion); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %d version %d not found", req.ModelId, req.ModelVersion)
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
	_, err := a.GetModel(ctx, &apiv1.GetModelRequest{ModelId: req.ModelId})
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

	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	respModelVersion := &apiv1.PostModelVersionResponse{}
	respModelVersion.ModelVersion = &modelv1.ModelVersion{}

	mdata, err := protojson.Marshal(req.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling ModelVersion.Metadata")
	}

	reqLabels := strings.Join(req.Labels, ",")

	err = a.m.db.QueryProto(
		"insert_model_version",
		respModelVersion.ModelVersion,
		req.ModelId,
		c.Uuid,
		mdata,
		reqLabels,
		req.Notes,
		user.User.Id,
	)

	return respModelVersion, errors.Wrapf(err, "error adding model version to model %d", req.ModelId)
}

func (a *apiServer) PatchModelVersion(
	ctx context.Context, req *apiv1.PatchModelVersionRequest) (*apiv1.PatchModelVersionResponse,
	error) {
	getResp, err := a.GetModelVersion(ctx,
		&apiv1.GetModelVersionRequest{ModelId: req.ModelId, ModelVersion: req.ModelVersionId})
	if err != nil {
		return nil, err
	}

	currModelVersion := getResp.ModelVersion
	madeChanges := false

	if req.ModelVersion.Name != nil && req.ModelVersion.Name.Value != currModelVersion.Name {
		log.Infof("model version (%d) name changing from \"%s\" to \"%s\"",
			req.ModelVersionId, currModelVersion.Name, req.ModelVersion.Name.Value)
		madeChanges = true
		currModelVersion.Name = req.ModelVersion.Name.Value
	}

	if req.ModelVersion.Comment != nil && req.ModelVersion.Comment.Value != currModelVersion.Comment {
		log.Infof("model version (%d) comment changing from \"%s\" to \"%s\"",
			req.ModelVersionId, currModelVersion.Comment, req.ModelVersion.Comment.Value)
		madeChanges = true
		currModelVersion.Comment = req.ModelVersion.Comment.Value
	}

	if req.ModelVersion.Notes != nil && req.ModelVersion.Notes.Value != currModelVersion.Notes {
		log.Infof("model version (%d) notes changing from \"%s\" to \"%s\"",
			req.ModelVersionId, currModelVersion.Notes, req.ModelVersion.Notes.Value)
		madeChanges = true
		currModelVersion.Notes = req.ModelVersion.Notes.Value
	}

	currMeta, err := protojson.Marshal(currModelVersion.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling database model version metadata")
	}
	if req.ModelVersion.Metadata != nil {
		newMeta, err2 := protojson.Marshal(req.ModelVersion.Metadata)
		if err2 != nil {
			return nil, errors.Wrap(err2, "error marshaling request model version metadata")
		}

		if !bytes.Equal(currMeta, newMeta) {
			log.Infof("model version (%d) metadata changing from %s to %s",
				req.ModelVersionId, currMeta, newMeta)
			madeChanges = true
			currMeta = newMeta
		}
	}

	currLabels := strings.Join(currModelVersion.Labels, ",")
	if req.ModelVersion.Labels != nil {
		var reqLabelList []string
		for _, el := range req.ModelVersion.Labels.Values {
			sval := el.String()
			reqLabelList = append(reqLabelList, sval[14:len(sval)-1])
		}
		reqLabels := strings.Join(reqLabelList, ",")
		if currLabels != reqLabels {
			log.Infof("model version (%d) labels changing from %s to %s",
				req.ModelVersionId, currModelVersion.Labels, reqLabels)
			madeChanges = true
		}
		currLabels = reqLabels
	}

	if !madeChanges {
		return &apiv1.PatchModelVersionResponse{ModelVersion: currModelVersion}, nil
	}

	finalModelVersion := &modelv1.ModelVersion{}
	err = a.m.db.QueryProto("update_model_version", finalModelVersion, req.ModelVersionId,
		req.ModelId, currModelVersion.Name, currModelVersion.Comment, currModelVersion.Notes,
		currMeta, currLabels)

	return &apiv1.PatchModelVersionResponse{ModelVersion: finalModelVersion},
		errors.Wrapf(err, "error updating model version %d in database", req.ModelVersionId)
}

func (a *apiServer) DeleteModelVersion(
	ctx context.Context, req *apiv1.DeleteModelVersionRequest) (*apiv1.DeleteModelVersionResponse,
	error) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	holder := &modelv1.ModelVersion{}
	err = a.m.db.QueryProto("delete_model_version", holder, req.ModelVersionId,
		user.User.Id, user.User.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "model version %d does not exist or not delete-able by this user",
			req.ModelVersionId)
	}

	return &apiv1.DeleteModelVersionResponse{},
		errors.Wrapf(err, "error deleting model version %d", req.ModelVersionId)
}
