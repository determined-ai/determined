package command

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type commandRequest struct {
	Config    model.CommandConfig
	Data      map[string]interface{}
	UserFiles archive.Archive

	Owner          commandOwner
	AgentUserGroup *model.AgentUserGroup
	TaskSpec       tasks.TaskSpec
}

// CommandParams describes parameters for launching a command.
type CommandParams struct {
	ConfigBytes json.RawMessage        `json:"config"`
	Template    *string                `json:"template"`
	UserFiles   archive.Archive        `json:"user_files"`
	Data        map[string]interface{} `json:"data"`
}

func respondBadRequest(ctx *actor.Context, err error) {
	ctx.Respond(echo.NewHTTPError(http.StatusBadRequest, err.Error()))
}

// parseCommandRequest parses an API request from the following components:
//
// - config: The command configuration.
// - template: The configuration template name.
// - user_files: The files to run with the command.
// - data: Additional data for a command.
//
// mustBeZeroSlot indicates that this type of command may never use more than
// zero slots (as of Jan 2021, this is only Tensorboards). This is important
// when building up the config, so that we can route the command to the
// correct default resource pool.
func parseCommandRequest(
	system *actor.System,
	db *db.PgDB,
	user model.User,
	params *CommandParams,
	makeTaskSpec tasks.MakeTaskSpecFn,
	mustBeZeroSlot bool,
) (*commandRequest, error) {
	resources := model.ParseJustResources([]byte(params.ConfigBytes))
	taskSpec := makeTaskSpec(resources.ResourcePool, resources.Slots)
	config := DefaultConfig(&taskSpec.TaskContainerDefaults)
	if params.Template != nil {
		template, err := db.TemplateByName(*params.Template)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(template.Config, &config); err != nil {
			return nil, err
		}
	}

	if len(params.ConfigBytes) != 0 {
		dec := json.NewDecoder(bytes.NewBuffer(params.ConfigBytes))
		dec.DisallowUnknownFields()

		if err := dec.Decode(&config); err != nil {
			return nil, errors.Wrapf(
				err,
				"unable to parse the config in the parameters: %s",
				string(params.ConfigBytes),
			)
		}
	}

	agentUserGroup, err := db.AgentUserGroup(user.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find user and group information for user %s", user.Username)
	}

	// If the user didn't explicitly set the 'slots' field, the DefaultConfig will set
	// slots=1. We need to correct that before attempting to fill in the default resource
	// pool as otherwise we will mistakenly route this CPU task to the default GPU pool.
	if mustBeZeroSlot {
		config.Resources.Slots = 0
	}

	if err := sproto.ValidateRP(system, config.Resources.ResourcePool); err != nil {
		return nil, err
	}

	// If the resource pool isn't set, fill in the default at creation time.
	if config.Resources.ResourcePool == "" {
		if config.Resources.Slots == 0 {
			config.Resources.ResourcePool = sproto.GetDefaultCPUResourcePool(system)
		} else {
			config.Resources.ResourcePool = sproto.GetDefaultGPUResourcePool(system)
		}
	}

	return &commandRequest{
		Config:    config,
		UserFiles: params.UserFiles,
		Data:      params.Data,

		Owner: commandOwner{
			ID:       user.ID,
			Username: user.Username,
		},
		AgentUserGroup: agentUserGroup,
		TaskSpec:       taskSpec,
	}, nil
}
