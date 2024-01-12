package internal

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/storage"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) RunPrepareForReport(
	ctx context.Context, req *apiv1.RunPrepareForReportRequest,
) (*apiv1.RunPrepareForReportResponse, error) {
	// TODO(runs) run specific RBAC.
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.RunId),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	var storageID *int32
	if req.CheckpointStorage != nil {
		bytes, err := req.CheckpointStorage.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshaling checkpoint storage %+v: %w", req.CheckpointStorage, err)
		}
		cs := &expconf.CheckpointStorageConfig{} //nolint:exhaustruct
		if err := cs.UnmarshalJSON(bytes); err != nil {
			return nil, fmt.Errorf("unmarshaling json bytes %s: %w", string(bytes), err)
		}

		id, err := storage.AddBackend(ctx, cs)
		if err != nil {
			return nil, fmt.Errorf("adding storage ID for runID %d: %w", req.RunId, err)
		}
		storageID = ptrs.Ptr(int32(id))
	}

	return &apiv1.RunPrepareForReportResponse{
		StorageId: storageID,
	}, nil
}
