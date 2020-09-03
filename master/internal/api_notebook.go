package internal

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
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

func logToProtoNotebookLog(log *logger.Entry) *apiv1.NotebookLogsResponse {
	return &apiv1.NotebookLogsResponse{Id: int32(log.ID), Message: log.Message}
}

func fetchCommandLogs(
	eventMgrAddr actor.Address,
	system *actor.System,
) api.FetchLogs {
	return func(req api.LogStreamRequest) ([]*logger.Entry, error) {
		logEntries := make([]*logger.Entry, 0)
		err := api.ActorRequest(system, eventMgrAddr, req, &logEntries)
		return logEntries, err
	}
}

func (a *apiServer) NotebookLogs(
	req *apiv1.NotebookLogsRequest, resp apiv1.Determined_NotebookLogsServer) error {
	// We push off calculating effective offset & limit to the actor to avoid having to synchronize
	// between two actor messages.
	logRequest := api.LogStreamRequest{
		Offset: int(req.Offset),
		Limit:  int(req.Limit),
		Follow: req.Follow,
	}
	eventManagerAddr := actor.Addr("notebooks", req.NotebookId, "events")

	onLogEntry := func(log *logger.Entry) error {
		return resp.Send(logToProtoNotebookLog(log))
	}

	return api.ProcessLogs(logRequest, fetchCommandLogs(eventManagerAddr, a.m.system), onLogEntry)
}
