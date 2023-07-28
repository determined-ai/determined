package trials

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
)

// CanGetTrialsExperimentAndCheckCanDoAction is a utility function for generalizing
// RBAC support for trials and experiments.
func CanGetTrialsExperimentAndCheckCanDoAction(ctx context.Context,
	trialID int, actionFunc func(context.Context, model.User, *model.Experiment) error,
) error {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}

	trialNotFound := api.NotFoundErrs("trial", fmt.Sprint(trialID), true)
	exp, err := db.ExperimentByTrialID(ctx, trialID)
	if errors.Is(err, db.ErrNotFound) {
		return trialNotFound
	} else if err != nil {
		return err
	}
	if err = experiment.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp); err != nil {
		return authz.SubIfUnauthorized(err, trialNotFound)
	}

	if err = actionFunc(ctx, *curUser, exp); err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	return nil
}
