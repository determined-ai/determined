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

	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/internal/templates"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	pkgCommand "github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

const (
	commandEntrypoint = "/run/determined/command-entrypoint.sh"
)

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

	managerName, poolName, launchWarnings, err := a.m.ResolveResources(
		resources.ResourceManager,
		resources.ResourcePool,
		resources.Slots,
		int(cmdSpec.Metadata.WorkspaceID),
		true,
	)
	if err != nil {
		return nil, launchWarnings, err
	}

	// Get the base TaskSpec.
	taskSpec, err := a.m.fillTaskSpec(managerName, poolName, agentUserGroup, userModel)
	if err != nil {
		return nil, nil, err
	}

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
	fillTaskConfig(resources.Slots, taskSpec, &config.Environment)
	config.Resources.ResourcePool = poolName
	config.Resources.Slots = resources.Slots

	var contextDirectory []byte
	config.WorkDir, contextDirectory, err = fillContextDir(config.WorkDir, workDirInDefaults, req.Files)
	if err != nil {
		return nil, nil, err
	}

	token, err := getTaskSessionToken(ctx, userModel)
	if err != nil {
		return nil, nil, err
	}
	taskSpec.UserSessionToken = token

	cmdSpec.Base = taskSpec
	cmdSpec.Config = config

	return &command.CreateGeneric{
		Spec:             &cmdSpec,
		ContextDirectory: contextDirectory,
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
	resp, err = command.DefaultCmdService.GetCommands(req)
	if err != nil {
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
) (*apiv1.GetCommandResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := command.DefaultCmdService.GetCommand(req)
	if err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.CommandId)
	if err := command.AuthZProvider.Get().CanGetNSC(
		ctx, *curUser, model.AccessScopeID(resp.Command.WorkspaceId)); err != nil {
		return nil, authz.SubIfUnauthorized(err, api.NotFoundErrs("command", req.CommandId, true))
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

	cmd, err := command.DefaultCmdService.KillNTSC(req.CommandId, model.TaskTypeCommand)
	if err != nil {
		return nil, err
	}

	return &apiv1.KillCommandResponse{Command: cmd.ToV1Command()}, nil
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

	cmd, err := command.DefaultCmdService.SetNTSCPriority(req.CommandId, int(req.Priority), model.TaskTypeCommand)
	if err != nil {
		return nil, err
	}

	return &apiv1.SetCommandPriorityResponse{Command: cmd.ToV1Command()}, nil
}

func (a *apiServer) getOIDCPachydermEnvVars(
	session *model.UserSession,
) (map[string]string, error) {
	if session == nil { // Can happen with allocation token.
		return map[string]string{}, nil
	}

	envVars := make(map[string]string)

	if val, ok := session.InheritedClaims["OIDCRawIDToken"]; ok {
		envVars["DEX_TOKEN"] = val
	}

	if a.m.config.Integrations.Pachyderm.Address != "" {
		envVars["PACHD_ADDRESS"] = a.m.config.Integrations.Pachyderm.Address
	}
	return envVars, nil
}

func (a *apiServer) LaunchCommand(
	ctx context.Context, req *apiv1.LaunchCommandRequest,
) (*apiv1.LaunchCommandResponse, error) {
	user, session, err := grpcutil.GetUser(ctx)
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

	if err = a.isNTSCPermittedToLaunch(ctx, launchReq.Spec, user); err != nil {
		return nil, err
	}

	// Postprocess the launchReq.Spec.
	if launchReq.Spec.Config.Description == "" {
		launchReq.Spec.Config.Description = fmt.Sprintf(
			"Command (%s)",
			petname.Generate(expconf.TaskNameGeneratorWords, expconf.TaskNameGeneratorSep),
		)
	}

	launchReq.Spec.Config.Entrypoint = append(
		[]string{commandEntrypoint}, launchReq.Spec.Config.Entrypoint...,
	)
	launchReq.Spec.AdditionalFiles = archive.Archive{
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(
			commandEntrypoint,
			etc.MustStaticFile(etc.CommandEntrypointResource),
			0o700,
			tar.TypeReg,
		),
	}

	if err = check.Validate(launchReq.Spec.Config); err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid command config: %s",
			err.Error(),
		)
	}

	launchReq.Spec.Base.ExtraEnvVars = map[string]string{
		"DET_TASK_TYPE": string(model.TaskTypeCommand),
	}

	OIDCPachydermEnvVars, err := a.getOIDCPachydermEnvVars(session)
	if err != nil {
		return nil, err
	}
	maps.Copy(launchReq.Spec.Base.ExtraEnvVars, OIDCPachydermEnvVars)

	// Launch a command.
	cmd, err := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeCommand,
		model.JobTypeCommand,
		launchReq)
	if err != nil {
		return nil, err
	}

	return &apiv1.LaunchCommandResponse{
		Command:  cmd.ToV1Command(),
		Config:   protoutils.ToStruct(launchReq.Spec.Config),
		Warnings: pkgCommand.LaunchWarningToProto(launchWarnings),
	}, nil
}
