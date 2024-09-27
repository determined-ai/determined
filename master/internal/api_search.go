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
		Tags:                  exp.Labels,
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

func (a *apiServer) GetSearcherEventsV2(
	ctx context.Context, req *apiv2.GetSearcherEventsV2Request,
) (*apiv2.GetSearcherEventsV2Response, error) {
	expReq := apiv1.GetSearcherEventsRequest{
		ExperimentId: req.SearchId,
	}
	expRes, err := a.GetSearcherEvents(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.GetSearcherEventsV2Response{
		SearcherEvents: expRes.SearcherEvents,
	}
	return &res, nil
}

func (a *apiServer) PostSearcherOperationsV2(
	ctx context.Context, req *apiv2.PostSearcherOperationsV2Request,
) (*apiv2.PostSearcherOperationsV2Response, error) {
	expReq := apiv1.PostSearcherOperationsRequest{
		ExperimentId: req.SearchId,
	}
	_, err := a.PostSearcherOperations(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.PostSearcherOperationsV2Response{}
	return &res, nil
}

func (a *apiServer) PutSearchRetainLogs(
	ctx context.Context, req *apiv2.PutSearchRetainLogsRequest,
) (*apiv2.PutSearchRetainLogsResponse, error) {
	expReq := apiv1.PutExperimentRetainLogsRequest{
		ExperimentId: req.SearchId,
		NumDays:      req.NumDays,
	}
	_, err := a.PutExperimentRetainLogs(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.PutSearchRetainLogsResponse{}
	return &res, nil
}

func bulkFiltersSearchToExperiment(filter *apiv2.BulkSearchFilters) *apiv1.BulkExperimentFilters {
	states := []experimentv1.State{}
	for _, e := range filter.States {
		states = append(states, experimentv1.State(e))
	}
	return &apiv1.BulkExperimentFilters{
		Description:           filter.Description,
		Name:                  filter.Name,
		Labels:                filter.Labels,
		Archived:              filter.Archived,
		States:                states,
		UserIds:               filter.UserIds,
		ProjectId:             filter.ProjectId,
		ExcludedExperimentIds: filter.ExcludedSearchIds,
	}
}

func (a *apiServer) ResumeSearch(ctx context.Context, req *apiv2.ResumeSearchRequest,
) (*apiv2.ResumeSearchResponse, error) {
	expReq := apiv1.ActivateExperimentRequest{
		Id: req.Id,
	}
	_, err := a.ActivateExperiment(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.ResumeSearchResponse{}
	return &res, nil
}

func (a *apiServer) ResumeSearches(ctx context.Context, req *apiv2.ResumeSearchesRequest,
) (*apiv2.ResumeSearchesResponse, error) {
	expReq := apiv1.ActivateExperimentsRequest{
		ExperimentIds: req.SearchIds,
		Filters:       bulkFiltersSearchToExperiment(req.Filters),
		ProjectId:     req.ProjectId,
	}
	expRes, err := a.ActivateExperiments(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.ResumeSearchesResponse{
		Results: expRes.Results,
	}
	return &res, nil
}

func (a *apiServer) PauseSearch(ctx context.Context, req *apiv2.PauseSearchRequest,
) (*apiv2.PauseSearchResponse, error) {
	expReq := apiv1.PauseExperimentRequest{
		Id: req.Id,
	}
	_, err := a.PauseExperiment(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.PauseSearchResponse{}
	return &res, nil
}

func (a *apiServer) PauseSearches(ctx context.Context, req *apiv2.PauseSearchesRequest,
) (*apiv2.PauseSearchesResponse, error) {
	expReq := apiv1.PauseExperimentsRequest{
		ExperimentIds: req.SearchIds,
		Filters:       bulkFiltersSearchToExperiment(req.Filters),
		ProjectId:     req.ProjectId,
	}
	expRes, err := a.PauseExperiments(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.PauseSearchesResponse{
		Results: expRes.Results,
	}
	return &res, nil
}

func (a *apiServer) CancelSearch(ctx context.Context, req *apiv2.CancelSearchRequest,
) (*apiv2.CancelSearchResponse, error) {
	expReq := apiv1.CancelExperimentRequest{
		Id: req.Id,
	}
	_, err := a.CancelExperiment(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.CancelSearchResponse{}
	return &res, nil
}

func (a *apiServer) CancelSearches(ctx context.Context, req *apiv2.CancelSearchesRequest,
) (*apiv2.CancelSearchesResponse, error) {
	expReq := apiv1.CancelExperimentsRequest{
		ExperimentIds: req.SearchIds,
		Filters:       bulkFiltersSearchToExperiment(req.Filters),
		ProjectId:     req.ProjectId,
	}
	expRes, err := a.CancelExperiments(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.CancelSearchesResponse{
		Results: expRes.Results,
	}
	return &res, nil
}

func (a *apiServer) KillSearch(ctx context.Context, req *apiv2.KillSearchRequest,
) (*apiv2.KillSearchResponse, error) {
	expReq := apiv1.KillExperimentRequest{
		Id: req.Id,
	}
	_, err := a.KillExperiment(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.KillSearchResponse{}
	return &res, nil
}

func (a *apiServer) KillSearches(ctx context.Context, req *apiv2.KillSearchesRequest,
) (*apiv2.KillSearchesResponse, error) {
	expReq := apiv1.KillExperimentsRequest{
		ExperimentIds: req.SearchIds,
		Filters:       bulkFiltersSearchToExperiment(req.Filters),
		ProjectId:     req.ProjectId,
	}
	expRes, err := a.KillExperiments(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.KillSearchesResponse{
		Results: expRes.Results,
	}
	return &res, nil
}

func (a *apiServer) ArchiveSearch(ctx context.Context, req *apiv2.ArchiveSearchRequest,
) (*apiv2.ArchiveSearchResponse, error) {
	expReq := apiv1.ArchiveExperimentRequest{
		Id: req.Id,
	}
	_, err := a.ArchiveExperiment(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.ArchiveSearchResponse{}
	return &res, nil
}

func (a *apiServer) ArchiveSearches(ctx context.Context, req *apiv2.ArchiveSearchesRequest,
) (*apiv2.ArchiveSearchesResponse, error) {
	expReq := apiv1.ArchiveExperimentsRequest{
		ExperimentIds: req.SearchIds,
		Filters:       bulkFiltersSearchToExperiment(req.Filters),
		ProjectId:     req.ProjectId,
	}
	expRes, err := a.ArchiveExperiments(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.ArchiveSearchesResponse{
		Results: expRes.Results,
	}
	return &res, nil
}

func (a *apiServer) UnarchiveSearch(ctx context.Context, req *apiv2.UnarchiveSearchRequest,
) (*apiv2.UnarchiveSearchResponse, error) {
	expReq := apiv1.UnarchiveExperimentRequest{
		Id: req.Id,
	}
	_, err := a.UnarchiveExperiment(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.UnarchiveSearchResponse{}
	return &res, nil
}

func (a *apiServer) UnarchiveSearches(ctx context.Context, req *apiv2.UnarchiveSearchesRequest,
) (*apiv2.UnarchiveSearchesResponse, error) {
	expReq := apiv1.UnarchiveExperimentsRequest{
		ExperimentIds: req.SearchIds,
		Filters:       bulkFiltersSearchToExperiment(req.Filters),
		ProjectId:     req.ProjectId,
	}
	expRes, err := a.UnarchiveExperiments(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.UnarchiveSearchesResponse{
		Results: expRes.Results,
	}
	return &res, nil
}

func (a *apiServer) DeleteSearch(ctx context.Context, req *apiv2.DeleteSearchRequest,
) (*apiv2.DeleteSearchResponse, error) {
	expReq := apiv1.DeleteExperimentRequest{
		ExperimentId: req.SearchId,
	}
	_, err := a.DeleteExperiment(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.DeleteSearchResponse{}
	return &res, nil
}

func (a *apiServer) DeleteSearches(ctx context.Context, req *apiv2.DeleteSearchesRequest,
) (*apiv2.DeleteSearchesResponse, error) {
	expReq := apiv1.DeleteExperimentsRequest{
		ExperimentIds: req.SearchIds,
		Filters:       bulkFiltersSearchToExperiment(req.Filters),
		ProjectId:     req.ProjectId,
	}
	expRes, err := a.DeleteExperiments(ctx, &expReq)
	if err != nil {
		return nil, err
	}
	res := apiv2.DeleteSearchesResponse{
		Results: expRes.Results,
	}
	return &res, nil
}
