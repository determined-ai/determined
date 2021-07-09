package internal

import (
	"archive/tar"
	"context"
	"fmt"
	"net/http"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/sproto"
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

var notebooksAddr = actor.Addr("notebooks")

func (a *apiServer) GetNotebooks(
	_ context.Context, req *apiv1.GetNotebooksRequest,
) (resp *apiv1.GetNotebooksResponse, err error) {
	err = a.actorRequest("/notebooks", req, &resp)
	if err != nil {
		return nil, err
	}
	a.sort(resp.Notebooks, req.OrderBy, req.SortBy, apiv1.GetNotebooksRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Notebooks, req.Offset, req.Limit)
}

func (a *apiServer) GetNotebook(
	_ context.Context, req *apiv1.GetNotebookRequest) (resp *apiv1.GetNotebookResponse, err error) {
	return resp, a.actorRequest(fmt.Sprintf("/notebooks/%s", req.NotebookId), req, &resp)
}

func (a *apiServer) KillNotebook(
	_ context.Context, req *apiv1.KillNotebookRequest) (resp *apiv1.KillNotebookResponse, err error) {
	return resp, a.actorRequest(fmt.Sprintf("/notebooks/%s", req.NotebookId), req, &resp)
}

func (a *apiServer) NotebookLogs(
	req *apiv1.NotebookLogsRequest, resp apiv1.Determined_NotebookLogsServer) error {
	if err := grpcutil.ValidateRequest(
		grpcutil.ValidateLimit(req.Limit),
	); err != nil {
		return err
	}

	cmdManagerAddr := actor.Addr("notebooks", req.NotebookId)
	eventManager := a.m.system.Get(cmdManagerAddr.Child("events"))

	logRequest := api.BatchRequest{
		Offset: int(req.Offset),
		Limit:  int(req.Limit),
		Follow: req.Follow,
	}

	onBatch := func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			lr := r.(*logger.Entry)
			return resp.Send(&apiv1.NotebookLogsResponse{
				LogEntry: &logv1.LogEntry{Id: int32(lr.ID), Message: lr.Message},
			})
		})
	}

	return a.m.system.MustActorOf(
		cmdManagerAddr.Child("logStream-"+uuid.New().String()),
		api.NewLogStreamProcessor(
			resp.Context(),
			eventManager,
			logRequest,
			onBatch,
		),
	).AwaitTermination()
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
		return nil, api.APIErr2GRPC(errors.Wrapf(err, "failed to prepare launch params"))
	}

	if req.Preview {
		return &apiv1.LaunchNotebookResponse{
			Notebook: &notebookv1.Notebook{},
			Config:   protoutils.ToStruct(spec.Config),
		}, nil
	}

	const (
		jupyterDir        = "/run/determined/jupyter/"
		jupyterConfigDir  = "/run/determined/jupyter/config"
		jupyterDataDir    = "/run/determined/jupyter/data"
		jupyterRuntimeDir = "/run/determined/jupyter/runtime"
		jupyterEntrypoint = "/run/determined/jupyter/notebook-entrypoint.sh"
		// Agent ports 2600 - 3500 are split between TensorBoards, Notebooks, and Shells.
		minNotebookPort     = 2900
		maxNotebookPort     = minNotebookPort + 299
		notebookDefaultPage = "/run/determined/workdir/test.ipynb"
	)
	var (
		notebookEntrypoint = []string{jupyterEntrypoint}
	)

	config := &spec.Config

	// Postprocess the config. Add Jupyter and configuration to the container.

	// Select a random port from the range to assign to the notebook. In host
	// mode, this mitigates the risk of multiple notebook processes binding
	// the same port on an agent.
	port := getPort(minNotebookPort, maxNotebookPort)
	notebookPorts := map[string]int{"notebook": port}
	portVar := fmt.Sprintf("NOTEBOOK_PORT=%d", port)

	config.Environment.Ports = notebookPorts
	config.Environment.EnvironmentVariables.CPU = append(
		config.Environment.EnvironmentVariables.CPU, portVar)
	config.Environment.EnvironmentVariables.GPU = append(
		config.Environment.EnvironmentVariables.GPU, portVar)

	config.Entrypoint = notebookEntrypoint

	if config.Description == "" {
		petName := petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep)
		config.Description = fmt.Sprintf("Notebook (%s)", petName)
	}

	if err = check.Validate(config); err != nil {
		return nil, echo.NewHTTPError(
			http.StatusBadRequest, errors.Wrap(err, "failed to launch notebook").Error())
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
			notebookDefaultPage,
			etc.MustStaticFile(etc.NotebookTemplateResource),
			0644,
			tar.TypeReg,
		),
	}
	spec.Port = &port
	notebookIDFut := a.m.system.AskAt(notebooksAddr, *spec)
	if err = api.ProcessActorResponseError(&notebookIDFut); err != nil {
		return nil, err
	}

	notebookID := notebookIDFut.Get().(sproto.TaskID)
	notebook := a.m.system.AskAt(notebooksAddr.Child(notebookID), &notebookv1.Notebook{})
	if err = api.ProcessActorResponseError(&notebook); err != nil {
		return nil, err
	}

	return &apiv1.LaunchNotebookResponse{
		Notebook: notebook.Get().(*notebookv1.Notebook),
		Config:   protoutils.ToStruct(spec.Config),
	}, nil
}
