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

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
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
		ExperimentID   int  `path:"experiment_id"`
		ExperimentBest *int `query:"save_experiment_best"`
		TrialBest      *int `query:"save_trial_best"`
		TrialLatest    *int `query:"save_trial_latest"`
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
	cleanDescription := reg.ReplaceAllString(expConfig.Description, "")

	// Truncate description to a smaller size to both accommodate file name and path size
	// limits on different platforms as well as get users more accustom to picking shorter
	// descriptions as we move toward "description as mnemonic for an experiment".
	maxDescriptionLength := 50
	if len(cleanDescription) > maxDescriptionLength {
		cleanDescription = cleanDescription[0:maxDescriptionLength]
	}

	c.Response().Header().Set(
		"Content-Disposition",
		fmt.Sprintf(
			`attachment; filename="exp%d_%s_model_def.tar.gz"`,
			args.ExperimentID,
			cleanDescription))
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
		if patch.Resources.MaxSlots.IsPresent {
			dbExp.Config.Resources.MaxSlots = patch.Resources.MaxSlots.Value
		}
		if patch.Resources.Weight != nil {
			dbExp.Config.Resources.Weight = *patch.Resources.Weight
		}
	}
	if patch.Description != nil {
		dbExp.Config.Description = *patch.Description
	}
	for label, keep := range patch.Labels {
		switch _, ok := dbExp.Config.Labels[label]; {
		case ok && keep == nil:
			delete(dbExp.Config.Labels, label)
		case !ok && keep != nil:
			if dbExp.Config.Labels == nil {
				dbExp.Config.Labels = make(model.Labels)
			}
			dbExp.Config.Labels[label] = true
		}
	}
	if patch.CheckpointStorage != nil {
		dbExp.Config.CheckpointStorage.SaveExperimentBest = patch.CheckpointStorage.SaveExperimentBest
		dbExp.Config.CheckpointStorage.SaveTrialBest = patch.CheckpointStorage.SaveTrialBest
		dbExp.Config.CheckpointStorage.SaveTrialLatest = patch.CheckpointStorage.SaveTrialLatest
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
	}

	if patch.CheckpointStorage != nil {
		m.system.ActorOf(actor.Addr(fmt.Sprintf("patch-checkpoint-gc-%s", uuid.New().String())),
			&checkpointGCTask{
				agentUserGroup: agentUserGroup,
				taskSpec:       m.taskSpec,
				rp:             m.rp,
				db:             m.db,
				experiment:     dbExp,
			})
	}

	return nil, nil
}

func (m *Master) parseExperiment(body []byte) (*model.Experiment, bool, error) {
	var params struct {
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
	if err := json.Unmarshal(body, &params); err != nil {
		return nil, false, errors.Wrap(err, "invalid experiment params")
	}

	config := model.DefaultExperimentConfig(&m.config.TaskContainerDefaults)

	checkpointStorage, err := m.config.CheckpointStorage.ToModel()
	if err != nil {
		return nil, false, errors.Wrap(err, "invalid experiment configuration")
	}

	config.CheckpointStorage = *checkpointStorage

	if params.Template != nil {
		template, terr := m.db.TemplateByName(*params.Template)
		if terr != nil {
			return nil, false, terr
		}
		if yerr := yaml.Unmarshal(template.Config, &config, yaml.DisallowUnknownFields); yerr != nil {
			return nil, false, yerr
		}
	}

	if yerr := yaml.Unmarshal(
		[]byte(params.ConfigBytes), &config, yaml.DisallowUnknownFields,
	); yerr != nil {
		return nil, false, errors.Wrap(yerr, "invalid experiment configuration")
	}

	if config.Environment.PodSpec == nil {
		if config.Resources.SlotsPerTrial == 0 {
			config.Environment.PodSpec = m.config.TaskContainerDefaults.CPUPodSpec
		} else {
			config.Environment.PodSpec = m.config.TaskContainerDefaults.GPUPodSpec
		}
	}

	if cerr := check.Validate(config); cerr != nil {
		return nil, false, errors.Wrap(cerr, "invalid experiment configuration")
	}

	var modelBytes []byte
	if params.ParentID != nil {
		var dbErr error
		modelBytes, dbErr = m.db.ExperimentModelDefinitionRaw(*params.ParentID)
		if dbErr != nil {
			return nil, false, errors.Wrapf(
				dbErr, "unable to find parent experiment %v", *params.ParentID)
		}
	} else {
		var compressErr error
		modelBytes, compressErr = archive.ToTarGz(params.ModelDef)
		if compressErr != nil {
			return nil, false, errors.Wrapf(
				compressErr, "unable to find compress model definition")
		}
	}

	dbExp, err := model.NewExperiment(
		config, modelBytes, params.ParentID, params.Archived,
		params.GitRemote, params.GitCommit, params.GitCommitter, params.GitCommitDate)
	return dbExp, params.ValidateOnly, err
}

func (m *Master) postExperiment(c echo.Context) (interface{}, error) {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	user := c.(*context.DetContext).MustGetUser()

	dbExp, validateOnly, err := m.parseExperiment(body)

	if err != nil {
		return nil, echo.NewHTTPError(
			http.StatusBadRequest,
			errors.Wrap(err, "invalid experiment"))
	}

	if validateOnly {
		return nil, c.NoContent(http.StatusNoContent)
	}

	dbExp.OwnerID = &user.ID
	e, err := newExperiment(m, dbExp)
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
	return c.JSON(http.StatusCreated, response), nil
}

func (m *Master) deleteExperiment(c echo.Context) (interface{}, error) {
	args := struct {
		ExperimentID int `path:"experiment_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}
	expID := args.ExperimentID
	dbExp, err := m.db.ExperimentByID(expID)
	if err != nil {
		return nil, errors.Wrapf(err, "loading experiment %v to delete", expID)
	}
	if _, ok := model.TerminalStates[dbExp.State]; !ok {
		return nil, errors.Errorf("cannot delete experiment %v in state %v", expID, dbExp.State)
	}

	agentUserGroup, err := m.db.AgentUserGroup(*dbExp.OwnerID)
	if err != nil {
		return nil, errors.Errorf("cannot find user and group for experiment %v", expID)
	}
	if agentUserGroup == nil {
		agentUserGroup = &m.config.Security.DefaultTask
	}

	// Change the GC policy to remove all checkpoints. This will trigger a checkpoint GC task,
	// if needed, to remove the checkpoint files.
	dbExp.Config.CheckpointStorage.SaveExperimentBest = 0
	dbExp.Config.CheckpointStorage.SaveTrialBest = 0
	dbExp.Config.CheckpointStorage.SaveTrialLatest = 0
	if serr := m.db.SaveExperimentConfig(dbExp); serr != nil {
		return nil, errors.Wrapf(serr, "patching experiment %d", dbExp.ID)
	}
	addr := actor.Addr(fmt.Sprintf("delete-checkpoint-gc-%s", uuid.New().String()))
	m.system.ActorOf(addr, &checkpointGCTask{
		agentUserGroup: agentUserGroup,
		taskSpec:       m.taskSpec,
		rp:             m.rp,
		db:             m.db,
		experiment:     dbExp,
	})

	c.Logger().Infof("deleting experiment %v from database", expID)
	if err = m.db.DeleteExperiment(expID); err != nil {
		return nil, errors.Wrapf(err, "deleting experiment %v from database", expID)
	}
	return nil, nil
}

func (m *Master) postExperimentKill(c echo.Context) (interface{}, error) {
	args := struct {
		ExperimentID int `path:"experiment_id"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}

	resp := m.system.AskAt(actor.Addr("experiments", args.ExperimentID), killExperiment{})
	if resp.Source() == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("active experiment not found: %d", args.ExperimentID))
	}
	if _, notTimedOut := resp.GetOrTimeout(defaultAskTimeout); !notTimedOut {
		return nil, errors.Errorf("attempt to kill experiment timed out")
	}
	return nil, nil
}
