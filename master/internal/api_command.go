package internal

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	petname "github.com/dustinkirkland/golang-petname"
	pstruct "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"

	k8sV1 "k8s.io/api/core/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	mconfig "github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/internal/templates"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	pkgCommand "github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

const (
	commandEntrypoint = "/run/determined/command-entrypoint.sh"
)

var commandsAddr = actor.Addr("commands")

func getRandomPort(min, max int) int {
	//nolint:gosec // Weak RNG doesn't matter here.
	return rand.Intn(max-min) + min
}

type protoCommandParams struct {
	TemplateName string
	WorkspaceID  int32
	Config       *pstruct.Struct
	Files        []*utilv1.File
	MustZeroSlot bool
}

func (a *apiServer) getCommandLaunchParams(ctx context.Context, req *protoCommandParams,
	aUser *model.User) (
	*command.CreateGeneric, []pkgCommand.LaunchWarning, error,
) {
	var err error
	cmdSpec := tasks.GenericCommandSpec{}

	cmdSpec.Metadata.WorkspaceID = model.DefaultWorkspaceID
	if req.WorkspaceID != 0 {
		cmdSpec.Metadata.WorkspaceID = model.AccessScopeID(req.WorkspaceID)
	}

	// Validate the userModel and get the agent userModel group.
	userModel, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil,
			nil,
			status.Errorf(codes.Unauthenticated, "failed to get the user: %s", err)
	}

	// TODO(ilia): When commands are workspaced, also use workspace AgentUserGroup here.
	agentUserGroup, err := user.GetAgentUserGroup(ctx, userModel.ID, int(cmdSpec.Metadata.WorkspaceID))
	if err != nil {
		return nil, nil, err
	}

	var configBytes []byte
	if req.Config != nil {
		configBytes, err = protojson.Marshal(req.Config)
		if err != nil {
			return nil, nil, status.Errorf(
				codes.InvalidArgument, "failed to parse config %s: %s", configBytes, err)
		}
	}

	// Validate the resource configuration.
	resources := model.ParseJustResources(configBytes)
	if req.MustZeroSlot {
		resources.Slots = 0
	}
	poolName, err := a.m.rm.ResolveResourcePool(
		a.m.system, resources.ResourcePool, int(req.WorkspaceID), resources.Slots)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err = a.m.rm.ValidateResources(a.m.system, poolName, resources.Slots, true); err != nil {
		return nil, nil, fmt.Errorf("validating resources: %v", err)
	}

	launchWarnings, err := a.m.rm.ValidateResourcePoolAvailability(
		a.m.system,
		poolName,
		resources.Slots,
	)
	if err != nil {
		return nil, launchWarnings, fmt.Errorf("checking resource availability: %v", err.Error())
	}
	if a.m.config.ResourceManager.AgentRM != nil &&
		a.m.config.LaunchError &&
		len(launchWarnings) > 0 {
		return nil, nil, errors.New("slots requested exceeds cluster capacity")
	}
	// Get the base TaskSpec.
	taskContainerDefaults, err := a.m.rm.TaskContainerDefaults(
		a.m.system,
		poolName,
		a.m.config.TaskContainerDefaults,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("getting TaskContainerDefaults: %v", err)
	}
	taskSpec := *a.m.taskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults
	taskSpec.AgentUserGroup = agentUserGroup
	taskSpec.Owner = userModel

	// Get the full configuration.
	config := model.DefaultConfig(&taskSpec.TaskContainerDefaults)
	if req.TemplateName != "" {
		err := templates.UnmarshalTemplateConfig(ctx, req.TemplateName, aUser, &config, false)
		if err != nil {
			return nil, launchWarnings, err
		}
	}
	workDirInDefaults := config.WorkDir
	if len(configBytes) != 0 {
		dec := json.NewDecoder(bytes.NewBuffer(configBytes))
		dec.DisallowUnknownFields()

		if err := dec.Decode(&config); err != nil {
			return nil, launchWarnings, status.Errorf(codes.InvalidArgument,
				errors.Wrapf(err,
					"unable to decode the merged config: %s", string(configBytes)).Error())
		}
	}
	// Copy discovered (default) resource pool name and slot count.
	config.Resources.ResourcePool = poolName
	config.Resources.Slots = resources.Slots

	if req.MustZeroSlot {
		config.Resources.Slots = 0
	}

	taskContainerPodSpec := taskSpec.TaskContainerDefaults.GPUPodSpec
	if config.Resources.Slots == 0 {
		taskContainerPodSpec = taskSpec.TaskContainerDefaults.CPUPodSpec
	}
	config.Environment.PodSpec = (*k8sV1.Pod)(schemas.Merge(
		(*expconf.PodSpec)(config.Environment.PodSpec),
		(*expconf.PodSpec)(taskContainerPodSpec),
	))

	modelDef := make([]byte, 0) // Non nil for database persistance.
	fmt.Println("REQ FILES", len(req.Files))
	if len(req.Files) > 0 {
		userFiles := filesToArchive(req.Files)

		workdirSetInReq := config.WorkDir != nil &&
			(workDirInDefaults == nil || *workDirInDefaults != *config.WorkDir)
		if workdirSetInReq {
			return nil, launchWarnings, status.Errorf(codes.InvalidArgument,
				"cannot set work_dir and context directory at the same time")
		}
		config.WorkDir = nil

		// TODO(nick) do we need a path like ToRelocatedTarGz?
		modelDef, err = archive.ToTarGz(userFiles)
		if err != nil {
			return nil, launchWarnings, status.Errorf(codes.InvalidArgument,
				fmt.Errorf("compressing files context files: %w", err).Error())
		}
	}
	fmt.Println("LEN MODEL DEF", len(modelDef))

	extConfig := mconfig.GetMasterConfig().InternalConfig.ExternalSessions
	var token string
	if extConfig.JwtKey != "" {
		token, err = grpcutil.GetUserExternalToken(ctx)
		if err != nil {
			return nil, launchWarnings, status.Errorf(codes.Internal,
				errors.Wrapf(err,
					"unable to get external user token").Error())
		}
		err = nil
	} else {
		token, err = user.StartSession(ctx, userModel)
		if err != nil {
			return nil, launchWarnings, status.Errorf(codes.Internal,
				errors.Wrapf(err,
					"unable to create user session inside task").Error())
		}
	}
	taskSpec.UserSessionToken = token

	cmdSpec.Base = taskSpec
	cmdSpec.Config = config

	return &command.CreateGeneric{
		Spec:     &cmdSpec,
		ModelDef: modelDef,
	}, launchWarnings, nil
}

func (a *apiServer) GetCommands(
	ctx context.Context, req *apiv1.GetCommandsRequest,
) (resp *apiv1.GetCommandsResponse, err error) {
	defer func() {
		if status.Code(err) == codes.Unknown {
			err = apiutils.MapAndFilterErrors(err, nil, nil)
		}
	}()
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	workspaceNotFoundErr := api.NotFoundErrs("workspace", fmt.Sprint(req.WorkspaceId), true)

	if req.WorkspaceId != 0 {
		// check if the workspace exists.
		_, err := a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, false)
		if errors.Is(err, db.ErrNotFound) {
			return nil, workspaceNotFoundErr
		} else if err != nil {
			return nil, err
		}
	}
	if err = a.ask(commandsAddr, req, &resp); err != nil {
		return nil, err
	}

	limitedScopes, err := command.AuthZProvider.Get().AccessibleScopes(
		ctx, *curUser, model.AccessScopeID(req.WorkspaceId),
	)
	if err != nil {
		return nil, err
	}
	if req.WorkspaceId != 0 && len(limitedScopes) == 0 {
		return nil, workspaceNotFoundErr
	}

	api.Where(&resp.Commands, func(i int) bool {
		return limitedScopes[model.AccessScopeID(resp.Commands[i].WorkspaceId)]
	})

	api.Sort(resp.Commands, req.OrderBy, req.SortBy, apiv1.GetCommandsRequest_SORT_BY_ID)
	return resp, api.Paginate(&resp.Pagination, &resp.Commands, req.Offset, req.Limit)
}

func (a *apiServer) GetCommand(
	ctx context.Context, req *apiv1.GetCommandRequest,
) (resp *apiv1.GetCommandResponse, err error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	addr := commandsAddr.Child(req.CommandId)
	if err := a.ask(addr, req, &resp); err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.CommandId)
	if err := command.AuthZProvider.Get().CanGetNSC(
		ctx, *curUser, model.AccessScopeID(resp.Command.WorkspaceId)); err != nil {
		return nil, authz.SubIfUnauthorized(err, api.NotFoundErrs("actor", fmt.Sprint(addr), true))
	}
	return resp, nil
}

func (a *apiServer) KillCommand(
	ctx context.Context, req *apiv1.KillCommandRequest,
) (resp *apiv1.KillCommandResponse, err error) {
	defer func() {
		if status.Code(err) == codes.Unknown {
			err = apiutils.MapAndFilterErrors(err, nil, nil)
		}
	}()

	targetCmd, err := a.GetCommand(ctx, &apiv1.GetCommandRequest{CommandId: req.CommandId})
	if err != nil {
		return nil, err
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.CommandId)
	if err = command.AuthZProvider.Get().CanTerminateNSC(
		ctx, *curUser, model.AccessScopeID(targetCmd.Command.WorkspaceId),
	); err != nil {
		return nil, err
	}

	return resp, a.ask(commandsAddr.Child(req.CommandId), req, &resp)
}

func (a *apiServer) SetCommandPriority(
	ctx context.Context, req *apiv1.SetCommandPriorityRequest,
) (resp *apiv1.SetCommandPriorityResponse, err error) {
	defer func() {
		if status.Code(err) == codes.Unknown {
			err = apiutils.MapAndFilterErrors(err, nil, nil)
		}
	}()
	targetCmd, err := a.GetCommand(ctx, &apiv1.GetCommandRequest{CommandId: req.CommandId})
	if err != nil {
		return nil, err
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.CommandId)
	if err = command.AuthZProvider.Get().CanSetNSCsPriority(
		ctx, *curUser, model.AccessScopeID(targetCmd.Command.WorkspaceId), int(req.Priority),
	); err != nil {
		return nil, err
	}

	return resp, a.ask(commandsAddr.Child(req.CommandId), req, &resp)
}

func (a *apiServer) LaunchCommand(
	ctx context.Context, req *apiv1.LaunchCommandRequest,
) (*apiv1.LaunchCommandResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	launchReq, launchWarnings, err := a.getCommandLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		WorkspaceID:  req.WorkspaceId,
		Config:       req.Config,
		Files:        req.Files,
	}, user)
	if err != nil {
		return nil, api.WrapWithFallbackCode(err, codes.InvalidArgument,
			"failed to prepare launch params")
	}
	spec := launchReq.Spec

	if err = a.isNTSCPermittedToLaunch(ctx, spec, user); err != nil {
		return nil, err
	}

	// Postprocess the spec.
	if spec.Config.Description == "" {
		spec.Config.Description = fmt.Sprintf(
			"Command (%s)",
			petname.Generate(expconf.TaskNameGeneratorWords, expconf.TaskNameGeneratorSep),
		)
	}

	spec.Config.Entrypoint = append([]string{commandEntrypoint}, spec.Config.Entrypoint...)
	spec.AdditionalFiles = archive.Archive{
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			commandEntrypoint,
			etc.MustStaticFile(etc.CommandEntrypointResource),
			0o700,
			tar.TypeReg,
		),
	}

	if err = check.Validate(spec.Config); err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid command config: %s",
			err.Error(),
		)
	}
	spec.Base.ExtraEnvVars = map[string]string{"DET_TASK_TYPE": string(model.TaskTypeCommand)}

	// Launch a command actor.
	var cmdID model.TaskID
	if err = a.ask(commandsAddr, launchReq, &cmdID); err != nil {
		return nil, err
	}

	var cmd *commandv1.Command
	if err = a.ask(commandsAddr.Child(cmdID), &commandv1.Command{}, &cmd); err != nil {
		return nil, err
	}

	return &apiv1.LaunchCommandResponse{
		Command:  cmd,
		Config:   protoutils.ToStruct(spec.Config),
		Warnings: pkgCommand.LaunchWarningToProto(launchWarnings),
	}, nil
}
