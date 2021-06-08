package command

import (
	"archive/tar"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"text/template"

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
	notebookDefaultPage = "/run/determined/workdir/Notebook.ipynb"
)

var (
	notebookEntrypoint  = []string{jupyterEntrypoint}
	jupyterReadyPattern = regexp.MustCompile("Jupyter Server .*is running at")
)

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

type notebookManager struct {
	db *db.PgDB

	defaultAgentUserGroup model.AgentUserGroup
	makeTaskSpec          tasks.MakeTaskSpecFn
}

// NotebookLaunchRequest describes a request to launch a new notebook.
type NotebookLaunchRequest struct {
	CommandParams *CommandParams
}

func (n *notebookManager) processLaunchRequest(
	ctx *actor.Context,
	req NotebookLaunchRequest,
) (*summary, int, error) {
	ctx.Log().Info("creating notebook")

	notebook, err := n.newNotebook(req.CommandParams)
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
		users := make(map[string]bool)
		for _, user := range msg.Users {
			users[user] = true
		}
		for _, notebook := range ctx.AskAll(&notebookv1.Notebook{}, ctx.Children()...).GetAll() {
			if typed := notebook.(*notebookv1.Notebook); len(users) == 0 || users[typed.Username] {
				resp.Notebooks = append(resp.Notebooks, typed)
			}
		}
		ctx.Respond(resp)

	case NotebookLaunchRequest:
		summary, statusCode, err := n.processLaunchRequest(ctx, msg)
		if err != nil || statusCode > 200 {
			ctx.Respond(echo.NewHTTPError(statusCode, errors.Wrap(err, "failed to launch shell").Error()))
			return nil
		}
		ctx.Respond(summary.ID)
	}
	return nil
}

func (n *notebookManager) newNotebook(params *CommandParams) (*command, error) {
	config := params.FullConfig
	taskID := sproto.NewTaskID()

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

	setPodSpec(config, params.TaskSpec.TaskContainerDefaults)

	if config.Description == "" {
		petName := petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep)
		config.Description = fmt.Sprintf("Notebook (%s)", petName)
	}

	serviceAddress, err := generateServiceAddress(string(taskID))
	if err != nil {
		return nil, errors.Wrap(err, "generating service address")
	}

	return &command{
		taskID:    taskID,
		config:    *config,
		userFiles: params.UserFiles,
		additionalFiles: archive.Archive{
			params.AgentUserGroup.OwnedArchiveItem(jupyterDir, nil, 0700, tar.TypeDir),
			params.AgentUserGroup.OwnedArchiveItem(jupyterConfigDir, nil, 0700, tar.TypeDir),
			params.AgentUserGroup.OwnedArchiveItem(jupyterDataDir, nil, 0700, tar.TypeDir),
			params.AgentUserGroup.OwnedArchiveItem(jupyterRuntimeDir, nil, 0700, tar.TypeDir),
			params.AgentUserGroup.OwnedArchiveItem(
				jupyterEntrypoint,
				etc.MustStaticFile(etc.NotebookEntrypointResource),
				0700,
				tar.TypeReg,
			),
			params.AgentUserGroup.OwnedArchiveItem(
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

		owner: commandOwner{
			ID:       params.User.ID,
			Username: params.User.Username,
		},
		agentUserGroup: params.AgentUserGroup,
		taskSpec:       params.TaskSpec,

		db: n.db,
	}, nil
}
