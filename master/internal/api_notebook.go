package internal

import (
	"archive/tar"
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
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

var notebooksAddr = actor.Addr("notebooks")

func (a *apiServer) GetNotebooks(
	_ context.Context, req *apiv1.GetNotebooksRequest,
) (resp *apiv1.GetNotebooksResponse, err error) {
	err = a.actorRequest(notebooksAddr, req, &resp)
	if err != nil {
		return nil, err
	}
	a.sort(resp.Notebooks, req.OrderBy, req.SortBy, apiv1.GetNotebooksRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Notebooks, req.Offset, req.Limit)
}

func (a *apiServer) GetNotebook(
	_ context.Context, req *apiv1.GetNotebookRequest) (resp *apiv1.GetNotebookResponse, err error) {
	return resp, a.actorRequest(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) KillNotebook(
	_ context.Context, req *apiv1.KillNotebookRequest) (resp *apiv1.KillNotebookResponse, err error) {
	return resp, a.actorRequest(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) SetNotebookPriority(
	_ context.Context, req *apiv1.SetNotebookPriorityRequest,
) (resp *apiv1.SetNotebookPriorityResponse, err error) {
	return resp, a.actorRequest(notebooksAddr.Child(req.NotebookId), req, &resp)
}

func (a *apiServer) NotebookLogs(
	req *apiv1.NotebookLogsRequest, resp apiv1.Determined_NotebookLogsServer) error {
	if err := grpcutil.ValidateRequest(
		grpcutil.ValidateLimit(req.Limit),
	); err != nil {
		return err
	}

	cmdManagerAddr := notebooksAddr.Child(req.NotebookId)
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

	// Postprocess the spec.
	if spec.Config.Description == "" {
		petName := petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep)
		spec.Config.Description = fmt.Sprintf("Notebook (%s)", petName)
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
	spec.Base.ExtraEnvVars = map[string]string{
		"NOTEBOOK_PORT": strconv.Itoa(port),
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
			notebookDefaultPage,
			etc.MustStaticFile(etc.NotebookTemplateResource),
			0644,
			tar.TypeReg,
		),
	}

	// Launch a Notebook actor.
	notebookIDFut := a.m.system.AskAt(notebooksAddr, *spec)
	if err = api.ProcessActorResponseError(&notebookIDFut); err != nil {
		return nil, err
	}

	notebookID := notebookIDFut.Get().(model.TaskID)
	notebook := a.m.system.AskAt(notebooksAddr.Child(notebookID), &notebookv1.Notebook{})
	if err = api.ProcessActorResponseError(&notebook); err != nil {
		return nil, err
	}

	return &apiv1.LaunchNotebookResponse{
		Notebook: notebook.Get().(*notebookv1.Notebook),
		Config:   protoutils.ToStruct(spec.Config),
	}, nil
}
