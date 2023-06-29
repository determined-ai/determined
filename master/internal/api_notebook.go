package internal

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/internal/task/idle"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	pkgCommand "github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

const (
	jupyterDir        = "/run/determined/jupyter/"
	jupyterConfigDir  = "/run/determined/jupyter/config"
	jupyterDataDir    = "/run/determined/jupyter/data"
	jupyterRuntimeDir = "/run/determined/jupyter/runtime"
	jupyterEntrypoint = "/run/determined/jupyter/notebook-entrypoint.sh"
	jupyterIdleCheck  = "/run/determined/jupyter/check_idle.py"
	// Agent ports 2600 - 3500 are split between TensorBoards, Notebooks, and Shells.
	minNotebookPort     = 2900
	maxNotebookPort     = minNotebookPort + 299
	notebookDefaultPage = "/run/determined/workdir/README.ipynb"
)

var notebooksAddr = actor.Addr(command.NotebookActorPath)

func (a *apiServer) GetNotebooks(
	ctx context.Context, req *apiv1.GetNotebooksRequest,
) (resp *apiv1.GetNotebooksResponse, err error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.WorkspaceId != 0 {
		// check if the workspace exists.
		_, err := a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, false)
		if err != nil {
			return nil, err
		}
	}

	if err = a.ask(notebooksAddr, req, &resp); err != nil {
		return nil, err
	}

	limitedScopes, err := command.AuthZProvider.Get().AccessibleScopes(
		ctx, *curUser, model.AccessScopeID(req.WorkspaceId),
	)
	if err != nil {
		return nil, apiutils.MapAndFilterErrors(err, nil, nil)
	}
	a.filter(&resp.Notebooks, func(i int) bool {
		return limitedScopes[model.AccessScopeID(resp.Notebooks[i].WorkspaceId)]
	})

	a.sort(resp.Notebooks, req.OrderBy, req.SortBy, apiv1.GetNotebooksRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Notebooks, req.Offset, req.Limit)
}

func (a *apiServer) GetNotebook(
	ctx context.Context, req *apiv1.GetNotebookRequest,
) (resp *apiv1.GetNotebookResponse, err error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	addr := notebooksAddr.Child(req.NotebookId)
	if err = a.ask(addr, req, &resp); err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.NotebookId)
	if err := command.AuthZProvider.Get().CanGetNSC(
		ctx, *curUser, model.AccessScopeID(resp.Notebook.WorkspaceId),
	); err != nil {
		return nil, authz.SubIfUnauthorized(err,
			api.NotFoundErrs("actor", fmt.Sprint(addr), true))
	}
	return resp, nil
}

func (a *apiServer) validateToKillNotebook(ctx context.Context, notebookID string) error {
	targetNotebook, err := a.GetNotebook(ctx, &apiv1.GetNotebookRequest{NotebookId: notebookID})
	if err != nil {
		return err
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}

	ctx = audit.SupplyEntityID(ctx, notebookID)
	err = command.AuthZProvider.Get().CanTerminateNSC(
		ctx, *curUser, model.AccessScopeID(targetNotebook.Notebook.WorkspaceId),
	)
	return apiutils.MapAndFilterErrors(err, nil, nil)
}

func (a *apiServer) IdleNotebook(
	ctx context.Context, req *apiv1.IdleNotebookRequest,
) (*apiv1.IdleNotebookResponse, error) {
	err := a.validateToKillNotebook(ctx, req.NotebookId)
	if err != nil {
		return nil, err
	}
	if !req.Idle {
		idle.RecordActivity(req.NotebookId)
	}
	return &apiv1.IdleNotebookResponse{}, nil
}

func (a *apiServer) KillNotebook(
	ctx context.Context, req *apiv1.KillNotebookRequest,
) (resp *apiv1.KillNotebookResponse, err error) {
	err = a.validateToKillNotebook(ctx, req.NotebookId)
	if err != nil {
		return nil, err
	}
	return resp, a.ask(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) SetNotebookPriority(
	ctx context.Context, req *apiv1.SetNotebookPriorityRequest,
) (resp *apiv1.SetNotebookPriorityResponse, err error) {
	targetNotebook, err := a.GetNotebook(ctx, &apiv1.GetNotebookRequest{NotebookId: req.NotebookId})
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.NotebookId)
	err = command.AuthZProvider.Get().CanSetNSCsPriority(
		ctx, *curUser, model.AccessScopeID(targetNotebook.Notebook.WorkspaceId), int(req.Priority),
	)
	if err != nil {
		return nil, apiutils.MapAndFilterErrors(err, nil, nil)
	}

	return resp, a.ask(notebooksAddr.Child(req.NotebookId), req, &resp)
}

// isNTSCPermittedToLaunch checks authorization to launch in a given
// workspace.
func (a *apiServer) isNTSCPermittedToLaunch(
	ctx context.Context, spec *tasks.GenericCommandSpec, user *model.User,
) error {
	workspaceID := spec.Metadata.WorkspaceID
	if workspaceID == 0 {
		return status.Errorf(codes.InvalidArgument, "workspace_id is required")
	}

	w := &workspacev1.Workspace{}
	notFoundErr := api.NotFoundErrs("workspace", fmt.Sprint(workspaceID), true)
	if err := a.m.db.QueryProto(
		"get_workspace", w, workspaceID, user.ID,
	); errors.Is(err, db.ErrNotFound) {
		return notFoundErr
	} else if err != nil {
		return errors.Wrapf(err, "error fetching workspace (%d) from database", workspaceID)
	}
	if w.Archived {
		return notFoundErr
	}

	if spec.TaskType == model.TaskTypeTensorboard {
		if err := command.AuthZProvider.Get().CanGetTensorboard(
			ctx, *user, workspaceID, spec.Metadata.ExperimentIDs, spec.Metadata.TrialIDs,
		); err != nil {
			return authz.SubIfUnauthorized(err, apiutils.MapAndFilterErrors(err, nil, nil))
		}
	} else {
		if err := command.AuthZProvider.Get().CanCreateNSC(
			ctx, *user, workspaceID); err != nil {
			return apiutils.MapAndFilterErrors(err, nil, nil)
		}
	}

	return nil
}

func (a *apiServer) LaunchNotebook(
	ctx context.Context, req *apiv1.LaunchNotebookRequest,
) (*apiv1.LaunchNotebookResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	spec, launchWarnings, err := a.getCommandLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		WorkspaceID:  req.WorkspaceId,
		Config:       req.Config,
		Files:        req.Files,
	}, user)
	if err != nil {
		return nil, api.WrapWithFallbackCode(err, codes.InvalidArgument,
			"failed to prepare launch params")
	}

	if err = a.isNTSCPermittedToLaunch(ctx, spec, user); err != nil {
		return nil, err
	}

	spec.WatchProxyIdleTimeout = true
	spec.WatchRunnerIdleTimeout = true

	// Postprocess the spec.
	if spec.Config.IdleTimeout == nil && a.m.config.NotebookTimeout != nil {
		spec.Config.IdleTimeout = ptrs.Ptr(model.Duration(
			time.Second * time.Duration(*a.m.config.NotebookTimeout)))
	}
	if spec.Config.Description == "" {
		petName := petname.Generate(expconf.TaskNameGeneratorWords, expconf.TaskNameGeneratorSep)
		spec.Config.Description = fmt.Sprintf("JupyterLab (%s)", petName)
	}

	if req.Preview {
		return &apiv1.LaunchNotebookResponse{
			Notebook: &notebookv1.Notebook{},
			Config:   protoutils.ToStruct(spec.Config),
		}, nil
	}

	// Selecting a random port mitigates the risk of multiple processes binding
	// the same port on an agent in host mode.
	port := getRandomPort(minNotebookPort, maxNotebookPort)
	configBytes, err := json.Marshal(spec.Config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot marshal notebook config: %s", err.Error())
	}
	spec.Base.ExtraEnvVars = map[string]string{
		"NOTEBOOK_PORT":      strconv.Itoa(port),
		"NOTEBOOK_CONFIG":    string(configBytes),
		"NOTEBOOK_IDLE_TYPE": spec.Config.NotebookIdleType,
		"DET_TASK_TYPE":      string(model.TaskTypeNotebook),
	}
	spec.Base.ExtraProxyPorts = append(spec.Base.ExtraProxyPorts, expconf.ProxyPort{
		RawProxyPort:        port,
		RawDefaultServiceID: ptrs.Ptr(true),
	})

	spec.Config.Entrypoint = []string{jupyterEntrypoint}

	if err = check.Validate(spec.Config); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid notebook config: %s", err.Error())
	}

	spec.AdditionalFiles = archive.Archive{
		spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterDir, nil, 0o700, tar.TypeDir),
		spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterConfigDir, nil, 0o700, tar.TypeDir),
		spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterDataDir, nil, 0o700, tar.TypeDir),
		spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterRuntimeDir, nil, 0o700, tar.TypeDir),
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			jupyterEntrypoint,
			etc.MustStaticFile(etc.NotebookEntrypointResource),
			0o700,
			tar.TypeReg,
		),
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			jupyterIdleCheck,
			etc.MustStaticFile(etc.NotebookIdleCheckResource),
			0o700,
			tar.TypeReg,
		),
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			taskReadyCheckLogs,
			etc.MustStaticFile(etc.TaskCheckReadyLogsResource),
			0o700,
			tar.TypeReg,
		),
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			notebookDefaultPage,
			etc.MustStaticFile(etc.NotebookTemplateResource),
			0o644,
			tar.TypeReg,
		),
	}

	// Launch a Notebook actor.
	var notebookID model.TaskID
	if err = a.ask(notebooksAddr, *spec, &notebookID); err != nil {
		return nil, err
	}

	var notebook *notebookv1.Notebook
	if err = a.ask(notebooksAddr.Child(notebookID), &notebookv1.Notebook{}, &notebook); err != nil {
		return nil, err
	}

	return &apiv1.LaunchNotebookResponse{
		Notebook: notebook,
		Config:   protoutils.ToStruct(spec.Config),
		Warnings: pkgCommand.LaunchWarningToProto(launchWarnings),
	}, nil
}
