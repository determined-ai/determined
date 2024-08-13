package experiment

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/db/bunutils"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

const (
	// GlobalProjectID is the project ID for requests that affect applicable experiments across all projects.
	GlobalProjectID = -1
)

// ExperimentActionResult contains an experiment's ID and associated error.
type ExperimentActionResult struct {
	Error error
	ID    int32
}

type archiveExperimentOKResult struct {
	Archived bool
	ID       int32
	State    bool
}

type deleteExperimentOKResult struct {
	ID       int32
	Versions int
	State    experimentv1.State
}

// For each experiment, try to retrieve an actor or append an error message.
func nonTerminalExperiments(
	expIDs []int32,
	results []ExperimentActionResult,
) (map[int32]Experiment, []ExperimentActionResult) {
	refs := make(map[int32]Experiment)
	for _, expID := range expIDs {
		ref, ok := ExperimentRegistry.Load(int(expID))
		if ref == nil || !ok {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.FailedPrecondition, "experiment in terminal state"),
				ID:    expID,
			})
		} else {
			refs[expID] = ref
		}
	}
	return refs, results
}

// Apply filters to a query.
func queryBulkExperiments(query *bun.SelectQuery,
	filters *apiv1.BulkExperimentFilters,
) *bun.SelectQuery {
	if len(filters.ExcludedExperimentIds) > 0 {
		query = query.Where("e.id NOT IN (?)", bun.In(filters.ExcludedExperimentIds))
	}

	if filters.Description != "" {
		query = query.Where("e.config->>'description' ILIKE ('%%' || ? || '%%')", filters.Description)
	}
	if filters.Name != "" {
		query = query.Where("e.config->>'name' ILIKE ('%%' || ? || '%%')",
			filters.Name)
	}

	if len(filters.Labels) > 0 {
		query = query.Where(`string_to_array(?, ',') <@ ARRAY(SELECT jsonb_array_elements_text(
				CASE WHEN e.config->'labels'::text = 'null'
				THEN NULL
				ELSE e.config->'labels' END
			))`, strings.Join(filters.Labels, ","))
	}

	if filters.Archived != nil {
		query = query.Where("e.archived = ?", filters.Archived.Value)
	}
	if len(filters.States) > 0 {
		var states []string
		for _, state := range filters.States {
			states = append(states, strings.TrimPrefix(state.String(), "STATE_"))
		}
		query = query.Where("e.state IN (?)", bun.In(states))
	}
	if len(filters.UserIds) > 0 {
		query = query.Where("e.owner_id IN (?)", bun.In(filters.UserIds))
	}
	if filters.ProjectId != 0 {
		query = query.Where("project_id = ?", filters.ProjectId)
	}
	return query
}

// FilterToExperimentIds applies a request's filters to get a list of matching experiment IDs.
func FilterToExperimentIds(ctx context.Context, filters *apiv1.BulkExperimentFilters) ([]int32,
	error,
) {
	var experimentIDList []int32
	query := db.Bun().NewSelect().
		Model(&experimentIDList).
		ModelTableExpr("experiments as e").
		Column("e.id")
	query = queryBulkExperiments(query, filters)

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}
	return experimentIDList, nil
}

// experimentsEditableByUser is a modifiable unexported func, exposed to allow for unit testing.
var experimentsEditableByUser = getExperimentsEditableByCurrentUser

func getExperimentsEditableByCurrentUser(
	ctx context.Context,
	projectID int32,
	experimentIDs []int32,
	filters *apiv1.BulkExperimentFilters,
) ([]int32, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	return getExperimentsEditableByUser(
		ctx, curUser, projectID, experimentIDs, filters,
	)
}

// getExperimentsEditableByUser returns a list of experiment ids which are editable by
// the provided user.
// If filters are provided, experimentIds are ignored.
func getExperimentsEditableByUser(
	ctx context.Context,
	user *model.User,
	projectID int32,
	experimentIDs []int32,
	filters *apiv1.BulkExperimentFilters,
) ([]int32, error) {
	var filteredExperimentIDs []int32
	var err error
	if filters == nil {
		filteredExperimentIDs = experimentIDs
	} else {
		filteredExperimentIDs, err = FilterToExperimentIds(ctx, filters)
	}
	if err != nil {
		return nil, err
	}

	var selectedExperimentIDs []int32
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments AS e").
		Model(&selectedExperimentIDs).
		Column("e.id").
		Join("JOIN projects p ON e.project_id = p.id").
		Where("NOT e.archived")

	if projectID != GlobalProjectID {
		query = query.Where("e.project_id = ?", projectID)
	}
	query = query.Where("e.id IN (?)", bun.In(filteredExperimentIDs))

	if query, err = AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *user, nil, query,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT}); err != nil {
		return nil, err
	}

	err = query.Scan(ctx)
	return selectedExperimentIDs, err
}

// ToAPIResults converts ExperimentActionResult type with error object to error strings.
func ToAPIResults(results []ExperimentActionResult) []*apiv1.ExperimentActionResult {
	var apiResults []*apiv1.ExperimentActionResult
	for _, result := range results {
		if result.Error == nil {
			apiResults = append(apiResults, &apiv1.ExperimentActionResult{
				Error: "",
				Id:    result.ID,
			})
		} else {
			apiResults = append(apiResults, &apiv1.ExperimentActionResult{
				Error: result.Error.Error(),
				Id:    result.ID,
			})
		}
	}
	return apiResults
}

// ActivateExperiments works on one or many experiments.
// If filters are provided, experimentIds are ignored.
func ActivateExperiments(
	ctx context.Context,
	projectID int32,
	experimentIds []int32,
	filters *apiv1.BulkExperimentFilters,
) ([]ExperimentActionResult, error) {
	if filters != nil && filters.States == nil {
		filters.States = []experimentv1.State{experimentv1.State_STATE_PAUSED}
	}
	expIDs, err := experimentsEditableByUser(ctx, projectID, experimentIds, filters)
	if err != nil {
		return nil, err
	}

	var results []ExperimentActionResult
	if filters == nil {
		for _, originalID := range experimentIds {
			if !slices.Contains(expIDs, originalID) {
				results = append(results, ExperimentActionResult{
					Error: api.NotFoundErrs("experiment", strconv.Itoa(int(originalID)), true),
					ID:    originalID,
				})
			}
		}
	}

	refs, results := nonTerminalExperiments(expIDs, results)
	for id, ref := range refs {
		results = append(results, ExperimentActionResult{
			Error: ref.ActivateExperiment(),
			ID:    id,
		})
	}
	return results, nil
}

// CancelExperiments works on one or many experiments.
// If filters are provided, experimentIds are ignored.
func CancelExperiments(
	ctx context.Context,
	projectID int32,
	experimentIds []int32,
	filters *apiv1.BulkExperimentFilters,
) ([]ExperimentActionResult, error) {
	if filters != nil && filters.States == nil {
		for _, s := range model.NonTerminalStates {
			filters.States = append(filters.States, model.StateToProto(s))
		}
	}
	expIDs, err := experimentsEditableByUser(ctx, projectID, experimentIds, filters)
	if err != nil {
		return nil, err
	}

	var results []ExperimentActionResult
	if filters == nil {
		for _, originalID := range experimentIds {
			if !slices.Contains(expIDs, originalID) {
				results = append(results, ExperimentActionResult{
					Error: api.NotFoundErrs("experiment", strconv.Itoa(int(originalID)), true),
					ID:    originalID,
				})
			}
		}
	}

	refs := make(map[int32]Experiment)
	for _, expID := range expIDs {
		ref, ok := ExperimentRegistry.Load(int(expID))
		if ref == nil || !ok {
			// For cancel/kill, it's OK if experiment already terminated.
			results = append(results, ExperimentActionResult{
				Error: nil,
				ID:    expID,
			})
		} else {
			refs[expID] = ref
		}
	}
	for id, ref := range refs {
		results = append(results, ExperimentActionResult{
			Error: ref.CancelExperiment(),
			ID:    id,
		})
	}
	return results, nil
}

// KillExperiments works on one or many experiments.
// If filters are provided, experimentIds are ignored.
func KillExperiments(
	ctx context.Context,
	projectID int32,
	experimentIds []int32,
	filters *apiv1.BulkExperimentFilters,
) ([]ExperimentActionResult, error) {
	if filters != nil && filters.States == nil {
		for _, s := range model.NonTerminalStates {
			filters.States = append(filters.States, model.StateToProto(s))
		}
	}
	expIDs, err := experimentsEditableByUser(ctx, projectID, experimentIds, filters)
	if err != nil {
		return nil, err
	}

	var results []ExperimentActionResult
	if filters == nil {
		for _, originalID := range experimentIds {
			if !slices.Contains(expIDs, originalID) {
				results = append(results, ExperimentActionResult{
					Error: api.NotFoundErrs("experiment", strconv.Itoa(int(originalID)), true),
					ID:    originalID,
				})
			}
		}
	}

	refs := make(map[int32]Experiment)
	for _, expID := range expIDs {
		ref, ok := ExperimentRegistry.Load(int(expID))
		if ref == nil || !ok {
			// For cancel/kill, it's OK if experiment already terminated.
			results = append(results, ExperimentActionResult{
				Error: nil,
				ID:    expID,
			})
		} else {
			refs[expID] = ref
		}
	}
	for id, ref := range refs {
		results = append(results, ExperimentActionResult{
			Error: ref.KillExperiment(),
			ID:    id,
		})
	}
	return results, nil
}

// PauseExperiments works on one or many experiments.
// If filters are provided, experimentIds are ignored.
func PauseExperiments(
	ctx context.Context,
	projectID int32,
	experimentIds []int32,
	filters *apiv1.BulkExperimentFilters,
) ([]ExperimentActionResult, error) {
	if filters != nil && filters.States == nil {
		filters.States = []experimentv1.State{experimentv1.State_STATE_ACTIVE}
	}
	expIDs, err := experimentsEditableByUser(ctx, projectID, experimentIds, filters)
	if err != nil {
		return nil, err
	}

	var results []ExperimentActionResult
	if filters == nil {
		for _, originalID := range experimentIds {
			if !slices.Contains(expIDs, originalID) {
				results = append(results, ExperimentActionResult{
					Error: api.NotFoundErrs("experiment", strconv.Itoa(int(originalID)), true),
					ID:    originalID,
				})
			}
		}
	}

	refs, results := nonTerminalExperiments(expIDs, results)
	for id, ref := range refs {
		results = append(results, ExperimentActionResult{
			Error: ref.PauseExperiment(),
			ID:    id,
		})
	}
	return results, nil
}

// DeleteExperiments works on one or many experiments.
// If filters are provided, experimentIds are ignored.
func DeleteExperiments(
	ctx context.Context,
	projectID int32,
	experimentIds []int32,
	filters *apiv1.BulkExperimentFilters,
) ([]ExperimentActionResult, []*model.Experiment, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, nil, err
	}

	var expChecks []deleteExperimentOKResult
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments as e").
		Model(&expChecks).
		Column("e.id").
		ColumnExpr(bunutils.ProtoStateDBCaseString(experimentv1.State_value, "e.state", "state", "STATE_")).
		ColumnExpr("COUNT(model_versions.id) AS versions").
		Join("JOIN projects p ON e.project_id = p.id").
		Join("LEFT JOIN checkpoints_view c ON c.experiment_id = e.id").
		Join("LEFT JOIN model_versions ON model_versions.checkpoint_uuid = c.uuid").
		Group("e.id")

	if projectID != GlobalProjectID {
		query = query.Where("e.project_id = ?", projectID)
	}

	switch {
	case filters == nil:
		query = query.Where("e.id IN (?)", bun.In(experimentIds))
	default:
		query = queryBulkExperiments(query, filters).
			Where("e.state IN (?)", bun.In(model.StatesToStrings(model.TerminalStates)))
	}

	query, err = AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, nil, query,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT})
	if err != nil {
		return nil, nil, err
	}

	if err = query.Scan(ctx); err != nil {
		return nil, nil, err
	}

	var results []ExperimentActionResult
	var visibleIDs []int32
	var validIDs []int32
	for _, check := range expChecks {
		visibleIDs = append(visibleIDs, check.ID)
		switch {
		case check.Versions > 0:
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.InvalidArgument, "checkpoints are registered as model versions"),
				ID:    check.ID,
			})
		case !model.ExperimentTransitions[model.StateFromProto(check.State)][model.DeletingState]:
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.FailedPrecondition, "cannot delete experiment in %s state",
					check.State),
				ID: check.ID,
			})
		default:
			validIDs = append(validIDs, check.ID)
		}
	}
	if filters == nil {
		for _, originalID := range experimentIds {
			if !slices.Contains(visibleIDs, originalID) {
				results = append(results, ExperimentActionResult{
					Error: api.NotFoundErrs("experiment", strconv.Itoa(int(originalID)), true),
					ID:    originalID,
				})
			}
		}
	}

	var acceptedExperiments []*model.Experiment
	if len(validIDs) > 0 {
		_, err = db.Bun().NewUpdate().
			ModelTableExpr("experiments as e").
			Set("state = ?", model.DeletingState).
			Where("id IN (?)", bun.In(validIDs)).
			Returning(`id, state, config, start_time, end_time, archived,
				   owner_id, notes, job_id, '' as username, project_id`).
			Model(&acceptedExperiments).
			Exec(ctx)
		if err != nil {
			return nil, nil, err
		}

		for _, exp := range acceptedExperiments {
			results = append(results, ExperimentActionResult{
				Error: nil,
				ID:    int32(exp.ID),
			})
		}
	}
	return results, acceptedExperiments, nil
}

// ArchiveExperiments works on one or many experiments.
// If filters are provided, experimentIds are ignored.
func ArchiveExperiments(
	ctx context.Context,
	projectID int32,
	experimentIds []int32,
	filters *apiv1.BulkExperimentFilters,
) ([]ExperimentActionResult, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	var expChecks []archiveExperimentOKResult
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments as e").
		Model(&expChecks).
		Column("e.archived").
		Column("e.id").
		ColumnExpr("e.state IN (?) AS state", bun.In(model.StatesToStrings(model.TerminalStates))).
		Join("JOIN projects p ON e.project_id = p.id")

	if projectID != GlobalProjectID {
		query = query.Where("e.project_id = ?", projectID)
	}

	switch {
	case filters == nil:
		query = query.Where("e.id IN (?)", bun.In(experimentIds))
	default:
		query = queryBulkExperiments(query, filters).
			Where("NOT e.archived").
			Where("e.state IN (?)", bun.In(model.StatesToStrings(model.TerminalStates)))
	}

	query, err = AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, nil, query,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT_METADATA})
	if err != nil {
		return nil, err
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []ExperimentActionResult
	var visibleIDs []int32
	var validIDs []int32
	for _, check := range expChecks {
		visibleIDs = append(visibleIDs, check.ID)
		switch {
		case check.Archived:
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.FailedPrecondition, "Experiment is already archived."),
				ID:    check.ID,
			})
		case !check.State:
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.FailedPrecondition, "Experiment is not in terminal state."),
				ID:    check.ID,
			})
		default:
			validIDs = append(validIDs, check.ID)
		}
	}
	if filters == nil {
		for _, originalID := range experimentIds {
			if !slices.Contains(visibleIDs, originalID) {
				results = append(results, ExperimentActionResult{
					Error: api.NotFoundErrs("experiment", strconv.Itoa(int(originalID)), true),
					ID:    originalID,
				})
			}
		}
	}

	if len(validIDs) > 0 {
		var acceptedIDs []int32
		_, err = db.Bun().NewUpdate().
			ModelTableExpr("experiments as e").
			Set("archived = true").
			Where("id IN (?)", bun.In(validIDs)).
			Returning("e.id").
			Model(&acceptedIDs).
			Exec(ctx)
		if err != nil {
			return nil, err
		}

		for _, acceptID := range acceptedIDs {
			results = append(results, ExperimentActionResult{
				Error: nil,
				ID:    acceptID,
			})
		}
	}
	return results, nil
}

// UnarchiveExperiments works on one or many experiments.
// If filters are provided, experimentIds are ignored.
func UnarchiveExperiments(
	ctx context.Context,
	projectID int32,
	experimentIds []int32,
	filters *apiv1.BulkExperimentFilters,
) ([]ExperimentActionResult, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	var expChecks []archiveExperimentOKResult
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments as e").
		Model(&expChecks).
		Column("e.archived").
		Column("e.id").
		ColumnExpr("e.state IN (?) AS state", bun.In(model.StatesToStrings(model.TerminalStates))).
		Join("JOIN projects p ON e.project_id = p.id")

	if projectID != GlobalProjectID {
		query = query.Where("e.project_id = ?", projectID)
	}

	switch {
	case filters == nil:
		query = query.Where("e.id IN (?)", bun.In(experimentIds))
	default:
		query = queryBulkExperiments(query, filters).
			Where("archived").
			Where("e.state IN (?)", bun.In(model.StatesToStrings(model.TerminalStates)))
	}

	query, err = AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, nil, query,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT_METADATA})
	if err != nil {
		return nil, err
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []ExperimentActionResult
	var visibleIDs []int32
	var validIDs []int32
	for _, check := range expChecks {
		visibleIDs = append(visibleIDs, check.ID)
		switch {
		case !check.Archived:
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.FailedPrecondition, "Experiment is not archived."),
				ID:    check.ID,
			})
		case !check.State:
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.FailedPrecondition, "Experiment is not in terminal state."),
				ID:    check.ID,
			})
		default:
			validIDs = append(validIDs, check.ID)
		}
	}
	if filters == nil {
		for _, originalID := range experimentIds {
			if !slices.Contains(visibleIDs, originalID) {
				results = append(results, ExperimentActionResult{
					Error: api.NotFoundErrs("experiment", strconv.Itoa(int(originalID)), true),
					ID:    originalID,
				})
			}
		}
	}

	if len(validIDs) > 0 {
		var acceptedIDs []int32
		_, err = db.Bun().NewUpdate().
			ModelTableExpr("experiments as e").
			Set("archived = false").
			Where("id IN (?)", bun.In(validIDs)).
			Returning("e.id").
			Model(&acceptedIDs).
			Exec(ctx)
		if err != nil {
			return nil, err
		}

		for _, acceptID := range acceptedIDs {
			results = append(results, ExperimentActionResult{
				Error: nil,
				ID:    acceptID,
			})
		}
	}
	return results, nil
}

// MoveExperiments works on one or many experiments.
// If filters are provided, experimentIds are ignored.
func MoveExperiments(
	ctx context.Context,
	projectID int32,
	experimentIds []int32,
	filters *apiv1.BulkExperimentFilters,
	destinationProjectID int32,
) ([]ExperimentActionResult, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	var expChecks []archiveExperimentOKResult
	getQ := db.Bun().NewSelect().
		ModelTableExpr("experiments AS e").
		Model(&expChecks).
		Column("e.id").
		ColumnExpr("(e.archived OR p.archived OR w.archived) AS archived").
		ColumnExpr("TRUE AS state").
		Join("JOIN projects p ON e.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id")

	if projectID != GlobalProjectID {
		getQ = getQ.Where("e.project_id = ?", projectID)
	}

	switch {
	case filters == nil:
		getQ = getQ.Where("e.id IN (?)", bun.In(experimentIds))
	default:
		getQ = queryBulkExperiments(getQ, filters).
			Where("NOT (e.archived OR p.archived OR w.archived)")
	}

	if filters == nil {
		getQ = getQ.Where("e.id IN (?)", bun.In(experimentIds))
	} else {
		getQ = queryBulkExperiments(getQ, filters).
			Where("NOT (e.archived OR p.archived OR w.archived)")
	}

	if getQ, err = AuthZProvider.Get().FilterExperimentsQuery(ctx, *curUser, nil, getQ,
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

	var results []ExperimentActionResult
	var visibleIDs []int32
	var validIDs []int32
	for _, check := range expChecks {
		visibleIDs = append(visibleIDs, check.ID)
		if check.Archived {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.FailedPrecondition, "Experiment is archived."),
				ID:    check.ID,
			})
		} else {
			validIDs = append(validIDs, check.ID)
		}
	}
	if filters == nil {
		for _, originalID := range experimentIds {
			if !slices.Contains(visibleIDs, originalID) {
				results = append(results, ExperimentActionResult{
					Error: api.NotFoundErrs("experiment", strconv.Itoa(int(originalID)), true),
					ID:    originalID,
				})
			}
		}
	}
	if len(validIDs) > 0 {
		tx, err := db.Bun().BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer func() {
			txErr := tx.Rollback()
			if txErr != nil && txErr != sql.ErrTxDone {
				log.WithError(txErr).Error("error rolling back transaction in MoveExperiments")
			}
		}()

		var acceptedIDs []int32
		if _, err = tx.NewUpdate().
			ModelTableExpr("experiments as e").
			Set("project_id = ?", destinationProjectID).
			Where("e.id IN (?)", bun.In(validIDs)).
			Returning("e.id").
			Model(&acceptedIDs).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("updating experiment's project IDs: %w", err)
		}

		// Move hyperparams for perf table
		var runIDs []int32
		err = tx.NewSelect().Table("runs").Column("id").
			Where("experiment_id IN (?)", bun.In(validIDs)).
			Scan(ctx, &runIDs)
		if err != nil {
			return nil, err
		}

		if len(runIDs) > 0 {
			var sourceProjectID int
			err = tx.NewSelect().Table("runs").Column("project_id").
				Where("id IN (?)", bun.In(runIDs)).
				Limit(1).
				Scan(ctx, &sourceProjectID)
			if err != nil {
				return nil, err
			}

			if _, err = tx.NewUpdate().Table("runs").
				Set("project_id = ?", destinationProjectID).
				Where("runs.experiment_id IN (?)", bun.In(validIDs)).
				Exec(ctx); err != nil {
				return nil, fmt.Errorf("updating run's project IDs: %w", err)
			}

			if err = db.AddProjectHparams(ctx, tx, int(destinationProjectID), runIDs); err != nil {
				return nil, err
			}
			if err = db.RemoveOutdatedProjectHparams(ctx, tx, sourceProjectID); err != nil {
				return nil, err
			}
		}

		if _, err = tx.NewRaw(`
		UPDATE runs SET local_id=s.local_id
		FROM
			(
				SELECT
					r.id as id,
					(p.max_local_id + ROW_NUMBER() OVER(PARTITION BY p.id ORDER BY r.id)) as local_id
				FROM
					projects p
					JOIN runs r ON r.project_id=p.id
				WHERE r.experiment_id IN (?)
			) as s 
		WHERE s.id=runs.id
		`,
			bun.In(acceptedIDs)).Exec(ctx); err != nil {
			return nil, fmt.Errorf("updating run's local IDs: %w", err)
		}

		if _, err = tx.NewRaw(`
		UPDATE projects SET max_local_id=s.max_local_id
		FROM 
			(
				SELECT 
					project_id,
					COALESCE(MAX(local_id), 1) as max_local_id
				FROM 
					runs
				GROUP BY
					project_id
				HAVING project_id=?
			) as s
		WHERE projects.id=?`,
			destinationProjectID, destinationProjectID).Exec(ctx); err != nil {
			return nil, fmt.Errorf("updating projects max local id: %w", err)
		}

		if _, err = tx.NewRaw(`
		INSERT INTO local_id_redirect (run_id, project_id, project_key, local_id)
		SELECT 
			r.id as run_id,
			p.id as project_id,
			p.key as project_key,
			r.local_id
		FROM 
			projects p
			JOIN runs r 
			ON r.project_id=p.id
		WHERE r.experiment_id IN (?)
		`, bun.In(acceptedIDs)).Exec(ctx); err != nil {
			return nil, fmt.Errorf("adding local id redirect: %w", err)
		}

		for _, acceptID := range acceptedIDs {
			results = append(results, ExperimentActionResult{
				Error: nil,
				ID:    acceptID,
			})
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func changeExperimentConfigLogRetention(ctx context.Context, database db.DB,
	expID int, numDays int16,
) error {
	exp, err := db.ExperimentByID(ctx, expID)
	if err != nil {
		return errors.Wrap(err, "fetching experiment from database")
	}

	// In long experiments like asha, if the user called this api right after creating the experiment,
	// only the trials that were created at that time have the log retention days changed, any trials
	// created after a while will have the same log retention days as in the task spec. After some
	// discussion, we came to the conclusion that it would be simplest to not allow the users to run
	// this command before an experiment is completed.
	if !model.TerminalStates[exp.State] {
		return status.Error(codes.FailedPrecondition, fmt.Sprintf(
			"experiment in non terminal state '%s', try again later", exp.State))
	}

	activeConfig, err := database.ActiveExperimentConfig(exp.ID)
	if err != nil {
		return errors.Wrapf(
			err, "unable to load config for experiment %v", exp.ID,
		)
	}
	activeConfig.SetRetentionPolicy(&expconf.RetentionPolicyConfigV0{RawLogRetentionDays: &numDays})

	if err := database.SaveExperimentConfig(exp.ID, activeConfig); err != nil {
		return errors.Wrapf(err, "patching experiment %d", exp.ID)
	}

	return nil
}

// BulkUpdateLogRetention retains logs for the given list of experiments.
// If filters are provided, experimentIds are ignored.
func BulkUpdateLogRetention(
	ctx context.Context,
	database db.DB,
	projectID int32,
	expIDs []int32,
	filters *apiv1.BulkExperimentFilters,
	numDays int16,
) ([]ExperimentActionResult, error) {
	var results []ExperimentActionResult
	editableExperimentIDList, err := experimentsEditableByUser(ctx, projectID, expIDs, filters)
	if err != nil {
		return nil, err
	}
	if filters == nil {
		for _, originalID := range expIDs {
			if !slices.Contains(editableExperimentIDList, originalID) {
				results = append(results, ExperimentActionResult{
					Error: api.NotFoundErrs("experiment", strconv.Itoa(int(originalID)), true),
					ID:    originalID,
				})
			}
		}
	}

	var intExpIDs []int
	for _, v := range editableExperimentIDList {
		err = changeExperimentConfigLogRetention(ctx, database, int(v), numDays)
		if err != nil {
			results = append(results, ExperimentActionResult{
				Error: err,
				ID:    v,
			})
		} else {
			intExpIDs = append(intExpIDs, int(v))
			results = append(results, ExperimentActionResult{
				Error: nil,
				ID:    v,
			})
		}
	}

	trialIDs, taskIDs, err := db.ExperimentsTrialAndTaskIDs(ctx, db.Bun(), intExpIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to gather trial IDs for experiments")
	}

	if len(taskIDs) == 0 {
		return results, nil
	}

	err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Table("runs").
			Set("log_retention_days = ?", numDays).
			Where("id IN (?)", bun.In(trialIDs)).
			Exec(ctx); err != nil {
			return fmt.Errorf("updating log retention days for tasks: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}
