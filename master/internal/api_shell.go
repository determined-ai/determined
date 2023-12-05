package internal

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"

	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	pkgCommand "github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	shellSSHDConfigFile   = "/run/determined/ssh/sshd_config"
	shellEntrypointScript = "/run/determined/ssh/shell-entrypoint.sh"
	// Agent ports 2600 - 3500 are split between TensorBoards, Notebooks, and Shells.
	minSshdPort = 3200
	maxSshdPort = minSshdPort + 299
)

func (a *apiServer) GetShells(
	ctx context.Context, req *apiv1.GetShellsRequest,
) (resp *apiv1.GetShellsResponse, err error) {
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
		_, err = a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, false)
		if errors.Is(err, db.ErrNotFound) {
			return nil, workspaceNotFoundErr
		} else if err != nil {
			return nil, err
		}
	}

	resp, err = command.DefaultCmdService.GetShells(req)
	if err != nil {
		return nil, err
	}

	limitedScopes, err := command.AuthZProvider.Get().AccessibleScopes(
		ctx, *curUser, model.AccessScopeID(req.WorkspaceId),
	)
	if err != nil {
		return nil, apiutils.MapAndFilterErrors(err, nil, nil)
	}

	if req.WorkspaceId != 0 && len(limitedScopes) == 0 {
		return nil, workspaceNotFoundErr
	}

	api.Where(&resp.Shells, func(i int) bool {
		return limitedScopes[model.AccessScopeID(resp.Shells[i].WorkspaceId)]
	})

	if err != nil {
		return nil, err
	}

	api.Sort(resp.Shells, req.OrderBy, req.SortBy, apiv1.GetShellsRequest_SORT_BY_ID)
	return resp, api.Paginate(&resp.Pagination, &resp.Shells, req.Offset, req.Limit)
}

func (a *apiServer) GetShell(
	ctx context.Context, req *apiv1.GetShellRequest,
) (*apiv1.GetShellResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := command.DefaultCmdService.GetShell(req)
	if err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.ShellId)
	if err := command.AuthZProvider.Get().CanGetNSC(
		ctx, *curUser, model.AccessScopeID(resp.Shell.WorkspaceId)); err != nil {
		return nil, authz.SubIfUnauthorized(err, api.NotFoundErrs("shell", req.ShellId, true))
	}
	return resp, nil
}

func (a *apiServer) KillShell(
	ctx context.Context, req *apiv1.KillShellRequest,
) (resp *apiv1.KillShellResponse, err error) {
	defer func() {
		if status.Code(err) == codes.Unknown {
			err = apiutils.MapAndFilterErrors(err, nil, nil)
		}
	}()

	getResponse, err := a.GetShell(ctx, &apiv1.GetShellRequest{ShellId: req.ShellId})
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.ShellId)
	err = command.AuthZProvider.Get().CanTerminateNSC(
		ctx, *curUser, model.AccessScopeID(getResponse.Shell.WorkspaceId))
	if err != nil {
		return nil, err
	}

	cmd, err := command.DefaultCmdService.KillNTSC(req.ShellId, model.TaskTypeShell)

	return &apiv1.KillShellResponse{Shell: cmd.ToV1Shell()}, nil
}

func (a *apiServer) SetShellPriority(
	ctx context.Context, req *apiv1.SetShellPriorityRequest,
) (resp *apiv1.SetShellPriorityResponse, err error) {
	defer func() {
		if status.Code(err) == codes.Unknown {
			err = apiutils.MapAndFilterErrors(err, nil, nil)
		}
	}()

	getResponse, err := a.GetShell(ctx, &apiv1.GetShellRequest{ShellId: req.ShellId})
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.ShellId)
	err = command.AuthZProvider.Get().CanSetNSCsPriority(
		ctx, *curUser, model.AccessScopeID(getResponse.Shell.WorkspaceId), int(req.Priority))
	if err != nil {
		return nil, err
	}

	cmd, err := command.DefaultCmdService.SetNTSCPriority(req.ShellId, int(req.Priority), model.TaskTypeShell)
	if err != nil {
		return nil, err
	}

	return &apiv1.SetShellPriorityResponse{Shell: cmd.ToV1Shell()}, nil
}

func (a *apiServer) LaunchShell(
	ctx context.Context, req *apiv1.LaunchShellRequest,
) (*apiv1.LaunchShellResponse, error) {
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
			"Shell (%s)",
			petname.Generate(expconf.TaskNameGeneratorWords, expconf.TaskNameGeneratorSep),
		)
	}

	// Selecting a random port mitigates the risk of multiple processes binding
	// the same port on an agent in host mode.
	port := getRandomPort(minSshdPort, maxSshdPort)
	// Shell authentication happens through SSH keys, instead.
	launchReq.Spec.Base.ExtraProxyPorts = append(launchReq.Spec.Base.ExtraProxyPorts, expconf.ProxyPort{
		RawProxyPort:        port,
		RawProxyTCP:         ptrs.Ptr(true),
		RawUnauthenticated:  ptrs.Ptr(true),
		RawDefaultServiceID: ptrs.Ptr(true),
	})

	launchReq.Spec.Config.Entrypoint = []string{
		shellEntrypointScript, "-f", shellSSHDConfigFile, "-p", strconv.Itoa(port), "-D", "-e",
	}

	if err = check.Validate(launchReq.Spec.Config); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	launchReq.Spec.AdditionalFiles = archive.Archive{
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(
			shellEntrypointScript,
			etc.MustStaticFile(etc.ShellEntrypointResource),
			0o700,
			tar.TypeReg,
		),
		launchReq.Spec.Base.AgentUserGroup.OwnedArchiveItem(
			taskReadyCheckLogs,
			etc.MustStaticFile(etc.TaskCheckReadyLogsResource),
			0o700,
			tar.TypeReg,
		),
	}

	launchReq.Spec.Base.ExtraEnvVars = map[string]string{"DET_TASK_TYPE": string(model.TaskTypeShell)}

	OIDCPachydermEnvVars, err := a.getOIDCPachydermEnvVars(session)
	if err != nil {
		return nil, err
	}
	maps.Copy(launchReq.Spec.Base.ExtraEnvVars, OIDCPachydermEnvVars)

	var passphrase *string
	if len(req.Data) > 0 {
		var data map[string]interface{}
		if err = json.Unmarshal(req.Data, &data); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse data %s: %s", req.Data, err)
		}
		if pwd, ok := data["passphrase"]; ok {
			if typed, typedOK := pwd.(string); typedOK {
				passphrase = &typed
			}
		}
	}

	keys, err := ssh.GenerateKey(launchReq.Spec.Base.SSHRsaSize, passphrase)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	launchReq.Spec.Metadata.PrivateKey = ptrs.Ptr(string(keys.PrivateKey))
	launchReq.Spec.Metadata.PublicKey = ptrs.Ptr(string(keys.PublicKey))
	launchReq.Spec.Keys = &keys

	// Launch a Shell.
	cmd, err := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeShell,
		model.JobTypeShell,
		launchReq)
	if err != nil {
		return nil, err
	}

	return &apiv1.LaunchShellResponse{
		Shell:    cmd.ToV1Shell(),
		Config:   protoutils.ToStruct(launchReq.Spec.Config),
		Warnings: pkgCommand.LaunchWarningToProto(launchWarnings),
	}, nil
}
