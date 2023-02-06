package internal

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	modelauth "github.com/determined-ai/determined/master/internal/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"

	structpb "github.com/golang/protobuf/ptypes/struct"
)

func (a *apiServer) ModelFromIdentifier(ctx context.Context, identifier string) (*modelv1.Model, error) {
	var err error
	m := &modelv1.Model{}

	query := db.Bun().NewSelect().
		Model(m).
		ModelTableExpr("models as m").
		Column("m.id").
		Column("m.name").
		Column("m.description").
		Column("m.notes").
		Column("m.metadata").
		Column("m.user_id").
		Column("m.workspace_id").
		Column("m.archived").
		ColumnExpr("array_to_json(m.labels) AS labels").
		ColumnExpr("proto_time(m.last_updated_time) AS last_updated_time").
		ColumnExpr("proto_time(m.creation_time) AS creation_time").
		Column("u.username").
		ColumnExpr("COUNT(mv.version) as num_versions").
		Join("LEFT JOIN model_versions as mv ON mv.model_id = m.id").
		Join("LEFT JOIN users as u ON u.id = m.user_id").
		GroupExpr("m.id").
		GroupExpr("u.id")

	allNumbers, _ := regexp.MatchString("^\\d+$", identifier)
	if allNumbers {
		query = query.Where("m.id = ?", identifier)
	} else {
		query = query.Where("m.name = ?", identifier)
	}
	err = query.Scan(ctx)

	switch err {
	case sql.ErrNoRows:
		return nil, status.Errorf(
			codes.NotFound, "model %q not found", identifier)
	default:
		return m, errors.Wrapf(err,
			"error fetching model %q from database", identifier)
	}
}

func (a *apiServer) ModelVersionFromID(ctx context.Context, modelIdentifier string,
	versionID int32,
) (*modelv1.ModelVersion, error) {
	mv := &modelv1.ModelVersion{}
	parentModel, err := a.ModelFromIdentifier(ctx, modelIdentifier)
	if err != nil {
		return nil, err
	}

	switch err = a.m.db.QueryProto(
		"get_model_version", mv, parentModel.Id, versionID); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "version %v for model %q not found", versionID, modelIdentifier)
	default:
		return mv, errors.Wrapf(err,
			"error fetching version %v for model %q from database", versionID, modelIdentifier)
	}
}

func (a *apiServer) GetModel(
	ctx context.Context, req *apiv1.GetModelRequest,
) (*apiv1.GetModelResponse, error) {
	m, err := a.ModelFromIdentifier(ctx, req.ModelName)
	if err != nil {
		return nil, err
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if ok, err := modelauth.AuthZProvider.Get().CanGetModel(ctx, *curUser, m,
		m.WorkspaceId); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.Errorf("current user %q doesn't have permissions to get model %q.",
			curUser.Username, m.Name)
	}
	return &apiv1.GetModelResponse{Model: m}, err
}

func (a *apiServer) GetModels(
	ctx context.Context, req *apiv1.GetModelsRequest,
) (*apiv1.GetModelsResponse, error) {
	resp := &apiv1.GetModelsResponse{}

	query := db.Bun().NewSelect().
		Model(&resp.Models).
		ModelTableExpr("models as m").
		Column("m.id").
		Column("m.name").
		Column("m.description").
		Column("m.notes").
		Column("m.metadata").
		Column("m.user_id").
		Column("m.workspace_id").
		Column("m.archived").
		ColumnExpr("array_to_json(m.labels) AS labels").
		ColumnExpr("proto_time(m.last_updated_time) AS last_updated_time").
		ColumnExpr("proto_time(m.creation_time) AS creation_time").
		Column("u.username").
		ColumnExpr("COUNT(mv.version) as num_versions").
		Join("LEFT JOIN model_versions as mv ON mv.model_id = m.id").
		Join("LEFT JOIN users as u ON u.id = m.user_id").
		GroupExpr("m.id").
		GroupExpr("u.id")

	if req.Id != 0 {
		query = query.Where("m.id = ?", req.Id)
	}
	if req.Archived != nil {
		query = query.Where("m.archived = ?", req.Archived.Value)
	}
	if len(req.Users) > 0 {
		query = query.Where("u.username IN (?)", bun.In(req.Users))
	}
	if len(req.UserIds) > 0 {
		query = query.Where("m.user_id IN (?)", bun.In(req.UserIds))
	}
	if len(req.Labels) > 0 {
		query = query.Where(
			`string_to_array(?, ',') <@ ARRAY(SELECT jsonb_array_elements_text(m.labels))`,
			strings.Join(req.Labels, ","))
	}
	if req.Name != "" {
		query = query.Where("m.name ILIKE ('%%' || ? || '%%')", req.Name)
	}
	if req.Description != "" {
		query = query.Where("m.description ILIKE ('%%' || ? || '%%')", req.Description)
	}

	// get current user (needed for permissions)
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	// filter on workspace name or ID
	workspaceIDFilterExpr := 0
	if req.WorkspaceId != nil { // default is to use workspace ID
		workspaceIDFilterExpr = int(*req.WorkspaceId)
	} else if req.WorkspaceName != nil {
		w := workspacev1.Workspace{}
		err := a.m.db.Query("get_workspace_from_name", &w, *req.WorkspaceName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get workspace %v", *req.WorkspaceName)
		}
		workspaceIDFilterExpr = int(w.Id)
	}
	if workspaceIDFilterExpr != 0 {
		// if workspace id isn't provided then no auth is required here. All models will be returned.
		if ok, err := modelauth.AuthZProvider.Get().CanGetModels(ctx, *curUser,
			int32(workspaceIDFilterExpr)); err != nil {
			return nil, err
		} else if !ok {
			return nil, errors.Errorf(
				"current user %q doesn't have view permissions in the given workspace with id: %v.",
				curUser.Username, workspaceIDFilterExpr)
		}
		query = query.Where("m.workspace_id = ?", workspaceIDFilterExpr)
	}

	// Construct the ordering expression.
	sortColMap := map[apiv1.GetModelsRequest_SortBy]string{
		apiv1.GetModelsRequest_SORT_BY_UNSPECIFIED:       "id",
		apiv1.GetModelsRequest_SORT_BY_NAME:              "name",
		apiv1.GetModelsRequest_SORT_BY_DESCRIPTION:       "description",
		apiv1.GetModelsRequest_SORT_BY_CREATION_TIME:     "m.creation_time",
		apiv1.GetModelsRequest_SORT_BY_LAST_UPDATED_TIME: "m.last_updated_time",
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
	case sortColMap[req.SortBy] != "id":
		orderExpr = fmt.Sprintf(
			"%s %s, id %s",
			sortColMap[req.SortBy], orderByMap[req.OrderBy], orderByMap[req.OrderBy],
		)
	default:
		orderExpr = fmt.Sprintf("id %s", orderByMap[req.OrderBy])
	}
	query = query.OrderExpr(orderExpr)

	// if requested editable models, filter through authz
	if req.Editable == true {
		if query, err = modelauth.AuthZProvider.Get().
			FilterEditableModelsQuery(ctx, *curUser, query); err != nil {
			return nil, err
		}
	}

	resp.Pagination, err = runPagedBunExperimentsQuery(ctx, query, int(req.Offset), int(req.Limit))
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *apiServer) GetModelLabels(
	_ context.Context, req *apiv1.GetModelLabelsRequest,
) (*apiv1.GetModelLabelsResponse, error) {
	resp := &apiv1.GetModelLabelsResponse{}
	err := a.m.db.QueryProto("get_model_labels", resp)
	if err != nil {
		return nil, err
	}

	return resp, errors.Wrapf(err, "error getting model labels")
}

func (a *apiServer) clearModelName(ctx context.Context, modelName string) error {
	if len(strings.ReplaceAll(modelName, " ", "")) == 0 {
		return status.Errorf(codes.InvalidArgument, "model names cannot be blank")
	}
	if strings.Contains(modelName, "  ") {
		return status.Errorf(codes.InvalidArgument, "model names cannot have excessive spacing")
	}
	if strings.Contains(modelName, "/") || strings.Contains(modelName, "\\") {
		return status.Errorf(codes.InvalidArgument, "model names cannot have slashes")
	}
	re := regexp.MustCompile(`^\d+$`)
	if len(re.FindAllString(modelName, 1)) > 0 {
		return status.Errorf(codes.InvalidArgument, "model names cannot be only numbers")
	}

	getResp := &apiv1.GetModelsResponse{}
	_ = db.Bun().NewSelect().
		Model(&getResp.Models).
		ModelTableExpr("models as m").
		Column("m.id").
		Where("m.name ILIKE ('%%' || ? || '%%')", modelName).
		Scan(ctx)

	if len(getResp.Models) > 0 {
		return status.Errorf(codes.AlreadyExists, "avoid names equal to other models (case-insensitive)")
	}
	return nil
}

func (a *apiServer) PostModel(
	ctx context.Context, req *apiv1.PostModelRequest,
) (*apiv1.PostModelResponse, error) {
	if err := a.clearModelName(ctx, req.Name); err != nil {
		return nil, err
	}

	b, err := protojson.Marshal(req.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling model.Metadata")
	}

	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}
	workspaceID := 1
	if req.WorkspaceId != nil { // default is to use workspace ID
		workspaceID = int(*req.WorkspaceId)
	} else if req.WorkspaceName != nil {
		w := workspacev1.Workspace{}
		err := a.m.db.Query("get_workspace_from_name", &w, *req.WorkspaceName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get workspace %v", *req.WorkspaceName)
		}
		workspaceID = int(w.Id)
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanCreateModel(ctx, *curUser,
		int32(workspaceID)); err != nil {
		return nil, err
	}
	m := &modelv1.Model{}
	reqLabels := strings.Join(req.Labels, ",")
	err = a.m.db.QueryProto(
		"insert_model", m, req.Name, req.Description, b,
		reqLabels, req.Notes, user.User.Id, workspaceID,
	)

	return &apiv1.PostModelResponse{Model: m},
		errors.Wrapf(err, "error creating model %q in database", req.Name)
}

func (a *apiServer) PatchModel(
	ctx context.Context, req *apiv1.PatchModelRequest,
) (*apiv1.PatchModelResponse, error) {
	currModel, err := a.ModelFromIdentifier(ctx, req.ModelName)
	if err != nil {
		return nil, err
	}

	if currModel.Archived {
		return nil, errors.Errorf("model %q is archived and cannot have attributes updated.",
			currModel.Name)
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanEditModel(ctx, *curUser, currModel,
		currModel.WorkspaceId); err != nil {
		return nil, err
	}

	madeChanges := false
	if req.Model.Name != nil && req.Model.Name.Value != currModel.Name {
		log.Infof("model (%v) name changing from %q to %q",
			currModel.Id, currModel.Name, req.Model.Name.Value)
		if err = a.clearModelName(ctx, req.Model.Name.Value); err != nil {
			return nil, err
		}
		madeChanges = true
		currModel.Name = req.Model.Name.Value
	}

	if req.Model.Description != nil && req.Model.Description.Value != currModel.Description {
		log.Infof("model %q description changing from %q to %q",
			currModel.Name, currModel.Description, req.Model.Description.Value)
		madeChanges = true
		currModel.Description = req.Model.Description.Value
	}

	if req.Model.Notes != nil && req.Model.Notes.Value != currModel.Notes {
		log.Infof("model %q notes changing from %q to %q",
			currModel.Name, currModel.Notes, req.Model.Notes.Value)
		madeChanges = true
		currModel.Notes = req.Model.Notes.Value
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
			log.Infof("model %q metadata changing from %q to %q",
				currModel.Name, currMeta, newMeta)
			madeChanges = true
			currMeta = newMeta
		}
	}

	currLabels := strings.Join(currModel.Labels, ",")
	if req.Model.Labels != nil {
		var reqLabelList []string
		for _, el := range req.Model.Labels.Values {
			if _, ok := el.GetKind().(*structpb.Value_StringValue); ok {
				reqLabelList = append(reqLabelList, el.GetStringValue())
			}
		}
		reqLabels := strings.Join(reqLabelList, ",")
		if currLabels != reqLabels {
			log.Infof("model %q labels changing from %q to %q",
				currModel.Name, currModel.Labels, reqLabels)
			madeChanges = true
		}
		currLabels = reqLabels
	}

	currWorkspaceID := currModel.WorkspaceId
	if req.Model.WorkspaceId != nil || req.Model.WorkspaceName != nil {
		w := workspacev1.Workspace{}
		if req.Model.WorkspaceId != nil { // default
			err := a.m.db.Query("get_workspace", &w, *req.Model.WorkspaceId)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get workspace with id %v", *req.Model.WorkspaceId)
			}
		} else {
			err := a.m.db.Query("get_workspace_from_name", &w, *req.Model.WorkspaceName)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get workspace %v", *req.Model.WorkspaceName)
			}
		}

		newWorkspaceID := w.Id
		if currWorkspaceID != newWorkspaceID {
			// check if user has permissions in new workspace.
			if err := modelauth.AuthZProvider.Get().CanEditModel(ctx, *curUser, currModel,
				newWorkspaceID); err != nil {
				return nil, err
			}
			currWorkspaceID = newWorkspaceID
			madeChanges = true
		}
	}

	if !madeChanges {
		return &apiv1.PatchModelResponse{Model: currModel}, nil
	}

	finalModel := &modelv1.Model{}
	err = a.m.db.QueryProto(
		"update_model", finalModel, currModel.Id, currModel.Name, currModel.Description,
		currModel.Notes, currMeta, currLabels, currWorkspaceID)

	return &apiv1.PatchModelResponse{Model: finalModel},
		errors.Wrapf(err, "error updating model %q in database", currModel.Name)
}

func (a *apiServer) ArchiveModel(
	ctx context.Context, req *apiv1.ArchiveModelRequest,
) (*apiv1.ArchiveModelResponse, error) {
	currModel, err := a.ModelFromIdentifier(ctx, req.ModelName)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanEditModel(ctx, *curUser, currModel,
		currModel.WorkspaceId); err != nil {
		return nil, err
	}

	holder := &modelv1.Model{}
	err = a.m.db.QueryProto("archive_model", holder, currModel.Name)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "model %q was not found and cannot be archived",
			req.ModelName)
	}

	return &apiv1.ArchiveModelResponse{},
		errors.Wrapf(err, "error archiving model %q", req.ModelName)
}

func (a *apiServer) UnarchiveModel(
	ctx context.Context, req *apiv1.UnarchiveModelRequest,
) (*apiv1.UnarchiveModelResponse, error) {
	currModel, err := a.ModelFromIdentifier(ctx, req.ModelName)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanEditModel(ctx, *curUser, currModel,
		currModel.WorkspaceId); err != nil {
		return nil, err
	}

	holder := &modelv1.Model{}
	err = a.m.db.QueryProto("unarchive_model", holder, currModel.Name)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "model %q was not found and cannot be un-archived",
			req.ModelName)
	}

	return &apiv1.UnarchiveModelResponse{},
		errors.Wrapf(err, "error unarchiving model %q", req.ModelName)
}

func (a *apiServer) DeleteModel(
	ctx context.Context, req *apiv1.DeleteModelRequest) (*apiv1.DeleteModelResponse,
	error,
) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	currModel, err := a.ModelFromIdentifier(ctx, req.ModelName)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanEditModel(ctx, *curUser, currModel,
		currModel.WorkspaceId); err != nil {
		return nil, err
	}
	holder := &modelv1.Model{}
	err = a.m.db.QueryProto("delete_model", holder, currModel.Name, user.User.Id,
		user.User.Admin)

	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "model %q does not exist or not deletable by this user",
			req.ModelName)
	}

	return &apiv1.DeleteModelResponse{},
		errors.Wrapf(err, "error deleting model %q", req.ModelName)
}

func (a *apiServer) GetModelVersion(
	ctx context.Context, req *apiv1.GetModelVersionRequest,
) (*apiv1.GetModelVersionResponse, error) {
	mv, err := a.ModelVersionFromID(ctx, req.ModelName, req.ModelVersionNum)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	currModel, err := a.ModelFromIdentifier(ctx, req.ModelName)
	if ok, err := modelauth.AuthZProvider.Get().CanGetModel(ctx, *curUser, currModel,
		currModel.WorkspaceId); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.Errorf("current user %q doesn't have permissions to get model %q.",
			curUser.Username, currModel.Name)
	}

	resp := &apiv1.GetModelVersionResponse{}
	resp.ModelVersion = mv
	return resp, err
}

func (a *apiServer) GetModelVersions(
	ctx context.Context, req *apiv1.GetModelVersionsRequest,
) (*apiv1.GetModelVersionsResponse, error) {
	parentModel, err := a.ModelFromIdentifier(ctx, req.ModelName)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	if ok, err := modelauth.AuthZProvider.Get().CanGetModel(ctx, *curUser, parentModel,
		parentModel.WorkspaceId); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.Errorf("current user %q doesn't have permissions to get model %q.",
			curUser.Username, parentModel.Name)
	}

	resp := &apiv1.GetModelVersionsResponse{Model: parentModel}
	err = a.m.db.QueryProto("get_model_versions", &resp.ModelVersions, parentModel.Id)
	if err != nil {
		return nil, err
	}

	a.sort(resp.ModelVersions, req.OrderBy, req.SortBy, apiv1.GetModelVersionsRequest_SORT_BY_VERSION)
	return resp, a.paginate(&resp.Pagination, &resp.ModelVersions, req.Offset, req.Limit)
}

func (a *apiServer) PostModelVersion(
	ctx context.Context, req *apiv1.PostModelVersionRequest,
) (*apiv1.PostModelVersionResponse, error) {
	// make sure that the model exists before adding a version
	modelResp, err := a.ModelFromIdentifier(ctx, req.ModelName)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanEditModel(ctx, *curUser, modelResp,
		modelResp.WorkspaceId); err != nil {
		return nil, err
	}

	if modelResp.Archived {
		return nil, errors.Errorf("model %q is archived and cannot register new versions.",
			modelResp.Name)
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
		modelResp.Id,
		c.Uuid,
		req.Name,
		req.Comment,
		mdata,
		reqLabels,
		req.Notes,
		user.User.Id,
	)

	return respModelVersion, errors.Wrapf(err, "error adding model version to model %q",
		req.ModelName)
}

func (a *apiServer) PatchModelVersion(
	ctx context.Context, req *apiv1.PatchModelVersionRequest) (*apiv1.PatchModelVersionResponse,
	error,
) {
	currModelVersion, err := a.ModelVersionFromID(ctx, req.ModelName, req.ModelVersionNum)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	currModel, err := a.ModelFromIdentifier(ctx, req.ModelName)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanEditModel(ctx, *curUser, currModel,
		currModel.WorkspaceId); err != nil {
		return nil, err
	}

	parentModel := currModelVersion.Model
	madeChanges := false
	modelVersionName := fmt.Sprintf("%v:%v", req.ModelName, req.ModelVersionNum)

	if req.ModelVersion.Name != nil && req.ModelVersion.Name.Value != currModelVersion.Name {
		log.Infof("model version (%v) name changing from %q to %q",
			modelVersionName, currModelVersion.Name, req.ModelVersion.Name.Value)
		madeChanges = true
		currModelVersion.Name = req.ModelVersion.Name.Value
	}

	if req.ModelVersion.Comment != nil && req.ModelVersion.Comment.Value != currModelVersion.Comment {
		log.Infof("model version (%v) comment changing from %q to %q",
			modelVersionName, currModelVersion.Comment, req.ModelVersion.Comment.Value)
		madeChanges = true
		currModelVersion.Comment = req.ModelVersion.Comment.Value
	}

	if req.ModelVersion.Notes != nil && req.ModelVersion.Notes.Value != currModelVersion.Notes {
		log.Infof("model version (%v) notes changing from %q to %q",
			modelVersionName, currModelVersion.Notes, req.ModelVersion.Notes.Value)
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
			log.Infof("model version (%v) metadata changing from %q to %q",
				modelVersionName, currMeta, newMeta)
			madeChanges = true
			currMeta = newMeta
		}
	}

	currLabels := strings.Join(currModelVersion.Labels, ",")
	if req.ModelVersion.Labels != nil {
		var reqLabelList []string
		for _, el := range req.ModelVersion.Labels.Values {
			if _, ok := el.GetKind().(*structpb.Value_StringValue); ok {
				reqLabelList = append(reqLabelList, el.GetStringValue())
			}
		}
		reqLabels := strings.Join(reqLabelList, ",")
		if currLabels != reqLabels {
			log.Infof("model version (%v) labels changing from %q to %q",
				modelVersionName, currModelVersion.Labels, reqLabels)
			madeChanges = true
		}
		currLabels = reqLabels
	}

	if !madeChanges {
		return &apiv1.PatchModelVersionResponse{ModelVersion: currModelVersion}, nil
	}

	finalModelVersion := &modelv1.ModelVersion{}
	err = a.m.db.QueryProto("update_model_version", finalModelVersion, currModelVersion.Id,
		parentModel.Id, currModelVersion.Name, currModelVersion.Comment, currModelVersion.Notes,
		currMeta, currLabels)

	return &apiv1.PatchModelVersionResponse{ModelVersion: finalModelVersion},
		errors.Wrapf(err, "error updating model version (%v) in database", modelVersionName)
}

func (a *apiServer) DeleteModelVersion(
	ctx context.Context, req *apiv1.DeleteModelVersionRequest) (*apiv1.DeleteModelVersionResponse,
	error,
) {
	user, err := a.CurrentUser(ctx, &apiv1.CurrentUserRequest{})
	if err != nil {
		return nil, err
	}

	modelVersion, err := a.ModelVersionFromID(ctx, req.ModelName, req.ModelVersionNum)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	currModel, err := a.ModelFromIdentifier(ctx, req.ModelName)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanEditModel(ctx, *curUser, currModel,
		currModel.WorkspaceId); err != nil {
		return nil, err
	}

	holder := &modelv1.ModelVersion{}
	err = a.m.db.QueryProto("delete_model_version", holder, modelVersion.Id,
		user.User.Id, user.User.Admin)

	modelVersionName := fmt.Sprintf("%v:%v", req.ModelName, req.ModelVersionNum)
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "model version %v does not exist or not deletable by this user",
			modelVersionName)
	}

	return &apiv1.DeleteModelVersionResponse{},
		errors.Wrapf(err, "error deleting model version %v", modelVersionName)
}
