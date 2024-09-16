package internal

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/job/jobservice"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv2"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/jobv2"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
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
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	resp := &apiv2.GetSearchTagsResponse{}
	var tags [][]string
	query := db.Bun().NewSelect().
		Table("experiments").
		Model(&tags).
		ColumnExpr("config->'labels' AS labels").
		Distinct()

	var proj *projectv1.Project
	if req.ProjectId != 0 {
		proj, err = a.GetProjectByID(ctx, req.ProjectId, *curUser)
		if err != nil {
			return nil, err
		}

		query = query.Where("project_id = ?", req.ProjectId)
	}

	if query, err = experiment.AuthZProvider.Get().
		FilterExperimentLabelsQuery(ctx, *curUser, proj, query); err != nil {
		return nil, err
	}

	if err = query.Scan(ctx); err != nil {
		return nil, err
	}

	// Sort tags by usage.
	tagUsage := make(map[string]int)
	for _, tagArr := range tags {
		for _, l := range tagArr {
			tagUsage[l]++
		}
	}

	resp.Tags = make([]string, len(tagUsage))
	i := 0
	for label := range tagUsage {
		resp.Tags[i] = label
		i++
	}
	sort.Slice(resp.Tags, func(i, j int) bool {
		return tagUsage[resp.Tags[i]] > tagUsage[resp.Tags[j]]
	})
	return resp, nil
}

func (a *apiServer) PutSearchTag(
	ctx context.Context, req *apiv2.PutSearchTagRequest,
) (*apiv2.PutSearchTagResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.WithError(err).Error("error rolling back transaction in create workspace")
		}
	}()

	exp := &experimentv1.Experiment{}
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments as e").
		Model(exp).
		Apply(getExperimentColumns).
		Where("e.id = ?", req.SearchId)
	if err = query.Scan(ctx); err != nil {
		return nil, err
	}
	modelExp, err := model.ExperimentFromProto(exp)
	if err != nil {
		return nil, err
	}

	if err = experiment.AuthZProvider.Get().CanEditExperimentsMetadata(
		ctx, *curUser, modelExp); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	if slices.Contains(exp.Labels, req.Tag) {
		return &apiv2.PutSearchTagResponse{Tags: exp.Labels}, nil
	}
	exp.Labels = append(exp.Labels, req.Tag)

	_, err = tx.NewUpdate().Model(modelExp).
		Set("config = jsonb_set(config, '{labels}', ?, true)", exp.Labels).
		Where("id = ?", exp.Id).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("error updating experiment %v in database %w", exp.Id, err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit patch experiment tags transaction %w", err)
	}

	return &apiv2.PutSearchTagResponse{Tags: exp.Labels}, nil
}

func (a *apiServer) DeleteSearchTag(
	ctx context.Context, req *apiv2.DeleteSearchTagRequest,
) (*apiv2.DeleteSearchTagResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.WithError(err).Error("error rolling back transaction in create workspace")
		}
	}()

	exp := &experimentv1.Experiment{}
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments as e").
		Model(exp).
		Apply(getExperimentColumns).
		Where("e.id = ?", req.SearchId)
	if err = query.Scan(ctx); err != nil {
		return nil, err
	}

	modelExp, err := model.ExperimentFromProto(exp)
	if err != nil {
		return nil, err
	}

	if err = experiment.AuthZProvider.Get().CanEditExperimentsMetadata(
		ctx, *curUser, modelExp); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	i := slices.Index(exp.Labels, req.Tag)
	if i == -1 {
		return &apiv2.DeleteSearchTagResponse{Tags: exp.Labels}, nil
	}
	exp.Labels = slices.Delete(exp.Labels, i, i+1)

	_, err = tx.NewUpdate().Model(modelExp).
		Set("config = jsonb_set(config, '{labels}', ?, true)", exp.Labels).
		Where("id = ?", exp.Id).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("error updating experiment %v in database: %w", exp.Id, err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit delete experiment tags transaction: %w", err)
	}

	return &apiv2.DeleteSearchTagResponse{Tags: exp.Labels}, nil
}
