package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/labstack/echo/v4"

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

func getPort(min, max int) int {
	return rand.Intn(max-min) + min
}

var commandsAddr = actor.Addr("commands")

type protoCommandParams struct {
	TemplateName string
	Config       *pstruct.Struct
	Files        []*utilv1.File
	Data         []byte
	MustZeroSlot bool
}

func (a *apiServer) getCommandLaunchParams(ctx context.Context, req *protoCommandParams) (
	*tasks.GenericCommandSpec, error,
) {
	var err error

	// Must get the user and the agent user group
	user, _, err := grpcutil.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get the user: %s", err)
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

	// Get the full configuration.
	var configBytes []byte
	if req.Config != nil {
		configBytes, err = protojson.Marshal(req.Config)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal, "failed to parse config %s: %s", configBytes, err)
		}
	}

	resources := model.ParseJustResources(configBytes)
	taskSpec := a.m.makeTaskSpec(resources.ResourcePool, resources.Slots)
	taskSpec.AgentUserGroup = agentUserGroup
	taskSpec.Owner = user

	config := command.DefaultConfig(&taskSpec.TaskContainerDefaults)
	if req.TemplateName != "" {
		template, err := a.m.db.TemplateByName(req.TemplateName)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to make command spec: %s",
				errors.Wrapf(err, "failed to find template: %s", req.TemplateName))
		}
		if err := yaml.Unmarshal(template.Config, &config); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to make command spec: %s",
				errors.Wrapf(err, "failed to unmarshal template: %s", req.TemplateName))
		}
	}

	if len(configBytes) != 0 {
		dec := json.NewDecoder(bytes.NewBuffer(configBytes))
		dec.DisallowUnknownFields()

		if err := dec.Decode(&config); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to make command spec: %s",
				errors.Wrapf(
					err,
					"unable to parse the config in the parameters: %s",
					string(configBytes),
				))
		}
	}

	// mustBeZeroSlot indicates that this type of command may never use more than
	// zero slots (as of Jan 2021, this is only Tensorboards). This is important
	// when building up the config, so that we can route the command to the
	// correct default resource pool. If the user didn't explicitly set the 'slots' field,
	// the DefaultConfig will set slots=1. We need to correct that before attempting to fill
	// in the default resource pool as otherwise we will mistakenly route this CPU task to the
	// default GPU pool.
	if req.MustZeroSlot {
		config.Resources.Slots = 0
	}

	if err := sproto.ValidateRP(a.m.system, config.Resources.ResourcePool); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to make command spec: %s",
			errors.Wrapf(err, "resource pool does not exist: %s", config.Resources.ResourcePool))
	}

	// If the resource pool isn't set, fill in the default at creation time.
	if config.Resources.ResourcePool == "" {
		if config.Resources.Slots == 0 {
			config.Resources.ResourcePool = sproto.GetDefaultAuxResourcePool(a.m.system)
		} else {
			config.Resources.ResourcePool = sproto.GetDefaultComputeResourcePool(a.m.system)
		}
	}

	if config.Resources.Slots > 0 {
		fillable, err := sproto.ValidateRPResources(
			a.m.system, config.Resources.ResourcePool, config.Resources.Slots)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to make command spec: %s", errors.Wrapf(
				err, "failed to check resource pool resources: "))
		}
		if !fillable {
			return nil, api.AsErrBadRequest(
				"resource request unfulfillable, please try requesting less slots")
		}
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
	spec, err := a.getCommandLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		Config:       req.Config,
		Files:        req.Files,
		Data:         req.Data,
	})
	if err != nil {
		return nil, api.APIErr2GRPC(err)
	}

	// Postprocess the config.
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
		return nil, echo.NewHTTPError(
			http.StatusBadRequest,
			errors.Wrap(err, "failed to launch command").Error(),
		)
	}

	commandIDFut := a.m.system.AskAt(commandsAddr, *spec)
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
		Config:  protoutils.ToStruct(spec.Config),
	}, nil
}
