package internal

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	petname "github.com/dustinkirkland/golang-petname"

	"github.com/pkg/errors"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/db"
	exputil "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/check"
	pkgCommand "github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

const (
	// Agent ports 2600 - 3500 are split between TensorBoards, Notebooks, and Shells.
	minTensorBoardPort        = 2600
	maxTensorBoardPort        = minTensorBoardPort + 299
	tensorboardEntrypointFile = "/run/determined/tensorboard/tensorboard-entrypoint.sh"
	storageConfPath           = "/run/determined/tensorboard/storage_config.json"
)

var tensorboardsAddr = actor.Addr("tensorboard")

func filesToArchive(files []*utilv1.File) archive.Archive {
	filesArchive := make([]archive.Item, 0, len(files))
	for _, file := range files {
		item := archive.Item{
			Content:      file.Content,
			FileMode:     os.FileMode(file.Mode),
			GroupID:      int(file.Gid),
			ModifiedTime: archive.UnixTime{Time: time.Unix(file.Mtime, 0)},
			Path:         file.Path,
			Type:         byte(file.Type),
			UserID:       int(file.Uid),
		}
		filesArchive = append(filesArchive, item)
	}
	return filesArchive
}

func (a *apiServer) GetTensorboards(
	ctx context.Context, req *apiv1.GetTensorboardsRequest,
) (resp *apiv1.GetTensorboardsResponse, err error) {
	defer func() {
		if status.Code(err) == codes.Unknown {
			err = apiutils.MapAndFilterErrors(err, nil, nil)
		}
	}()

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.WorkspaceId != 0 {
		// check if the workspace exists.
		_, err := a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, false)
		if err != nil {
			return nil, err
		}
	}

	if err = a.ask(tensorboardsAddr, req, &resp); err != nil {
		return nil, err
	}

	filteredTensorboards, err := command.AuthZProvider.Get().FilterTensorboards(
		ctx, *curUser, model.AccessScopeID(req.WorkspaceId), resp.Tensorboards)
	if err != nil {
		return nil, err
	}

	resp.Tensorboards = filteredTensorboards

	api.Sort(resp.Tensorboards, req.OrderBy, req.SortBy, apiv1.GetTensorboardsRequest_SORT_BY_ID)
	return resp, api.Paginate(&resp.Pagination, &resp.Tensorboards, req.Offset, req.Limit)
}

func (a *apiServer) GetTensorboard(
	ctx context.Context, req *apiv1.GetTensorboardRequest,
) (resp *apiv1.GetTensorboardResponse, err error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	addr := tensorboardsAddr.Child(req.TensorboardId)
	if err := a.ask(addr, req, &resp); err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.TensorboardId)
	if err := command.AuthZProvider.Get().CanGetTensorboard(
		ctx, *curUser, model.AccessScopeID(resp.Tensorboard.WorkspaceId),
		resp.Tensorboard.ExperimentIds, resp.Tensorboard.TrialIds); err != nil {
		return nil, authz.SubIfUnauthorized(err, api.NotFoundErrs("actor", fmt.Sprint(addr), true))
	}
	return resp, nil
}

func (a *apiServer) KillTensorboard(
	ctx context.Context, req *apiv1.KillTensorboardRequest,
) (resp *apiv1.KillTensorboardResponse, err error) {
	defer func() {
		if status.Code(err) == codes.Unknown {
			err = apiutils.MapAndFilterErrors(err, nil, nil)
		}
	}()

	getResponse, err := a.GetTensorboard(ctx,
		&apiv1.GetTensorboardRequest{TensorboardId: req.TensorboardId})
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.TensorboardId)
	err = command.AuthZProvider.Get().CanTerminateTensorboard(
		ctx, *curUser, model.AccessScopeID(getResponse.Tensorboard.WorkspaceId))
	if err != nil {
		return nil, err
	}

	return resp, a.ask(tensorboardsAddr.Child(req.TensorboardId), req, &resp)
}

func (a *apiServer) SetTensorboardPriority(
	ctx context.Context, req *apiv1.SetTensorboardPriorityRequest,
) (resp *apiv1.SetTensorboardPriorityResponse, err error) {
	defer func() {
		if status.Code(err) == codes.Unknown {
			err = apiutils.MapAndFilterErrors(err, nil, nil)
		}
	}()

	getResponse, err := a.GetTensorboard(ctx,
		&apiv1.GetTensorboardRequest{TensorboardId: req.TensorboardId})
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	ctx = audit.SupplyEntityID(ctx, req.TensorboardId)
	err = command.AuthZProvider.Get().CanSetNSCsPriority(
		ctx, *curUser, model.AccessScopeID(getResponse.Tensorboard.WorkspaceId), int(req.Priority))
	if err != nil {
		return nil, err
	}

	return resp, a.ask(tensorboardsAddr.Child(req.TensorboardId), req, &resp)
}

func (a *apiServer) LaunchTensorboard(
	ctx context.Context, req *apiv1.LaunchTensorboardRequest,
) (*apiv1.LaunchTensorboardResponse, error) {
	var err error

	// Validate the request.
	if len(req.ExperimentIds) == 0 && len(req.TrialIds) == 0 && req.Filters == nil {
		err = errors.New("must set experiment, trial ids, or filters")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	launchReq, launchWarnings, err := a.getCommandLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		WorkspaceID:  req.WorkspaceId,
		Config:       req.Config,
		Files:        req.Files,
		MustZeroSlot: true,
	}, user)
	if err != nil {
		return nil, api.WrapWithFallbackCode(err, codes.InvalidArgument,
			"failed to prepare launch params")
	}
	spec := launchReq.Spec
	spec.TaskType = model.TaskTypeTensorboard
	spec.Metadata.ExperimentIDs = req.ExperimentIds
	spec.Metadata.TrialIDs = req.TrialIds

	if err = a.isNTSCPermittedToLaunch(ctx, spec, user); err != nil {
		return nil, err
	}

	exps, err := a.getTensorBoardConfigsFromReq(ctx, a.m.db, req)
	if err != nil {
		return nil, err
	}
	if len(exps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no experiments found")
	}

	spec.WatchProxyIdleTimeout = true

	// Postprocess the spec.
	if spec.Config.IdleTimeout == nil {
		masterTensorBoardIdleTimeout := model.Duration(
			time.Duration(a.m.config.TensorBoardTimeout) * time.Second)
		spec.Config.IdleTimeout = &masterTensorBoardIdleTimeout
	}

	spec.Config.Description = fmt.Sprintf(
		"TensorBoard (%s)",
		petname.Generate(expconf.TaskNameGeneratorWords, expconf.TaskNameGeneratorSep),
	)

	// Selecting a random port mitigates the risk of multiple processes binding
	// the same port on an agent in host mode.
	port := getRandomPort(minTensorBoardPort, maxTensorBoardPort)
	spec.Base.ExtraProxyPorts = append(spec.Base.ExtraProxyPorts, expconf.ProxyPort{
		RawProxyPort:        port,
		RawDefaultServiceID: ptrs.Ptr(true),
	})

	logDirs := make([]string, 0)
	uniqMounts := map[string]model.BindMount{}

	// Multiple experiments may have different s3 credentials. We sort the
	// experiments in ascending experiment ID order and dedupicate the
	// environment variables by key name. This gives the behavior of selecting
	// the most recent s3 credentials to start the tensorboard process with.
	uniqEnvVars := map[string]string{
		"TENSORBOARD_PORT":     strconv.Itoa(port),
		"TF_CPP_MIN_LOG_LEVEL": "3",
		"DET_TASK_TYPE":        string(model.TaskTypeTensorboard),
	}

	if spec.Config.Debug {
		uniqEnvVars["DET_DEBUG"] = "true"
	}

	for _, exp := range exps {
		var logBasePath string

		switch c := exp.Config.CheckpointStorage.GetUnionMember().(type) {
		case expconf.SharedFSConfig:
			// Mount the checkpoint location into the TensorBoard container to
			// make the logs visible to TensorBoard. Bind mounts must be unique
			// and therefore we use a map here to deduplicate mounts.
			sharedFSMount := schemas.WithDefaults(expconf.BindMount{
				RawContainerPath: expconf.DefaultSharedFSContainerPath,
				RawHostPath:      c.HostPath(),
				RawPropagation:   ptrs.Ptr(expconf.DefaultSharedFSPropagation),
			})
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
					return nil, status.Error(codes.Internal,
						"unable to parse checkpoint_storage.s3.endpoint_url")
				}

				// The TensorBoard container needs access to the original URL
				// and the URL in "host:port" form.
				uniqEnvVars["DET_S3_ENDPOINT_URL"] = *c.EndpointURL()
				uniqEnvVars["S3_ENDPOINT"] = endpoint.Host

				uniqEnvVars["S3_USE_HTTPS"] = "0"
				if endpoint.Scheme == "https" {
					uniqEnvVars["S3_USE_HTTPS"] = "1"
				}
			}

			uniqEnvVars["AWS_BUCKET"] = c.Bucket()

			prefix := c.Prefix()
			if prefix != nil {
				logBasePath = fmt.Sprintf("s3://%s", filepath.Join(c.Bucket(), *prefix))
			} else {
				logBasePath = fmt.Sprintf("s3://%s", c.Bucket())
			}

		case expconf.AzureConfig:
			logBasePath = "azure://" + c.Container()

		case expconf.GCSConfig:
			prefix := c.Prefix()
			if prefix != nil {
				logBasePath = fmt.Sprintf("gs://%s", filepath.Join(c.Bucket(), *prefix))
			} else {
				logBasePath = fmt.Sprintf("gs://%s", c.Bucket())
			}

		default:
			return nil, status.Errorf(codes.Internal,
				"unknown storage backend for experiment: %T", c)
		}

		if len(exp.TrialIDs) == 0 {
			expDir := fmt.Sprintf("%s/%s/tensorboard/experiment/%d/",
				logBasePath, spec.Base.ClusterID, exp.ExperimentID)
			logDirs = append(logDirs, expDir)
			continue
		}

		for _, id := range exp.TrialIDs {
			trialDir := fmt.Sprintf("%s/%s/tensorboard/experiment/%d/trial/%d/",
				logBasePath, spec.Base.ClusterID, exp.ExperimentID, id)

			logDirs = append(logDirs, trialDir)
		}
	}

	// Get the most recent experiment config as raw json and add it to the container. This
	// is used for automatically configuring checkpoint storage, registry auth, etc.
	mostRecentExpID := exps[len(exps)-1].ExperimentID
	exp, err := db.ExperimentByID(ctx, int(mostRecentExpID))
	if err != nil {
		return nil, errors.Wrapf(err, "error loading experiment: %d", mostRecentExpID)
	}

	spec.Config.Entrypoint = append(
		[]string{tensorboardEntrypointFile, storageConfPath, strings.Join(logDirs, ",")},
		spec.Config.TensorBoardArgs...)

	spec.Base.ExtraEnvVars = uniqEnvVars

	if !model.UsingCustomImage(req) {
		spec.Config.Environment.Image = model.RuntimeItem{
			CPU:  exp.Config.Environment.Image().CPU(),
			CUDA: exp.Config.Environment.Image().CUDA(),
			ROCM: exp.Config.Environment.Image().ROCM(),
		}

		// Inherit ImagePullSecrets too, if we inherit the image.
		presentPod := spec.Config.Environment.PodSpec
		experimentPod := exp.Config.Environment.PodSpec()
		if experimentPod != nil && len(experimentPod.Spec.ImagePullSecrets) > 0 {
			if presentPod != nil {
				// Update the k8sV1.Pod with the experiment's pod's ImagePullSecrets.
				presentPod.Spec.ImagePullSecrets = append(
					presentPod.Spec.ImagePullSecrets, experimentPod.Spec.ImagePullSecrets...,
				)
			} else {
				// Construct a new k8sV1.Pod with just ImagePullSecrets set.
				spec.Config.Environment.PodSpec = &k8sV1.Pod{
					Spec: k8sV1.PodSpec{
						ImagePullSecrets: experimentPod.Spec.ImagePullSecrets,
					},
				}
			}
		}
	}
	// Prefer RegistryAuth already present over the one from inferred from the experiment.
	if spec.Config.Environment.RegistryAuth == nil {
		spec.Config.Environment.RegistryAuth = exp.Config.Environment.RegistryAuth()
	}

	var bindMounts []model.BindMount
	for _, uniqMount := range uniqMounts {
		bindMounts = append(bindMounts, uniqMount)
	}
	spec.Config.BindMounts = append(spec.Config.BindMounts, bindMounts...)

	confBytes, err := json.Marshal(exp.Config.CheckpointStorage)
	if err != nil {
		return nil, errors.Wrapf(err, "error marshaling checkpoint_storage")
	}

	spec.AdditionalFiles = archive.Archive{
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			tensorboardEntrypointFile,
			etc.MustStaticFile(etc.TensorboardEntryScriptResource), 0o700,
			tar.TypeReg,
		),
		spec.Base.AgentUserGroup.OwnedArchiveItem(storageConfPath, confBytes, 0o700, tar.TypeReg),
		spec.Base.AgentUserGroup.OwnedArchiveItem(
			taskReadyCheckLogs,
			etc.MustStaticFile(etc.TaskCheckReadyLogsResource),
			0o700,
			tar.TypeReg,
		),
	}

	if err = check.Validate(req.Config); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid TensorBoard config: %s", err.Error())
	}

	// Launch a TensorBoard actor.
	var tbID model.TaskID
	if err = a.ask(tensorboardsAddr, launchReq, &tbID); err != nil {
		return nil, err
	}

	var tb *tensorboardv1.Tensorboard
	if err = a.ask(tensorboardsAddr.Child(tbID), &tensorboardv1.Tensorboard{}, &tb); err != nil {
		return nil, err
	}

	return &apiv1.LaunchTensorboardResponse{
		Tensorboard: tb,
		Config:      protoutils.ToStruct(spec.Config),
		Warnings:    pkgCommand.LaunchWarningToProto(launchWarnings),
	}, err
}

type tensorboardConfig struct {
	Config       expconf.LegacyConfig
	ExperimentID int32
	TrialIDs     []int32
}

func (a *apiServer) getTensorBoardConfigsFromReq(
	ctx context.Context, db *db.PgDB, req *apiv1.LaunchTensorboardRequest,
) ([]*tensorboardConfig, error) {
	confByID := map[int32]*tensorboardConfig{}

	var err error
	var originalExpIDs []int32
	if req.Filters == nil {
		originalExpIDs = req.ExperimentIds
	} else {
		originalExpIDs, err = exputil.FilterToExperimentIds(ctx, req.Filters)
		if err != nil {
			return nil, err
		}
	}

	for _, expID := range originalExpIDs {
		if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(expID),
			exputil.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
			return nil, err
		}

		conf, err := db.LegacyExperimentConfigByID(int(expID))
		if err != nil {
			return nil, err
		}

		confByID[expID] = &tensorboardConfig{ExperimentID: expID, Config: conf}
	}

	for _, trialID := range req.TrialIds {
		if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(trialID),
			exputil.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
			return nil, err
		}

		expID, err := db.ExperimentIDByTrialID(int(trialID))
		if err != nil {
			return nil, err
		}

		conf, err := db.LegacyExperimentConfigByID(expID)
		if err != nil {
			return nil, err
		}

		if conf, ok := confByID[int32(expID)]; ok {
			conf.TrialIDs = append(conf.TrialIDs, trialID)
			continue
		}

		confByID[int32(expID)] = &tensorboardConfig{
			ExperimentID: int32(expID), Config: conf, TrialIDs: []int32{trialID},
		}
	}

	var expIDs []int
	for expID := range confByID {
		expIDs = append(expIDs, int(expID))
	}

	sort.Ints(expIDs)
	var configs []*tensorboardConfig
	for _, expID := range expIDs {
		configs = append(configs, confByID[int32(expID)])
	}

	return configs, nil
}
