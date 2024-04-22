package internal

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/exp/maps"
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
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/internal/task/idle"
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
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

const (
	jupyterDir        = "/run/determined/jupyter/"
	jupyterConfigDir  = "/run/determined/jupyter/config"
	jupyterDataDir    = "/run/determined/jupyter/data"
	jupyterRuntimeDir = "/run/determined/jupyter/runtime"
	jupyterEntrypoint = "/run/determined/jupyter/notebook-entrypoint.sh"
	jupyterIdleCheck  = "/run/determined/jupyter/check_idle.py"
	jupyterCertPath   = "/run/determined/jupyter/jupyterCert.pem"
	jupyterKeyPath    = "/run/determined/jupyter/jupyterKey.key"
	// Agent ports 2600 - 3500 are split between TensorBoards, Notebooks, and Shells.
	minNotebookPort     = 2900
	maxNotebookPort     = minNotebookPort + 299
	notebookDefaultPage = "/run/determined/workdir/README.ipynb"
)

func (a *apiServer) GetNotebooks(
	ctx context.Context, req *apiv1.GetNotebooksRequest,
) (*apiv1.GetNotebooksResponse, error) {
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

	resp, err := command.DefaultCmdService.GetNotebooks(req)
	if err != nil {
		return nil, err
	}

	limitedScopes, err := command.AuthZProvider.Get().AccessibleScopes(
		ctx, *curUser, model.AccessScopeID(req.WorkspaceId),
	)
	if err != nil {
		return nil, apiutils.MapAndFilterErrors(err, nil, nil)
	}
	api.Where(&resp.Notebooks, func(i int) bool {
		return limitedScopes[model.AccessScopeID(resp.Notebooks[i].WorkspaceId)]
	})

	api.Sort(resp.Notebooks, req.OrderBy, req.SortBy, apiv1.GetNotebooksRequest_SORT_BY_ID)
	return resp, api.Paginate(&resp.Pagination, &resp.Notebooks, req.Offset, req.Limit)
}

func (a *apiServer) GetNotebook(
	ctx context.Context, req *apiv1.GetNotebookRequest,
) (*apiv1.GetNotebookResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := command.DefaultCmdService.GetNotebook(req)
	if err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.NotebookId)
	if err := command.AuthZProvider.Get().CanGetNSC(
		ctx, *curUser, model.AccessScopeID(resp.Notebook.WorkspaceId),
	); err != nil {
		return nil, authz.SubIfUnauthorized(err,
			api.NotFoundErrs("notebook", req.NotebookId, true))
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
	cmd, err := command.DefaultCmdService.KillNTSC(req.NotebookId, model.TaskTypeNotebook)
	if err != nil {
		return nil, err
	}
	return &apiv1.KillNotebookResponse{Notebook: cmd.ToV1Notebook()}, nil
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

	cmd, err := command.DefaultCmdService.SetNTSCPriority(req.NotebookId, int(req.Priority), model.TaskTypeNotebook)
	if err != nil {
		return nil, err
	}

	return &apiv1.SetNotebookPriorityResponse{Notebook: cmd.ToV1Notebook()}, nil
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
			var pdErr authz.PermissionDeniedError
			if errors.As(err, &pdErr) {
				for _, perm := range pdErr.RequiredPermissions {
					if perm == rbacv1.PermissionType_PERMISSION_TYPE_VIEW_WORKSPACE {
						return apiutils.ErrNotFound
					}
				}
			}

			return apiutils.MapAndFilterErrors(err, nil, nil)
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
	user, session, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	launchReq, launchWarnings, err := a.getCommandLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		WorkspaceID:  req.WorkspaceId,
		Config:       req.Config,
		Files:        req.Files,
	}, user)
	if err != nil {
		return nil, api.WrapWithFallbackCode(err, codes.InvalidArgument,
			"failed to prepare launch params")
	}

	if err = a.isNTSCPermittedToLaunch(ctx, launchReq.Spec, user); err != nil {
		return nil, err
	}

	notebookKey, notebookCert, err := proxy.GenSignedCert()
	if err != nil {
		return nil, err
	}

	launchReq.Spec.WatchProxyIdleTimeout = true
	launchReq.Spec.WatchRunnerIdleTimeout = true

	// Postprocess the launchReq.Spec.
	if launchReq.Spec.Config.IdleTimeout == nil && a.m.config.NotebookTimeout != nil {
		launchReq.Spec.Config.IdleTimeout = ptrs.Ptr(model.Duration(
			time.Second * time.Duration(*a.m.config.NotebookTimeout)))
	}
	if launchReq.Spec.Config.Description == "" {
		petName := petname.Generate(expconf.TaskNameGeneratorWords, expconf.TaskNameGeneratorSep)
		launchReq.Spec.Config.Description = fmt.Sprintf("JupyterLab (%s)", petName)
	}

	if req.Preview {
		return &apiv1.LaunchNotebookResponse{
			Notebook: &notebookv1.Notebook{},
			Config:   protoutils.ToStruct(launchReq.Spec.Config),
		}, nil
	}

	// Selecting a random port mitigates the risk of multiple processes binding
	// the same port on an agent in host mode.
	port := getRandomPort(minNotebookPort, maxNotebookPort)
	configBytes, err := json.Marshal(launchReq.Spec.Config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot marshal notebook config: %s", err.Error())
	}

	launchReq.Spec.Base.ExtraEnvVars = map[string]string{
		"NOTEBOOK_PORT":      strconv.Itoa(port),
		"NOTEBOOK_CONFIG":    string(configBytes),
		"NOTEBOOK_IDLE_TYPE": launchReq.Spec.Config.NotebookIdleType,
		"DET_TASK_TYPE":      string(model.TaskTypeNotebook),
	}

	OIDCPachydermEnvVars, err := a.getOIDCPachydermEnvVars(session)
	if err != nil {
		return nil, err
	}
	maps.Copy(launchReq.Spec.Base.ExtraEnvVars, OIDCPachydermEnvVars)

	launchReq.Spec.Base.ExtraProxyPorts = append(launchReq.Spec.Base.ExtraProxyPorts,
		expconf.ProxyPort{
			RawProxyPort:        port,
			RawDefaultServiceID: ptrs.Ptr(true),
		})

	launchReq.Spec.Config.Entrypoint = []string{jupyterEntrypoint}

	if err = check.Validate(launchReq.Spec.Config); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid notebook config: %s", err.Error())
	}

	launchReq.Spec.AdditionalFiles = archive.Archive{
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterDir, nil, 0o700, tar.TypeDir),
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterConfigDir, nil, 0o700, tar.TypeDir),
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterDataDir, nil, 0o700, tar.TypeDir),
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterRuntimeDir, nil, 0o700, tar.TypeDir),
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(
			jupyterEntrypoint,
			etc.MustStaticFile(etc.NotebookEntrypointResource),
			0o700,
			tar.TypeReg,
		),
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(
			jupyterIdleCheck,
			etc.MustStaticFile(etc.NotebookIdleCheckResource),
			0o700,
			tar.TypeReg,
		),
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(
			taskReadyCheckLogs,
			etc.MustStaticFile(etc.TaskCheckReadyLogsResource),
			0o700,
			tar.TypeReg,
		),
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(
			notebookDefaultPage,
			etc.MustStaticFile(etc.NotebookTemplateResource),
			0o644,
			tar.TypeReg,
		),
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(
			jupyterKeyPath,
			notebookKey,
			0o600,
			tar.TypeReg,
		),
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(
			jupyterCertPath,
			notebookCert,
			0o600,
			tar.TypeReg,
		),
	}

	// Launch a Notebook.
	genericCmd, err := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeNotebook,
		model.JobTypeNotebook,
		launchReq)
	if err != nil {
		return nil, err
	}

	return &apiv1.LaunchNotebookResponse{
		Notebook: genericCmd.ToV1Notebook(),
		Config:   protoutils.ToStruct(launchReq.Spec.Config),
		Warnings: pkgCommand.LaunchWarningToProto(launchWarnings),
	}, nil
}
