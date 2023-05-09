package internal

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/determined-ai/determined/proto/pkg/projectv1"

	"github.com/ghodss/yaml"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/project"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
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
	e, err := m.db.ExperimentByID(expID)

	expNotFound := echo.NewHTTPError(http.StatusNotFound, "experiment not found: %d", expID)
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

// CreateExperimentParams defines a request to create an experiment.
type CreateExperimentParams struct {
	Activate      bool            `json:"activate"`
	ConfigBytes   string          `json:"experiment_config"`
	Template      *string         `json:"template"`
	ModelDef      archive.Archive `json:"model_definition"`
	ParentID      *int            `json:"parent_id"`
	Archived      bool            `json:"archived"`
	GitRemote     *string         `json:"git_remote"`
	GitCommit     *string         `json:"git_commit"`
	GitCommitter  *string         `json:"git_committer"`
	GitCommitDate *time.Time      `json:"git_commit_date"`
	ValidateOnly  bool            `json:"validate_only"`
	ProjectID     *int            `json:"project_id"`
}

// ErrProjectNotFound is returned in parseCreateExperiment for when project cannot be found
// or when project cannot be viewed due to RBAC restrictions.
type ErrProjectNotFound string

// Error implements the error interface.
func (p ErrProjectNotFound) Error() string {
	return string(p)
}

func getCreateExperimentsProject(
	m *Master, params *CreateExperimentParams, user *model.User, config expconf.ExperimentConfig,
) (*projectv1.Project, error) {
	// Place experiment in Uncategorized, unless project set in config or CreateExperimentParams.
	// CreateExperimentParams has highest priority.
	var err error
	projectID := 1
	errProjectNotFound := ErrProjectNotFound(fmt.Sprintf("project (%d) not found", projectID))
	if params.ProjectID != nil {
		projectID = *params.ProjectID
		errProjectNotFound = ErrProjectNotFound(fmt.Sprintf("project (%d) not found", projectID))
	} else {
		if (config.Workspace() == "") != (config.Project() == "") {
			return nil,
				errors.New("workspace and project must both be included in config if one is provided")
		}
		if config.Workspace() != "" && config.Project() != "" {
			errProjectNotFound = ErrProjectNotFound(fmt.Sprintf(
				"workspace '%s' or project '%s' not found",
				config.Workspace(), config.Project()))

			projectID, err = m.db.ProjectByName(config.Workspace(), config.Project())
			if errors.Is(err, db.ErrNotFound) {
				return nil, errProjectNotFound
			} else if err != nil {
				return nil, err
			}
		}
	}

	p := &projectv1.Project{}
	if err = m.db.QueryProto("get_project", p, projectID); errors.Is(err, db.ErrNotFound) {
		return nil, errProjectNotFound
	} else if err != nil {
		return nil, err
	}
	if err = project.AuthZProvider.Get().CanGetProject(context.TODO(), *user, p); err != nil {
		return nil, authz.SubIfUnauthorized(err, errProjectNotFound)
	}
	return p, nil
}

func (m *Master) parseCreateExperiment(params *CreateExperimentParams, user *model.User) (
	*model.Experiment, expconf.ExperimentConfig, *projectv1.Project, bool, *tasks.TaskSpec, error,
) {
	// Read the config as the user provided it.
	config, err := expconf.ParseAnyExperimentConfigYAML([]byte(params.ConfigBytes))
	if err != nil {
		return nil, config, nil, false, nil, errors.Wrap(err, "invalid experiment configuration")
	}

	// Apply the template that the user specified.
	if params.Template != nil {
		template, terr := m.db.TemplateByName(*params.Template)
		if terr != nil {
			return nil, config, nil, false, nil, errors.Wrapf(
				terr, "TemplateByName(%q)", *params.Template,
			)
		}
		var tc expconf.ExperimentConfig
		if yerr := yaml.Unmarshal(template.Config, &tc, yaml.DisallowUnknownFields); yerr != nil {
			return nil, config, nil, false, nil, errors.Wrapf(
				terr, "yaml.Unmarshal(template=%q)", *params.Template,
			)
		}
		// Merge the template into the config.
		config = schemas.Merge(config, tc)
	}

	defaulted := schemas.WithDefaults(config)
	resources := defaulted.Resources()
	poolName, err := m.rm.ResolveResourcePool(
		m.system, resources.ResourcePool(), resources.SlotsPerTrial())
	if err != nil {
		return nil, config, nil, false, nil, errors.Wrapf(err, "invalid resource configuration")
	}
	if err = m.rm.ValidateResources(m.system, poolName, resources.SlotsPerTrial(), false); err != nil {
		return nil, config, nil, false, nil, errors.Wrapf(err, "error validating resources")
	}
	taskContainerDefaults, err := m.rm.TaskContainerDefaults(
		m.system,
		poolName,
		m.config.TaskContainerDefaults,
	)
	if err != nil {
		return nil, config, nil, false, nil, errors.Wrapf(err, "error getting TaskContainerDefaults")
	}
	taskSpec := *m.taskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults
	taskSpec.TaskContainerDefaults.MergeIntoExpConfig(&config)

	project, err := getCreateExperimentsProject(m, params, user, defaulted)
	if err != nil {
		return nil, config, nil, false, nil, err
	}

	// Merge in workspace's checkpoint storage into the conifg.
	w := &model.Workspace{}
	if err = db.Bun().NewSelect().Model(w).
		Where("id = ?", project.WorkspaceId).
		Column("checkpoint_storage_config").
		Scan(context.TODO()); err != nil {
		return nil, config, nil, false, nil, err
	}
	config.RawCheckpointStorage = schemas.Merge(
		config.RawCheckpointStorage, w.CheckpointStorageConfig)

	// Merge in the master's checkpoint storage into the config.
	config.RawCheckpointStorage = schemas.Merge(
		config.RawCheckpointStorage, &m.config.CheckpointStorage,
	)

	// Lastly, apply any json-schema-defined defaults.
	config = schemas.WithDefaults(config)

	// Make sure the experiment config has all eventuallyRequired fields.
	if err = schemas.IsComplete(config); err != nil {
		return nil, config, nil, false, nil, errors.Wrap(err, "invalid experiment configuration")
	}

	// Disallow EOL searchers.
	if err = config.Searcher().AssertCurrent(); err != nil {
		return nil, config, nil, false, nil, errors.Wrap(err, "invalid experiment configuration")
	}

	var modelBytes []byte
	if params.ParentID != nil {
		var dbErr error
		modelBytes, dbErr = m.db.ExperimentModelDefinitionRaw(*params.ParentID)
		if dbErr != nil {
			return nil, config, nil, false, nil, errors.Wrapf(
				dbErr, "unable to find parent experiment %v", *params.ParentID)
		}
	} else {
		var compressErr error
		modelBytes, compressErr = archive.ToTarGz(params.ModelDef)
		if compressErr != nil {
			return nil, config, nil, false, nil, errors.Wrapf(
				compressErr, "unable to find compress model definition")
		}
	}

	token, createSessionErr := m.db.StartUserSession(user)
	if createSessionErr != nil {
		return nil, config, nil, false, nil, errors.Wrapf(
			createSessionErr, "unable to create user session inside task")
	}
	taskSpec.UserSessionToken = token
	taskSpec.Owner = user

	dbExp, err := model.NewExperiment(
		config, params.ConfigBytes, modelBytes, params.ParentID, params.Archived,
		params.GitRemote, params.GitCommit, params.GitCommitter, params.GitCommitDate,
		int(project.Id),
	)
	if user != nil {
		dbExp.OwnerID = &user.ID
		dbExp.Username = user.Username
	}

	taskSpec.Project = config.Project()
	taskSpec.Workspace = config.Workspace()
	for label := range config.Labels() {
		taskSpec.Labels = append(taskSpec.Labels, label)
	}

	return dbExp, config, project, params.ValidateOnly, &taskSpec, err
}
