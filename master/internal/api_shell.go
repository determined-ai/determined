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
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
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
	_ context.Context, req *apiv1.GetShellsRequest,
) (resp *apiv1.GetShellsResponse, err error) {
	err = a.actorRequest(shellsAddr, req, &resp)
	if err != nil {
		return nil, err
	}
	a.sort(resp.Shells, req.OrderBy, req.SortBy, apiv1.GetShellsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Shells, req.Offset, req.Limit)
}

func (a *apiServer) GetShell(
	_ context.Context, req *apiv1.GetShellRequest) (resp *apiv1.GetShellResponse, err error) {
	return resp, a.actorRequest(shellsAddr.Child(req.ShellId), req, &resp)
}

func (a *apiServer) KillShell(
	_ context.Context, req *apiv1.KillShellRequest) (resp *apiv1.KillShellResponse, err error) {
	return resp, a.actorRequest(shellsAddr.Child(req.ShellId), req, &resp)
}

func (a *apiServer) SetShellPriority(
	_ context.Context, req *apiv1.SetShellPriorityRequest,
) (resp *apiv1.SetShellPriorityResponse, err error) {
	return resp, a.actorRequest(shellsAddr.Child(req.ShellId), req, &resp)
}

func (a *apiServer) LaunchShell(
	ctx context.Context, req *apiv1.LaunchShellRequest,
) (*apiv1.LaunchShellResponse, error) {
	spec, err := a.getCommandLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		Config:       req.Config,
		Files:        req.Files,
	})
	if err != nil {
		return nil, api.APIErr2GRPC(errors.Wrapf(err, "failed to prepare launch params"))
	}

	// Postprocess the spec.
	if spec.Config.Description == "" {
		spec.Config.Description = fmt.Sprintf(
			"Shell (%s)",
			petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep),
		)
	}

	// Selecting a random port mitigates the risk of multiple processes binding
	// the same port on an agent in host mode.
	port := getRandomPort(minSshdPort, maxSshdPort)
	spec.Port = &port
	spec.Config.Environment.Ports = map[string]int{"shell": port}

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
			0700,
			tar.TypeReg,
		),
	}

	var keys ssh.PrivateAndPublicKeys
	if len(req.Data) > 0 {
		var data map[string]interface{}
		if err = json.Unmarshal(req.Data, &data); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to parse data %s: %s", req.Data, err)
		}
		var passphrase *string
		if pwd, ok := data["passphrase"]; ok {
			if typed, typedOK := pwd.(string); typedOK {
				passphrase = &typed
			}
		}
		keys, err = ssh.GenerateKey(passphrase)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	spec.Metadata = map[string]interface{}{
		"privateKey": string(keys.PrivateKey),
		"publicKey":  string(keys.PublicKey),
	}
	spec.Keys = &keys

	spec.ProxyTCP = true

	// Launch a Shell actor.
	shellIDFut := a.m.system.AskAt(shellsAddr, *spec)
	if err = api.ProcessActorResponseError(&shellIDFut); err != nil {
		return nil, err
	}

	shellID := shellIDFut.Get().(model.AllocationID)
	shell := a.m.system.AskAt(shellsAddr.Child(shellID), &shellv1.Shell{})
	if err = api.ProcessActorResponseError(&shell); err != nil {
		return nil, err
	}

	return &apiv1.LaunchShellResponse{
		Shell:  shell.Get().(*shellv1.Shell),
		Config: protoutils.ToStruct(spec.Config),
	}, nil
}
