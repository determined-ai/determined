package command

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
)

type commandRequest struct {
	Config    model.CommandConfig
	Data      map[string]interface{}
	UserFiles archive.Archive

	Owner          commandOwner
	AgentUserGroup *model.AgentUserGroup
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
func parseCommandRequest(
	system *actor.System,
	db *db.PgDB,
	user model.User,
	params *CommandParams,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
) (*commandRequest, error) {
	config := DefaultConfig(taskContainerDefaults)
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
	}, nil
}
