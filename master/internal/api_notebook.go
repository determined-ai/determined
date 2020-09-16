package internal

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/pkg/errors"
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

func (a *apiServer) NotebookLogs(
	req *apiv1.NotebookLogsRequest, resp apiv1.Determined_NotebookLogsServer) error {
	if err := grpc.ValidateRequest(
		grpc.ValidateLimit(req.Limit),
	); err != nil {
		return err
	}

	cmdManagerAddr := actor.Addr("notebooks", req.NotebookId)
	eventManagerAddr := cmdManagerAddr.Child("events")

	// TODO create an actor (logActor)
	// register actor using logReq and context with eventManager
	// block while the actor is running.
	// Q how does the actor shutdown? check context, and termination?
	// receive:
	//   logmessages: push through the context
	//   unregistering: command and event mgr

	// TODO a general subscriber management for actors (for events and commands actors):
	// 1. add subscriber with parameters (logrequest, identifier)
	// 2. remove subscribers
	// 3. find active and matching subscribers for termination and log events
	//   a. publish to subscribers based on parameters for: new events and termination

	logRequest := api.LogsRequest{
		Offset: int(req.Offset),
		Limit:  int(req.Limit),
		Follow: req.Follow,
	}

	onLogEntry := func(log *logger.Entry) error {
		return resp.Send(&apiv1.NotebookLogsResponse{LogEntry: api.LogEntryToProtoLogEntry(log)})
	}

	streamID := rand.Int() // FIXME
	logStreamActorAddr := cmdManagerAddr.Child("logStream" + string(streamID))
	logStreamActor, created := a.m.system.ActorOf(
		logStreamActorAddr,
		api.NewCommandLogStreamActor(
			resp.Context(),
			a.m.system.Get(eventManagerAddr),
			logRequest,
			onLogEntry,
		),
	)

	if !created {
		// either there is a collision in actor address or actor creation failed.
		return errors.New("failed to create actor")
	}

	// We delegate checking for context closure to logStreamActor accepting that
	// it could stay up if there are no log messages coming in instead of busy checking
	// to kill the actor.
	return logStreamActor.AwaitTermination()
}
