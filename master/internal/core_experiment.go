package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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

func (m *Master) getExperimentSummaries(c echo.Context) (interface{}, error) {
	type ExperimentSummary struct {
		ID        int             `db:"id" json:"id"`
		State     string          `db:"state" json:"state"`
		OwnerID   int             `db:"owner_id" json:"owner_id"`
		Progress  *float64        `db:"progress" json:"progress"`
		Archived  bool            `db:"archived" json:"archived"`
		StartTime string          `db:"start_time" json:"start_time"`
		EndTime   *string         `db:"end_time" json:"end_time"`
		Config    json.RawMessage `db:"config" json:"config"`
	}
	states := c.QueryParam("states")
	if states == "" {
		var allStates []string
		for state := range model.ExperimentTransitions {
			allStates = append(allStates, string(state))
		}
		states = strings.Join(allStates, ",")
	}
	var results []ExperimentSummary
	err := m.db.Query("get_experiment_summaries", &results, states)
	return results, err
}

func (m *Master) getExperimentList(c echo.Context) (interface{}, error) {
	userFilter := c.QueryParam("user")
	skipInactive, err := strconv.ParseBool(c.QueryParam("skipInactive"))
	if err != nil {
		skipInactive = false
	}
	if userFilter != "" {
		return m.db.ExperimentDescriptorsRawForUser(true, skipInactive, userFilter)
	}
	return m.db.ExperimentDescriptorsRaw(true, skipInactive)
}

func (m *Master) getExperiments(c echo.Context) (interface{}, error) {
	query, err := ParseExperimentsQuery(c)
	if err != nil {
		return nil, err
	}

	skipArchived := query.Filter != "all"

	return m.db.ExperimentListRaw(skipArchived, query.User, query.Limit, query.Offset)
}

func (m *Master) getExperiment(c echo.Context) (interface{}, error) {
	args := struct {
		ExperimentID int `path:"experiment_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	return m.db.ExperimentRaw(args.ExperimentID)
}

func (m *Master) getExperimentCheckpoints(c echo.Context) (interface{}, error) {
	args := struct {
		ExperimentID int  `path:"experiment_id"`
		NumBest      *int `query:"best"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	return m.db.ExperimentCheckpointsRaw(args.ExperimentID, args.NumBest)
}

func (m *Master) getExperimentSummary(c echo.Context) (interface{}, error) {
	args := struct {
		ExperimentID int `path:"experiment_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	return m.db.ExperimentWithTrialSummariesRaw(args.ExperimentID)
}

func (m *Master) getExperimentConfig(c echo.Context) (interface{}, error) {
	args := struct {
		ExperimentID int `path:"experiment_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	return m.db.ExperimentConfigRaw(args.ExperimentID)
}

func (m *Master) getExperimentSummaryMetrics(c echo.Context) (interface{}, error) {
	args := struct {
		ExperimentID int `path:"experiment_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	return m.db.ExperimentWithSummaryMetricsRaw(args.ExperimentID)
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
	return m.db.ExperimentCheckpointsToGCRaw(
		args.ExperimentID, args.ExperimentBest, args.TrialBest, args.TrialLatest, false)
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
		State *model.State `json:"state"`
		// TODO: the config-level items like `description` are really at a different level
		// than the top-level items, we should reorganize this into ExperimentPatch and
		// ExperimentConfigPatch.
		Description *string `json:"description"`
		// Labels set to nil are deleted.
		Labels    map[string]*bool `json:"labels"`
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
		Archived *bool `json:"archived"`
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

	if patch.Archived != nil {
		dbExp.Archived = *patch.Archived
		if err := m.db.SaveExperimentArchiveStatus(dbExp); err != nil {
			return nil, errors.Wrapf(err, "archiving experiment %d", dbExp.ID)
		}
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
	if patch.Description != nil {
		dbExp.Config.SetDescription(patch.Description)
	}
	labels := dbExp.Config.Labels()
	for label, keep := range patch.Labels {
		switch _, ok := labels[label]; {
		case ok && keep == nil:
			delete(labels, label)
		case !ok && keep != nil:
			if labels == nil {
				labels = make(expconf.Labels)
			}
			labels[label] = true
		}
	}
	dbExp.Config.SetLabels(labels)
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

	if patch.State != nil {
		m.system.TellAt(actor.Addr("experiments", args.ExperimentID), *patch.State)
	}

	if patch.Resources != nil {
		if patch.Resources.MaxSlots.IsPresent {
			m.system.TellAt(actor.Addr("experiments", args.ExperimentID),
				sproto.SetGroupMaxSlots{MaxSlots: patch.Resources.MaxSlots.Value})
		}
		if patch.Resources.Weight != nil {
			m.system.TellAt(actor.Addr("experiments", args.ExperimentID),
				sproto.SetGroupWeight{Weight: *patch.Resources.Weight})
		}
		if patch.Resources.Priority != nil {
			m.system.TellAt(actor.Addr("experiments", args.ExperimentID),
				sproto.SetGroupPriority{Priority: patch.Resources.Priority})
		}
	}

	if patch.CheckpointStorage != nil {
		checkpoints, err := m.db.ExperimentCheckpointsToGCRaw(
			dbExp.ID,
			dbExp.Config.CheckpointStorage().SaveExperimentBest(),
			dbExp.Config.CheckpointStorage().SaveTrialBest(),
			dbExp.Config.CheckpointStorage().SaveTrialLatest(),
			true,
		)
		if err != nil {
			return nil, err
		}

		taskSpec := *m.taskSpec
		taskSpec.AgentUserGroup = agentUserGroup

		m.system.ActorOf(actor.Addr(fmt.Sprintf("patch-checkpoint-gc-%s", uuid.New().String())),
			&checkpointGCTask{
				GCCkptSpec: tasks.GCCkptSpec{
					Base:               taskSpec,
					ExperimentID:       dbExp.ID,
					LegacyConfig:       dbExp.Config.AsLegacy(),
					ToDelete:           checkpoints,
					DeleteTensorboards: true,
				},
				rm: m.rm,
				db: m.db,
			})
	}

	return nil, nil
}

// CreateExperimentParams defines a request to create an experiment.
type CreateExperimentParams struct {
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
}

func (m *Master) parseCreateExperiment(params *CreateExperimentParams) (
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
	poolName, err := sproto.GetResourcePool(
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

	dbExp, err := model.NewExperiment(
		config, params.ConfigBytes, modelBytes, params.ParentID, params.Archived,
		params.GitRemote, params.GitCommit, params.GitCommitter, params.GitCommitDate)
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

	dbExp, validateOnly, taskSpec, err := m.parseCreateExperiment(&params)
	if err != nil {
		return nil, echo.NewHTTPError(
			http.StatusBadRequest,
			errors.Wrap(err, "invalid experiment"))
	}

	if validateOnly {
		return nil, nil
	}

	dbExp.OwnerID = &user.ID
	e, err := newExperiment(m, dbExp, taskSpec)
	if err != nil {
		return nil, errors.Wrap(err, "starting experiment")
	}
	m.system.ActorOf(actor.Addr("experiments", e.ID), e)

	c.Response().Header().Set(echo.HeaderLocation, fmt.Sprintf("/experiments/%v", e.ID))
	response := model.ExperimentDescriptor{
		ID:       e.ID,
		Archived: false,
		Config:   e.Config,
		Labels:   make([]string, 0),
	}
	return response, nil
}

func (m *Master) postExperimentKill(c echo.Context) (interface{}, error) {
	args := struct {
		ExperimentID int `path:"experiment_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}

	exp := actor.Addr("experiments", args.ExperimentID)
	resp := m.system.AskAt(exp, &apiv1.KillExperimentRequest{})
	if resp.Source() == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("active experiment not found: %d", args.ExperimentID))
	}
	if _, notTimedOut := resp.GetOrTimeout(defaultAskTimeout); !notTimedOut {
		return nil, errors.Errorf("attempt to kill experiment timed out")
	}
	return nil, nil
}
