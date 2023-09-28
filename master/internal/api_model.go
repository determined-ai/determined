package internal

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	modelauth "github.com/determined-ai/determined/master/internal/model"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"

	structpb "github.com/golang/protobuf/ptypes/struct"
)

func (a *apiServer) ModelFromIdentifier(identifier string) (*modelv1.Model, error) {
	var err error
	m := &modelv1.Model{}

	allNumbers, _ := regexp.MatchString("^\\d+$", identifier)
	if allNumbers {
		err = a.m.db.QueryProto("get_model_by_id", m, identifier)
	} else {
		err = a.m.db.QueryProto("get_model", m, identifier)
	}
	switch err {
	case db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "model %q not found", identifier)
	default:
		return m, errors.Wrapf(err,
			"error fetching model %q from database", identifier)
	}
}

func (a *apiServer) ModelVersionFromID(modelIdentifier string,
	versionID int32,
) (*modelv1.ModelVersion, error) {
	mv := &modelv1.ModelVersion{}
	parentModel, err := a.ModelFromIdentifier(modelIdentifier)
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
	m, err := a.ModelFromIdentifier(req.ModelName)
	if err != nil {
		return nil, err
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = modelauth.AuthZProvider.Get().CanGetModel(ctx, *curUser, m,
		m.WorkspaceId); err != nil {
		return nil, authz.SubIfUnauthorized(err,
			errors.Errorf("current user %q doesn't have permissions to get model %q",
				curUser.Username, m.Name))
	}
	return &apiv1.GetModelResponse{Model: m}, err
}

func (a *apiServer) GetModels(
	ctx context.Context, req *apiv1.GetModelsRequest,
) (*apiv1.GetModelsResponse, error) {
	resp := &apiv1.GetModelsResponse{}
	idFilterExpr := req.Id
	nameFilter := "%" + req.Name + "%"
	descFilterExpr := "%" + req.Description + "%"
	archFilterExpr := ""
	if req.Archived != nil {
		archFilterExpr = strconv.FormatBool(req.Archived.Value)
	}
	userFilterExpr := strings.Join(req.Users, ",")
	userIds := make([]string, 0, len(req.UserIds))
	for _, userID := range req.UserIds {
		userIds = append(userIds, strconv.Itoa(int(userID)))
	}
	userIDFilterExpr := strings.Join(userIds, ",")
	labelFilterExpr := strings.Join(req.Labels, ",")
	// Construct the ordering expression.
	sortColMap := map[apiv1.GetModelsRequest_SortBy]string{
		apiv1.GetModelsRequest_SORT_BY_UNSPECIFIED:       "id",
		apiv1.GetModelsRequest_SORT_BY_NAME:              "name",
		apiv1.GetModelsRequest_SORT_BY_DESCRIPTION:       "description",
		apiv1.GetModelsRequest_SORT_BY_CREATION_TIME:     "creation_time",
		apiv1.GetModelsRequest_SORT_BY_LAST_UPDATED_TIME: "last_updated_time",
		apiv1.GetModelsRequest_SORT_BY_NUM_VERSIONS:      "num_versions",
		apiv1.GetModelsRequest_SORT_BY_WORKSPACE:         "w.name",
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

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	var workspaceIdsGiven []int32
	if req.WorkspaceIds != nil {
		// default is to use workspace ids.
		workspaceIdsGiven = req.WorkspaceIds
	} else if req.WorkspaceIds == nil && req.WorkspaceNames != nil {
		// get the ids of the corresponding workspaces
		if err := db.Bun().NewSelect().Table("workspaces").Column("id").
			Where("name in (?)", bun.In(req.WorkspaceNames)).Distinct().
			Scan(ctx, &workspaceIdsGiven); err != nil {
			return nil, fmt.Errorf("getting workspace ids from names: %w", err)
		}
	}
	// function below returns a list of workspaces that have permissions
	// filtered according to user given workspaces.
	// if global permissions and no filter list given by user then it's an empty list.
	workspaceIdsWithPermsAndFilterList, err := modelauth.AuthZProvider.Get().
		CanGetModels(ctx, *curUser, workspaceIdsGiven)
	if err != nil {
		return nil, authz.SubIfUnauthorized(err, errors.Errorf(
			"current user doesn't have view permissions in related workspaces"))
	}
	var workspaceIds []string
	var workspaceIdsWithPermsAndFilter string
	if workspaceIdsWithPermsAndFilterList == nil {
		workspaceIdsWithPermsAndFilter = ""
	} else {
		for _, wID := range workspaceIdsWithPermsAndFilterList {
			workspaceIds = append(workspaceIds, strconv.Itoa(int(wID)))
		}
		workspaceIdsWithPermsAndFilter = strings.Join(workspaceIds, ",")
	}

	err = a.m.db.QueryProtof(
		"get_models",
		[]interface{}{orderExpr},
		&resp.Models,
		idFilterExpr,
		archFilterExpr,
		userFilterExpr,
		userIDFilterExpr,
		labelFilterExpr,
		nameFilter,
		descFilterExpr,
		workspaceIdsWithPermsAndFilter,
	)
	if err != nil {
		return nil, err
	}
	return resp, api.Paginate(&resp.Pagination, &resp.Models, req.Offset, req.Limit)
}

func (a *apiServer) GetModelLabels(
	ctx context.Context, req *apiv1.GetModelLabelsRequest,
) (*apiv1.GetModelLabelsResponse, error) {
	resp := apiv1.GetModelLabelsResponse{}

	modelQuery := db.Bun().NewSelect().
		ModelTableExpr("models as m").
		Column("m.id").
		ColumnExpr("UNNEST(m.labels) AS label")

	if req.WorkspaceId != nil && int(*req.WorkspaceId) > 0 {
		modelQuery = modelQuery.Where("workspace_id = ?", req.WorkspaceId)
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if modelQuery, err = modelauth.AuthZProvider.Get().
		FilterReadableModelsQuery(ctx, *curUser, modelQuery); err != nil {
		return nil, err
	}

	labelQuery := db.Bun().NewSelect().
		ModelTableExpr("(?) AS all_labels", modelQuery).
		Column("all_labels.label").
		GroupExpr("all_labels.label").
		OrderExpr("COUNT(DISTINCT(all_labels.id)) DESC, all_labels.label ASC")

	opQuery := db.Bun().NewSelect().
		ModelTableExpr("(?) AS sorted_labels", labelQuery).
		Model(&resp.Labels).
		ColumnExpr("sorted_labels.label")
	err = opQuery.Scan(ctx)

	return &resp, errors.Wrapf(err, "error getting model labels")
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

	if err != nil && strings.Contains(err.Error(), db.CodeUniqueViolation) {
		return nil,
			status.Errorf(codes.AlreadyExists, "avoid names equal to other models (case-insensitive)")
	}
	return &apiv1.PostModelResponse{Model: m},
		errors.Wrapf(err, "error creating model %q in database", req.Name)
}

func (a *apiServer) PatchModel(
	ctx context.Context, req *apiv1.PatchModelRequest,
) (*apiv1.PatchModelResponse, error) {
	currModel, err := a.ModelFromIdentifier(req.ModelName)
	if err != nil {
		return nil, err
	}

	if currModel.Archived {
		return nil, errors.Errorf("model %q is archived and cannot have attributes updated",
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
		// avoid duplicate keys
		reqLabelSet := make(map[string]struct{}, len(req.Model.Labels.Values))
		reqLabelList := make([]string, len(reqLabelSet))
		for _, el := range req.Model.Labels.Values {
			if _, ok := el.GetKind().(*structpb.Value_StringValue); !ok {
				// Invalid label.
				continue
			}
			label := el.GetStringValue()
			if _, ok := reqLabelSet[label]; ok {
				// Duplicate key.
				continue
			}
			reqLabelSet[label] = struct{}{}
			reqLabelList = append(reqLabelList, label)
		}
		reqLabels := strings.Join(reqLabelList, ",")
		if currLabels != reqLabels {
			log.Infof("model %q labels changing from %q to %q",
				currModel.Name, currModel.Labels, reqLabels)
			madeChanges = true
			currLabels = reqLabels
		}
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

	if err != nil && strings.Contains(err.Error(), db.CodeUniqueViolation) {
		return nil,
			status.Errorf(codes.AlreadyExists, "avoid names equal to other models (case-insensitive)")
	}
	return &apiv1.PatchModelResponse{Model: finalModel},
		errors.Wrapf(err, "error updating model %q in database", currModel.Name)
}

func (a *apiServer) ArchiveModel(
	ctx context.Context, req *apiv1.ArchiveModelRequest,
) (*apiv1.ArchiveModelResponse, error) {
	currModel, err := a.ModelFromIdentifier(req.ModelName)
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
	currModel, err := a.ModelFromIdentifier(req.ModelName)
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

func (a *apiServer) MoveModel(
	ctx context.Context, req *apiv1.MoveModelRequest,
) (*apiv1.MoveModelResponse, error) {
	currModel, err := a.ModelFromIdentifier(req.ModelName)
	if err != nil {
		return nil, err
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	err = modelauth.AuthZProvider.
		Get().
		CanMoveModel(ctx, *curUser, currModel, currModel.WorkspaceId, req.DestinationWorkspaceId)
	if err != nil {
		return nil, err
	}

	holder := &modelv1.Model{}
	err = a.m.db.QueryProto("move_model", holder, currModel.Id, req.DestinationWorkspaceId)
	if err != nil {
		return nil, errors.Wrapf(err, "error moving a model (%s)", req.ModelName)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "Model (%s) does not exist or not moveable by this user",
			req.ModelName)
	}

	return &apiv1.MoveModelResponse{}, nil
}

func (a *apiServer) DeleteModel(
	ctx context.Context, req *apiv1.DeleteModelRequest) (*apiv1.DeleteModelResponse,
	error,
) {
	currModel, err := a.ModelFromIdentifier(req.ModelName)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanDeleteModel(ctx, *curUser, currModel,
		currModel.WorkspaceId); err != nil {
		return nil, err
	}
	holder := &modelv1.Model{}
	err = a.m.db.QueryProto("delete_model", holder, currModel.Name)

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
	mv, err := a.ModelVersionFromID(req.ModelName, req.ModelVersionNum)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	currModel, _ := a.ModelFromIdentifier(req.ModelName)
	if err = modelauth.AuthZProvider.Get().CanGetModel(ctx, *curUser, currModel,
		currModel.WorkspaceId); err != nil {
		return nil, authz.SubIfUnauthorized(err,
			errors.Errorf("current user %q doesn't have permissions to get model %q",
				curUser.Username, currModel.Name))
	}

	resp := &apiv1.GetModelVersionResponse{}
	resp.ModelVersion = mv
	return resp, nil
}

func (a *apiServer) GetModelVersions(
	ctx context.Context, req *apiv1.GetModelVersionsRequest,
) (*apiv1.GetModelVersionsResponse, error) {
	parentModel, err := a.ModelFromIdentifier(req.ModelName)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	if err := modelauth.AuthZProvider.Get().CanGetModel(ctx, *curUser, parentModel,
		parentModel.WorkspaceId); err != nil {
		return nil, authz.SubIfUnauthorized(err,
			errors.Errorf("current user %q doesn't have permissions to get model %q",
				curUser.Username, parentModel.Name))
	}

	resp := &apiv1.GetModelVersionsResponse{Model: parentModel}
	err = a.m.db.QueryProto("get_model_versions", &resp.ModelVersions, parentModel.Id)
	if err != nil {
		return nil, err
	}

	api.Sort(
		resp.ModelVersions, req.OrderBy, req.SortBy, apiv1.GetModelVersionsRequest_SORT_BY_VERSION,
	)
	return resp, api.Paginate(&resp.Pagination, &resp.ModelVersions, req.Offset, req.Limit)
}

func (a *apiServer) PostModelVersion(
	ctx context.Context, req *apiv1.PostModelVersionRequest,
) (*apiv1.PostModelVersionResponse, error) {
	// make sure that the model exists before adding a version
	modelResp, err := a.ModelFromIdentifier(req.ModelName)
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
		return nil, errors.Errorf("model %q is archived and cannot register new versions",
			modelResp.Name)
	}

	// make sure the checkpoint exists
	c := &checkpointv1.Checkpoint{}

	switch getCheckpointErr := a.m.db.QueryProto("get_checkpoint", c, req.CheckpointUuid); {
	case getCheckpointErr == db.ErrNotFound:
		return nil, api.NotFoundErrs("checkpoint", req.CheckpointUuid, true)
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
	currModelVersion, err := a.ModelVersionFromID(req.ModelName, req.ModelVersionNum)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	currModel, err := a.ModelFromIdentifier(req.ModelName)
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
		// avoid duplicate keys
		reqLabelSet := make(map[string]struct{}, len(req.ModelVersion.Labels.Values))
		for _, el := range req.ModelVersion.Labels.Values {
			if _, ok := el.GetKind().(*structpb.Value_StringValue); ok {
				reqLabelSet[el.GetStringValue()] = struct{}{}
			}
		}
		reqLabelList := make([]string, len(reqLabelSet))
		i := 0
		for key := range reqLabelSet {
			reqLabelList[i] = key
			i++
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
	modelVersion, err := a.ModelVersionFromID(req.ModelName, req.ModelVersionNum)
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	currModel, err := a.ModelFromIdentifier(req.ModelName)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanDeleteModelVersion(ctx, *curUser,
		modelVersion, currModel.WorkspaceId); err != nil {
		return nil, err
	}

	holder := &modelv1.ModelVersion{}
	err = a.m.db.QueryProto("delete_model_version", holder, modelVersion.Id)

	modelVersionName := fmt.Sprintf("%v:%v", req.ModelName, req.ModelVersionNum)
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "model version %v does not exist or not deletable by this user",
			modelVersionName)
	}

	return &apiv1.DeleteModelVersionResponse{},
		errors.Wrapf(err, "error deleting model version %v", modelVersionName)
}

// Query for all trials that use a given model_version and return their metrics.
func (a *apiServer) GetTrialMetricsByModelVersion(
	ctx context.Context, req *apiv1.GetTrialMetricsByModelVersionRequest,
) (*apiv1.GetTrialMetricsByModelVersionResponse, error) {
	modelVersionResp, err := a.ModelVersionFromID(
		req.ModelName, req.ModelVersionNum,
	)
	if err != nil {
		return nil, err
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err := modelauth.AuthZProvider.Get().CanGetModel(ctx, *curUser, modelVersionResp.Model,
		modelVersionResp.Model.WorkspaceId); err != nil {
		return nil, err
	}
	resp := &apiv1.GetTrialMetricsByModelVersionResponse{}
	trialIDsQuery := db.Bun().NewSelect().Table("trial_source_infos").
		Where("model_id = ?", modelVersionResp.Model.Id).
		Where("model_version = ?", modelVersionResp.Version)

	if req.TrialSourceInfoType != nil {
		trialIDsQuery.Where("trial_source_info_type = ?", req.TrialSourceInfoType.String())
	}

	metrics, err := trials.GetMetricsForTrialSourceInfoQuery(ctx, trialIDsQuery, req.MetricGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get trial source info %w", err)
	}

	resp.Metrics = append(resp.Metrics, metrics...)
	return resp, nil
}
