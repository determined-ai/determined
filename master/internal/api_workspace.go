package internal

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/templates"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/set"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

const kubernetesDefaultNamespace = "default"

func maskStorageConfigSecrets(w *workspacev1.Workspace) error {
	if w.CheckpointStorageConfig == nil {
		return nil
	}

	// Convert to expconf.
	bytes, err := w.CheckpointStorageConfig.MarshalJSON()
	if err != nil {
		return err
	}
	var checkpointStorageConfig expconf.CheckpointStorageConfig
	if err = (&checkpointStorageConfig).UnmarshalJSON(bytes); err != nil {
		return err
	}

	// Convert back to proto.Struct with .Printable() called.
	bytes, err = checkpointStorageConfig.Printable().MarshalJSON()
	if err != nil {
		return err
	}
	if err = w.CheckpointStorageConfig.UnmarshalJSON(bytes); err != nil {
		return err
	}
	return nil
}

func validatePostWorkspaceRequest(req *apiv1.PostWorkspaceRequest) error {
	if req.ClusterName != nil && req.NamespaceName == nil {
		return status.Errorf(codes.InvalidArgument,
			"Must specify either an existing Kubernetes namespace")
	}
	if req.NamespaceName != nil && req.ClusterName == nil {
		return status.Errorf(codes.InvalidArgument,
			"You must specify a cluster for the specified namespace that you would like to bind.")
	}
	return nil
}

func validateWorkspaceName(name string) error {
	switch {
	case len(name) < 1:
		return status.Errorf(codes.InvalidArgument, "name '%s' must be at least 1 character long", name)
	case len(name) > 53:
		return status.Errorf(codes.InvalidArgument, "name '%s' must be at most 53 character long", name)
	case len(strings.TrimFunc(name, unicode.IsSpace)) == 0:
		return status.Error(codes.InvalidArgument, "name must contain at least non-whitespace letter")
	default:
		return nil
	}
}

func generateNamespaceName(workspace string) (*string, error) {
	namespace := "det-" + workspace
	// Ensure the namespace name is <= 63 characters.
	if len(namespace) > 63 {
		return nil, status.Error(codes.InvalidArgument, "The namespace name must be at most 63 characters")
	}
	return &namespace, nil
}

func (a *apiServer) GetWorkspaceByID(
	ctx context.Context, id int32, curUser model.User, rejectImmutable bool,
) (*workspacev1.Workspace, error) {
	notFoundErr := api.NotFoundErrs("workspace", fmt.Sprint(id), true)
	w := &workspacev1.Workspace{}

	if err := a.m.db.QueryProto("get_workspace", w, id, curUser.ID); errors.Is(err, db.ErrNotFound) {
		return nil, notFoundErr
	} else if err != nil {
		return nil, errors.Wrapf(err, "error fetching workspace (%d) from database", id)
	}

	if err := workspace.AuthZProvider.Get().CanGetWorkspace(ctx, curUser, w); err != nil {
		return nil, authz.SubIfUnauthorized(err, notFoundErr)
	}

	if err := maskStorageConfigSecrets(w); err != nil {
		return nil, err
	}

	if rejectImmutable && w.Immutable {
		return nil, errors.Errorf("workspace (%v) is immutable and cannot add new projects", w.Id)
	}
	if rejectImmutable && w.Archived {
		return nil, errors.Errorf("workspace (%v) is archived and cannot add new projects", w.Id)
	}
	return w, nil
}

func (a *apiServer) getWorkspaceAndCheckCanDoActions(
	ctx context.Context, workspaceID int32, rejectImmutable bool,
	canDoActions ...func(context.Context, model.User, *workspacev1.Workspace) error,
) (*workspacev1.Workspace, model.User, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, model.User{}, err
	}
	w, err := a.GetWorkspaceByID(ctx, workspaceID, *curUser, rejectImmutable)
	if err != nil {
		return nil, model.User{}, err
	}

	for _, canDoAction := range canDoActions {
		if err = canDoAction(ctx, *curUser, w); err != nil {
			return nil, model.User{}, status.Error(codes.PermissionDenied, err.Error())
		}
	}
	return w, *curUser, nil
}

func (a *apiServer) workspaceHasModels(ctx context.Context, workspaceID int32) (bool, error) {
	exists, err := db.Bun().NewSelect().Table("models").
		Where("workspace_id=?", workspaceID).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("checking workspace for models: %w", err)
	}
	return exists, nil
}

func (a *apiServer) GetWorkspace(
	ctx context.Context, req *apiv1.GetWorkspaceRequest,
) (*apiv1.GetWorkspaceResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	w, err := a.GetWorkspaceByID(ctx, req.Id, *curUser, false)
	return &apiv1.GetWorkspaceResponse{Workspace: w}, err
}

func (a *apiServer) GetWorkspaceProjects(
	ctx context.Context, req *apiv1.GetWorkspaceProjectsRequest,
) (*apiv1.GetWorkspaceProjectsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if req.Id != 0 {
		if _, err = a.GetWorkspaceByID(ctx, req.Id, *curUser, false); err != nil {
			return nil, err
		}
	}

	nameFilter := req.Name
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
	// Construct the ordering expression.
	startTime := apiv1.GetWorkspaceProjectsRequest_SORT_BY_LAST_EXPERIMENT_START_TIME
	sortColMap := map[apiv1.GetWorkspaceProjectsRequest_SortBy]string{
		// `p` is an alias of `project` which is defined in master/static/srv/get_workspace_projects.sql
		apiv1.GetWorkspaceProjectsRequest_SORT_BY_UNSPECIFIED:   "p.id",
		apiv1.GetWorkspaceProjectsRequest_SORT_BY_CREATION_TIME: "p.created_at",
		startTime: "last_experiment_started_at",
		apiv1.GetWorkspaceProjectsRequest_SORT_BY_ID:          "p.id",
		apiv1.GetWorkspaceProjectsRequest_SORT_BY_NAME:        "p.name",
		apiv1.GetWorkspaceProjectsRequest_SORT_BY_DESCRIPTION: "p.description",
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

	resp := &apiv1.GetWorkspaceProjectsResponse{}
	err = a.m.db.QueryProtof(
		"get_workspace_projects",
		[]interface{}{orderExpr},
		&resp.Projects,
		req.Id,
		userFilterExpr,
		userIDFilterExpr,
		nameFilter,
		archFilterExpr,
	)
	if err != nil {
		return nil, err
	}

	resp.Projects, err = workspace.AuthZProvider.Get().
		FilterWorkspaceProjects(ctx, *curUser, resp.Projects)
	if err != nil {
		return nil, err
	}

	return resp, api.Paginate(&resp.Pagination, &resp.Projects, req.Offset, req.Limit)
}

func (a *apiServer) GetWorkspaces(
	ctx context.Context, req *apiv1.GetWorkspacesRequest,
) (*apiv1.GetWorkspacesResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	nameFilter := req.Name
	nameCaseSensitiveFilter := req.NameCaseSensitive
	archFilterExpr := ""
	if req.Archived != nil {
		archFilterExpr = strconv.FormatBool(req.Archived.Value)
	}
	pinFilterExpr := ""
	if req.Pinned != nil {
		pinFilterExpr = strconv.FormatBool(req.Pinned.Value)
	}
	userFilterExpr := strings.Join(req.Users, ",")
	userIds := make([]string, 0, len(req.UserIds))
	for _, userID := range req.UserIds {
		userIds = append(userIds, strconv.Itoa(int(userID)))
	}
	userIDFilterExpr := strings.Join(userIds, ",")
	// Construct the ordering expression.
	sortColMap := map[apiv1.GetWorkspacesRequest_SortBy]string{
		apiv1.GetWorkspacesRequest_SORT_BY_UNSPECIFIED: "id",
		apiv1.GetWorkspacesRequest_SORT_BY_ID:          "id",
		apiv1.GetWorkspacesRequest_SORT_BY_NAME:        "name",
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

	resp := &apiv1.GetWorkspacesResponse{}
	err = a.m.db.QueryProtof(
		"get_workspaces",
		[]interface{}{orderExpr},
		&resp.Workspaces,
		userFilterExpr,
		userIDFilterExpr,
		nameFilter,
		archFilterExpr,
		pinFilterExpr,
		curUser.ID,
		nameCaseSensitiveFilter,
	)
	if err != nil {
		return nil, err
	}

	resp.Workspaces, err = workspace.AuthZProvider.Get().
		FilterWorkspaces(ctx, *curUser, resp.Workspaces)
	if err != nil {
		return nil, err
	}

	return resp, api.Paginate(&resp.Pagination, &resp.Workspaces, req.Offset, req.Limit)
}

func (a *apiServer) PostWorkspace(
	ctx context.Context, req *apiv1.PostWorkspaceRequest,
) (*apiv1.PostWorkspaceResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)

	if err != nil {
		return nil, err
	}

	if err = workspace.AuthZProvider.Get().CanCreateWorkspace(ctx, *curUser); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if err = validateWorkspaceName(req.Name); err != nil {
		return nil, err
	}

	if req.AgentUserGroup != nil {
		err = workspace.AuthZProvider.Get().CanCreateWorkspaceWithAgentUserGroup(ctx, *curUser)
		if err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	if req.CheckpointStorageConfig != nil && len(req.CheckpointStorageConfig.Fields) > 0 {
		if err = workspace.AuthZProvider.Get().
			CanCreateWorkspaceWithCheckpointStorageConfig(ctx, *curUser); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
	}

	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.WithError(err).Error("error rolling back transaction in create workspace")
		}
	}()

	w := &model.Workspace{
		Name: req.Name, UserID: curUser.ID,
		DefaultComputePool: req.DefaultComputePool, DefaultAuxPool: req.DefaultAuxPool,
	}

	if req.AgentUserGroup != nil {
		w.AgentUID = req.AgentUserGroup.AgentUid
		w.AgentGID = req.AgentUserGroup.AgentGid
		w.AgentUser = req.AgentUserGroup.AgentUser
		w.AgentGroup = req.AgentUserGroup.AgentGroup
	}

	if req.CheckpointStorageConfig != nil && len(req.CheckpointStorageConfig.Fields) > 0 {
		var bytes []byte
		bytes, err = req.CheckpointStorageConfig.MarshalJSON()
		if err != nil {
			return nil, err
		}
		var sc expconf.CheckpointStorageConfig
		w.CheckpointStorageConfig = &sc
		if err = w.CheckpointStorageConfig.UnmarshalJSON(bytes); err != nil {
			return nil, err
		}
		if err = schemas.IsComplete(w.CheckpointStorageConfig); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
	}

	if err = workspace.AuthZProvider.Get().
		CanModifyWorkspaceNamespaceBindings(ctx, *curUser); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if err := validatePostWorkspaceRequest(req); err != nil {
		return nil, err
	}

	err = tx.NewInsert().Model(w).Scan(ctx, w)
	if err != nil {
		if strings.Contains(err.Error(), db.CodeUniqueViolation) {
			return nil,
				status.Errorf(codes.AlreadyExists, "avoid names equal to other workspaces (case-insensitive)")
		}
		return nil, errors.Wrapf(err, "error creating workspace %s in database", req.Name)
	}

	// Verify that the specified cluster name is also provided in the master config.
	if req.ClusterName != nil {
		newReq := &apiv1.ModifyWorkspaceNamespaceBindingRequest{
			Id:            int32(w.ID),
			ClusterName:   *req.ClusterName,
			NamespaceName: req.NamespaceName,
		}

		_, err := a.modifyWorkspaceNamespaceBinding(ctx, newReq, &tx, w)
		if err != nil {
			return nil, fmt.Errorf("Failed to create namespace binding: %w", err)
		}
	}

	pin := &model.WorkspacePin{WorkspaceID: w.ID, UserID: w.UserID}
	_, err = tx.NewInsert().Model(pin).Exec(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating workspace pin %s in database", req.Name)
	}

	if err = a.AssignWorkspaceAdminToUserTx(ctx, tx, w.ID, w.UserID); err != nil {
		return nil, errors.Wrap(err, "error assigning workspace admin")
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "could not commit create workspace transcation")
	}

	protoWorkspace, err := w.ToProto()
	if err != nil {
		return nil, err
	}
	protoWorkspace.Username = curUser.Username
	protoWorkspace.Pinned = true
	return &apiv1.PostWorkspaceResponse{Workspace: protoWorkspace}, nil
}

func (a *apiServer) PatchWorkspace(
	ctx context.Context, req *apiv1.PatchWorkspaceRequest,
) (*apiv1.PatchWorkspaceResponse, error) {
	currWorkspace, currUser, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, true)
	if err != nil {
		return nil, err
	}

	insertColumns := []string{}
	updatedWorkspace := model.Workspace{}

	if req.Workspace.Name != nil {
		err = validateWorkspaceName(req.Workspace.Name.Value)
		if err != nil {
			return nil, err
		}
	}

	if req.Workspace.Name != nil && req.Workspace.Name.Value != currWorkspace.Name {
		if err = workspace.AuthZProvider.Get().
			CanSetWorkspacesName(ctx, currUser, currWorkspace); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		log.Infof("workspace (%d) name changing from \"%s\" to \"%s\"",
			currWorkspace.Id, currWorkspace.Name, req.Workspace.Name.Value)
		insertColumns = append(insertColumns, "name")
		updatedWorkspace.Name = req.Workspace.Name.Value
	}

	if req.Workspace.AgentUserGroup != nil {
		if err = workspace.AuthZProvider.Get().
			CanSetWorkspacesAgentUserGroup(ctx, currUser, currWorkspace); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		updateAug := req.Workspace.AgentUserGroup

		updatedWorkspace.AgentUID = updateAug.AgentUid
		updatedWorkspace.AgentGID = updateAug.AgentGid
		updatedWorkspace.AgentUser = updateAug.AgentUser
		updatedWorkspace.AgentGroup = updateAug.AgentGroup

		insertColumns = append(insertColumns, "uid", "user_", "gid", "group_")
	}

	if req.Workspace.DefaultComputeResourcePool != nil ||
		req.Workspace.DefaultAuxResourcePool != nil {
		if err = workspace.AuthZProvider.Get().
			CanSetWorkspacesDefaultPools(ctx, currUser, currWorkspace); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		rpConfigs, err := a.resourcePoolsAsConfigs()
		if err != nil {
			return nil, err
		}
		rpNamesSlice, _, err := db.ReadRPsAvailableToWorkspace(
			ctx, currWorkspace.Id, 0, -1, rpConfigs,
		)
		if err != nil {
			return nil, err
		}

		rpNames := set.FromSlice(append(rpNamesSlice, ""))

		if req.Workspace.DefaultComputeResourcePool != nil {
			if !rpNames.Contains(*req.Workspace.DefaultComputeResourcePool) {
				return nil, status.Error(codes.FailedPrecondition, "unable to bind a resource "+
					"pool that does not exist or is not available to the workspace")
			}
			updatedWorkspace.DefaultComputePool = *req.Workspace.DefaultComputeResourcePool
			insertColumns = append(insertColumns, "default_compute_pool")
		}
		if req.Workspace.DefaultAuxResourcePool != nil {
			if !rpNames.Contains(*req.Workspace.DefaultAuxResourcePool) {
				return nil, status.Error(codes.FailedPrecondition, "unable to bind a resource "+
					"pool that does not exist or is not available to the workspace")
			}
			updatedWorkspace.DefaultAuxPool = *req.Workspace.DefaultAuxResourcePool
			insertColumns = append(insertColumns, "default_aux_pool")
		}
	}

	if req.Workspace.CheckpointStorageConfig != nil {
		if err = workspace.AuthZProvider.Get().
			CanSetWorkspacesCheckpointStorageConfig(ctx, currUser, currWorkspace); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		if len(req.Workspace.CheckpointStorageConfig.Fields) > 0 {
			var bytes []byte
			bytes, err = req.Workspace.CheckpointStorageConfig.MarshalJSON()
			if err != nil {
				return nil, err
			}
			var sc expconf.CheckpointStorageConfig
			updatedWorkspace.CheckpointStorageConfig = &sc
			if err = updatedWorkspace.CheckpointStorageConfig.UnmarshalJSON(bytes); err != nil {
				return nil, err
			}
			if err = schemas.IsComplete(updatedWorkspace.CheckpointStorageConfig); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
		}
		insertColumns = append(insertColumns, "checkpoint_storage_config")
	}

	if len(insertColumns) == 0 {
		return &apiv1.PatchWorkspaceResponse{Workspace: currWorkspace}, nil
	}

	_, err = db.Bun().NewUpdate().Model(&updatedWorkspace).
		Column(insertColumns...).
		Where("id = ?", currWorkspace.Id).
		Exec(ctx)
	if err != nil {
		if strings.Contains(err.Error(), db.CodeUniqueViolation) {
			return nil,
				status.Errorf(codes.AlreadyExists, "avoid names equal to other workspaces (case-insensitive)")
		}
		return nil, err
	}

	// TODO(ilia): Avoid second refetch.
	finalWorkspace, err := a.GetWorkspaceByID(ctx, currWorkspace.Id, currUser, false)
	return &apiv1.PatchWorkspaceResponse{Workspace: finalWorkspace},
		errors.Wrapf(err, "error refetching updated workspace (%d) from db", currWorkspace.Id)
}

func (a *apiServer) modifyWorkspaceNamespaceBinding(ctx context.Context,
	req *apiv1.ModifyWorkspaceNamespaceBindingRequest, tx *bun.Tx, w *model.Workspace) (*apiv1.ModifyWorkspaceNamespaceBindingResponse, error) {
	// Create the namespace in Kubernetes.
	err := a.m.rm.CreateNamespace(false, *req.NamespaceName, req.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s namespace: %w", err)
	}

	var wsns model.WorkspaceNamespace

	// If there is more than one namespace bound to a workspace within a given cluster, this query
	// should fail (because the result would have type []int and []string rather than int and string
	// string, respectively).
	err = tx.NewSelect().Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", w.ID).
		Where("cluster_name = ?", req.ClusterName).
		Scan(ctx, &wsns)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("error getting the current workspace-namespace binding: %w", err)
		} else { // The workspace didn't have a namespace binding.
			workspaceNamespace := &model.WorkspaceNamespace{
				NamespaceName: *req.NamespaceName,
				ClusterName:   req.ClusterName,
				WorkspaceID:   w.ID,
			}
			err = tx.NewInsert().Model(workspaceNamespace).Scan(ctx)
			if err != nil {
				return nil, errors.Wrapf(err, "error adding workspace-namespace %s to database",
					*req.NamespaceName)
			}
		}
	} else {
		_, err = tx.NewUpdate().Model(&model.WorkspaceNamespace{}).
			Set("namespace_name = ?", *req.NamespaceName).
			Where("id = ?", &wsns.ID).
			Exec(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not update workspace-namespace binding")
		}
	}

	res := apiv1.ModifyWorkspaceNamespaceBindingResponse{Name: *req.NamespaceName}

	return &res, nil
}

func (a *apiServer) ModifyWorkspaceNamespaceBinding(ctx context.Context,
	req *apiv1.ModifyWorkspaceNamespaceBindingRequest) (*apiv1.ModifyWorkspaceNamespaceBindingResponse, error) {
	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err = tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.WithError(err).Error("error rolling back transaction in bind workspace to namespace")
		}
	}()

	var w model.Workspace
	err = tx.NewSelect().Model(&model.Workspace{}).Where("id = ?", req.Id).Scan(ctx, &w)
	if err != nil {
		return nil, errors.Wrapf(err, "workspace with name %s not found", w.Name)
	}

	res, err := a.modifyWorkspaceNamespaceBinding(ctx, req, &tx, &w)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "could not commit modify workspace-namespace bindings transcation")
	}
	return res, nil
}

func (a *apiServer) deleteWorkspace(
	ctx context.Context, workspaceID int32, projects []*projectv1.Project,
) {
	log.Debugf("deleting workspace %d projects", workspaceID)
	holder := &workspacev1.Workspace{}
	for _, pj := range projects {
		expList, err := db.ProjectExperiments(context.TODO(), int(pj.Id))
		if err != nil {
			log.WithError(err).Errorf("error fetching experiments on project %d while deleting workspace %d",
				pj.Id, workspaceID)
			_ = a.m.db.QueryProto("delete_fail_workspace", holder, workspaceID, err.Error())
			return
		}
		err = a.deleteProject(ctx, pj.Id, expList)
		if err != nil {
			log.WithError(err).Errorf("error deleting project %d while deleting workspace %d", pj.Id,
				workspaceID)
			_ = a.m.db.QueryProto("delete_fail_workspace", holder, workspaceID, err.Error())
			return
		}
	}

	err := a.m.db.QueryProto("delete_workspace", holder, workspaceID)
	if err != nil {
		log.WithError(err).Errorf("failed to delete workspace %d", workspaceID)
		_ = a.m.db.QueryProto("delete_fail_workspace", holder, workspaceID, err.Error())
		return
	}
	log.Debugf("workspace %d deleted successfully", workspaceID)
}

func (a *apiServer) DeleteWorkspace(
	ctx context.Context,
	req *apiv1.DeleteWorkspaceRequest,
) (*apiv1.DeleteWorkspaceResponse, error) {
	_, _, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, false,
		workspace.AuthZProvider.Get().CanDeleteWorkspace)
	if err != nil {
		return nil, err
	}

	modelsExist, err := a.workspaceHasModels(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if modelsExist {
		return nil, status.Errorf(codes.FailedPrecondition, "workspace (%d) contains models; move or delete models first",
			req.Id)
	}

	holder := &workspacev1.Workspace{}
	// TODO(kristine): DET-10138 update workspace state in transaction with template delete
	err = a.m.db.QueryProto("deletable_workspace", holder, req.Id)
	if err != nil || holder.Id == 0 {
		return nil, fmt.Errorf("workspace (%d) does not exist or not deletable by this user: %w", req.Id, err)
	}

	projects := []*projectv1.Project{}
	err = a.m.db.QueryProtof(
		"get_workspace_projects",
		[]interface{}{"id ASC"},
		&projects,
		req.Id,
		"",
		"",
		"",
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("getting workspace projects: %w", err)
	}

	log.Debugf("deleting workspace %d NTSC", req.Id)
	command.DefaultCmdService.DeleteWorkspaceNTSC(req)

	log.Debugf("deleting workspace %d templates", req.Id)
	err = templates.DeleteWorkspaceTemplates(ctx, int(req.Id))
	if err != nil {
		return nil, fmt.Errorf("error deleting workspace (%d) templates: %w", req.Id, err)
	}

	if len(projects) == 0 {
		err = a.m.db.QueryProto("delete_workspace", holder, req.Id)
		if err != nil {
			return nil, fmt.Errorf("error deleting workspace (%d): %w", req.Id, err)
		}
		return &apiv1.DeleteWorkspaceResponse{Completed: true}, nil
	}

	go func() {
		a.deleteWorkspace(ctx, req.Id, projects)
	}()

	return &apiv1.DeleteWorkspaceResponse{Completed: false}, nil
}

func (a *apiServer) ArchiveWorkspace(
	ctx context.Context, req *apiv1.ArchiveWorkspaceRequest) (*apiv1.ArchiveWorkspaceResponse,
	error,
) {
	_, _, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, false,
		workspace.AuthZProvider.Get().CanArchiveWorkspace)
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	if err = a.m.db.QueryProto("archive_workspace", holder, req.Id, true); err != nil {
		return nil, errors.Wrapf(err, "error archiving workspace (%d)", req.Id)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "workspace (%d) does not exist or not archive-able by this user",
			req.Id)
	}
	return &apiv1.ArchiveWorkspaceResponse{}, nil
}

func (a *apiServer) UnarchiveWorkspace(
	ctx context.Context, req *apiv1.UnarchiveWorkspaceRequest) (*apiv1.UnarchiveWorkspaceResponse,
	error,
) {
	_, _, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, false,
		workspace.AuthZProvider.Get().CanUnarchiveWorkspace)
	if err != nil {
		return nil, err
	}

	holder := &workspacev1.Workspace{}
	if err = a.m.db.QueryProto("archive_workspace", holder, req.Id, false); err != nil {
		return nil, errors.Wrapf(err, "error unarchiving workspace (%d)", req.Id)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err,
			"workspace (%d) does not exist or not unarchive-able by this user", req.Id)
	}
	return &apiv1.UnarchiveWorkspaceResponse{}, nil
}

func (a *apiServer) PinWorkspace(
	ctx context.Context, req *apiv1.PinWorkspaceRequest,
) (*apiv1.PinWorkspaceResponse, error) {
	_, currUser, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, false,
		workspace.AuthZProvider.Get().CanPinWorkspace)
	if err != nil {
		return nil, err
	}

	err = a.m.db.QueryProto("pin_workspace", &workspacev1.Workspace{}, req.Id, currUser.ID)

	return &apiv1.PinWorkspaceResponse{},
		errors.Wrapf(err, "error pinning workspace (%d)", req.Id)
}

func (a *apiServer) UnpinWorkspace(
	ctx context.Context, req *apiv1.UnpinWorkspaceRequest,
) (*apiv1.UnpinWorkspaceResponse, error) {
	_, currUser, err := a.getWorkspaceAndCheckCanDoActions(ctx, req.Id, false,
		workspace.AuthZProvider.Get().CanUnpinWorkspace)
	if err != nil {
		return nil, err
	}

	err = a.m.db.QueryProto("unpin_workspace", &workspacev1.Workspace{}, req.Id, currUser.ID)

	return &apiv1.UnpinWorkspaceResponse{},
		errors.Wrapf(err, "error un-pinning workspace (%d)", req.Id)
}

func (a *apiServer) ListRPsBoundToWorkspace(
	ctx context.Context, req *apiv1.ListRPsBoundToWorkspaceRequest,
) (*apiv1.ListRPsBoundToWorkspaceResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	err = workspace.AuthZProvider.Get().CanGetWorkspaceID(
		ctx, *curUser, req.WorkspaceId,
	)
	if err != nil {
		return nil, err
	}

	rpConfigs, err := a.resourcePoolsAsConfigs()
	if err != nil {
		return nil, err
	}
	rpNames, pagination, err := db.ReadRPsAvailableToWorkspace(
		ctx, req.WorkspaceId, req.Offset, req.Limit, rpConfigs,
	)
	if err != nil {
		return nil, err
	}

	return &apiv1.ListRPsBoundToWorkspaceResponse{
		ResourcePools: rpNames,
		Pagination:    pagination,
	}, nil
}
