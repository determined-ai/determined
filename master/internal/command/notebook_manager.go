package command

import (
	"archive/tar"
	"bytes"
	"net/http"
	"strings"
	"text/template"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	jupyterDir          = "/run/determined/jupyter/"
	jupyterConfigDir    = "/run/determined/jupyter/config"
	jupyterDataDir      = "/run/determined/jupyter/data"
	jupyterRuntimeDir   = "/run/determined/jupyter/runtime"
	jupyterEntrypoint   = "/run/determined/jupyter/notebook-entrypoint.sh"
	notebookConfigFile  = "/run/determined/workdir/jupyter-conf.py"
	notebookDefaultPage = "/run/determined/workdir/Notebook.ipynb"
)

var (
	notebookEntrypoint = []string{jupyterEntrypoint}
	notebookPorts      = map[string]int{"notebook": 8888}
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
	clusterID             string
}

func (n *notebookManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
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

		ctx.Log().Info("creating notebook")

		notebook, err := n.newNotebook(req)
		if err != nil {
			ctx.Respond(errors.Wrap(err, "creating notebook"))
			return
		}

		if err = check.Validate(notebook.config); err != nil {
			respondBadRequest(ctx, err)
			return
		}

		a, _ := ctx.ActorOf(notebook.taskID, notebook)
		ctx.Respond(apiCtx.JSON(http.StatusOK, ctx.Ask(a, getSummary{})))
		ctx.Log().Infof("created notebook %s", a.Address().Local())

	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (n *notebookManager) newNotebook(req *commandRequest) (*command, error) {
	config := req.Config
	taskID := scheduler.NewTaskID()

	// Postprocess the config. Add Jupyter and configuration to the container.
	config.Environment.Ports = notebookPorts
	config.Entrypoint = notebookEntrypoint

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
				return strings.Contains(log.String(), "Jupyter Notebook is running")
			},
		},
		serviceAddress: &serviceAddress,

		owner:          req.Owner,
		agentUserGroup: req.AgentUserGroup,
	}, nil
}
