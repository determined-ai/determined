package internal

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/actor"
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
	"github.com/determined-ai/determined/proto/pkg/shellv1"
)

const (
	shellSSHDConfigFile   = "/run/determined/ssh/sshd_config"
	shellEntrypointScript = "/run/determined/ssh/shell-entrypoint.sh"
	// Agent ports 2600 - 3500 are split between TensorBoards, Notebooks, and Shells.
	minSshdPort = 3200
	maxSshdPort = minSshdPort + 299
)

var shellsAddr = actor.Addr("shells")

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

	if err = a.ask(shellsAddr, req, &resp); err != nil {
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

	a.filter(&resp.Shells, func(i int) bool {
		return limitedScopes[model.AccessScopeID(resp.Shells[i].WorkspaceId)]
	})

	if err != nil {
		return nil, err
	}

	a.sort(resp.Shells, req.OrderBy, req.SortBy, apiv1.GetShellsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Shells, req.Offset, req.Limit)
}

func (a *apiServer) GetShell(
	ctx context.Context, req *apiv1.GetShellRequest,
) (resp *apiv1.GetShellResponse, err error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	addr := shellsAddr.Child(req.ShellId)
	if err = a.ask(addr, req, &resp); err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.ShellId)
	if err := command.AuthZProvider.Get().CanGetNSC(
		ctx, *curUser, model.AccessScopeID(resp.Shell.WorkspaceId)); err != nil {
		return nil, authz.SubIfUnauthorized(err, api.NotFoundErrs("actor", fmt.Sprint(addr), true))
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

	return resp, a.ask(shellsAddr.Child(req.ShellId), req, &resp)
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

	return resp, a.ask(shellsAddr.Child(req.ShellId), req, &resp)
}

func (a *apiServer) LaunchShell(
	ctx context.Context, req *apiv1.LaunchShellRequest,
) (*apiv1.LaunchShellResponse, error) {
	spec, launchWarnings, err := a.getCommandLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		Config:       req.Config,
		Files:        req.Files,
		WorkspaceID:  int(req.WorkspaceId),
	})
	if err != nil {
		return nil, api.APIErrToGRPC(errors.Wrapf(err, "failed to prepare launch params"))
	}

	spec.Metadata.WorkspaceID = model.DefaultWorkspaceID
	if req.WorkspaceId != 0 {
		spec.Metadata.WorkspaceID = model.AccessScopeID(req.WorkspaceId)
	}
	if err = a.isNTSCPermittedToLaunch(ctx, spec); err != nil {
		return nil, err
	}

	// Postprocess the spec.
	if spec.Config.Description == "" {
		spec.Config.Description = fmt.Sprintf(
			"Shell (%s)",
			petname.Generate(expconf.TaskNameGeneratorWords, expconf.TaskNameGeneratorSep),
		)
	}

	// Selecting a random port mitigates the risk of multiple processes binding
	// the same port on an agent in host mode.
	port := getRandomPort(minSshdPort, maxSshdPort)
	// Shell authentication happens through SSH keys, instead.
	spec.Base.ExtraProxyPorts = append(spec.Base.ExtraProxyPorts, expconf.ProxyPort{
		RawProxyPort:        port,
		RawProxyTCP:         ptrs.Ptr(true),
		RawUnauthenticated:  ptrs.Ptr(true),
		RawDefaultServiceID: ptrs.Ptr(true),
	})

	spec.Config.Entrypoint = []string{
		shellEntrypointScript, "-f", shellSSHDConfigFile, "-p", strconv.Itoa(port), "-D", "-e",
	}

	if err = check.Validate(spec.Config); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	spec.AdditionalFiles = archive.Archive{
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			shellEntrypointScript,
			etc.MustStaticFile(etc.ShellEntrypointResource),
			0o700,
			tar.TypeReg,
		),
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			taskReadyCheckLogs,
			etc.MustStaticFile(etc.TaskCheckReadyLogsResource),
			0o700,
			tar.TypeReg,
		),
	}

	spec.Base.ExtraEnvVars = map[string]string{"DET_TASK_TYPE": string(model.TaskTypeShell)}

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

	keys, err := ssh.GenerateKey(spec.Base.SSHRsaSize, passphrase)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	spec.Metadata.PrivateKey = ptrs.Ptr(string(keys.PrivateKey))
	spec.Metadata.PublicKey = ptrs.Ptr(string(keys.PublicKey))
	spec.Keys = &keys

	// Launch a Shell actor.
	var shellID model.TaskID
	if err := a.ask(shellsAddr, *spec, &shellID); err != nil {
		return nil, err
	}

	var shell *shellv1.Shell
	if err := a.ask(shellsAddr.Child(shellID), &shellv1.Shell{}, &shell); err != nil {
		return nil, err
	}

	return &apiv1.LaunchShellResponse{
		Shell:    shell,
		Config:   protoutils.ToStruct(spec.Config),
		Warnings: pkgCommand.LaunchWarningToProto(launchWarnings),
	}, nil
}
