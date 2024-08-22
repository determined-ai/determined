//go:build integration
// +build integration

package trials

import (
	"context"
	"testing"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const testTrialCount = 5

func actionFuncAllow(context.Context, model.User, *model.Experiment) error {
	return nil
}

func actionFuncDeny(context.Context, model.User, *model.Experiment) error {
	return status.Error(codes.PermissionDenied, "")
}

func TestCanGetTrialsExperimentAndCheckCanDoAction(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, closeDB := db.MustResolveTestPostgres(t)
	defer closeDB()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)
	user := db.RequireMockUser(t, pgDB)

	externalID := uuid.New().String()
	exp := db.RequireMockExperimentParams(t, pgDB, user, db.MockExperimentParams{
		ExternalExperimentID: &externalID,
	}, db.DefaultProjectID)
	trial, _ := db.RequireMockTrial(t, pgDB, exp)

	// allowed
	err := CanGetTrialsExperimentAndCheckCanDoAction(ctx, trial.ID, &user, actionFuncAllow)
	require.NoError(t, err)
	// denied
	err = CanGetTrialsExperimentAndCheckCanDoAction(ctx, trial.ID, &user, actionFuncDeny)
	require.Error(t, err)
	// not found
	err = CanGetTrialsExperimentAndCheckCanDoAction(ctx, -999, &user, actionFuncAllow)
	require.ErrorIs(t, err, apiPkg.NotFoundErrs("trial", "-999", true))
}

func TestCanGetTrialsExperimentAndCheckCanDoActionBulk(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, closeDB := db.MustResolveTestPostgres(t)
	defer closeDB()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)
	user := db.RequireMockUser(t, pgDB)

	externalID := uuid.New().String()
	exp := db.RequireMockExperimentParams(t, pgDB, user, db.MockExperimentParams{
		ExternalExperimentID: &externalID,
	}, db.DefaultProjectID)
	trialIDs := []int{}
	for i := 0; i < testTrialCount; i++ {
		trial, _ := db.RequireMockTrial(t, pgDB, exp)
		trialIDs = append(trialIDs, trial.ID)
	}

	// allowed
	err := CanGetTrialsExperimentAndCheckCanDoActionBulk(ctx, trialIDs, &user, actionFuncAllow)
	require.NoError(t, err)
	// denied
	err = CanGetTrialsExperimentAndCheckCanDoActionBulk(ctx, trialIDs, &user, actionFuncDeny)
	require.Error(t, err)
	// not found
	err = CanGetTrialsExperimentAndCheckCanDoActionBulk(ctx, []int{-1, -2, -3}, &user, actionFuncAllow)
	require.ErrorIs(t, err, apiPkg.NotFoundErrs("trial", "-1,-2,-3", true))
}
