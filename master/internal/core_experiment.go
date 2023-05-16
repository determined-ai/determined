package internal

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/pkg/model"
)

// ExperimentRequestQuery contains values for the experiments request queries with defaults already
// applied. This should to be kept in sync with the expected queries from ParseExperimentsQuery.
type ExperimentRequestQuery struct {
	User   string
	Limit  int
	Offset int
	Filter string
}

// ParseExperimentsQuery parse queries for the experiments endpoint.
func ParseExperimentsQuery(apiCtx echo.Context) (*ExperimentRequestQuery, error) {
	args := struct {
		User   *string `query:"user"`
		Limit  *int    `query:"limit"`
		Offset *int    `query:"offset"`
		Filter *string `query:"filter"`
	}{}
	var err error
	if err = api.BindArgs(&args, apiCtx); err != nil {
		return nil, err
	}

	queries := ExperimentRequestQuery{}

	if args.User != nil {
		queries.User = *args.User
	}

	if args.Filter != nil {
		queries.Filter = *args.Filter
	}

	if args.Limit == nil || *args.Limit < 0 {
		queries.Limit = 0
	} else {
		queries.Limit = *args.Limit
	}

	if args.Offset == nil || *args.Offset < 0 {
		queries.Offset = 0
	} else {
		queries.Offset = *args.Offset
	}

	return &queries, nil
}

func echoGetExperimentAndCheckCanDoActions(ctx context.Context, c echo.Context, m *Master,
	expID int, actions ...func(context.Context, model.User, *model.Experiment) error,
) (*model.Experiment, model.User, error) {
	user := c.(*detContext.DetContext).MustGetUser()
	e, err := db.ExperimentByID(ctx, expID)
	expNotFound := api.NotFoundErrs("experiment", fmt.Sprint(expID), false)
	if errors.Is(err, db.ErrNotFound) {
		return nil, model.User{}, expNotFound
	} else if err != nil {
		return nil, model.User{}, err
	}
	if err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, user, e); err != nil {
		return nil, model.User{}, authz.SubIfUnauthorized(err, expNotFound)
	}

	for _, action := range actions {
		if err := action(ctx, user, e); err != nil {
			return nil, model.User{}, echo.NewHTTPError(http.StatusForbidden, err.Error())
		}
	}
	return e, user, nil
}

func (m *Master) getExperimentCheckpointsToGC(c echo.Context) (interface{}, error) {
	args := struct {
		ExperimentID   int `path:"experiment_id"`
		ExperimentBest int `query:"save_experiment_best"`
		TrialBest      int `query:"save_trial_best"`
		TrialLatest    int `query:"save_trial_latest"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	exp, _, err := echoGetExperimentAndCheckCanDoActions(
		c.Request().Context(), c, m, args.ExperimentID,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts,
	)
	if err != nil {
		return nil, err
	}

	checkpointUUIDs, err := m.db.ExperimentCheckpointsToGCRaw(
		args.ExperimentID, args.ExperimentBest, args.TrialBest, args.TrialLatest)
	if err != nil {
		return nil, err
	}
	checkpointsDB, err := m.db.CheckpointByUUIDs(checkpointUUIDs)
	if err != nil {
		return nil, err
	}

	checkpointsWithMetric := map[string]interface{}{
		"checkpoints": checkpointsDB, "metric_name": exp.Config.Searcher.Metric,
	}

	return checkpointsWithMetric, nil
}

//	@Summary	Get individual file from modal definitions for download.
//	@Tags		Experiments
//	@ID			get-experiment-model-file
//	@Accept		json
//	@Produce	text/plain; charset=utf-8
//	@Param		experiment_id	path	int		true	"Experiment ID"
//	@Param		path			query	string	true	"Path to the target file"
//	@Success	200				{}		string	""
//	@Router		/experiments/{experiment_id}/file/download [get]
//
// Read why this line exists on the comment on getAggregatedResourceAllocation in core.go.
func (m *Master) getExperimentModelFile(c echo.Context) error {
	args := struct {
		ExperimentID int    `path:"experiment_id"`
		Path         string `query:"path"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return err
	}
	if _, _, err := echoGetExperimentAndCheckCanDoActions(
		c.Request().Context(), c, m, args.ExperimentID,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts,
	); err != nil {
		return err
	}

	modelDefCache := GetModelDefCache()
	file, err := modelDefCache.FileContent(args.ExperimentID, args.Path)
	if err != nil {
		return err
	}
	c.Response().Header().Set(
		"Content-Disposition",
		fmt.Sprintf(
			`attachment; filename="exp%d/%s"`,
			args.ExperimentID,
			args.Path))
	return c.Blob(http.StatusOK, http.DetectContentType(file), file)
}

func (m *Master) getExperimentModelDefinition(c echo.Context) error {
	args := struct {
		ExperimentID int `path:"experiment_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return err
	}
	if _, _, err := echoGetExperimentAndCheckCanDoActions(
		c.Request().Context(), c, m, args.ExperimentID,
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts,
	); err != nil {
		return err
	}

	modelDef, err := m.db.ExperimentModelDefinitionRaw(args.ExperimentID)
	if err != nil {
		return err
	}

	var cleanName string

	// TODO(DET-8577): Remove unnecessary active config usage.
	activeConfig, err := m.db.ActiveExperimentConfig(args.ExperimentID)
	if err == nil {
		// Make a Regex to remove everything but a whitelist of characters.
		reg := regexp.MustCompile(`[^A-Za-z0-9_ \-()[\].{}]+`)
		cleanName = "_" + reg.ReplaceAllString(activeConfig.Name().String(), "")
	}

	// Truncate name to a smaller size to both accommodate file name and path size
	// limits on different platforms as well as get users more accustom to picking shorter
	// names as we move toward "name as mnemonic for an experiment".
	maxNameLength := 50
	if len(cleanName) > maxNameLength {
		cleanName = cleanName[0:maxNameLength]
	}

	c.Response().Header().Set(
		"Content-Disposition",
		fmt.Sprintf(
			`attachment; filename="exp%d%s_model_def.tar.gz"`,
			args.ExperimentID,
			cleanName))
	return c.Blob(http.StatusOK, "application/x-gtar", modelDef)
}

// ErrProjectNotFound is returned in parseCreateExperiment for when project cannot be found
// or when project cannot be viewed due to RBAC restrictions.
type ErrProjectNotFound string

// Error implements the error interface.
func (p ErrProjectNotFound) Error() string {
	return string(p)
}
