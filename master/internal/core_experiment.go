package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
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

	checkpointUUIDs, err := m.db.ExperimentCheckpointsToGCRaw(
		args.ExperimentID, args.ExperimentBest, args.TrialBest, args.TrialLatest)
	if err != nil {
		return nil, err
	}
	checkpointsDB, err := m.db.CheckpointByUUIDs(checkpointUUIDs)
	if err != nil {
		return nil, err
	}

	expConfig, err := m.db.ExperimentConfig(args.ExperimentID)
	if err != nil {
		return nil, err
	}

	metricName := expConfig.Searcher().Metric()
	checkpointsWithMetric := map[string]interface{}{
		"checkpoints": checkpointsDB, "metric_name": metricName,
	}

	return checkpointsWithMetric, nil
}

// @Summary Get individual file from modal definitions for download.
// @Tags Experiments
// @ID get-experiment-model-file
// @Accept  json
// @Produce  text/plain; charset=utf-8
// @Param   experiment_id path int  true  "Experiment ID"
// @Param   path query string true "Path to the target file"
// @Success 200 {} string ""
//nolint:godot
// @Router /experiments/{experiment_id}/file/download [get]
func (m *Master) getExperimentModelFile(c echo.Context) error {
	args := struct {
		ExperimentID int    `path:"experiment_id"`
		Path         string `query:"path"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
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

	modelDef, err := m.db.ExperimentModelDefinitionRaw(args.ExperimentID)
	if err != nil {
		return err
	}

	expConfig, err := m.db.ExperimentConfig(args.ExperimentID)
	if err != nil {
		return err
	}

	// Make a Regex to remove everything but a whitelist of characters.
	reg := regexp.MustCompile(`[^A-Za-z0-9_ \-()[\].{}]+`)
	cleanName := reg.ReplaceAllString(expConfig.Name().String(), "")

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
			`attachment; filename="exp%d_%s_model_def.tar.gz"`,
			args.ExperimentID,
			cleanName))
	return c.Blob(http.StatusOK, "application/x-gtar", modelDef)
}

func (m *Master) patchExperiment(c echo.Context) (interface{}, error) {
	// Allow clients to apply partial updates to an experiment via the JSON Merge Patch format
	// (RFC 7386). Clients can only update certain fields of the experiment.
	args := struct {
		ExperimentID int `path:"experiment_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	// `patch` represents the allowed mutations that can be performed on an experiment, in JSON
	// Merge Patch (RFC 7386) format.
	// TODO: check for extraneous fields.
	patch := struct {
		Resources *struct {
			MaxSlots api.MaybeInt `json:"max_slots"`
			Weight   *float64     `json:"weight"`
			Priority *int         `json:"priority"`
		} `json:"resources"`
		CheckpointStorage *struct {
			SaveExperimentBest int `json:"save_experiment_best"`
			SaveTrialBest      int `json:"save_trial_best"`
			SaveTrialLatest    int `json:"save_trial_latest"`
		} `json:"checkpoint_storage"`
	}{}
	if err := api.BindPatch(&patch, c); err != nil {
		return nil, err
	}

	dbExp, err := m.db.ExperimentByID(args.ExperimentID)
	if err != nil {
		return nil, errors.Wrapf(err, "loading experiment %v", args.ExperimentID)
	}

	agentUserGroup, err := m.db.AgentUserGroup(*dbExp.OwnerID)
	if err != nil {
		return nil, errors.Errorf("cannot find user and group for experiment %v", dbExp.OwnerID)
	}

	if agentUserGroup == nil {
		agentUserGroup = &m.config.Security.DefaultTask
	}

	ownerFullUser, err := m.db.UserByID(*dbExp.OwnerID)
	if err != nil {
		return nil, errors.Errorf("cannot find user %v who owns experiment", dbExp.OwnerID)
	}

	if patch.Resources != nil {
		resources := dbExp.Config.Resources()
		if patch.Resources.MaxSlots.IsPresent {
			resources.SetMaxSlots(patch.Resources.MaxSlots.Value)
		}
		if patch.Resources.Weight != nil {
			resources.SetWeight(*patch.Resources.Weight)
		}
		if patch.Resources.Priority != nil {
			resources.SetPriority(patch.Resources.Priority)
		}
		dbExp.Config.SetResources(resources)
	}
	if patch.CheckpointStorage != nil {
		storage := dbExp.Config.CheckpointStorage()
		storage.SetSaveExperimentBest(patch.CheckpointStorage.SaveExperimentBest)
		storage.SetSaveTrialBest(patch.CheckpointStorage.SaveTrialBest)
		storage.SetSaveTrialLatest(patch.CheckpointStorage.SaveTrialLatest)
		dbExp.Config.SetCheckpointStorage(storage)
	}

	if err := m.db.SaveExperimentConfig(dbExp); err != nil {
		return nil, errors.Wrapf(err, "patching experiment %d", dbExp.ID)
	}

	if patch.Resources != nil {
		if patch.Resources.MaxSlots.IsPresent {
			m.system.TellAt(actor.Addr("experiments", args.ExperimentID),
				sproto.SetGroupMaxSlots{MaxSlots: patch.Resources.MaxSlots.Value})
		}
		if patch.Resources.Weight != nil {
			resp := m.system.AskAt(actor.Addr("experiments", args.ExperimentID),
				sproto.SetGroupWeight{Weight: *patch.Resources.Weight})
			if resp.Error() != nil {
				return nil, errors.Errorf("cannot change experiment weight to %v", *patch.Resources.Weight)
			}
		}
		if patch.Resources.Priority != nil {
			resp := m.system.AskAt(actor.Addr("experiments", args.ExperimentID),
				sproto.SetGroupPriority{Priority: *patch.Resources.Priority})
			if resp.Error() != nil {
				return nil, errors.Errorf("cannot change experiment priority to %v", *patch.Resources.Priority)
			}
		}
	}

	if patch.CheckpointStorage != nil {
		checkpoints, err := m.db.ExperimentCheckpointsToGCRaw(
			dbExp.ID,
			dbExp.Config.CheckpointStorage().SaveExperimentBest(),
			dbExp.Config.CheckpointStorage().SaveTrialBest(),
			dbExp.Config.CheckpointStorage().SaveTrialLatest(),
		)
		if err != nil {
			return nil, err
		}

		taskSpec := *m.taskSpec
		user := &model.User{
			ID:       ownerFullUser.ID,
			Username: ownerFullUser.Username,
		}

		taskID := model.NewTaskID()
		ckptGCTask := newCheckpointGCTask(
			m.rm, m.db, m.taskLogger, taskID, dbExp.JobID, dbExp.StartTime, taskSpec, dbExp.ID,
			dbExp.Config.AsLegacy(), checkpoints, true, agentUserGroup, user, nil,
		)
		m.system.ActorOf(actor.Addr(fmt.Sprintf("patch-checkpoint-gc-%s", uuid.New().String())),
			ckptGCTask)
	}

	return nil, nil
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
	Project       *string         `json:"project"`
	ProjectID     *int            `json:"project_id"`
	Workspace     *string         `json:"workspace"`
}

func (m *Master) parseCreateExperiment(params *CreateExperimentParams, user *model.User) (
	*model.Experiment, bool, *tasks.TaskSpec, error,
) {
	// Read the config as the user provided it.
	config, err := expconf.ParseAnyExperimentConfigYAML([]byte(params.ConfigBytes))
	if err != nil {
		return nil, false, nil, errors.Wrap(err, "invalid experiment configuration")
	}

	// Apply the template that the user specified.
	if params.Template != nil {
		template, terr := m.db.TemplateByName(*params.Template)
		if terr != nil {
			return nil, false, nil, terr
		}
		var tc expconf.ExperimentConfig
		if yerr := yaml.Unmarshal(template.Config, &tc, yaml.DisallowUnknownFields); yerr != nil {
			return nil, false, nil, yerr
		}
		// Merge the template into the config.
		config = schemas.Merge(config, tc).(expconf.ExperimentConfig)
	}

	resources := schemas.WithDefaults(config).(expconf.ExperimentConfig).Resources()
	poolName, err := m.rm.ResolveResourcePool(
		m.system, resources.ResourcePool(), resources.SlotsPerTrial(), false)
	if err != nil {
		return nil, false, nil, errors.Wrapf(err, "invalid resource configuration")
	}

	taskContainerDefaults := m.getTaskContainerDefaults(poolName)
	taskSpec := *m.taskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults
	taskSpec.TaskContainerDefaults.MergeIntoExpConfig(&config)

	// Merge in the master's checkpoint storage into the config.
	config.RawCheckpointStorage = schemas.Merge(
		config.RawCheckpointStorage, &m.config.CheckpointStorage,
	).(*expconf.CheckpointStorageConfig)

	// Lastly, apply any json-schema-defined defaults.
	config = schemas.WithDefaults(config).(expconf.ExperimentConfig)

	// Make sure the experiment config has all eventuallyRequired fields.
	if err = schemas.IsComplete(config); err != nil {
		return nil, false, nil, errors.Wrap(err, "invalid experiment configuration")
	}

	// Disallow EOL searchers.
	if err = config.Searcher().AssertCurrent(); err != nil {
		return nil, false, nil, errors.Wrap(err, "invalid experiment configuration")
	}

	var modelBytes []byte
	if params.ParentID != nil {
		var dbErr error
		modelBytes, dbErr = m.db.ExperimentModelDefinitionRaw(*params.ParentID)
		if dbErr != nil {
			return nil, false, nil, errors.Wrapf(
				dbErr, "unable to find parent experiment %v", *params.ParentID)
		}
	} else {
		var compressErr error
		modelBytes, compressErr = archive.ToTarGz(params.ModelDef)
		if compressErr != nil {
			return nil, false, nil, errors.Wrapf(
				compressErr, "unable to find compress model definition")
		}
	}

	token, createSessionErr := m.db.StartUserSession(user)
	if createSessionErr != nil {
		return nil, false, nil, errors.Wrapf(
			createSessionErr, "unable to create user session inside task")
	}
	taskSpec.UserSessionToken = token
	taskSpec.Owner = user

	// Place experiment in Uncategorized, unless project set in config or CreateExperimentParams
	// CreateExperimentParams has highest priority.
	projectID := 1
	if params.ProjectID == nil {
		if (config.Workspace() == "") != (config.Project() == "") {
			return nil, false, nil,
				errors.New("workspace and project must both be included in config if one is provided")
		}
		if config.Workspace() != "" && config.Project() != "" {
			projectID, err = m.db.ProjectByName(config.Workspace(), config.Project())
			if err != nil {
				return nil, false, nil, errors.Wrapf(
					err, "unable to find parent workspace and project")
			}
		}
	} else {
		projectID = *params.ProjectID
	}

	dbExp, err := model.NewExperiment(
		config, params.ConfigBytes, modelBytes, params.ParentID, params.Archived,
		params.GitRemote, params.GitCommit, params.GitCommitter, params.GitCommitDate,
		projectID,
	)
	if user != nil {
		dbExp.OwnerID = &user.ID
		dbExp.Username = user.Username
	}

	return dbExp, params.ValidateOnly, &taskSpec, err
}

func (m *Master) postExperiment(c echo.Context) (interface{}, error) {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	user := c.(*context.DetContext).MustGetUser()

	var params CreateExperimentParams
	if err = json.Unmarshal(body, &params); err != nil {
		return nil, errors.Wrap(err, "invalid experiment params")
	}

	dbExp, validateOnly, taskSpec, err := m.parseCreateExperiment(&params, &user)
	if err != nil {
		return nil, echo.NewHTTPError(
			http.StatusBadRequest,
			errors.Wrap(err, "invalid experiment"))
	}

	if validateOnly {
		return nil, nil
	}

	e, err := newExperiment(m, dbExp, taskSpec)
	if err != nil {
		return nil, errors.Wrap(err, "starting experiment")
	}
	config, ok := schemas.Copy(e.Config).(expconf.ExperimentConfig)
	if !ok {
		return nil, errors.Errorf("could not copy experiment's config to return")
	}
	m.system.ActorOf(actor.Addr("experiments", e.ID), e)

	if params.Activate {
		exp := actor.Addr("experiments", e.ID)
		resp := m.system.AskAt(exp, &apiv1.ActivateExperimentRequest{Id: int32(e.ID)})
		if resp.Source() == nil {
			return nil, echo.NewHTTPError(http.StatusNotFound,
				fmt.Sprintf("experiment not found: %d", e.ID))
		}
		if _, notTimedOut := resp.GetOrTimeout(defaultAskTimeout); !notTimedOut {
			return nil, errors.Errorf("attempt to activate experiment timed out")
		}
	}

	c.Response().Header().Set(echo.HeaderLocation, fmt.Sprintf("/experiments/%v", e.ID))
	response := model.ExperimentDescriptor{
		ID:       e.ID,
		Archived: false,
		Config:   config,
		Labels:   make([]string, 0),
	}
	return response, nil
}
