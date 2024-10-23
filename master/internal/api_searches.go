package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

type searchCandidateResult struct {
	Archived   bool
	ID         int32
	IsTerminal bool
}

func filterSearchQuery(getQ *bun.SelectQuery, filter *string) (*bun.SelectQuery, error) {
	var efr experimentFilterRoot
	err := json.Unmarshal([]byte(*filter), &efr)
	if err != nil {
		return nil, err
	}
	getQ = getQ.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		_, err = efr.toSQL(q)
		return q
	}).WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		if !efr.ShowArchived {
			return q.Where(`NOT e.archived`)
		}
		return q
	})
	if err != nil {
		return nil, err
	}
	return getQ, nil
}

func getSelectSearchesQueryTables() *bun.SelectQuery {
	return db.Bun().NewSelect().
		ModelTableExpr("experiments AS e").
		Join("JOIN projects p ON e.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id")
}

func (a *apiServer) MoveSearches(
	ctx context.Context, req *apiv1.MoveSearchesRequest,
) (*apiv1.MoveSearchesResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// check that user can view source project
	srcProject, err := a.GetProjectByID(ctx, req.SourceProjectId, *curUser)
	if err != nil {
		return nil, err
	}
	if srcProject.Archived {
		return nil, errors.Errorf("project (%v) is archived and cannot have searches moved from it",
			srcProject.Id)
	}

	// check suitable destination project
	destProject, err := a.GetProjectByID(ctx, req.DestinationProjectId, *curUser)
	if err != nil {
		return nil, err
	}
	if destProject.Archived {
		return nil, errors.Errorf("project (%v) is archived and cannot add new searches",
			req.DestinationProjectId)
	}
	if err = experiment.AuthZProvider.Get().CanCreateExperiment(ctx, *curUser, destProject); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if req.SourceProjectId == req.DestinationProjectId {
		return &apiv1.MoveSearchesResponse{Results: []*apiv1.SearchActionResult{}}, nil
	}

	var searchChecks []searchCandidateResult
	getQ := db.Bun().NewSelect().
		ModelTableExpr("experiments AS e").
		Model(&searchChecks).
		Column("e.id").
		ColumnExpr("COALESCE((e.archived OR p.archived OR w.archived), FALSE) AS archived").
		Join("JOIN projects p ON e.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id").
		Where("e.project_id = ?", req.SourceProjectId)

	if req.Filter == nil {
		getQ = getQ.Where("e.id IN (?)", bun.In(req.SearchIds))
	} else {
		getQ, err = filterSearchQuery(getQ, req.Filter)
		if err != nil {
			return nil, err
		}
	}

	if getQ, err = experiment.AuthZProvider.Get().FilterExperimentsQuery(ctx, *curUser, nil, getQ,
		[]rbacv1.PermissionType{
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA,
			rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT,
		}); err != nil {
		return nil, err
	}

	err = getQ.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []*apiv1.SearchActionResult
	visibleIDs := set.New[int32]()
	var validIDs []int32
	for _, check := range searchChecks {
		visibleIDs.Insert(check.ID)
		if check.Archived {
			results = append(results, &apiv1.SearchActionResult{
				Error: "Search is archived.",
				Id:    check.ID,
			})
			continue
		}
		validIDs = append(validIDs, check.ID)
	}
	if req.Filter == nil {
		for _, originalID := range req.SearchIds {
			if !visibleIDs.Contains(originalID) {
				results = append(results, &apiv1.SearchActionResult{
					Error: fmt.Sprintf("Search with id '%d' not found in project with id '%d'", originalID, req.SourceProjectId),
					Id:    originalID,
				})
			}
		}
	}
	if len(validIDs) > 0 {
		expMoveResults, err := experiment.MoveExperiments(ctx, srcProject.Id, validIDs, nil, req.DestinationProjectId)
		if err != nil {
			return nil, err
		}
		failedExpMoveIds := []int32{-1}
		successExpMoveIds := []int32{-1}
		for _, res := range expMoveResults {
			if res.Error != nil {
				failedExpMoveIds = append(failedExpMoveIds, res.ID)
			} else {
				successExpMoveIds = append(successExpMoveIds, res.ID)
			}
		}

		tx, err := db.Bun().BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer func() {
			txErr := tx.Rollback()
			if txErr != nil && txErr != sql.ErrTxDone {
				log.WithError(txErr).Error("error rolling back transaction in MoveSearches")
			}
		}()

		if err = db.RemoveOutdatedProjectHparams(ctx, tx, int(req.SourceProjectId)); err != nil {
			return nil, err
		}

		var failedSearchIDs []int32
		if err = tx.NewSelect().Table("experiments").
			Column("id").
			Where("experiments.id IN (?)", bun.In(validIDs)).
			Where("experiments.id IN (?)", bun.In(failedExpMoveIds)).
			Scan(ctx, &failedSearchIDs); err != nil {
			return nil, fmt.Errorf("getting failed experiment move IDs: %w", err)
		}
		for _, failedSearchID := range failedSearchIDs {
			results = append(results, &apiv1.SearchActionResult{
				Error: "Failed to move experiment",
				Id:    failedSearchID,
			})
		}

		var successSearchIDs []int32
		if err = tx.NewSelect().Table("experiments").
			Column("id").
			Where("experiments.id IN (?)", bun.In(successExpMoveIds)).
			Scan(ctx, &successSearchIDs); err != nil {
			return nil, fmt.Errorf("getting failed experiment move search IDs: %w", err)
		}
		for _, successSearchID := range successSearchIDs {
			results = append(results, &apiv1.SearchActionResult{
				Error: "",
				Id:    successSearchID,
			})
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
	}
	return &apiv1.MoveSearchesResponse{Results: results}, nil
}

func (a *apiServer) CancelSearches(ctx context.Context, req *apiv1.CancelSearchesRequest,
) (*apiv1.CancelSearchesResponse, error) {
	results, validIDs, err := validateBulkSearchActionCandidates(ctx, req.ProjectId, req.SearchIds, req.Filter)
	if err != nil {
		return nil, err
	}

	for _, searchID := range validIDs {
		// Could use a.CancelExperiments instead, but would still need to
		// iterate over results to build out response structure
		_, err = a.CancelExperiment(ctx, &apiv1.CancelExperimentRequest{
			Id: searchID,
		})
		if err != nil {
			results = append(results, &apiv1.SearchActionResult{
				Error: fmt.Sprintf("Failed to kill search: %s", err),
				Id:    searchID,
			})
		} else {
			results = append(results, &apiv1.SearchActionResult{
				Error: "",
				Id:    searchID,
			})
		}
	}
	return &apiv1.CancelSearchesResponse{Results: results}, nil
}

func (a *apiServer) KillSearches(ctx context.Context, req *apiv1.KillSearchesRequest,
) (*apiv1.KillSearchesResponse, error) {
	results, validIDs, err := validateBulkSearchActionCandidates(ctx, req.ProjectId, req.SearchIds, req.Filter)
	if err != nil {
		return nil, err
	}
	for _, searchID := range validIDs {
		_, err = a.KillExperiment(ctx, &apiv1.KillExperimentRequest{
			Id: searchID,
		})
		if err != nil {
			results = append(results, &apiv1.SearchActionResult{
				Error: fmt.Sprintf("Failed to kill search: %s", err),
				Id:    searchID,
			})
		} else {
			results = append(results, &apiv1.SearchActionResult{
				Error: "",
				Id:    searchID,
			})
		}
	}
	return &apiv1.KillSearchesResponse{Results: results}, nil
}

func (a *apiServer) DeleteSearches(ctx context.Context, req *apiv1.DeleteSearchesRequest,
) (*apiv1.DeleteSearchesResponse, error) {
	if len(req.SearchIds) > 0 && req.Filter != nil {
		return nil, fmt.Errorf("if filter is provided search id list must be empty")
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// get searches to delete
	var deleteCandidates []searchCandidateResult
	getQ := getSelectSearchesQueryTables().
		Model(&deleteCandidates).
		Column("e.id").
		ColumnExpr("COALESCE((e.archived OR p.archived OR w.archived), FALSE) AS archived").
		ColumnExpr("e.state IN (?) AS is_terminal", bun.In(model.StatesToStrings(model.TerminalStates))).
		Where("e.project_id = ?", req.ProjectId)

	if req.Filter == nil {
		getQ = getQ.Where("e.id IN (?)", bun.In(req.SearchIds))
	} else {
		getQ, err = filterSearchQuery(getQ, req.Filter)
		if err != nil {
			return nil, err
		}
	}

	if getQ, err = experiment.AuthZProvider.Get().FilterExperimentsQuery(ctx, *curUser, nil, getQ,
		[]rbacv1.PermissionType{
			rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT,
		}); err != nil {
		return nil, err
	}

	err = getQ.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []*apiv1.SearchActionResult
	visibleIDs := set.New[int32]()
	var validIDs []int32
	for _, cand := range deleteCandidates {
		visibleIDs.Insert(cand.ID)
		if !cand.IsTerminal {
			results = append(results, &apiv1.SearchActionResult{
				Error: "Search is not in a terminal state.",
				Id:    cand.ID,
			})
		} else {
			validIDs = append(validIDs, cand.ID)
		}
	}
	if req.Filter == nil {
		for _, originalID := range req.SearchIds {
			if !visibleIDs.Contains(originalID) {
				results = append(results, &apiv1.SearchActionResult{
					Error: fmt.Sprintf("Search with id '%d' not found in project with id '%d'", originalID, req.ProjectId),
					Id:    originalID,
				})
			}
		}
	}
	if len(validIDs) == 0 {
		return &apiv1.DeleteSearchesResponse{Results: results}, nil
	}
	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		txErr := tx.Rollback()
		if txErr != nil && txErr != sql.ErrTxDone {
			log.WithError(txErr).Error("error rolling back transaction in DeleteSearches")
		}
	}()

	var deleteRunIDs []int32
	getQ = db.Bun().NewSelect().
		TableExpr("runs AS r").
		Model(&deleteRunIDs).
		Column("r.id").
		Where("r.experiment_id IN (?)", bun.In(validIDs))

	err = getQ.Scan(ctx)
	if err != nil {
		return nil, err
	}

	// delete run logs
	if _, err = tx.NewDelete().Table("trial_logs").
		Where("trial_logs.trial_id IN (?)", bun.In(deleteRunIDs)).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete run logs: %w", err)
	}

	// delete task logs
	trialTaskQuery := tx.NewSelect().Table("run_id_task_id").
		ColumnExpr("task_id").
		Where("run_id IN (?)", bun.In(deleteRunIDs))
	if _, err = tx.NewDelete().Table("task_logs").
		Where("task_logs.task_id IN (?)", trialTaskQuery).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete task logs: %w", err)
	}

	// delete runs
	if _, err = tx.NewDelete().Table("runs").
		Where("runs.id IN (?)", bun.In(deleteRunIDs)).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete runs: %w", err)
	}

	var acceptedIDs []int
	if _, err = tx.NewDelete().Table("experiments").
		Where("id IN (?)", bun.In(validIDs)).
		Returning("id").
		Model(&acceptedIDs).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete runs: %w", err)
	}

	for _, acceptID := range acceptedIDs {
		results = append(results, &apiv1.SearchActionResult{
			Error: "",
			Id:    int32(acceptID),
		})
	}

	// delete run hparams
	if _, err = tx.NewDelete().Table("run_hparams").
		Where("run_id IN (?)", bun.In(deleteRunIDs)).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("deleting run hparams: %w", err)
	}
	// remove project hparams
	if err = db.RemoveOutdatedProjectHparams(ctx, tx, int(req.ProjectId)); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &apiv1.DeleteSearchesResponse{Results: results}, nil
}

func (a *apiServer) ArchiveSearches(
	ctx context.Context, req *apiv1.ArchiveSearchesRequest,
) (*apiv1.ArchiveSearchesResponse, error) {
	results, err := archiveUnarchiveSearchAction(ctx, true, req.SearchIds, req.ProjectId, req.Filter)
	if err != nil {
		return nil, err
	}
	return &apiv1.ArchiveSearchesResponse{Results: results}, nil
}

func (a *apiServer) UnarchiveSearches(
	ctx context.Context, req *apiv1.UnarchiveSearchesRequest,
) (*apiv1.UnarchiveSearchesResponse, error) {
	results, err := archiveUnarchiveSearchAction(ctx, false, req.SearchIds, req.ProjectId, req.Filter)
	if err != nil {
		return nil, err
	}
	return &apiv1.UnarchiveSearchesResponse{Results: results}, nil
}

func archiveUnarchiveSearchAction(ctx context.Context, archive bool, runIDs []int32,
	projectID int32, filter *string,
) ([]*apiv1.SearchActionResult, error) {
	if len(runIDs) > 0 && filter != nil {
		return nil, fmt.Errorf("if filter is provided run id list must be empty")
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	var searchCandidates []searchCandidateResult
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments AS e").
		Model(&searchCandidates).
		Column("e.id").
		ColumnExpr("COALESCE((e.archived OR p.archived OR w.archived), FALSE) AS archived").
		ColumnExpr("e.state IN (?) AS is_terminal", bun.In(model.StatesToStrings(model.TerminalStates))).
		Join("JOIN projects p ON e.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id").
		Where("e.project_id = ?", projectID)

	if filter == nil {
		query = query.Where("e.id IN (?)", bun.In(runIDs))
	} else {
		query, err = filterSearchQuery(query, filter)
		if err != nil {
			return nil, err
		}
	}

	query, err = experiment.AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, nil, query,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT_METADATA})
	if err != nil {
		return nil, err
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []*apiv1.SearchActionResult
	visibleIDs := set.New[int32]()
	var validIDs []int32
	for _, cand := range searchCandidates {
		visibleIDs.Insert(cand.ID)
		switch {
		case !cand.IsTerminal:
			results = append(results, &apiv1.SearchActionResult{
				Error: "Search is not in terminal state.",
				Id:    cand.ID,
			})
		case cand.Archived == archive:
			results = append(results, &apiv1.SearchActionResult{
				Id: cand.ID,
			})
		default:
			validIDs = append(validIDs, cand.ID)
		}
	}

	if filter == nil {
		for _, originalID := range runIDs {
			if !visibleIDs.Contains(originalID) {
				results = append(results, &apiv1.SearchActionResult{
					Error: fmt.Sprintf("Search with id '%d' not found in project with id '%d'", originalID, projectID),
					Id:    originalID,
				})
			}
		}
	}

	if len(validIDs) > 0 {
		var acceptedIDs []int32
		_, err = db.Bun().NewUpdate().
			ModelTableExpr("experiments as e").
			Set("archived = ?", archive).
			Where("id IN (?)", bun.In(validIDs)).
			Returning("id").
			Model(&acceptedIDs).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to archive/unarchive searches: %w", err)
		}
		for _, acceptID := range acceptedIDs {
			results = append(results, &apiv1.SearchActionResult{
				Error: "",
				Id:    acceptID,
			})
		}
	}

	return results, nil
}

func (a *apiServer) PauseSearches(ctx context.Context, req *apiv1.PauseSearchesRequest,
) (*apiv1.PauseSearchesResponse, error) {
	results, err := pauseResumeSearchAction(ctx, true, req.ProjectId, req.SearchIds, req.Filter)
	if err != nil {
		return nil, err
	}
	return &apiv1.PauseSearchesResponse{Results: results}, nil
}

func (a *apiServer) ResumeSearches(ctx context.Context, req *apiv1.ResumeSearchesRequest,
) (*apiv1.ResumeSearchesResponse, error) {
	results, err := pauseResumeSearchAction(ctx, false, req.ProjectId, req.SearchIds, req.Filter)
	if err != nil {
		return nil, err
	}
	return &apiv1.ResumeSearchesResponse{Results: results}, nil
}

func pauseResumeSearchAction(ctx context.Context, isPause bool, projectID int32,
	searchIds []int32, filter *string) (
	[]*apiv1.SearchActionResult, error,
) {
	if len(searchIds) > 0 && filter != nil {
		return nil, fmt.Errorf("if filter is provided search id list must be empty")
	}
	// Get experiment ids
	var err error
	var searchCandidates []searchCandidateResult
	isSearchIDAction := (len(searchIds) > 0)
	getQ := db.Bun().NewSelect().
		ModelTableExpr("experiments AS e").
		Model(&searchCandidates).
		Column("e.id").
		ColumnExpr("COALESCE((e.archived OR p.archived OR w.archived), FALSE) AS archived").
		Join("JOIN projects p ON e.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id").
		Where("e.project_id = ?", projectID)

	if isSearchIDAction {
		getQ = getQ.Where("e.id IN (?)", bun.In(searchIds))
	} else {
		getQ, err = filterSearchQuery(getQ, filter)
		if err != nil {
			return nil, err
		}
	}

	err = getQ.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []*apiv1.SearchActionResult
	visibleIDs := set.New[int32]()
	expIDs := set.New[int32]()
	for _, cand := range searchCandidates {
		visibleIDs.Insert(cand.ID)
		if cand.Archived {
			results = append(results, &apiv1.SearchActionResult{
				Error: "Search is archived.",
				Id:    cand.ID,
			})
			continue
		}
		expIDs.Insert(cand.ID)
	}
	if isSearchIDAction {
		for _, originalID := range searchIds {
			if !visibleIDs.Contains(originalID) {
				results = append(results, &apiv1.SearchActionResult{
					Error: fmt.Sprintf("Search with id '%d' not found in project with id '%d'", originalID, projectID),
					Id:    originalID,
				})
			}
		}
	}
	// Pause/Resume experiments
	var expResults []experiment.ExperimentActionResult
	var errMsg string
	if isPause {
		expResults, err = experiment.PauseExperiments(ctx, projectID, expIDs.ToSlice(), nil)
		errMsg = "Failed to pause experiment: %s"
	} else {
		expResults, err = experiment.ActivateExperiments(ctx, projectID, expIDs.ToSlice(), nil)
		errMsg = "Failed to resume experiment: %s"
	}
	if err != nil {
		return nil, err
	}
	for _, expRes := range expResults {
		if expRes.Error != nil {
			results = append(results, &apiv1.SearchActionResult{
				Error: fmt.Sprintf(errMsg, expRes.Error),
				Id:    expRes.ID,
			})
		} else {
			results = append(results, &apiv1.SearchActionResult{
				Error: "",
				Id:    expRes.ID,
			})
		}
	}
	return results, nil
}

func validateBulkSearchActionCandidates(
	ctx context.Context, projectID int32, searchIds []int32, filter *string,
) ([]*apiv1.SearchActionResult, []int32, error) {
	if len(searchIds) > 0 && filter != nil {
		return nil, nil, fmt.Errorf("if filter is provided search id list must be empty")
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, nil, err
	}

	type searchResult struct {
		ID         int32
		IsTerminal bool
	}

	var candidates []searchResult
	getQ := getSelectSearchesQueryTables().
		Model(&candidates).
		Column("e.id").
		ColumnExpr("e.state IN (?) AS is_terminal", bun.In(model.StatesToStrings(model.TerminalStates))).
		Where("e.project_id = ?", projectID)

	if filter == nil {
		getQ = getQ.Where("e.id IN (?)", bun.In(searchIds))
	} else {
		getQ, err = filterSearchQuery(getQ, filter)
		if err != nil {
			return nil, nil, err
		}
	}

	if getQ, err = experiment.AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, nil, getQ,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT}); err != nil {
		return nil, nil, err
	}

	err = getQ.Scan(ctx)
	if err != nil {
		return nil, nil, err
	}

	var results []*apiv1.SearchActionResult
	visibleIDs := set.New[int32]()
	var validIDs []int32
	for _, cand := range candidates {
		visibleIDs.Insert(cand.ID)
		switch {
		case cand.IsTerminal:
			results = append(results, &apiv1.SearchActionResult{
				Error: "",
				Id:    cand.ID,
			})
		default:
			validIDs = append(validIDs, cand.ID)
		}
	}
	if filter == nil {
		for _, originalID := range searchIds {
			if !visibleIDs.Contains(originalID) {
				results = append(results, &apiv1.SearchActionResult{
					Error: fmt.Sprintf("Search with id '%d' not found", originalID),
					Id:    originalID,
				})
			}
		}
	}
	return results, validIDs, nil
}

func (a *apiServer) LaunchTensorboardSearches(ctx context.Context, req *apiv1.LaunchTensorboardSearchesRequest,
) (*apiv1.LaunchTensorboardSearchesResponse, error) {
	_, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	type searchResult struct {
		ID int32
	}

	var targets []searchResult
	getQ := getSelectSearchesQueryTables().
		Model(&targets).
		Column("e.id").
		Where("w.id = ?", req.GetWorkspaceId()).
		Where("e.state NOT IN (?)", bun.In(model.StatesToStrings(model.TerminalStates)))

	getQ, err = filterSearchQuery(getQ, &req.Filter)
	if err != nil {
		return nil, err
	}

	err = getQ.Scan(ctx)
	if err != nil {
		return nil, err
	}

	expIds := make([]int32, len(targets))
	for i := range expIds {
		expIds[i] = targets[i].ID
	}

	launchResp, err := a.LaunchTensorboard(ctx, &apiv1.LaunchTensorboardRequest{
		ExperimentIds: expIds,
		Config:        req.GetConfig(),
		TemplateName:  req.GetTemplateName(),
		Files:         req.GetFiles(),
		WorkspaceId:   req.GetWorkspaceId(),
	})

	return &apiv1.LaunchTensorboardSearchesResponse{
		Tensorboard: launchResp.GetTensorboard(),
		Config:      launchResp.GetConfig(),
		Warnings:    launchResp.GetWarnings(),
	}, err
}
