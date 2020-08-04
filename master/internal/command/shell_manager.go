package command

import (
	"archive/tar"
	"fmt"
	"net/http"
	"strconv"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/shellv1"
)

const (
	shellSSHDir             = "/run/determined/ssh"
	shellAuthorizedKeysFile = "/run/determined/ssh/authorized_keys"
	shellSSHDConfigFile     = "/run/determined/ssh/sshd_config"
	shellHostPrivKeyFile    = "/run/determined/ssh/id_rsa"
	shellHostPubKeyFile     = "/run/determined/ssh/id_rsa.pub"
	sshdPort                = 2222
)

var (
	shellEntrypoint = []string{
		"/usr/sbin/sshd", "-f", shellSSHDConfigFile, "-p", strconv.Itoa(sshdPort), "-D",
	}
	shellPorts = map[string]int{"shell": sshdPort}
)

type shellManager struct {
	db *db.PgDB

	defaultAgentUserGroup model.AgentUserGroup
	clusterID             string
}

func (n *shellManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *apiv1.GetShellsRequest:
		resp := &apiv1.GetShellsResponse{}
		for _, shell := range ctx.AskAll(&shellv1.Shell{}, ctx.Children()...).GetAll() {
			resp.Shells = append(resp.Shells, shell.(*shellv1.Shell))
		}
		ctx.Respond(resp)

	case echo.Context:
		n.handleAPIRequest(ctx, msg)
	}
	return nil
}

func (n *shellManager) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
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

		req, err := parseCommandRequest(apiCtx, n.db, &params)
		if err != nil {
			respondBadRequest(ctx, err)
			return
		}

		if req.AgentUserGroup == nil {
			req.AgentUserGroup = &n.defaultAgentUserGroup
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

		shell := n.newShell(req, generatedKeys)
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

func (n *shellManager) newShell(
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
	config.Environment.Ports = shellPorts
	config.Entrypoint = shellEntrypoint

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
	}

	return &command{
		taskID:          scheduler.NewTaskID(),
		config:          config,
		userFiles:       req.UserFiles,
		additionalFiles: additionalFiles,
		metadata: map[string]interface{}{
			"privateKey": string(keyPair.PrivateKey),
			"publicKey":  string(keyPair.PublicKey),
		},

		owner:          req.Owner,
		agentUserGroup: req.AgentUserGroup,
	}
}
