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
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
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
	tensorboardServiceAddress = "/proxy/%s/"
	tickInterval              = 5 * time.Second
)

// TensorboardRequest describes a request for a new Tensorboard.
type TensorboardRequest struct {
	CommandParams *CommandParams

	ExperimentIDs []int `json:"experiment_ids"`
	TrialIDs      []int `json:"trial_ids"`
}

type tensorboardConfig struct {
	Config       expconf.LegacyConfig
	ExperimentID int
	TrialIDs     []int
}

type tensorboardManager struct {
	db *db.PgDB

	defaultAgentUserGroup model.AgentUserGroup
	timeout               time.Duration
	proxyRef              *actor.Ref
	makeTaskSpec          tasks.MakeTaskSpecFn
}

type tensorboardTick struct{}

func (t *tensorboardManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, tickInterval, tensorboardTick{})
	case *apiv1.GetTensorboardsRequest:
		resp := &apiv1.GetTensorboardsResponse{}
		users := make(map[string]bool)
		for _, user := range msg.Users {
			users[user] = true
		}
		for _, tensorboard := range ctx.AskAll(&tensorboardv1.Tensorboard{}, ctx.Children()...).GetAll() {
			if typed := tensorboard.(*tensorboardv1.Tensorboard); len(users) == 0 || users[typed.Username] {
				resp.Tensorboards = append(resp.Tensorboards, typed)
			}
		}
		ctx.Respond(resp)

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
	case TensorboardRequest:
		summary, statusCode, err := t.processLaunchRequest(ctx, &msg)
		if err != nil || statusCode > 200 {
			ctx.Respond(echo.NewHTTPError(statusCode,
				errors.Wrap(err, "failed to launch Tensorboard").Error(),
			))
			return nil
		}
		ctx.Respond(summary.ID)
	}

	return nil
}

func (t *tensorboardManager) processLaunchRequest(
	ctx *actor.Context,
	req *TensorboardRequest,
) (*summary, int, error) {
	var err error
	params := req.CommandParams

	if len(req.ExperimentIDs) == 0 && len(req.TrialIDs) == 0 {
		err = errors.New("must set experiment or trial ids")
		return nil, http.StatusBadRequest, err
	}

	ctx.Log().Infof("creating tensorboard (experiment id(s): %v trial id(s): %v)",
		req.ExperimentIDs, req.TrialIDs)

	b, err := t.newTensorBoard(params, *req)

	if err != nil {
		err = errors.Wrap(err, "failed to create tensorboard")
		return nil, http.StatusInternalServerError, err
	}

	if err := check.Validate(b.config); err != nil {
		err = errors.Wrap(err, "failed to validate tensorboard config")
		return nil, http.StatusBadRequest, err
	}

	a, _ := ctx.ActorOf(b.taskID, b)
	summaryFut := ctx.Ask(a, getSummary{})
	if err := summaryFut.Error(); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	ctx.Log().Infof("created tensorboard %s", a.Address().Local())
	summary := summaryFut.Get().(summary)
	return &summary, http.StatusOK, nil
}

func (t *tensorboardManager) newTensorBoard(
	params *CommandParams,
	req TensorboardRequest,
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
		params.AgentUserGroup.OwnedArchiveItem(
			tensorboardEntrypointFile,
			etc.MustStaticFile(etc.TensorboardEntryScriptResource), 0700,
			tar.TypeReg,
		),
	}

	uniqMounts := map[string]model.BindMount{}
	uniqEnvVars := map[string]string{}

	taskID := sproto.NewTaskID()
	serviceAddress := fmt.Sprintf(tensorboardServiceAddress, taskID)

	config := params.FullConfig

	uniqEnvVars["TF_CPP_MIN_LOG_LEVEL"] = "3"

	for _, exp := range exps {
		var logBasePath string

		switch c := exp.Config.CheckpointStorage().GetUnionMember().(type) {
		case expconf.SharedFSConfig:
			// Mount the checkpoint location into the TensorBoard container to
			// make the logs visible to TensorBoard. Bind mounts must be unique
			// and therefore we use a map here to deduplicate mounts.
			sharedFSMount := schemas.WithDefaults(expconf.BindMount{
				RawContainerPath: model.DefaultSharedFSContainerPath,
				RawHostPath:      c.HostPath(),
				RawPropagation:   ptrs.StringPtr(model.DefaultSharedFSPropagation),
			}).(expconf.BindMount)
			uniqMounts[sharedFSMount.ContainerPath()] = model.ToModelBindMount(sharedFSMount)
			logBasePath = c.PathInContainer()

		case expconf.S3Config:
			if c.AccessKey() != nil {
				uniqEnvVars["AWS_ACCESS_KEY_ID"] = *c.AccessKey()
			}
			if c.SecretKey() != nil {
				uniqEnvVars["AWS_SECRET_ACCESS_KEY"] = *c.SecretKey()
			}
			if c.EndpointURL() != nil {
				endpoint, urlErr := url.Parse(*c.EndpointURL())
				if urlErr != nil {
					return nil, echo.NewHTTPError(http.StatusInternalServerError,
						"unable to parse checkpoint_storage.s3.endpoint_url")
				}

				// The TensorBoard container needs access to the original URL
				// and the URL in "host:port" form.
				uniqEnvVars["DET_S3_ENDPOINT"] = *c.EndpointURL()
				uniqEnvVars["S3_ENDPOINT"] = endpoint.Host

				uniqEnvVars["S3_USE_HTTPS"] = "0"
				if endpoint.Scheme == "https" {
					uniqEnvVars["S3_USE_HTTPS"] = "1"
				}
			}

			uniqEnvVars["AWS_BUCKET"] = c.Bucket()

			logBasePath = "s3://" + c.Bucket()

		case expconf.AzureConfig:
			logBasePath = "azure://" + c.Container()

		case expconf.GCSConfig:
			logBasePath = "gs://" + c.Bucket()

		case expconf.HDFSConfig:
			logBasePath = "hdfs://" + c.Path()

			// The credentials files for HDFS exist on agent machines and are
			// bind mounted into the container.
			for _, mount := range exp.Config.BindMounts() {
				uniqMounts[mount.ContainerPath()] = model.ToModelBindMount(mount)
			}

		default:
			return nil, echo.NewHTTPError(
				http.StatusBadRequest, fmt.Sprintf(
					"unknown storage backend for experiment: %T", c,
				),
			)
		}

		if len(exp.TrialIDs) == 0 {
			expDir := fmt.Sprintf("%s/%s/tensorboard/experiment/%d/",
				logBasePath, params.TaskSpec.ClusterID, exp.ExperimentID)
			logDirs = append(logDirs, expDir)
			continue
		}

		for _, id := range exp.TrialIDs {
			trialDir := fmt.Sprintf("trial_%d:%s/%s/tensorboard/experiment/%d/trial/%d/",
				id, logBasePath, params.TaskSpec.ClusterID, exp.ExperimentID, id)

			logDirs = append(logDirs, trialDir)
		}
	}

	// Get the most recent experiment config as raw json and add it to the container. This
	// is used to determine if the experiment is backed by S3.
	mostRecentExpID := exps[len(exps)-1].ExperimentID
	confBytes, err := t.db.ExperimentConfigRaw(mostRecentExpID)
	if err != nil {
		return nil, errors.Wrapf(err, "error loading experiment config: %d", mostRecentExpID)
	}

	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			"unable to marshal experiment configuration")
	}

	expConf, err := expconf.ParseAnyExperimentConfigYAML(confBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing experiment config: %d", mostRecentExpID)
	}
	expConf = schemas.WithDefaults(expConf).(expconf.ExperimentConfig)

	additionalFiles = append(additionalFiles,
		params.AgentUserGroup.OwnedArchiveItem(expConfPath, confBytes, 0700, tar.TypeReg))

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
		[]string{tensorboardEntrypointFile, strings.Join(logDirs, ",")},
		config.TensorBoardArgs...)

	cpuEnvVars := append(config.Environment.EnvironmentVariables.CPU, envVars...)
	gpuEnvVars := append(config.Environment.EnvironmentVariables.GPU, envVars...)
	config.Environment.EnvironmentVariables = model.RuntimeItems{CPU: cpuEnvVars, GPU: gpuEnvVars}
	config.Environment.Image = model.RuntimeItem{CPU: expConf.Environment().Image().CPU(),
		GPU: expConf.Environment().Image().GPU()}

	var bindMounts []model.BindMount

	for _, uniqMount := range uniqMounts {
		bindMounts = append(bindMounts, uniqMount)
	}

	config.BindMounts = append(config.BindMounts, bindMounts...)

	setPodSpec(config, params.TaskSpec.TaskContainerDefaults)

	return &command{
		taskID:          taskID,
		config:          *config,
		userFiles:       params.UserFiles,
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
		owner: commandOwner{
			ID:       params.User.ID,
			Username: params.User.Username,
		},
		agentUserGroup: params.AgentUserGroup,
		taskSpec:       params.TaskSpec,

		db: t.db,
	}, nil
}

func (t *tensorboardManager) getTensorBoardConfigs(req TensorboardRequest) (
	[]*tensorboardConfig, error) {
	confByID := map[int]*tensorboardConfig{}

	for _, expID := range req.ExperimentIDs {
		conf, err := t.db.LegacyExperimentConfigByID(expID)
		if err != nil {
			return nil, err
		}

		confByID[expID] = &tensorboardConfig{ExperimentID: expID, Config: conf}
	}

	for _, trialID := range req.TrialIDs {
		expID, err := t.db.ExperimentIDByTrialID(trialID)
		if err != nil {
			return nil, err
		}

		conf, err := t.db.LegacyExperimentConfigByID(expID)
		if err != nil {
			return nil, err
		}

		if conf, ok := confByID[expID]; ok {
			conf.TrialIDs = append(conf.TrialIDs, trialID)
			continue
		}

		confByID[expID] = &tensorboardConfig{
			ExperimentID: expID, Config: conf, TrialIDs: []int{trialID},
		}
	}

	var expIDs []int
	for expID := range confByID {
		expIDs = append(expIDs, expID)
	}

	sort.Ints(expIDs)
	var configs []*tensorboardConfig
	for _, expID := range expIDs {
		configs = append(configs, confByID[expID])
	}

	return configs, nil
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
