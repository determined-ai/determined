package experiment

import (
	"context"
	"strconv"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
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

// ExperimentsAddr is the address to direct experiment actions.
var ExperimentsAddr = actor.Addr("experiments")

// For each experiment, based on the actor, add an error or non-error to results.
func loadMultiExperimentActionResults(results []ExperimentActionResult,
	resps map[*actor.Ref]actor.Message,
) ([]ExperimentActionResult, error) {
	for ref, actorResp := range resps {
		originalID, err := strconv.ParseInt(ref.Address().Local(), 10, 32)
		if err != nil {
			return nil, err
		}
		if actorResp == nil {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.Internal, "actorResp nil."),
				ID:    int32(originalID),
			})
		} else if typed, ok := actorResp.(error); ok && typed != nil {
			results = append(results, ExperimentActionResult{
				Error: typed,
				ID:    int32(originalID),
			})
		} else {
			results = append(results, ExperimentActionResult{
				Error: nil,
				ID:    int32(originalID),
			})
		}
	}
	return results, nil
}

// For each experiment, try to retrieve an actor or append an error message.
func nonTerminalExperiments(system *actor.System, expIDs []int32,
	results []ExperimentActionResult,
) ([]*actor.Ref, []ExperimentActionResult) {
	refs := []*actor.Ref{}
	for _, expID := range expIDs {
		addr := ExperimentsAddr.Child(expID)
		ref := system.Get(addr)
		if ref == nil {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.FailedPrecondition, "experiment in terminal state"),
				ID:    expID,
			})
		} else {
			refs = append(refs, ref)
		}
	}
	return refs, results
}

// A Bun query for editable experiments in multi-experiment actions.
func editableExperimentIds(ctx context.Context, requestedIds []int32) ([]int32,
	error,
) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	expIDs := []int32{}
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments AS e").
		Model(&expIDs).
		Column("e.id").
		Where("id IN (?)", bun.In(requestedIds))

	if query, err = AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, nil, query,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT}); err != nil {
		return nil, err
	}

	err = query.Scan(ctx)
	return expIDs, err
}

// ToAPIResults converts ExperimentActionResult type with error object to error strings.
func ToAPIResults(results []ExperimentActionResult) []*apiv1.ExperimentActionResult {
	apiResults := []*apiv1.ExperimentActionResult{}
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
func ActivateExperiments(ctx context.Context, system *actor.System,
	experimentIds []int32,
) ([]ExperimentActionResult, error) {
	expIDs, err := editableExperimentIds(ctx, experimentIds)
	if err != nil {
		return nil, err
	}

	results := []ExperimentActionResult{}
	for _, originalID := range experimentIds {
		if !slices.Contains(expIDs, originalID) {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.NotFound, "experiment not found: %d", originalID),
				ID:    originalID,
			})
		}
	}

	refs, results := nonTerminalExperiments(system, expIDs, results)
	resps := system.AskAll(&apiv1.ActivateExperimentRequest{}, refs...).GetAll()
	return loadMultiExperimentActionResults(results, resps)
}

// CancelExperiments works on one or many experiments.
func CancelExperiments(ctx context.Context, system *actor.System,
	experimentIds []int32,
) ([]ExperimentActionResult, error) {
	expIDs, err := editableExperimentIds(ctx, experimentIds)
	if err != nil {
		return nil, err
	}

	results := []ExperimentActionResult{}
	for _, originalID := range experimentIds {
		if !slices.Contains(expIDs, originalID) {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.NotFound, "experiment not found: %d", originalID),
				ID:    originalID,
			})
		}
	}

	refs := []*actor.Ref{}
	for _, expID := range expIDs {
		addr := ExperimentsAddr.Child(expID)
		ref := system.Get(addr)
		if ref == nil {
			// For cancel/kill, it's OK if experiment already terminated.
			results = append(results, ExperimentActionResult{
				Error: nil,
				ID:    expID,
			})
		} else {
			refs = append(refs, ref)
		}
	}
	resps := system.AskAll(&apiv1.CancelExperimentRequest{}, refs...).GetAll()
	return loadMultiExperimentActionResults(results, resps)
}

// KillExperiments works on one or many experiments.
func KillExperiments(ctx context.Context, system *actor.System,
	experimentIds []int32,
) ([]ExperimentActionResult, error) {
	expIDs, err := editableExperimentIds(ctx, experimentIds)
	if err != nil {
		return nil, err
	}

	results := []ExperimentActionResult{}
	for _, originalID := range experimentIds {
		if !slices.Contains(expIDs, originalID) {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.NotFound, "experiment not found: %d", originalID),
				ID:    originalID,
			})
		}
	}

	refs := []*actor.Ref{}
	for _, expID := range expIDs {
		addr := ExperimentsAddr.Child(expID)
		ref := system.Get(addr)
		if ref == nil {
			// For cancel/kill, it's OK if experiment already terminated.
			results = append(results, ExperimentActionResult{
				Error: nil,
				ID:    expID,
			})
		} else {
			refs = append(refs, ref)
		}
	}
	resps := system.AskAll(&apiv1.KillExperimentRequest{}, refs...).GetAll()
	return loadMultiExperimentActionResults(results, resps)
}

// PauseExperiments works on one or many experiments.
func PauseExperiments(ctx context.Context, system *actor.System,
	experimentIds []int32,
) ([]ExperimentActionResult, error) {
	expIDs, err := editableExperimentIds(ctx, experimentIds)
	if err != nil {
		return nil, err
	}

	results := []ExperimentActionResult{}
	for _, originalID := range experimentIds {
		if !slices.Contains(expIDs, originalID) {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.NotFound, "experiment not found: %d", originalID),
				ID:    originalID,
			})
		}
	}

	refs, results := nonTerminalExperiments(system, expIDs, results)
	resps := system.AskAll(&apiv1.PauseExperimentRequest{}, refs...).GetAll()
	return loadMultiExperimentActionResults(results, resps)
}

// ArchiveExperiments works on one or many experiments.
func ArchiveExperiments(ctx context.Context, system *actor.System,
	experimentIds []int32,
) ([]ExperimentActionResult, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	expChecks := []archiveExperimentOKResult{}
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments as e").
		Model(&expChecks).
		Column("e.archived").
		Column("e.id").
		ColumnExpr("e.state IN (?) AS state", bun.In([]string{
			"CANCELED",
			"COMPLETED",
			"ERROR",
		})).
		Where("e.id IN (?)", bun.In(experimentIds))

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

	results := []ExperimentActionResult{}
	visibleIDs := []int32{}
	validIDs := []int32{}
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
	for _, originalID := range experimentIds {
		if !slices.Contains(visibleIDs, originalID) {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.NotFound, "experiment not found: %d", originalID),
				ID:    originalID,
			})
		}
	}

	if len(validIDs) > 0 {
		acceptedIDs := []int32{}
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
func UnarchiveExperiments(ctx context.Context, system *actor.System,
	experimentIds []int32,
) ([]ExperimentActionResult, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	expChecks := []archiveExperimentOKResult{}
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments as e").
		Model(&expChecks).
		Column("e.archived").
		Column("e.id").
		ColumnExpr("e.state IN (?) AS state", bun.In([]string{
			"CANCELED",
			"COMPLETED",
			"ERROR",
		})).
		Where("e.id IN (?)", bun.In(experimentIds))

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

	results := []ExperimentActionResult{}
	visibleIDs := []int32{}
	validIDs := []int32{}
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
	for _, originalID := range experimentIds {
		if !slices.Contains(visibleIDs, originalID) {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.NotFound, "experiment not found: %d", originalID),
				ID:    originalID,
			})
		}
	}

	if len(validIDs) > 0 {
		acceptedIDs := []int32{}
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
func MoveExperiments(ctx context.Context, system *actor.System,
	experimentIds []int32, destinationProjectID int32,
) ([]ExperimentActionResult, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	expChecks := []archiveExperimentOKResult{}
	getQ := db.Bun().NewSelect().
		ModelTableExpr("experiments AS exp").
		Model(&expChecks).
		Column("exp.id").
		ColumnExpr("(exp.archived OR p.archived OR w.archived) AS archived").
		ColumnExpr("TRUE AS state").
		Join("JOIN projects p ON exp.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id").
		Where("exp.id IN (?)", bun.In(experimentIds))

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

	results := []ExperimentActionResult{}
	visibleIDs := []int32{}
	validIDs := []int32{}
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
	for _, originalID := range experimentIds {
		if !slices.Contains(visibleIDs, originalID) {
			results = append(results, ExperimentActionResult{
				Error: status.Errorf(codes.NotFound, "experiment not found: %d", originalID),
				ID:    originalID,
			})
		}
	}

	if len(validIDs) > 0 {
		acceptedIDs := []int32{}
		_, err = db.Bun().NewUpdate().
			ModelTableExpr("experiments as e").
			Set("project_id = ?", destinationProjectID).
			Where("e.id IN (?)", bun.In(validIDs)).
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
