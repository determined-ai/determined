package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	petname "github.com/dustinkirkland/golang-petname"
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
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

var commandsAddr = actor.Addr("commands")

func getRandomPort(min, max int) int {
	return rand.Intn(max-min) + min
}

type protoCommandParams struct {
	TemplateName string
	Config       *pstruct.Struct
	Files        []*utilv1.File
	MustZeroSlot bool
}

func (a *apiServer) getCommandLaunchParams(ctx context.Context, req *protoCommandParams) (
	*tasks.GenericCommandSpec, error,
) {
	var err error

	// Validate the user and get the agent user group.
	user, _, err := grpcutil.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to get the user: %s", err)
	}
	agentUserGroup, err := a.m.db.AgentUserGroup(user.ID)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"cannot find user and group information for user %s: %s",
			user.Username,
			err,
		)
	}
	if agentUserGroup == nil {
		agentUserGroup = &a.m.config.Security.DefaultTask
	}

	var configBytes []byte
	if req.Config != nil {
		configBytes, err = protojson.Marshal(req.Config)
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument, "failed to parse config %s: %s", configBytes, err)
		}
	}

	// Validate the resource configuration.
	resources := model.ParseJustResources(configBytes)
	if req.MustZeroSlot {
		resources.Slots = 0
	}
	poolName, err := sproto.GetResourcePool(
		a.m.system, resources.ResourcePool, resources.Slots, true)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// Get the base TaskSpec.
	taskContainerDefaults := a.m.getTaskContainerDefaults(poolName)
	taskSpec := *a.m.taskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults
	taskSpec.AgentUserGroup = agentUserGroup
	taskSpec.Owner = user

	// Get the full configuration.
	config := command.DefaultConfig(&taskSpec.TaskContainerDefaults)
	if req.TemplateName != "" {
		template, err := a.m.db.TemplateByName(req.TemplateName)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument,
				errors.Wrapf(err, "failed to find template: %s", req.TemplateName).Error())
		}
		if err := yaml.Unmarshal(template.Config, &config); err != nil {
			return nil, status.Errorf(codes.InvalidArgument,
				errors.Wrapf(err, "failed to unmarshal template: %s", req.TemplateName).Error())
		}
	}
	if len(configBytes) != 0 {
		dec := json.NewDecoder(bytes.NewBuffer(configBytes))
		dec.DisallowUnknownFields()

		if err := dec.Decode(&config); err != nil {
			return nil, status.Errorf(codes.InvalidArgument,
				errors.Wrapf(err,
					"unable to decode the merged config: %s", string(configBytes)).Error())
		}
	}
	config.Resources.ResourcePool = poolName
	if req.MustZeroSlot {
		config.Resources.Slots = 0
	}
	if config.Environment.PodSpec == nil {
		if config.Resources.Slots == 0 {
			config.Environment.PodSpec = taskSpec.TaskContainerDefaults.CPUPodSpec
		} else {
			config.Environment.PodSpec = taskSpec.TaskContainerDefaults.GPUPodSpec
		}
	}

	var userFiles archive.Archive
	if len(req.Files) > 0 {
		userFiles = filesToArchive(req.Files)
	}

	return &tasks.GenericCommandSpec{
		Base:      taskSpec,
		Config:    config,
		UserFiles: userFiles,
	}, nil
}

func (a *apiServer) GetCommands(
	_ context.Context, req *apiv1.GetCommandsRequest,
) (resp *apiv1.GetCommandsResponse, err error) {
	err = a.actorRequest(commandsAddr, req, &resp)
	if err != nil {
		return nil, err
	}
	a.sort(resp.Commands, req.OrderBy, req.SortBy, apiv1.GetCommandsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Commands, req.Offset, req.Limit)
}

func (a *apiServer) GetCommand(
	_ context.Context, req *apiv1.GetCommandRequest) (resp *apiv1.GetCommandResponse, err error) {
	return resp, a.actorRequest(commandsAddr.Child(req.CommandId), req, &resp)
}

func (a *apiServer) KillCommand(
	_ context.Context, req *apiv1.KillCommandRequest) (resp *apiv1.KillCommandResponse, err error) {
	return resp, a.actorRequest(commandsAddr.Child(req.CommandId), req, &resp)
}

func (a *apiServer) SetCommandPriority(
	_ context.Context, req *apiv1.SetCommandPriorityRequest,
) (resp *apiv1.SetCommandPriorityResponse, err error) {
	return resp, a.actorRequest(commandsAddr.Child(req.CommandId), req, &resp)
}

func (a *apiServer) LaunchCommand(
	ctx context.Context, req *apiv1.LaunchCommandRequest,
) (*apiv1.LaunchCommandResponse, error) {
	spec, err := a.getCommandLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		Config:       req.Config,
		Files:        req.Files,
	})
	if err != nil {
		return nil, api.APIErr2GRPC(err)
	}

	// Postprocess the spec.
	if spec.Config.Description == "" {
		spec.Config.Description = fmt.Sprintf(
			"Command (%s)",
			petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep),
		)
	}
	if len(spec.Config.Entrypoint) == 1 {
		// If an entrypoint is specified as a singleton string, Determined will follow the "shell form"
		// convention of Docker that executes the Command with "/bin/sh -c" prepended.
		//
		// https://docs.docker.com/engine/reference/builder/#shell-form-entrypoint-example
		var shellFormEntrypoint = []string{"/bin/sh", "-c"}

		spec.Config.Entrypoint = append(shellFormEntrypoint, spec.Config.Entrypoint...)
	}
	if err = check.Validate(spec.Config); err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid command config: %s",
			err.Error(),
		)
	}

	// Launch a command actor.
	commandIDFut := a.m.system.AskAt(commandsAddr, *spec)
	if err = api.ProcessActorResponseError(&commandIDFut); err != nil {
		return nil, err
	}
	cmdID := commandIDFut.Get().(model.AllocationID)
	cmd := a.m.system.AskAt(commandsAddr.Child(cmdID), &commandv1.Command{})
	if err = api.ProcessActorResponseError(&cmd); err != nil {
		return nil, err
	}

	return &apiv1.LaunchCommandResponse{
		Command: cmd.Get().(*commandv1.Command),
		Config:  protoutils.ToStruct(spec.Config),
	}, nil
}
