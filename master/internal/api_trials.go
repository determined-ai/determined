package internal

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	batchSize     = 1000
	batchWaitTime = 100 * time.Millisecond
)

func trialStatus(d *db.PgDB, trialID int32) (model.State, int, error) {
	trialStatus := struct {
		State   model.State
		NumLogs int
	}{}
	err := d.Query("trial_status", &trialStatus, trialID)
	if err == db.ErrNotFound {
		err = status.Error(codes.NotFound, "trial not found")
	}
	return trialStatus.State, trialStatus.NumLogs, err
}

func (a *apiServer) TrialLogs(
	req *apiv1.TrialLogsRequest, resp apiv1.Determined_TrialLogsServer) error {
	if err := grpc.ValidateRequest(
		grpc.ValidateLimit(req.Limit),
	); err != nil {
		return err
	}
	_, total, err := trialStatus(a.m.db, req.TrialId)
	if err != nil {
		return err
	}

	offset := effectiveOffset(int(req.Offset), total)

	if limit := int32(total - offset); !req.Follow && (limit < req.Limit || req.Limit == 0) {
		req.Limit = limit
	}
	count := 0
	for {
		queryLimit := int(req.Limit) - count
		if req.Limit == 0 || queryLimit > batchSize {
			queryLimit = batchSize
		}
		if queryLimit <= 0 {
			return nil
		}
		var logs []*apiv1.TrialLogsResponse
		if err := a.m.db.QueryProto("stream_logs", &logs, req.TrialId, offset, queryLimit); err != nil {
			return err
		}
		for i, log := range logs {
			log.Id = int32(offset + i)
			if err := resp.Send(log); err != nil {
				return err
			}
		}
		newRecords := len(logs)
		count += newRecords
		offset += newRecords
		if newRecords < queryLimit {
			state, _, err := trialStatus(a.m.db, req.TrialId)
			if err != nil || model.TerminalStates[state] {
				return err
			}
		}
		time.Sleep(batchWaitTime)
	}
}
