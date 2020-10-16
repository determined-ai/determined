package command

import (
	"archive/tar"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	requestContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

const (
	expConfPath = "/run/determined/workdir/experiment_config.json"
	// Agent ports 2600 - 3500 are split between TensorBoards, Notebooks, and Shells.
	minTensorBoardPort        = 2600
	maxTensorBoardPort        = minTensorBoardPort + 299
	tensorboardEntrypointFile = "/run/determined/workdir/tensorboard-entrypoint.sh"
	tensorboardResourcesSlots = 0
	tensorboardServiceAddress = "/proxy/%s/"
	tickInterval              = 5 * time.Second
)

// TensorboardRequest describes a request for a new Tensorboard.
type TensorboardRequest struct {
	*CommandParams

	ExperimentIDs []int `json:"experiment_ids"`
	TrialIDs      []int `json:"trial_ids"`
}

// TensorboardRequestWithUser accompanies TensorboardRequest with a user.
type TensorboardRequestWithUser struct {
	Tensorboard TensorboardRequest
	User        *model.User
}

type tensorboardConfig struct {
	model.Experiment
	TrialIDs []int
}

type tensorboardManager struct {
	db *db.PgDB

	defaultAgentUserGroup model.AgentUserGroup
	timeout               time.Duration
	proxyRef              *actor.Ref
	taskSpec              *tasks.TaskSpec
}

type tensorboardTick struct{}

func (t *tensorboardManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, tickInterval, tensorboardTick{})
	case *apiv1.GetTensorboardsRequest:
		resp := &apiv1.GetTensorboardsResponse{}
		for _, tensorboard := range ctx.AskAll(&tensorboardv1.Tensorboard{}, ctx.Children()...).GetAll() {
			resp.Tensorboards = append(resp.Tensorboards, tensorboard.(*tensorboardv1.Tensorboard))
		}
		ctx.Respond(resp)

	case echo.Context:
		t.handleAPIRequest(ctx, msg)
	case tensorboardTick:
		services := ctx.Ask(t.proxyRef, proxy.GetSummary{}).Get().(map[string]proxy.Service)
		for _, boardRef := range ctx.Children() {
			boardSummary := ctx.Ask(boardRef, getSummary{}).Get().(summary)
			if boardSummary.State != container.Running.String() {
				continue
			}

			service, ok := services[string(boardSummary.ID)]
			if !ok {
				continue
			}

			if time.Now().After(service.LastRequested.Add(t.timeout)) {
				ctx.Log().Infof("killing %s due to inactivity", boardSummary.Config.Description)
				ctx.Ask(boardRef, &apiv1.KillTensorboardRequest{})
			}
		}

		actors.NotifyAfter(ctx, tickInterval, tensorboardTick{})
	case TensorboardRequestWithUser:
		summary, err := t.processTensorboardRequest(ctx, msg.User, &msg.Tensorboard)
		if err != nil {
			ctx.Respond(errors.Wrap(err, "failed to launch tensorboard"))
		} else {
			ctx.Respond(summary.ID)
		}
	}

	return nil
}

func (t *tensorboardManager) processTensorboardRequest(
	ctx *actor.Context,
	user *model.User,
	req *TensorboardRequest,
) (*summary, error) {
	commandReq, err := parseCommandRequest(
		*user, t.db, req.CommandParams, &t.taskSpec.TaskContainerDefaults)
	if err != nil {
		return nil, err
	}

	if commandReq.AgentUserGroup == nil {
		commandReq.AgentUserGroup = &t.defaultAgentUserGroup
	}

	if len(req.ExperimentIDs) == 0 && len(req.TrialIDs) == 0 {
		err = errors.New("must set experiment or trial ids")
		return nil, err
	}

	ctx.Log().Infof("creating tensorboard (experiment id(s): %v trial id(s): %v)",
		req.ExperimentIDs, req.TrialIDs)

	b, err := t.newTensorBoard(commandReq, *req)

	if err != nil {
		err = errors.Wrap(err, "failed to create tensorboard")
		return nil, err
	}

	if err := check.Validate(b.config); err != nil {
		err = errors.Wrap(err, "failed to validate tensorboard config")
		return nil, err
	}

	a, _ := ctx.ActorOf(b.taskID, b)
	summaryResponse := ctx.Ask(a, getSummary{})
	if err := summaryResponse.Error(); err != nil {
		return nil, err
	}
	ctx.Log().Infof("created tensorboard %s", a.Address().Local())
	summary := summaryResponse.Get().(summary)
	// REMOVEME
	// jConfig, err := json.Marshal(summary.Config)
	// if err != nil {
	// 	fmt.Println("failed to marshal config")
	// }
	// fmt.Printf("config description %s\n", jConfig)
	return &summary, nil
}

func (t *tensorboardManager) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		userFilter := apiCtx.QueryParam("user")
		ctx.Respond(apiCtx.JSON(
			http.StatusOK,
			ctx.AskAll(getSummary{userFilter: userFilter}, ctx.Children()...)))

	case echo.POST:
		req := TensorboardRequest{}
		if err := apiCtx.Bind(&req); err != nil {
			respondBadRequest(ctx, err)
			return
		}
		user := apiCtx.(*requestContext.DetContext).MustGetUser()
		summary, err := t.processTensorboardRequest(ctx, &user, &req)
		if err != nil {
			respondBadRequest(ctx, err)
		} else {
			ctx.Respond(apiCtx.JSON(http.StatusOK, summary))
		}

	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (t *tensorboardManager) newTensorBoard(
	commandReq *commandRequest,
	req TensorboardRequest,
) (*command, error) {
	// Warning! Since certain fields are incompatible with the current model.Experiment,
	// internally this avoids loading certain parts of the experiment configuration so
	// we can load their tensorboards still.
	// TODO(DET-4009): Fix this in the experiment configuration backwards compatibility project.
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

	taskID := resourcemanagers.NewTaskID()
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
			expDir := fmt.Sprintf("%s/%s/tensorboard/experiment/%d/",
				logBasePath, t.taskSpec.ClusterID, exp.ID)
			logDirs = append(logDirs, expDir)
			continue
		}

		for _, id := range exp.TrialIDs {
			trialDir := fmt.Sprintf("trial_%d:%s/%s/tensorboard/experiment/%d/trial/%d/",
				id, logBasePath, t.taskSpec.ClusterID, exp.ID, id)

			logDirs = append(logDirs, trialDir)
		}
	}

	// Get the most recent experiment config as raw json and add it to the container. This
	// is used to determine if the experiment is backed by S3.
	mostRecentExpID := exps[len(exps)-1].ID
	confBytes, err := t.db.ExperimentConfigRaw(mostRecentExpID)
	if err != nil {
		return nil, errors.Wrapf(err, "error loading raw experiment config: %d", mostRecentExpID)
	}

	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			"unable to marshal experiment configuration")
	}

	additionalFiles = append(additionalFiles,
		commandReq.AgentUserGroup.OwnedArchiveItem(expConfPath, confBytes, 0700, tar.TypeReg))

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

	refineArgs(config.TensorBoardArgs)
	config.Entrypoint = append(
		[]string{tensorboardEntrypointFile, "--logdir", strings.Join(logDirs, ",")},
		config.TensorBoardArgs...)

	config.Resources.Slots = tensorboardResourcesSlots

	cpuEnvVars := append(config.Environment.EnvironmentVariables.CPU, envVars...)
	gpuEnvVars := append(config.Environment.EnvironmentVariables.GPU, envVars...)
	config.Environment.EnvironmentVariables = model.RuntimeItems{CPU: cpuEnvVars, GPU: gpuEnvVars}
	config.BindMounts = append(config.BindMounts, getMounts(uniqMounts)...)

	setPodSpec(&config, t.taskSpec.TaskContainerDefaults)

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
		taskSpec:       t.taskSpec,
	}, nil
}

func (t *tensorboardManager) getTensorBoardConfigs(req TensorboardRequest) (
	[]*tensorboardConfig, error) {
	confByID := map[int]*tensorboardConfig{}
	var exp *model.Experiment
	var err error

	for _, id := range req.ExperimentIDs {
		exp, err = t.db.ExperimentWithoutBackwardsIncompatibleFieldsByID(id)
		if err != nil {
			return nil, err
		}

		confByID[id] = &tensorboardConfig{Experiment: *exp}
	}

	for _, id := range req.TrialIDs {
		expID, err := t.db.ExperimentIDByTrialID(id)
		if err != nil {
			return nil, err
		}

		exp, err = t.db.ExperimentWithoutBackwardsIncompatibleFieldsByID(expID)
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

func refineArgs(s []string) {
	trimmed := ""
	for x := range s {
		trimmed = strings.TrimLeft(s[x], "-")
		if trimmed == "h" {
			s[x] = "-h"
		} else {
			s[x] = "--" + trimmed
		}
	}
}
