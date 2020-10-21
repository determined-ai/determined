package command

import (
	"archive/tar"
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	requestContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
)

const (
	jupyterDir        = "/run/determined/jupyter/"
	jupyterConfigDir  = "/run/determined/jupyter/config"
	jupyterDataDir    = "/run/determined/jupyter/data"
	jupyterRuntimeDir = "/run/determined/jupyter/runtime"
	jupyterEntrypoint = "/run/determined/jupyter/notebook-entrypoint.sh"
	// Agent ports 2600 - 3500 are split between TensorBoards, Notebooks, and Shells.
	minNotebookPort     = 2900
	maxNotebookPort     = minNotebookPort + 299
	notebookConfigFile  = "/run/determined/workdir/jupyter-conf.py"
	notebookDefaultPage = "/run/determined/workdir/Notebook.ipynb"
)

var (
	notebookEntrypoint  = []string{jupyterEntrypoint}
	jupyterReadyPattern = regexp.MustCompile("Jupyter Notebook .*is running at")
)

func generateNotebookDescription() (string, error) {
	tmpl := "Notebook ({{.PetName}})"

	t, err := template.New("").Parse(strings.TrimSpace(tmpl))
	if err != nil {
		return "", errors.Wrap(err, "parsing template")
	}

	petName := petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep)

	var buf strings.Builder
	err = t.Execute(&buf, map[string]string{"PetName": petName})
	if err != nil {
		return "", errors.Wrap(err, "executing template")
	}
	return buf.String(), nil
}

func generateServiceAddress(taskID string) (string, error) {
	tmpl := "/proxy/{{.TaskID}}/lab/tree/Notebook.ipynb?reset"

	t, err := template.New("").Parse(strings.TrimSpace(tmpl))
	if err != nil {
		return "", errors.Wrap(err, "parsing template")
	}

	var buf strings.Builder
	err = t.Execute(&buf, map[string]string{"TaskID": taskID})
	if err != nil {
		return "", errors.Wrap(err, "executing template")
	}
	return buf.String(), nil
}

func generateNotebookConfig(taskID string) ([]byte, error) {
	tmpl := `
c.NotebookApp.base_url       = "/proxy/{{.TaskID}}/"
c.NotebookApp.allow_origin   = "*"
c.NotebookApp.trust_xheaders = True
c.NotebookApp.open_browser   = False
c.NotebookApp.allow_root     = True
c.NotebookApp.ip             = "0.0.0.0"
c.NotebookApp.token          = ""
	`

	t, err := template.New("").Parse(strings.TrimSpace(tmpl))
	if err != nil {
		return nil, errors.Wrap(err, "parsing template")
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]string{"TaskID": taskID})
	if err != nil {
		return nil, errors.Wrap(err, "executing template")
	}
	return buf.Bytes(), nil
}

type notebookManager struct {
	db *db.PgDB

	defaultAgentUserGroup model.AgentUserGroup
	taskSpec              *tasks.TaskSpec
}

// NotebookLaunchRequest describes a request to launch a new notebook.
type NotebookLaunchRequest struct {
	CommandParams *CommandParams
	User          *model.User
}

func (n *notebookManager) processLaunchRequest(
	ctx *actor.Context,
	req NotebookLaunchRequest,
) (*summary, int, error) {
	commandReq, err := parseCommandRequest(
		*req.User, n.db, req.CommandParams, &n.taskSpec.TaskContainerDefaults,
	)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	if commandReq.AgentUserGroup == nil {
		commandReq.AgentUserGroup = &n.defaultAgentUserGroup
	}

	ctx.Log().Info("creating notebook")

	notebook, err := n.newNotebook(commandReq)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if err = check.Validate(notebook.config); err != nil {
		return nil, http.StatusBadRequest, err
	}

	a, _ := ctx.ActorOf(notebook.taskID, notebook)
	summaryFut := ctx.Ask(a, getSummary{})
	if err := summaryFut.Error(); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	summary := summaryFut.Get().(summary)
	ctx.Log().Infof("created notebook %s", a.Address().Local())
	return &summary, http.StatusOK, nil
}

func (n *notebookManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *apiv1.GetNotebooksRequest:
		resp := &apiv1.GetNotebooksResponse{}
		for _, notebook := range ctx.AskAll(&notebookv1.Notebook{}, ctx.Children()...).GetAll() {
			resp.Notebooks = append(resp.Notebooks, notebook.(*notebookv1.Notebook))
		}
		ctx.Respond(resp)

	case NotebookLaunchRequest:
		summary, statusCode, err := n.processLaunchRequest(ctx, msg)
		if err != nil || statusCode > 200 {
			ctx.Respond(echo.NewHTTPError(statusCode, errors.Wrap(err, "failed to launch shell").Error()))
			return nil
		}
		ctx.Respond(summary.ID)

	case echo.Context:
		n.handleAPIRequest(ctx, msg)
	}
	return nil
}

func (n *notebookManager) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		userFilter := apiCtx.QueryParam("user")
		ctx.Respond(apiCtx.JSON(
			http.StatusOK,
			ctx.AskAll(getSummary{userFilter: userFilter}, ctx.Children()...)))

	case echo.POST:
		var params CommandParams
		if err := apiCtx.Bind(&params); err != nil {
			respondBadRequest(ctx, err)
			return
		}
		user := apiCtx.(*requestContext.DetContext).MustGetUser()
		req := NotebookLaunchRequest{
			User:          &user,
			CommandParams: &params,
		}
		summary, statusCode, err := n.processLaunchRequest(ctx, req)
		if err != nil || statusCode > 200 {
			ctx.Respond(echo.NewHTTPError(statusCode, err.Error()))
			return
		}
		ctx.Respond(apiCtx.JSON(http.StatusOK, summary))

	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (n *notebookManager) newNotebook(req *commandRequest) (*command, error) {
	config := req.Config
	taskID := resourcemanagers.NewTaskID()

	// Postprocess the config. Add Jupyter and configuration to the container.

	// Select a random port from the range to assign to the notebook. In host
	// mode, this mitigates the risk of multiple notebook processes binding
	// the same port on an agent.
	port := getPort(minNotebookPort, maxNotebookPort)
	notebookPorts := map[string]int{"notebook": port}
	portVar := fmt.Sprintf("NOTEBOOK_PORT=%d", port)

	config.Environment.Ports = notebookPorts
	config.Environment.EnvironmentVariables.CPU = append(
		config.Environment.EnvironmentVariables.CPU, portVar)
	config.Environment.EnvironmentVariables.GPU = append(
		config.Environment.EnvironmentVariables.GPU, portVar)

	config.Entrypoint = notebookEntrypoint

	setPodSpec(&config, n.taskSpec.TaskContainerDefaults)

	if config.Description == "" {
		var err error
		config.Description, err = generateNotebookDescription()
		if err != nil {
			return nil, errors.Wrap(err, "generating notebook name")
		}
	}

	serviceAddress, err := generateServiceAddress(string(taskID))
	if err != nil {
		return nil, errors.Wrap(err, "generating service address")
	}

	notebookConfigContent, err := generateNotebookConfig(string(taskID))
	if err != nil {
		return nil, errors.Wrap(err, "generating notebook config")
	}

	return &command{
		taskID:    taskID,
		config:    config,
		userFiles: req.UserFiles,
		additionalFiles: archive.Archive{
			req.AgentUserGroup.OwnedArchiveItem(jupyterDir, nil, 0700, tar.TypeDir),
			req.AgentUserGroup.OwnedArchiveItem(jupyterConfigDir, nil, 0700, tar.TypeDir),
			req.AgentUserGroup.OwnedArchiveItem(jupyterDataDir, nil, 0700, tar.TypeDir),
			req.AgentUserGroup.OwnedArchiveItem(jupyterRuntimeDir, nil, 0700, tar.TypeDir),
			req.AgentUserGroup.OwnedArchiveItem(
				jupyterEntrypoint,
				etc.MustStaticFile(etc.NotebookEntrypointResource),
				0700,
				tar.TypeReg,
			),
			req.AgentUserGroup.OwnedArchiveItem(
				notebookConfigFile, notebookConfigContent, 0644, tar.TypeReg,
			),
			req.AgentUserGroup.OwnedArchiveItem(
				notebookDefaultPage,
				etc.MustStaticFile(etc.NotebookTemplateResource),
				0644,
				tar.TypeReg,
			),
		},

		readinessChecks: map[string]readinessCheck{
			"notebook": func(log sproto.ContainerLog) bool {
				return jupyterReadyPattern.MatchString(log.String())
			},
		},
		serviceAddress: &serviceAddress,

		owner:          req.Owner,
		agentUserGroup: req.AgentUserGroup,
		taskSpec:       n.taskSpec,
	}, nil
}
