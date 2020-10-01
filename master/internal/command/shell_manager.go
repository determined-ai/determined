package command

import (
	"archive/tar"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/shellv1"
)

const (
	shellSSHDir             = "/run/determined/ssh"
	shellAuthorizedKeysFile = "/run/determined/ssh/authorized_keys_unmodified"
	shellSSHDConfigFile     = "/run/determined/ssh/sshd_config"
	shellHostPrivKeyFile    = "/run/determined/ssh/id_rsa"
	shellHostPubKeyFile     = "/run/determined/ssh/id_rsa.pub"
	shellEntrypointScript   = "/run/determined/ssh/shell-entrypoint.sh"
	// Agent ports 2600 - 3500 are split between TensorBoards, Notebooks, and Shells.
	minSshdPort = 3200
	maxSshdPort = minSshdPort + 299
)

type shellManager struct {
	db *db.PgDB

	defaultAgentUserGroup model.AgentUserGroup
	taskSpec              *tasks.TaskSpec
}

func (s *shellManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *apiv1.GetShellsRequest:
		resp := &apiv1.GetShellsResponse{}
		for _, shell := range ctx.AskAll(&shellv1.Shell{}, ctx.Children()...).GetAll() {
			resp.Shells = append(resp.Shells, shell.(*shellv1.Shell))
		}
		ctx.Respond(resp)

	case echo.Context:
		s.handleAPIRequest(ctx, msg)
	}
	return nil
}

func (s *shellManager) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		userFilter := apiCtx.QueryParam("user")
		ctx.Respond(apiCtx.JSON(
			http.StatusOK,
			ctx.AskAll(getSummary{userFilter: userFilter}, ctx.Children()...)))

	case echo.POST:
		var params commandParams
		if err := apiCtx.Bind(&params); err != nil {
			respondBadRequest(ctx, err)
			return
		}

		req, err := parseCommandRequest(apiCtx, s.db, &params, &s.taskSpec.TaskContainerDefaults)
		if err != nil {
			respondBadRequest(ctx, err)
			return
		}

		if req.AgentUserGroup == nil {
			req.AgentUserGroup = &s.defaultAgentUserGroup
		}

		var passphrase *string
		if pwd, ok := req.Data["passphrase"]; ok {
			if typed, typedOK := pwd.(string); typedOK {
				passphrase = &typed
			}
		}
		generatedKeys, err := ssh.GenerateKey(passphrase)
		if err != nil {
			ctx.Respond(err)
			return
		}

		ctx.Log().Info("creating shell")

		shell := s.newShell(req, generatedKeys)
		if err := check.Validate(shell.config); err != nil {
			respondBadRequest(ctx, err)
			return
		}

		a, _ := ctx.ActorOf(shell.taskID, shell)
		ctx.Respond(apiCtx.JSON(http.StatusOK, ctx.Ask(a, getSummary{})))
		ctx.Log().Infof("created shell %s", a.Address().Local())

	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (s *shellManager) newShell(
	req *commandRequest,
	keyPair ssh.PrivateAndPublicKeys,
) *command {
	config := req.Config

	// Postprocess the config.
	if config.Description == "" {
		config.Description = fmt.Sprintf(
			"Shell (%s)",
			petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep),
		)
	}

	taskID := resourcemanagers.NewTaskID()
	serviceAddress := fmt.Sprintf("/proxy/%s/", taskID)

	// Select a random port from the range to assign to sshd. In host
	// mode, this mitigates the risk of multiple sshd processes binding
	// the same port on an agent.
	port := getPort(minSshdPort, maxSshdPort)

	config.Environment.Ports = map[string]int{"shell": port}
	config.Entrypoint = []string{
		shellEntrypointScript, "-f", shellSSHDConfigFile, "-p", strconv.Itoa(port), "-D", "-e",
	}

	setPodSpec(&config, s.taskSpec.TaskContainerDefaults)

	additionalFiles := archive.Archive{
		req.AgentUserGroup.OwnedArchiveItem(shellSSHDir, nil, 0700, tar.TypeDir),
		req.AgentUserGroup.OwnedArchiveItem(
			shellAuthorizedKeysFile, keyPair.PublicKey, 0644, tar.TypeReg,
		),
		req.AgentUserGroup.OwnedArchiveItem(
			shellHostPrivKeyFile, keyPair.PrivateKey, 0600, tar.TypeReg,
		),
		req.AgentUserGroup.OwnedArchiveItem(
			shellHostPubKeyFile, keyPair.PublicKey, 0600, tar.TypeReg,
		),
		req.AgentUserGroup.OwnedArchiveItem(
			shellSSHDConfigFile,
			etc.MustStaticFile(etc.SSHDConfigResource),
			0644,
			tar.TypeReg,
		),
		req.AgentUserGroup.OwnedArchiveItem(
			shellEntrypointScript,
			etc.MustStaticFile(etc.ShellEntrypointResource),
			0700,
			tar.TypeReg,
		),
	}

	return &command{
		taskID:          taskID,
		config:          config,
		userFiles:       req.UserFiles,
		additionalFiles: additionalFiles,
		metadata: map[string]interface{}{
			"privateKey": string(keyPair.PrivateKey),
			"publicKey":  string(keyPair.PublicKey),
		},
		readinessChecks: map[string]readinessCheck{
			"shell": func(log sproto.ContainerLog) bool {
				return strings.Contains(log.String(), "Server listening on")
			},
		},

		serviceAddress: &serviceAddress,
		owner:          req.Owner,
		agentUserGroup: req.AgentUserGroup,
		taskSpec:       s.taskSpec,
	}
}
