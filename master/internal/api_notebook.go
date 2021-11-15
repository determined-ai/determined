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
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/logv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
)

const (
	jupyterDir         = "/run/determined/jupyter/"
	jupyterConfigDir   = "/run/determined/jupyter/config"
	jupyterDataDir     = "/run/determined/jupyter/data"
	jupyterRuntimeDir  = "/run/determined/jupyter/runtime"
	jupyterEntrypoint  = "/run/determined/jupyter/notebook-entrypoint.sh"
	jupyterIdleCheck   = "/run/determined/jupyter/check_idle.py"
	taskReadyCheckLogs = "/run/determined/check_ready_logs.py"
	// Agent ports 2600 - 3500 are split between TensorBoards, Notebooks, and Shells.
	minNotebookPort     = 2900
	maxNotebookPort     = minNotebookPort + 299
	notebookDefaultPage = "/run/determined/workdir/README.ipynb"
)

var notebooksAddr = actor.Addr("notebooks")

func (a *apiServer) GetNotebooks(
	_ context.Context, req *apiv1.GetNotebooksRequest,
) (resp *apiv1.GetNotebooksResponse, err error) {
	if err = a.ask(notebooksAddr, req, &resp); err != nil {
		return nil, err
	}
	a.sort(resp.Notebooks, req.OrderBy, req.SortBy, apiv1.GetNotebooksRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Notebooks, req.Offset, req.Limit)
}

func (a *apiServer) GetNotebook(
	_ context.Context, req *apiv1.GetNotebookRequest) (resp *apiv1.GetNotebookResponse, err error) {
	return resp, a.ask(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) IdleNotebook(
	_ context.Context, req *apiv1.IdleNotebookRequest) (resp *apiv1.IdleNotebookResponse, err error) {
	return resp, a.ask(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) KillNotebook(
	_ context.Context, req *apiv1.KillNotebookRequest) (resp *apiv1.KillNotebookResponse, err error) {
	return resp, a.ask(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) SetNotebookPriority(
	_ context.Context, req *apiv1.SetNotebookPriorityRequest,
) (resp *apiv1.SetNotebookPriorityResponse, err error) {
	return resp, a.ask(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) NotebookLogs(
	req *apiv1.NotebookLogsRequest, resp apiv1.Determined_NotebookLogsServer,
) error {
	if err := grpcutil.ValidateRequest(
		grpcutil.ValidateLimit(req.Limit),
	); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	res := make(chan api.BatchResult, 1)
	go a.taskLogs(ctx, &apiv1.TaskLogsRequest{
		TaskId: req.NotebookId,
		Limit:  req.Limit,
		Follow: req.Follow,
	}, res)

	return processBatches(res, func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			lr := r.(*logger.Entry)
			return resp.Send(&apiv1.NotebookLogsResponse{
				LogEntry: &logv1.LogEntry{Id: int32(lr.ID), Message: lr.Message},
			})
		})
	})
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
		petName := petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep)
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
		"NOTEBOOK_PORT":   strconv.Itoa(port),
		"NOTEBOOK_CONFIG": string(configBytes),
		"DET_TASK_TYPE":   model.TaskTypeNotebook,
	}
	spec.Port = &port
	spec.Config.Environment.Ports = map[string]int{"notebook": port}

	spec.Config.Entrypoint = []string{jupyterEntrypoint}

	if err = check.Validate(spec.Config); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid notebook config: %s", err.Error())
	}

	spec.AdditionalFiles = archive.Archive{
		spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterDir, nil, 0700, tar.TypeDir),
		spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterConfigDir, nil, 0700, tar.TypeDir),
		spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterDataDir, nil, 0700, tar.TypeDir),
		spec.Base.AgentUserGroup.OwnedArchiveItem(jupyterRuntimeDir, nil, 0700, tar.TypeDir),
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			jupyterEntrypoint,
			etc.MustStaticFile(etc.NotebookEntrypointResource),
			0700,
			tar.TypeReg,
		),
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			jupyterIdleCheck,
			etc.MustStaticFile(etc.NotebookIdleCheckResource),
			0700,
			tar.TypeReg,
		),
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			taskReadyCheckLogs,
			etc.MustStaticFile(etc.TaskCheckReadyLogsResource),
			0700,
			tar.TypeReg,
		),
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			notebookDefaultPage,
			etc.MustStaticFile(etc.NotebookTemplateResource),
			0644,
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
