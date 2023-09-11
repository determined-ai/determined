package internal

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
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
	pkgTemplate "github.com/determined-ai/determined/master/internal/template"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/ptrs"
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

func getCreateExperimentsProject(
	m *Master, req *apiv1.CreateExperimentRequest, user *model.User, config expconf.ExperimentConfig,
) (*projectv1.Project, error) {
	// Place experiment in Uncategorized, unless project set in request params or config.
	var err error
	projectID := model.DefaultProjectID
	errProjectNotFound := api.NotFoundErrs("project", fmt.Sprint(projectID), true)
	if req.ProjectId > 1 {
		projectID = int(req.ProjectId)
		errProjectNotFound = api.NotFoundErrs("project", fmt.Sprint(projectID), true)
	} else {
		if (config.Workspace() == "") != (config.Project() == "") {
			return nil,
				errors.New("workspace and project must both be included in config if one is provided")
		}
		if config.Workspace() != "" && config.Project() != "" {
			errProjectNotFound = api.NotFoundErrs("workspace/project",
				config.Workspace()+"/"+config.Project(), true)

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

// unmarshalTemplateConfig unmarshals the template config into `o` and returns api-ready errors.
func (m *Master) unmarshalTemplateConfig(ctx context.Context, templateName string,
	user *model.User, out interface{}, disallowUnknownFields bool,
) error {
	notFoundErr := status.Errorf(codes.InvalidArgument,
		api.NotFoundErrMsg("temlpate", fmt.Sprint(templateName)))
	template, err := m.db.TemplateByName(templateName)
	if err != nil {
		return notFoundErr
	}
	permErr, err := pkgTemplate.AuthZProvider.Get().CanViewTemplate(ctx,
		user, model.AccessScopeID(template.WorkspaceID))
	if err != nil {
		return err
	}
	if permErr != nil {
		return notFoundErr
	}
	if disallowUnknownFields {
		err = yaml.Unmarshal(template.Config, out, yaml.DisallowUnknownFields)
	} else {
		err = yaml.Unmarshal(template.Config, out)
	}
	if err != nil {
		return errors.Wrapf(err, "yaml.Unmarshal(template=%s)", templateName)
	}
	return nil
}

func (m *Master) parseCreateExperiment(req *apiv1.CreateExperimentRequest, user *model.User) (
	*model.Experiment, expconf.ExperimentConfig, *projectv1.Project, *tasks.TaskSpec, error,
) {
	ctx := context.TODO()
	// Read the config as the user provided it.
	config, err := expconf.ParseAnyExperimentConfigYAML([]byte(req.Config))
	if err != nil {
		return nil, config, nil, nil, errors.Wrap(err, "invalid experiment configuration")
	}

	// Apply the template that the user specified.
	if req.Template != nil {
		var tc expconf.ExperimentConfig
		err := m.unmarshalTemplateConfig(ctx, *req.Template, user, &tc, true)
		if err != nil {
			return nil, config, nil, nil, err
		}
		config = schemas.Merge(config, tc)
	}

	defaulted := schemas.WithDefaults(config)
	resources := defaulted.Resources()

	p, err := getCreateExperimentsProject(m, req, user, defaulted)
	if err != nil {
		return nil, config, nil, nil, err
	}
	workspaceModel, err := workspace.WorkspaceByProjectID(ctx, int(p.Id))
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, config, nil, nil, err
	}
	workspaceID := resolveWorkspaceID(workspaceModel)
	poolName, err := m.rm.ResolveResourcePool(
		m.system, resources.ResourcePool(), workspaceID, resources.SlotsPerTrial())
	if err != nil {
		return nil, config, nil, nil, errors.Wrapf(err, "invalid resource configuration")
	}
	if err = m.rm.ValidateResources(m.system, poolName, resources.SlotsPerTrial(), false); err != nil {
		return nil, config, nil, nil, errors.Wrapf(err, "error validating resources")
	}
	taskContainerDefaults, err := m.rm.TaskContainerDefaults(
		m.system,
		poolName,
		m.config.TaskContainerDefaults,
	)
	if err != nil {
		return nil, config, nil, nil, errors.Wrapf(err, "error getting TaskContainerDefaults")
	}
	taskSpec := *m.taskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults
	taskSpec.TaskContainerDefaults.MergeIntoExpConfig(&config)
	if defaulted.RawEntrypoint == nil && (req.Unmanaged == nil || !*req.Unmanaged) {
		return nil, config, nil, nil, errors.New("managed experiments require entrypoint")
	}

	// Merge in workspace's checkpoint storage into the conifg.
	w := &model.Workspace{}
	if err = db.Bun().NewSelect().Model(w).
		Where("id = ?", p.WorkspaceId).
		Column("checkpoint_storage_config").
		Scan(ctx); err != nil {
		return nil, config, nil, nil, err
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
		return nil, config, nil, nil, errors.Wrap(err, "invalid experiment configuration")
	}

	// Disallow EOL searchers.
	if err = config.Searcher().AssertCurrent(); err != nil {
		return nil, config, nil, nil, errors.Wrap(err, "invalid experiment configuration")
	}

	var modelBytes []byte
	var parentID *int
	if req.ParentId != 0 {
		parentID = ptrs.Ptr(int(req.ParentId))
		var dbErr error
		modelBytes, dbErr = m.db.ExperimentModelDefinitionRaw(int(req.ParentId))
		if dbErr != nil {
			return nil, config, nil, nil, errors.Wrapf(
				dbErr, "unable to find parent experiment %v", req.ParentId)
		}
	} else {
		var compressErr error
		if req.ModelDefinition != nil {
			modelBytes, compressErr = archive.ToTarGz(filesToArchive(req.ModelDefinition))
			if compressErr != nil {
				return nil, config, nil, nil, errors.Wrapf(
					compressErr, "unable to find compress model definition")
			}
		} else {
			modelBytes = []byte{}
		}

	}

	token, createSessionErr := m.db.StartUserSession(user)
	if createSessionErr != nil {
		return nil, config, nil, nil, errors.Wrapf(
			createSessionErr, "unable to create user session inside task")
	}
	taskSpec.UserSessionToken = token
	taskSpec.Owner = user

	var commitDate *time.Time
	pt, err := protoutils.ToTime(req.GitCommitDate)
	if err == nil {
		commitDate = &pt
	}

	dbExp, err := model.NewExperiment(
		config, req.Config, modelBytes, parentID, false,
		req.GitRemote, req.GitCommit, req.GitCommitter, commitDate,
		int(p.Id), req.Unmanaged != nil && *req.Unmanaged,
	)
	if err != nil {
		return nil, config, nil, nil, err
	}
	if user != nil {
		dbExp.OwnerID = &user.ID
		dbExp.Username = user.Username
	}

	taskSpec.Project = config.Project()
	taskSpec.Workspace = config.Workspace()
	for label := range config.Labels() {
		taskSpec.Labels = append(taskSpec.Labels, label)
	}

	return dbExp, config, p, &taskSpec, err
}
