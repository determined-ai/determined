package command

import (
	"archive/tar"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
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
	makeTaskSpec          tasks.MakeTaskSpecFn
}

// ShellLaunchRequest describes a request to launch a new shell.
type ShellLaunchRequest struct {
	CommandParams *CommandParams
}

func (s *shellManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *apiv1.GetShellsRequest:
		resp := &apiv1.GetShellsResponse{}
		users := make(map[string]bool)
		for _, user := range msg.Users {
			users[user] = true
		}
		for _, shell := range ctx.AskAll(&shellv1.Shell{}, ctx.Children()...).GetAll() {
			if typed := shell.(*shellv1.Shell); len(users) == 0 || users[typed.Username] {
				resp.Shells = append(resp.Shells, typed)
			}
		}
		ctx.Respond(resp)

	case ShellLaunchRequest:
		summary, statusCode, err := s.processLaunchRequest(ctx, msg)
		if err != nil || statusCode > 200 {
			ctx.Respond(echo.NewHTTPError(statusCode, errors.Wrap(err, "failed to launch shell").Error()))
			return nil
		}
		ctx.Respond(summary.ID)
	}
	return nil
}

func (s *shellManager) processLaunchRequest(
	ctx *actor.Context,
	req ShellLaunchRequest,
) (*summary, int, error) {
	params := req.CommandParams

	var passphrase *string
	if pwd, ok := params.Data["passphrase"]; ok {
		if typed, typedOK := pwd.(string); typedOK {
			passphrase = &typed
		}
	}
	keys, err := ssh.GenerateKey(passphrase)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	ctx.Log().Info("creating shell")

	shell := s.newShell(params, keys)
	if err = check.Validate(shell.config); err != nil {
		return nil, http.StatusBadRequest, err
	}

	a, _ := ctx.ActorOf(shell.taskID, shell)
	summaryFut := ctx.Ask(a, getSummary{})
	if err = summaryFut.Error(); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	summary := summaryFut.Get().(summary)
	ctx.Log().Infof("created shell %s", a.Address().Local())
	return &summary, http.StatusOK, err
}

func (s *shellManager) newShell(
	params *CommandParams,
	keyPair ssh.PrivateAndPublicKeys,
) *command {
	config := params.FullConfig

	// Postprocess the config.
	if config.Description == "" {
		config.Description = fmt.Sprintf(
			"Shell (%s)",
			petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep),
		)
	}

	taskID := sproto.NewTaskID()
	serviceAddress := fmt.Sprintf("/proxy/%s/", taskID)

	// Select a random port from the range to assign to sshd. In host
	// mode, this mitigates the risk of multiple sshd processes binding
	// the same port on an agent.
	port := getPort(minSshdPort, maxSshdPort)

	config.Environment.Ports = map[string]int{"shell": port}
	config.Entrypoint = []string{
		shellEntrypointScript, "-f", shellSSHDConfigFile, "-p", strconv.Itoa(port), "-D", "-e",
	}

	setPodSpec(config, params.TaskSpec.TaskContainerDefaults)

	additionalFiles := archive.Archive{
		params.AgentUserGroup.OwnedArchiveItem(shellSSHDir, nil, 0700, tar.TypeDir),
		params.AgentUserGroup.OwnedArchiveItem(
			shellAuthorizedKeysFile, keyPair.PublicKey, 0644, tar.TypeReg,
		),
		params.AgentUserGroup.OwnedArchiveItem(
			shellHostPrivKeyFile, keyPair.PrivateKey, 0600, tar.TypeReg,
		),
		params.AgentUserGroup.OwnedArchiveItem(
			shellHostPubKeyFile, keyPair.PublicKey, 0600, tar.TypeReg,
		),
		params.AgentUserGroup.OwnedArchiveItem(
			shellSSHDConfigFile,
			etc.MustStaticFile(etc.SSHDConfigResource),
			0644,
			tar.TypeReg,
		),
		params.AgentUserGroup.OwnedArchiveItem(
			shellEntrypointScript,
			etc.MustStaticFile(etc.ShellEntrypointResource),
			0700,
			tar.TypeReg,
		),
	}

	return &command{
		taskID:          taskID,
		config:          *config,
		userFiles:       params.UserFiles,
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
		owner: commandOwner{
			ID:       params.User.ID,
			Username: params.User.Username,
		},
		agentUserGroup: params.AgentUserGroup,
		taskSpec:       params.TaskSpec,

		proxyTCP: true,

		db: s.db,
	}
}
