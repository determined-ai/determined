package internal

import (
	"context"

	"github.com/determined-ai/determined/master/internal/cluster"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/logretention"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) CleanupLogs(
	ctx context.Context, req *apiv1.CleanupLogsRequest,
) (*apiv1.CleanupLogsResponse, error) {
	u, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// Check if the user has permission to update the server config, then they should be able to
	// change the schedule and cleanup logs immediately.
	// TODO: Update to IsClusterAdmin eventually.
	permErr, err := cluster.AuthZProvider.Get().CanUpdateMasterConfig(ctx, u)
	if err != nil {
		return nil, err
	} else if permErr != nil {
		return nil, permErr
	}

	resp := &apiv1.CleanupLogsResponse{}
	resp.RemovedCount, err = logretention.DeleteExpiredTaskLogs(a.m.taskSpec.LogRetentionDays)
	return resp, err
}
