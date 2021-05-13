package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	pstruct "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

var commandsAddr = actor.Addr("commands")

type protoCommandParams struct {
	TemplateName string
	Config       *pstruct.Struct
	Files        []*utilv1.File
	Data         []byte
	MustZeroSlot bool
	Preview      bool
}

func (a *apiServer) makeFullCommandSpec(
	configBytes []byte, templateName *string, mustBeZeroSlot bool,
) (*model.CommandConfig, *tasks.TaskSpec, error) {
	resources := model.ParseJustResources(configBytes)
	taskSpec := a.m.makeTaskSpec(resources.ResourcePool, resources.Slots)
	config := command.DefaultConfig(&taskSpec.TaskContainerDefaults)
	if templateName != nil && *templateName != "" {
		template, err := a.m.db.TemplateByName(*templateName)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to find template: %s", *templateName)
		}
		if err := yaml.Unmarshal(template.Config, &config); err != nil {
			return nil, nil, errors.Wrapf(err, "failed to unmarshal template: %s", *templateName)
		}
	}

	if len(configBytes) != 0 {
		dec := json.NewDecoder(bytes.NewBuffer(configBytes))
		dec.DisallowUnknownFields()

		if err := dec.Decode(&config); err != nil {
			return nil, nil, errors.Wrapf(
				err,
				"unable to parse the config in the parameters: %s",
				string(configBytes),
			)
		}
	}

	// mustBeZeroSlot indicates that this type of command may never use more than
	// zero slots (as of Jan 2021, this is only Tensorboards). This is important
	// when building up the config, so that we can route the command to the
	// correct default resource pool. If the user didn't explicitly set the 'slots' field,
	// the DefaultConfig will set slots=1. We need to correct that before attempting to fill
	// in the default resource pool as otherwise we will mistakenly route this CPU task to the
	// default GPU pool.
	if mustBeZeroSlot {
		config.Resources.Slots = 0
	}

	if err := sproto.ValidateRP(a.m.system, config.Resources.ResourcePool); err != nil {
		return nil, nil, errors.Wrapf(
			err, "resource pool does not exist: %s", config.Resources.ResourcePool,
		)
	}

	// If the resource pool isn't set, fill in the default at creation time.
	if config.Resources.ResourcePool == "" {
		if config.Resources.Slots == 0 {
			config.Resources.ResourcePool = sproto.GetDefaultCPUResourcePool(a.m.system)
		} else {
			config.Resources.ResourcePool = sproto.GetDefaultGPUResourcePool(a.m.system)
		}
	}

	return &config, &taskSpec, nil
}

// prepareLaunchParams prepares launch parameters for Commands, Notebooks, Shells, and TensorBoards.
func (a *apiServer) prepareLaunchParams(ctx context.Context, req *protoCommandParams) (
	*command.CommandParams, error,
) {
	params := command.CommandParams{}
	var err error

	// Must get the user and the agent user group
	params.User, _, err = grpcutil.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get the user: %s", err)
	}
	params.AgentUserGroup, err = a.m.db.AgentUserGroup(params.User.ID)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"cannot find user and group information for user %s: %s",
			params.User.Username,
			err,
		)
	}
	if params.AgentUserGroup == nil {
		params.AgentUserGroup = &a.m.config.Security.DefaultTask
	}

	// Get the full configuration.
	var configBytes []byte
	if req.Config != nil {
		configBytes, err = protojson.Marshal(req.Config)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal, "failed to parse config %s: %s", configBytes, err)
		}
	}

	params.FullConfig, params.TaskSpec, err = a.makeFullCommandSpec(
		configBytes, &req.TemplateName, req.MustZeroSlot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to make command spec: %s", err)
	}

	if len(req.Files) > 0 {
		params.UserFiles = filesToArchive(req.Files)
	}

	if len(req.Data) > 0 {
		var data map[string]interface{}
		if err = json.Unmarshal(req.Data, &data); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse data %s: %s", req.Data, err)
		}
		params.Data = data
	}

	return &params, nil
}

func (a *apiServer) GetCommands(
	_ context.Context, req *apiv1.GetCommandsRequest,
) (resp *apiv1.GetCommandsResponse, err error) {
	err = a.actorRequest("/commands", req, &resp)
	if err != nil {
		return nil, err
	}
	a.sort(resp.Commands, req.OrderBy, req.SortBy, apiv1.GetCommandsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Commands, req.Offset, req.Limit)
}

func (a *apiServer) GetCommand(
	_ context.Context, req *apiv1.GetCommandRequest) (resp *apiv1.GetCommandResponse, err error) {
	return resp, a.actorRequest(fmt.Sprintf("/commands/%s", req.CommandId), req, &resp)
}

func (a *apiServer) KillCommand(
	_ context.Context, req *apiv1.KillCommandRequest) (resp *apiv1.KillCommandResponse, err error) {
	return resp, a.actorRequest(fmt.Sprintf("/commands/%s", req.CommandId), req, &resp)
}

func (a *apiServer) LaunchCommand(
	ctx context.Context, req *apiv1.LaunchCommandRequest,
) (*apiv1.LaunchCommandResponse, error) {
	params, err := a.prepareLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		Config:       req.Config,
		Files:        req.Files,
		Data:         req.Data,
	})
	if err != nil {
		return nil, err
	}

	commandLaunchReq := command.CommandLaunchRequest{CommandParams: params}
	commandIDFut := a.m.system.AskAt(commandsAddr, commandLaunchReq)
	if err = api.ProcessActorResponseError(&commandIDFut); err != nil {
		return nil, err
	}

	commandID := commandIDFut.Get().(sproto.TaskID)
	command := a.m.system.AskAt(commandsAddr.Child(commandID), &commandv1.Command{})
	if err = api.ProcessActorResponseError(&command); err != nil {
		return nil, err
	}

	return &apiv1.LaunchCommandResponse{
		Command: command.Get().(*commandv1.Command),
		Config:  protoutils.ToStruct(*params.FullConfig),
	}, nil
}
