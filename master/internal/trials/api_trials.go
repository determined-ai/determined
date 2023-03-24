package trials

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// TrialsAPIServer is an embedded api server struct.
type TrialsAPIServer struct{}

func checkTrialFiltersEmpty(f *apiv1.TrialFilters) error {
	emptyFilters := status.Error(
		codes.InvalidArgument,
		"at least one filter required",
	)

	if f == nil {
		return emptyFilters
	}

	filtersLength := len(f.ExperimentIds) +
		len(f.ProjectIds) +
		len(f.WorkspaceIds) +
		len(f.TrialIds) +
		len(f.ValidationMetrics) +
		len(f.TrainingMetrics) +
		len(f.Hparams) +
		len(f.Searcher) +
		len(f.UserIds) +
		len(f.Tags) +
		len(f.States) +
		len(f.SearcherMetric)

	if filtersLength == 0 &&
		f.RankWithinExp == nil &&
		f.StartTime == nil && f.EndTime == nil &&
		f.SearcherMetricValue == nil {
		return emptyFilters
	}
	return nil
}

// QueryTrials returns a list of AugmentedTrials filtered according to the
// filters provided.
func (a *TrialsAPIServer) QueryTrials(ctx context.Context,
	req *apiv1.QueryTrialsRequest,
) (*apiv1.QueryTrialsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("error querying for trials %w", err)
	}

	err = checkTrialFiltersEmpty(req.Filters)
	if err != nil {
		return nil, err
	}

	q, err := BuildFilterTrialsQuery(req.Filters, true)
	if err != nil {
		return nil, fmt.Errorf("error querying for trials %w", err)
	}

	q, err = AuthZProvider.Get().AuthFilterTrialsQuery(ctx, curUser, q, false)
	if err != nil {
		return nil, fmt.Errorf("error querying for trials %w", err)
	}

	orderColumn := "trial_id"
	orderDirection := db.SortDirectionAsc
	if req.Sorter != nil {
		orderColumn, err = TrialsColumnForNamespace(req.Sorter.Namespace, req.Sorter.Field)
		if err != nil {
			return nil, fmt.Errorf("error querying for trials, bad order by column %w", err)
		}

		if req.Sorter.OrderBy == apiv1.OrderBy_ORDER_BY_DESC {
			orderDirection = db.SortDirectionDescNullsLast
		}
	}

	if req.Limit == 0 {
		req.Limit = 1000
	}

	q = db.PaginateBunUnsafe(
		q,
		orderColumn,
		orderDirection,
		int(req.Offset),
		int(req.Limit),
	)

	trials := []TrialsAugmented{}
	err = q.Scan(ctx, &trials)
	if err != nil {
		return nil, fmt.Errorf("error querying for trials %w", err)
	}

	resp := apiv1.QueryTrialsResponse{Trials: []*apiv1.AugmentedTrial{}}
	for _, trial := range trials {
		resp.Trials = append(resp.Trials, trial.Proto())
	}

	return &resp, nil
}

// UpdateTrialTags patches a target set of trials, specified either by a list
// of trial ids, or a set of filters, according to the provided patch.
func (a *TrialsAPIServer) UpdateTrialTags(ctx context.Context,
	req *apiv1.UpdateTrialTagsRequest,
) (*apiv1.UpdateTrialTagsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update trial tags %w", err)
	}

	q, err := BuildTrialPatchQuery(req.Patch)
	if err != nil {
		return nil, fmt.Errorf("failed to construct set clause for trial tags %w", err)
	}

	var subQ *bun.SelectQuery
	switch targetType := req.Target.(type) {
	case *apiv1.UpdateTrialTagsRequest_Filters:
		filters := req.GetFilters()

		err = checkTrialFiltersEmpty(filters)
		if err != nil {
			return nil, err
		}

		subQ, err = BuildFilterTrialsQuery(filters, false)
		if err != nil {
			return nil, fmt.Errorf("failed to update trial tags %w", err)
		}

		subQ, err = AuthZProvider.Get().AuthFilterTrialsQuery(ctx, curUser, subQ, true)
		if err != nil {
			return nil, fmt.Errorf("failed to update trial tags %w", err)
		}

	case *apiv1.UpdateTrialTagsRequest_Trial:
		trialIds := req.GetTrial().Ids
		if len(trialIds) == 0 {
			return nil, fmt.Errorf("no trial ids provided to update trial tags")
		}

		subQ = db.Bun().NewSelect().Model((*TrialsAugmented)(nil)).
			Where("trial_id in (?)", bun.In(trialIds))
		subQ, err = AuthZProvider.Get().AuthFilterTrialsQuery(ctx, curUser, subQ, true)
		if err != nil {
			return nil, fmt.Errorf("failed to update trial tags %w", err)
		}

	default:
		return nil, fmt.Errorf("bad target for trials patch %f", targetType)
	}

	subQ.Column("trial_id")
	q.Where("id IN (?)", subQ)
	res, err := q.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update trial tags %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Warn("unable to determined number of rows affected")
		rowsAffected = 0
	}

	return &apiv1.UpdateTrialTagsResponse{RowsAffected: int32(rowsAffected)}, nil
}

// GetTrialsCollections returns the list of collections for the (optionally) specified project.
func (a *TrialsAPIServer) GetTrialsCollections(
	ctx context.Context, req *apiv1.GetTrialsCollectionsRequest,
) (*apiv1.GetTrialsCollectionsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get trials collections %w", err)
	}

	collections := []*TrialsCollection{}
	q := db.Bun().
		NewSelect().
		Model(&collections)
	if req.ProjectId != 0 {
		q = q.Where("project_id = ?", req.ProjectId)
	}
	q, err = AuthZProvider.Get().AuthFilterCollectionsReadQuery(ctx, curUser, q)
	if err != nil {
		return nil, fmt.Errorf("failed to get trials collections %w", err)
	}

	err = q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get trials collections %w", err)
	}

	resp := &apiv1.GetTrialsCollectionsResponse{
		Collections: []*apiv1.TrialsCollection{},
	}
	for _, c := range collections {
		resp.Collections = append(resp.Collections, c.Proto())
	}
	return resp, nil
}

// CreateTrialsCollection creates a collection based on the provided
// name, filters, project, and sorter.
func (a *TrialsAPIServer) CreateTrialsCollection(
	ctx context.Context, req *apiv1.CreateTrialsCollectionRequest,
) (*apiv1.CreateTrialsCollectionResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create trials collection %w", err)
	}

	err = checkTrialFiltersEmpty(req.Filters)
	if err != nil {
		return nil, err
	}

	if req.ProjectId == 0 {
		return nil, errors.New("failed to create trials collection: must specify project_id")
	}
	canCreate, err := AuthZProvider.Get().CanCreateTrialCollection(ctx, curUser, req.ProjectId)
	if !canCreate {
		return nil, status.Error(
			codes.PermissionDenied,
			"unable to create collection",
		)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to create collection %w", err)
	}

	collection := TrialsCollection{
		UserID:    int32(curUser.ID),
		Name:      req.Name,
		ProjectID: req.ProjectId,
		Filters:   req.Filters,
		Sorter:    req.Sorter,
	}
	_, err = db.Bun().NewInsert().
		Model(&collection).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("error in creating collection %w", err)
	}

	resp := &apiv1.CreateTrialsCollectionResponse{Collection: collection.Proto()}
	return resp, nil
}

// PatchTrialsCollection patches a collection based on the (optionally) provided
// name, filters, and sorter.
func (a *TrialsAPIServer) PatchTrialsCollection(
	ctx context.Context, req *apiv1.PatchTrialsCollectionRequest,
) (*apiv1.PatchTrialsCollectionResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to patch trials collection %w", err)
	}

	collection := TrialsCollection{
		ID:      req.Id,
		Name:    req.Name,
		Filters: req.Filters,
		Sorter:  req.Sorter,
	}
	q := db.Bun().NewUpdate().
		Model(&collection).
		Returning("*").
		WherePK()

	q, err = AuthZProvider.Get().AuthFilterCollectionsUpdateQuery(ctx, curUser, q)
	if err != nil {
		return nil, fmt.Errorf("failed to patch trials collection %w", err)
	}

	if req.Name != "" {
		q.Column("name")
	}
	if req.Filters != nil {
		q.Column("filters")
	}
	if req.Sorter != nil {
		q.Column("sorter")
	}

	_, err = q.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to patch trials collection %w", err)
	}

	resp := &apiv1.PatchTrialsCollectionResponse{Collection: collection.Proto()}
	return resp, nil
}

// DeleteTrialsCollection deletes the specified collection.
func (a *TrialsAPIServer) DeleteTrialsCollection(
	ctx context.Context, req *apiv1.DeleteTrialsCollectionRequest,
) (*apiv1.DeleteTrialsCollectionResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to delete trials collection %w", err)
	}

	collection := TrialsCollection{
		ID: req.Id,
	}
	q := db.Bun().NewDelete().
		Model(&collection).
		WherePK()

	q, err = AuthZProvider.Get().AuthFilterCollectionsDeleteQuery(ctx, curUser, q)
	if err != nil {
		return nil, fmt.Errorf("failed to delete trials collection %w", err)
	}

	_, err = q.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to delete trials collection %w", err)
	}

	return &apiv1.DeleteTrialsCollectionResponse{}, nil
}
