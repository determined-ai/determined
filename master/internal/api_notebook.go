package internal

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

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

/* Command Helpers */
// TODO reorder args?
// CHECK parts of this code could live on the command and events actors but the idea is to
// keep the api code out of there.
func fetchCommandLogs(
	eventMgrAddr actor.Address,
	system *actor.System,
) api.LogFetcher {
	return func(req api.LogsRequest) ([]*logger.Entry, error) {
		logEntries := make([]*logger.Entry, 0)
		err := api.ActorRequest(system, eventMgrAddr, req, &logEntries)
		return logEntries, err
	}
}

func commandIsTermianted(
	cmdManagerAddr actor.Address,
	system *actor.System,
) api.TerminationCheck {
	return func() (bool, error) {
		cmd := command.Summary{}
		err := api.ActorRequest(system, cmdManagerAddr, command.GetSummary{}, &cmd)
		isTerminated := cmd.State == container.Terminated.String()
		return isTerminated, err
	}
}

func (a *apiServer) NotebookLogs(
	req *apiv1.NotebookLogsRequest, resp apiv1.Determined_NotebookLogsServer) error {
	if err := grpc.ValidateRequest(
		grpc.ValidateLimit(req.Limit),
	); err != nil {
		return err
	}

	cmdManagerAddr := actor.Addr("notebooks", req.NotebookId)
	eventManagerAddr := cmdManagerAddr.Child("events")

	// CHECK Here we might have a synchronization issue which would only affect negative offsets since
	// those are reliant on the total event number. Still it wouldn't cause any real miscalculations
	// from an external POV would it? same situation with trial logs

	var total int
	err := api.ActorRequest(a.m.system, eventManagerAddr, command.GetEventCount{}, &total)
	if err != nil {
		return errors.Wrapf(err, "failed to get event count from %s actor", eventManagerAddr)
	}

	offset, limit := api.EffectiveOffsetNLimit(int(req.Offset), int(req.Limit), total)

	logRequest := api.LogsRequest{
		Offset: offset,
		Limit:  limit,
		Follow: req.Follow,
	}

	onLogEntry := func(log *logger.Entry) error {
		return resp.Send(&apiv1.NotebookLogsResponse{LogEntry: api.LogEntryToProtoLogEntry(log)})
	}

	terminationCheck := commandIsTermianted(cmdManagerAddr, a.m.system)

	return api.ProcessLogs(
		resp.Context(),
		logRequest,
		fetchCommandLogs(eventManagerAddr, a.m.system),
		onLogEntry,
		&terminationCheck,
	)
}
