package internal

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/job/jobservice"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/apiv2"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/jobv2"
	"github.com/determined-ai/determined/proto/pkg/searchv2"
)

func convertExpToSearch(exp *experimentv1.Experiment) *searchv2.Search {
	return &searchv2.Search{
		Id:                    exp.Id,
		Description:           exp.Description,
		Labels:                exp.Labels,
		StartTime:             exp.StartTime,
		EndTime:               exp.EndTime,
		State:                 searchv2.State(exp.State),
		Archived:              exp.Archived,
		NumRuns:               exp.NumTrials,
		RunIds:                exp.TrialIds,
		DisplayName:           exp.DisplayName,
		UserId:                exp.UserId,
		Username:              exp.Username,
		ResourcePool:          exp.ResourcePool,
		SearcherType:          exp.SearcherType,
		SearcherMetric:        exp.SearcherMetric,
		Hyperparameters:       exp.Hyperparameters,
		Name:                  exp.Name,
		Notes:                 exp.Notes,
		JobId:                 exp.JobId,
		ForkedFrom:            exp.ForkedFrom,
		Progress:              exp.Progress,
		ProjectId:             exp.ProjectId,
		ProjectName:           exp.ProjectName,
		WorkspaceId:           exp.WorkspaceId,
		WorkspaceName:         exp.WorkspaceName,
		ParentArchived:        exp.ParentArchived,
		Config:                exp.Config, //nolint:staticcheck
		OriginalConfig:        exp.OriginalConfig,
		ProjectOwnerId:        exp.ProjectOwnerId,
		CheckpointSize:        exp.CheckpointSize,
		CheckpointCount:       exp.CheckpointCount,
		BestRunSearcherMetric: exp.BestTrialSearcherMetric,
		BestRunId:             exp.BestTrialId,
		Unmanaged:             exp.Unmanaged,
		Duration:              exp.Duration,
		ExternalSearchId:      exp.ExternalExperimentId,
		ExternalRunId:         exp.ExternalTrialId,
		ModelDefinitionSize:   exp.ModelDefinitionSize,
		PachydermIntegration:  exp.PachydermIntegration,
	}
}

func upgradeJobSummaryVersion(v1 *jobv1.JobSummary) *jobv2.JobSummary {
	return &jobv2.JobSummary{
		State:     jobv2.State(v1.State),
		JobsAhead: v1.JobsAhead,
	}
}

func (a *apiServer) GetSearch(
	ctx context.Context, req *apiv2.GetSearchRequest,
) (*apiv2.GetSearchResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}
	exp, err := a.getExperiment(ctx, *user, int(req.SearchId))
	if err != nil {
		return nil, err
	}

	search := convertExpToSearch(exp)

	// Update this when we remove the proto type.
	resp := apiv2.GetSearchResponse{
		Search: search,
		Config: search.Config, //nolint:staticcheck
	}

	// Only continue to add a job summary if it's an active search.
	if !isActiveExperimentState(exp.State) {
		return &resp, nil
	}

	jobID := model.JobID(exp.JobId)
	jobSummary, err := jobservice.DefaultService.GetJobSummary(jobID, rm.ResourcePoolName(exp.ResourcePool))
	if err != nil {
		// An error here either is real or just that the experiment was not yet terminal in the DB
		// when we first queried it but was by the time it got around to handling out ask. We can't
		// just refresh our DB state to see which it was, since there is a time between an actor
		// closing and PostStop (where the DB state is set) being received where the actor may not
		// respond but still is not terminal -- more clearly, there is a time where the actor is
		// truly non-terminal and not reachable. We _could_ await its stop and recheck, but it's not
		// easy deducible how long that would block. So the best we can really do is return without
		// an error if we're in this case and log. This is a debug log because of how often the
		// happens when polling for an experiment to end.
		if !strings.Contains(err.Error(), sproto.ErrJobNotFound(jobID).Error()) {
			return nil, err
		}
		log.WithError(err).Debugf("asking for job summary")
	} else {
		resp.JobSummary = upgradeJobSummaryVersion(jobSummary)
	}

	return &resp, nil
}

func (a *apiServer) GetSearchTags(
	ctx context.Context, req *apiv2.GetSearchTagsRequest,
) (*apiv2.GetSearchTagsResponse, error) {
	expReq := apiv1.GetExperimentLabelsRequest{
		ProjectId: req.ProjectId,
	}
	expRes, err := a.GetExperimentLabels(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.GetSearchTagsResponse{
		Tags: expRes.Labels,
	}
	return &res, nil
}

func (a *apiServer) PutSearchTag(
	ctx context.Context, req *apiv2.PutSearchTagRequest,
) (*apiv2.PutSearchTagResponse, error) {
	expReq := apiv1.PutExperimentLabelRequest{
		ExperimentId: req.SearchId,
		Label:        req.Tag,
	}
	expRes, err := a.PutExperimentLabel(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.PutSearchTagResponse{
		Tags: expRes.Labels,
	}
	return &res, nil
}

func (a *apiServer) DeleteSearchTag(
	ctx context.Context, req *apiv2.DeleteSearchTagRequest,
) (*apiv2.DeleteSearchTagResponse, error) {
	expReq := apiv1.DeleteExperimentLabelRequest{
		ExperimentId: req.SearchId,
		Label:        req.Tag,
	}
	expRes, err := a.DeleteExperimentLabel(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.DeleteSearchTagResponse{
		Tags: expRes.Labels,
	}
	return &res, nil
}
