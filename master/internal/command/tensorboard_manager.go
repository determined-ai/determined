package command

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

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
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

const (
	expConfPath = "/run/determined/workdir/experiment_config.json"
	// Agent port range is 2600 - 3200. Ports are split between TensorBoard and Notebooks.
	minTensorBoardPort        = 2600
	maxTensorBoardPort        = minTensorBoardPort + 299
	tensorboardEntrypointFile = "/run/determined/workdir/tensorboard-entrypoint.sh"
	tensorboardResourcesSlots = 0
	tensorboardServiceAddress = "/proxy/%s/"
)

type tensorboardRequest struct {
	commandParams

	ExperimentIDs []int `json:"experiment_ids"`
	TrialIDs      []int `json:"trial_ids"`
}

type tensorboardConfig struct {
	model.Experiment
	TrialIDs []int
}

type tensorboardManager struct {
	db *db.PgDB

	defaultAgentUserGroup model.AgentUserGroup
	clusterID             string
}

func (t *tensorboardManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *apiv1.GetTensorboardsRequest:
		resp := &apiv1.GetTensorboardsResponse{}
		for _, tensorboard := range ctx.AskAll(&tensorboardv1.Tensorboard{}, ctx.Children()...).GetAll() {
			resp.Tensorboards = append(resp.Tensorboards, tensorboard.(*tensorboardv1.Tensorboard))
		}
		ctx.Respond(resp)

	case echo.Context:
		t.handleAPIRequest(ctx, msg)
	}
	return nil
}

func (t *tensorboardManager) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		userFilter := apiCtx.QueryParam("user")
		ctx.Respond(apiCtx.JSON(
			http.StatusOK,
			ctx.AskAll(getSummary{userFilter: userFilter}, ctx.Children()...)))

	case echo.POST:
		req := tensorboardRequest{}
		if err := apiCtx.Bind(&req); err != nil {
			respondBadRequest(ctx, err)
			return
		}

		commandReq, err := parseCommandRequest(apiCtx, t.db, &req.commandParams)
		if err != nil {
			respondBadRequest(ctx, err)
			return
		}

		if commandReq.AgentUserGroup == nil {
			commandReq.AgentUserGroup = &t.defaultAgentUserGroup
		}

		if len(req.ExperimentIDs) == 0 && len(req.TrialIDs) == 0 {
			respondBadRequest(ctx, errors.New("must set experiment or trial ids"))
			return
		}

		ctx.Log().Infof("creating tensorboard (experiment id(s): %v trial id(s): %v)",
			req.ExperimentIDs, req.TrialIDs)

		b, err := t.newTensorBoard(commandReq, req)

		if err != nil {
			ctx.Respond(err)
			return
		}

		if err := check.Validate(b.config); err != nil {
			ctx.Respond(err)
			return
		}

		a, _ := ctx.ActorOf(b.taskID, b)
		ctx.Respond(apiCtx.JSON(http.StatusOK, ctx.Ask(a, getSummary{})))
		ctx.Log().Infof("created tensorboard %s", a.Address().Local())

	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (t *tensorboardManager) newTensorBoard(
	commandReq *commandRequest,
	req tensorboardRequest,
) (*command, error) {
	exps, err := t.getTensorBoardConfigs(req)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if len(exps) == 0 {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "no experiments found")
	}

	var logDirs []string

	additionalFiles := archive.Archive{
		commandReq.AgentUserGroup.OwnedArchiveItem(
			tensorboardEntrypointFile,
			etc.MustStaticFile(etc.TensorboardEntryScriptResource), 0700,
			tar.TypeReg,
		),
	}

	uniqMounts := map[model.BindMount]bool{}
	uniqEnvVars := map[string]string{}

	taskID := scheduler.NewTaskID()
	serviceAddress := fmt.Sprintf(tensorboardServiceAddress, taskID)

	config := commandReq.Config

	uniqEnvVars["TF_CPP_MIN_LOG_LEVEL"] = "3"

	for _, exp := range exps {
		var logBasePath string

		switch c := exp.Config.CheckpointStorage; {
		case c.SharedFSConfig != nil:
			// Mount the checkpoint location into the TensorBoard container to
			// make the logs visible to TensorBoard. Bind mounts must be unique
			// and therefore we use a map here to deduplicate mounts.
			uniqMounts[model.BindMount{
				ContainerPath: model.DefaultSharedFSContainerPath,
				HostPath:      c.SharedFSConfig.HostPath,
				Propagation:   model.DefaultSharedFSPropagation,
			}] = true

			logBasePath = model.DefaultSharedFSContainerPath
			if c.SharedFSConfig.StoragePath != nil {
				logBasePath = fmt.Sprintf("%s/%s", logBasePath, *c.SharedFSConfig.StoragePath)
			}

		case c.S3Config != nil:
			if c.S3Config.AccessKey != nil {
				uniqEnvVars["AWS_ACCESS_KEY_ID"] = *c.S3Config.AccessKey
			}
			if c.S3Config.SecretKey != nil {
				uniqEnvVars["AWS_SECRET_ACCESS_KEY"] = *c.S3Config.SecretKey
			}
			if c.S3Config.EndpointURL != nil {
				endpoint, urlErr := url.Parse(*c.S3Config.EndpointURL)
				if urlErr != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError,
						"unable to parse checkpoint_storage.s3.endpoint_url")
				}

				// The TensorBoard container needs access to the original URL
				// and the URL in "host:port" form.
				uniqEnvVars["DET_S3_ENDPOINT"] = *c.S3Config.EndpointURL
				uniqEnvVars["S3_ENDPOINT"] = endpoint.Host

				uniqEnvVars["S3_USE_HTTPS"] = "0"
				if endpoint.Scheme == "https" {
					uniqEnvVars["S3_USE_HTTPS"] = "1"
				}
			}

			uniqEnvVars["AWS_BUCKET"] = c.S3Config.Bucket

			logBasePath = "s3://" + c.S3Config.Bucket

		case c.GCSConfig != nil:
			logBasePath = "gs://" + c.GCSConfig.Bucket

		case c.HDFSConfig != nil:
			logBasePath = "hdfs://" + c.HDFSConfig.Path

			// The credentials files for HDFS exist on agent machines and are
			// bind mounted into the container.
			for _, mount := range exp.Config.BindMounts {
				uniqMounts[mount] = true
			}

		default:
			return nil, echo.NewHTTPError(http.StatusBadRequest, "unknown storage backend for experiment")
		}

		if len(exp.TrialIDs) == 0 {
			expDir := fmt.Sprintf("%s/%s/tensorboard/experiment/%d/", logBasePath, t.clusterID, exp.ID)
			logDirs = append(logDirs, expDir)
			continue
		}

		for _, id := range exp.TrialIDs {
			trialDir := fmt.Sprintf("trial_%d:%s/%s/tensorboard/experiment/%d/trial/%d/",
				id, logBasePath, t.clusterID, exp.ID, id)

			logDirs = append(logDirs, trialDir)
		}
	}

	// Take the most recent experiment config and add it to the container. This
	// is used to determine if the experiment is backed by S3.
	finalExpConf := exps[len(exps)-1].Config

	eConf, err := json.Marshal(finalExpConf)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			"unable to marshal experiment configuration")
	}

	additionalFiles = append(additionalFiles,
		commandReq.AgentUserGroup.OwnedArchiveItem(expConfPath, eConf, 0700, tar.TypeReg))

	// Multiple experiments may have different s3 credentials. We sort the
	// experiments in ascending experiment ID order and dedupicate the
	// environment variables by key name. This gives the behavior of selecting
	// the most recent s3 credentials to start the tensorboard process with.
	envVars := getEnvVars(uniqEnvVars)

	// Select a random port from the range to assign to TensorBoard. In host
	// mode, this mitigates the risk of multiple TensorBoard processes binding
	// the same port on an agent.
	port := getPort(minTensorBoardPort, maxTensorBoardPort)
	config.Environment.Ports = map[string]int{"tensorboard": port}
	envVars = append(envVars, fmt.Sprintf("TENSORBOARD_PORT=%d", port))

	config.Description = fmt.Sprintf(
		"TensorBoard (%s)",
		petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep),
	)
	config.Entrypoint = []string{tensorboardEntrypointFile, "--logdir", strings.Join(logDirs, ",")}
	config.Resources.Slots = tensorboardResourcesSlots
	config.Environment.EnvironmentVariables = model.RuntimeItems{CPU: envVars, GPU: envVars}
	config.BindMounts = getMounts(uniqMounts)

	return &command{
		taskID:          taskID,
		config:          config,
		userFiles:       commandReq.UserFiles,
		additionalFiles: additionalFiles,
		metadata: map[string]interface{}{
			"experiment_ids": req.ExperimentIDs,
			"trial_ids":      req.TrialIDs,
		},
		readinessChecks: map[string]readinessCheck{
			"tensorboard": func(log sproto.ContainerLog) bool {
				return strings.Contains(log.String(), "TensorBoard contains metrics")
			},
		},
		serviceAddress: &serviceAddress,
		owner:          commandReq.Owner,
		agentUserGroup: commandReq.AgentUserGroup,
	}, nil
}

func (t *tensorboardManager) getTensorBoardConfigs(req tensorboardRequest) (
	[]*tensorboardConfig, error) {
	confByID := map[int]*tensorboardConfig{}
	var exp *model.Experiment
	var err error

	for _, id := range req.ExperimentIDs {
		exp, err = t.db.ExperimentByID(id)
		if err != nil {
			return nil, err
		}

		confByID[id] = &tensorboardConfig{Experiment: *exp}
	}

	for _, id := range req.TrialIDs {
		exp, err = t.db.ExperimentByTrialID(id)
		if err != nil {
			return nil, err
		}

		if conf, ok := confByID[exp.ID]; ok {
			conf.TrialIDs = append(conf.TrialIDs, id)
			continue
		}

		confByID[exp.ID] = &tensorboardConfig{Experiment: *exp, TrialIDs: []int{id}}
	}

	var expIDs []int
	for id := range confByID {
		expIDs = append(expIDs, id)
	}

	sort.Ints(expIDs)
	var configs []*tensorboardConfig
	for _, id := range expIDs {
		configs = append(configs, confByID[id])
	}

	return configs, nil
}

func getMounts(m map[model.BindMount]bool) []model.BindMount {
	var bindMounts []model.BindMount

	for mount := range m {
		bindMounts = append(bindMounts, mount)
	}

	return bindMounts
}

func getEnvVars(m map[string]string) []string {
	var envVars []string

	for k, v := range m {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	return envVars
}
