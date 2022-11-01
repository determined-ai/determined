package internal

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
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

var notebooksAddr = actor.Addr("notebooks")

func (a *apiServer) GetNotebooks(
	ctx context.Context, req *apiv1.GetNotebooksRequest,
) (resp *apiv1.GetNotebooksResponse, err error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	if err = a.ask(notebooksAddr, req, &resp); err != nil {
		return nil, err
	}

	a.filter(&resp.Notebooks, func(i int) bool {
		if err != nil {
			return false
		}
		ok, serverError := user.AuthZProvider.Get().CanAccessNTSCTask(
			ctx, *curUser, model.UserID(resp.Notebooks[i].UserId))
		if serverError != nil {
			err = serverError
		}
		return ok
	})
	if err != nil {
		return nil, err
	}

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

	if ok, err := user.AuthZProvider.Get().CanAccessNTSCTask(
		ctx, *curUser, model.UserID(resp.Notebook.UserId)); err != nil {
		return nil, err
	} else if !ok {
		return nil, errActorNotFound(addr)
	}
	return resp, nil
}

func (a *apiServer) IdleNotebook(
	ctx context.Context, req *apiv1.IdleNotebookRequest,
) (resp *apiv1.IdleNotebookResponse, err error) {
	if _, err := a.GetNotebook(ctx,
		&apiv1.GetNotebookRequest{NotebookId: req.NotebookId}); err != nil {
		return nil, err
	}

	return resp, a.ask(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) KillNotebook(
	ctx context.Context, req *apiv1.KillNotebookRequest,
) (resp *apiv1.KillNotebookResponse, err error) {
	if _, err := a.GetNotebook(ctx,
		&apiv1.GetNotebookRequest{NotebookId: req.NotebookId}); err != nil {
		return nil, err
	}

	return resp, a.ask(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) SetNotebookPriority(
	ctx context.Context, req *apiv1.SetNotebookPriorityRequest,
) (resp *apiv1.SetNotebookPriorityResponse, err error) {
	if _, err := a.GetNotebook(ctx,
		&apiv1.GetNotebookRequest{NotebookId: req.NotebookId}); err != nil {
		return nil, err
	}

	return resp, a.ask(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) LaunchNotebook(
	ctx context.Context, req *apiv1.LaunchNotebookRequest,
) (*apiv1.LaunchNotebookResponse, error) {
	spec, err := a.getCommandLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		Config:       req.Config,
		Files:        req.Files,
	})
	if err != nil {
		return nil, api.APIErrToGRPC(errors.Wrapf(err, "failed to prepare launch params"))
	}

	spec.WatchProxyIdleTimeout = true
	spec.WatchRunnerIdleTimeout = true

	// Postprocess the spec.
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
	spec.Port = &port
	spec.Config.Environment.Ports = map[string]int{"notebook": port}

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
	}, nil
}
