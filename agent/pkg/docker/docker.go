package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"syscall"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

const (
	// ContainerTypeLabel describes container type.
	ContainerTypeLabel = "ai.determined.container.type"
	// ContainerTypeValue is the corresponding value for 'normal tasks'.
	ContainerTypeValue = "task-container"
	// ContainerVersionLabel describes the container version.
	ContainerVersionLabel = "ai.determined.container.version"
	// ContainerVersionValue is the current (and only) container version.
	ContainerVersionValue = "0"
	// ContainerIDLabel gives the Determined container ID (cproto.ID).
	ContainerIDLabel = "ai.determined.container.id"
	// ContainerDevicesLabel describes the devices allocated to the container.
	ContainerDevicesLabel = "ai.determined.container.devices"
	// ContainerDescriptionLabel describes the container.
	ContainerDescriptionLabel = "ai.determined.container.description"
	// AgentLabel gives the agent the container is managed by.
	AgentLabel = "ai.determined.container.agent"
	// ClusterLabel gives the cluster the container is managed by.
	ClusterLabel = "ai.determined.container.cluster"
	// MasterLabel gives the master the container is managed by.
	MasterLabel = "ai.determined.container.master"

	// ImagePullStatsKind describes the IMAGEPULL event.
	ImagePullStatsKind = "IMAGEPULL"
)

// Docker error strings returned by the Docker API.
var (
	NoSuchContainer   = "No such container"
	RemovalInProgress = regexp.MustCompile(`removal of container ([a-f0-9]+) is already in progress`)
)

var ForceRemoveOpts = types.ContainerRemoveOptions{Force: true}

type (
	// ContainerWaiter contains channels to wait on the termination of a running container.
	// Results on the Waiter channel indicate changes in container state, while results on the
	// Errs channel indicate failures to watch for updates.
	ContainerWaiter struct {
		Waiter <-chan dcontainer.ContainerWaitOKBody
		Errs   <-chan error
	}
	// Container contains details about a running container and waiters to await its termination.
	Container struct {
		ContainerInfo   types.ContainerJSON
		ContainerWaiter ContainerWaiter
	}
)

// Client wraps the Docker client, augmenting it with a few higher level convenience APIs.
type Client struct {
	// Configuration details. Set during initialization, never modified afterwards.
	credentialStores map[string]*credentialStore
	authConfigs      map[string]types.AuthConfig

	// System dependencies. Also set during initialization, never modified afterwards.
	cl  *client.Client
	log *logrus.Entry
}

// NewClient populates credentials from the Docker Daemon config and returns a new Client that uses
// them.
func NewClient(cl *client.Client) *Client {
	d := &Client{
		cl:  cl,
		log: logrus.WithField("component", "docker-client"),
	}

	stores, auths, err := processDockerConfig()
	if err != nil {
		d.log.Infof("couldn't process ~/.docker/config.json %v", err)
	}
	if len(stores) == 0 {
		d.log.Info("can't find any docker credential stores, continuing without them")
	}
	if len(auths) == 0 {
		d.log.Info("can't find any auths in ~/.docker/config.json, continuing without them")
	}
	d.credentialStores, d.authConfigs = stores, auths

	return d
}

// Inner returns the underlying Docker client, to be used sparingly.
// TODO(DET-8628): Consolidate around usage of the wrapper client, remove this.
func (d *Client) Inner() *client.Client {
	return d.cl
}

// ReattachContainer looks for a single running or terminated container that matches the given
// filters and returns whether it can be reattached or has terminated.
func (d *Client) ReattachContainer(
	ctx context.Context,
	id cproto.ID,
) (*Container, *aproto.ExitCode, error) {
	filter := LabelFilter(ContainerIDLabel, id.String())
	containers, err := d.cl.ContainerList(ctx, types.ContainerListOptions{Filters: filter})
	if err != nil {
		return nil, nil, fmt.Errorf("while reattaching container: %w", err)
	}

	if len(containers) > 1 {
		return nil, nil, errors.New("reattach filters matched more than one container")
	}

	for _, cont := range containers {
		// Subscribe to termination notifications first, to not miss immediate exits.
		waiter, errs := d.cl.ContainerWait(ctx, cont.ID, dcontainer.WaitConditionNextExit)

		// Restore containerInfo.
		containerInfo, err := d.cl.ContainerInspect(ctx, cont.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("inspecting reattached container: %w", err)
		}

		// Check if container has exited while we were trying to reattach it.
		if !containerInfo.State.Running {
			return nil, ptrs.Ptr(aproto.ExitCode(containerInfo.State.ExitCode)), nil
		}
		return &Container{ //nolint: staticcheck // We mean to terminate this loop.
			ContainerInfo: containerInfo,
			ContainerWaiter: ContainerWaiter{
				Waiter: waiter,
				Errs:   errs,
			},
		}, nil, nil
	}
	return nil, nil, nil
}

// PullImage describes a request to pull an image.
type PullImage struct {
	Name      string
	ForcePull bool
	Registry  *types.AuthConfig
}

// PullImage pulls an image according to the given request and credentials initialized at client
// creation from the Daemon config or credentials helpers configured there. It takes a
// caller-provided channel on which docker events are sent. Slow receivers will block the call.
func (d *Client) PullImage(ctx context.Context, req PullImage, p events.Publisher[Event]) error {
	ref, err := reference.ParseNormalizedNamed(req.Name)
	if err != nil {
		return fmt.Errorf("error parsing image name %s: %w", req.Name, err)
	}
	ref = reference.TagNameOnly(ref)

	switch _, _, err = d.cl.ImageInspectWithRaw(ctx, ref.String()); {
	case req.ForcePull:
		if err != nil {
			break
		}

		if err = p.Publish(ctx, NewLogEvent(model.LogLevelInfo, fmt.Sprintf(
			"image present, but force_pull_image is set; checking for updates: %s",
			ref.String(),
		))); err != nil {
			return err
		}
	case client.IsErrNotFound(err):
		if err = p.Publish(ctx, NewLogEvent(model.LogLevelInfo, fmt.Sprintf(
			"image not found, pulling image: %s", ref.String(),
		))); err != nil {
			return err
		}
	case err != nil:
		return fmt.Errorf("error checking if image exists %s: %w", ref.String(), err)
	default:
		if err = p.Publish(ctx, NewLogEvent(model.LogLevelInfo, fmt.Sprintf(
			"image already found, skipping pull phase: %s",
			ref.String(),
		))); err != nil {
			return err
		}
		return nil
	}

	if err = p.Publish(ctx, NewBeginStatsEvent(ImagePullStatsKind)); err != nil {
		return err
	}
	defer func() {
		if scErr := p.Publish(ctx, NewEndStatsEvent(ImagePullStatsKind)); scErr != nil {
			d.log.WithError(scErr).Warn("did not send image pull done stats")
		}
	}()

	auth, err := d.getDockerAuths(ctx, ref, req.Registry, p)
	if err != nil {
		return fmt.Errorf("could not get docker authentication: %w", err)
	}

	authString, err := registryToString(*auth)
	if err != nil {
		return fmt.Errorf("error encoding docker credentials: %w", err)
	}

	logs, err := d.cl.ImagePull(ctx, ref.String(), types.ImagePullOptions{
		All:          false,
		RegistryAuth: authString,
	})
	if err != nil {
		return errors.Wrapf(err, "error pulling image: %s", ref.String())
	}
	defer func() {
		if err = logs.Close(); err != nil {
			d.log.WithError(err).Error("error closing log stream")
		}
	}()

	if err = d.sendPullLogs(ctx, logs, p); err != nil {
		return fmt.Errorf("error processing pull log stream: %w", err)
	}
	return nil
}

// CreateContainer creates a container according to the given spec, returning a docker container ID
// to start it. It takes a caller-provided channel on which docker events are sent. Slow receivers
// will block the call.
func (d *Client) CreateContainer(
	ctx context.Context,
	id cproto.ID,
	req cproto.RunSpec,
	p events.Publisher[Event],
) (string, error) {
	response, err := d.cl.ContainerCreate(
		ctx, &req.ContainerConfig, &req.HostConfig, &req.NetworkingConfig, nil, "")
	if err != nil {
		return "", fmt.Errorf("creating container: %w", err)
	}
	dockerID := response.ID
	for _, w := range response.Warnings {
		if err = p.Publish(ctx, NewLogEvent(model.LogLevelWarning, fmt.Sprintf(
			"warning when creating container: %s", w,
		))); err != nil {
			return "", err
		}
	}

	for _, copyArx := range req.Archives {
		if err = p.Publish(ctx, NewLogEvent(model.LogLevelInfo, fmt.Sprintf(
			"copying files to container: %s", copyArx.Path,
		))); err != nil {
			return "", err
		}

		files, cErr := archive.ToIOReader(copyArx.Archive)
		if cErr != nil {
			return "", fmt.Errorf("converting RunSpec Archive files to io.Reader: %w", cErr)
		}
		if err = d.cl.CopyToContainer(
			ctx,
			dockerID,
			copyArx.Path,
			files,
			copyArx.CopyOptions,
		); err != nil {
			return "", fmt.Errorf("copying files to container: %w", err)
		}
	}
	return dockerID, nil
}

// RunContainer runs a container by docker container ID. It takes a caller-provided channel on which
// docker events are sent. Slow receivers will block the call. RunContainer takes two contexts: one
// to govern cancellation of running the container, and another to govern the lifetime of the waiter
// returned.
// nolint: golint // Both contexts can't both be first.
func (d *Client) RunContainer(
	ctx context.Context,
	waitCtx context.Context,
	id string,
	p events.Publisher[Event],
) (*Container, error) {
	// Wait before start to not miss immediate exits.
	waiter, errs := d.cl.ContainerWait(waitCtx, id, dcontainer.WaitConditionNextExit)

	if err := d.cl.ContainerStart(ctx, id, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("starting container: %w", err)
	}

	containerInfo, err := d.cl.ContainerInspect(ctx, id)
	if err != nil {
		if cErr := d.RemoveContainer(ctx, id, true); cErr != nil {
			d.log.
				WithError(cErr).
				WithField("dcl-container-id", id).
				Errorf("removing container %s after inspect failure", id)
		}
		return nil, fmt.Errorf("inspecting, container may be orphaned: %w", err)
	}
	return &Container{
		ContainerInfo: containerInfo,
		ContainerWaiter: ContainerWaiter{
			Waiter: waiter,
			Errs:   errs,
		},
	}, nil
}

// SignalContainer signals the container, by docker container ID, with the requested signal,
// returning an error if the Docker daemon is unable to process our request.
func (d *Client) SignalContainer(ctx context.Context, id string, sig syscall.Signal) error {
	return d.cl.ContainerKill(ctx, id, unix.SignalName(sig))
}

// RemoveContainer removes a Docker container by ID.
func (d *Client) RemoveContainer(ctx context.Context, id string, force bool) error {
	return d.cl.ContainerRemove(ctx, id, types.ContainerRemoveOptions{Force: force})
}

// ListRunningContainers lists running Docker containers satisfying the given filters.
func (d *Client) ListRunningContainers(ctx context.Context, fs filters.Args) (
	map[cproto.ID]types.Container, error,
) {
	// List "our" running containers, based on `dockerAgentLabel`.
	// This doesn't include Fluent Bit or containers spawned by other agents.
	containers, err := d.cl.ContainerList(ctx, types.ContainerListOptions{All: false, Filters: fs})
	if err != nil {
		return nil, err
	}

	result := make(map[cproto.ID]types.Container, len(containers))
	for _, cont := range containers {
		containerID, ok := cont.Labels[ContainerIDLabel]
		if ok {
			result[cproto.ID(containerID)] = cont
		} else {
			d.log.Warnf("container %v has agent label but no container ID", cont.ID)
		}
	}
	return result, nil
}

// LabelFilter is a convenience that takes a key and value and returns a docker label filter.
func LabelFilter(key, val string) filters.Args {
	return filters.NewArgs(filters.Arg("label", key+"="+val))
}

func (d *Client) getDockerAuths(
	ctx context.Context,
	image reference.Named,
	userRegistry *types.AuthConfig,
	p events.Publisher[Event],
) (*types.AuthConfig, error) {
	imageDomain := reference.Domain(image)
	// Try user submitted registry auth config.
	if userRegistry != nil {
		// TODO: remove didNotPassServerAddress when it becomes required.
		didNotPassServerAddress := userRegistry.ServerAddress == ""
		if didNotPassServerAddress {
			if err := p.Publish(ctx, NewLogEvent(model.LogLevelWarning,
				"setting registry_auth without registry_auth.serveraddress is deprecated "+
					"and the latter will soon be required")); err != nil {
				return nil, err
			}
		}

		registryDomain := registry.ConvertToHostname(userRegistry.ServerAddress)
		if registryDomain == imageDomain || didNotPassServerAddress {
			return userRegistry, nil
		}
		if err := p.Publish(ctx, NewLogEvent(model.LogLevelWarning, fmt.Sprintf(
			"not using expconfig registry_auth since expconf "+
				"registry_auth.serverAddress %s did not match the image serverAddress %s",
			registryDomain, imageDomain,
		))); err != nil {
			return nil, err
		}
	}

	// Try using credential stores specified in ~/.docker/config.json.
	if store, ok := d.credentialStores[imageDomain]; ok {
		creds, err := store.get()
		if err != nil {
			return nil, fmt.Errorf("unable to get credentials from helper: %w", err)
		}

		if err := p.Publish(ctx, NewLogEvent(model.LogLevelInfo, fmt.Sprintf(
			"domain '%s' found in 'credHelpers' config, using credentials helper",
			imageDomain,
		))); err != nil {
			return nil, err
		}

		return &creds, nil
	}

	// Finally try using auths section of user's ~/.docker/config.json.
	index, err := registry.ParseSearchIndexInfo(image.String())
	if err != nil {
		return nil, fmt.Errorf("error invalid docker repo name: %w", err)
	}
	reg := registry.ResolveAuthConfig(d.authConfigs, index)
	if reg == (types.AuthConfig{}) {
		return &reg, nil
	}

	if err := p.Publish(ctx, NewLogEvent(model.LogLevelInfo, fmt.Sprintf(
		"domain '%s' found in 'auths' ~/.docker/config.json", imageDomain,
	))); err != nil {
		return nil, err
	}
	return &reg, nil
}

// registryToString converts the Registry struct to a base64 encoding for json strings.
func registryToString(reg types.AuthConfig) (string, error) {
	// Docker stores the username and password in an auth section types.AuthConfig
	// formatted as user:pass then base64ed. This is not documented clearly.
	// https://github.com/docker/cli/blob/master/cli/config/configfile/file.go#L76
	if reg.Auth != "" {
		bytes, err := base64.StdEncoding.DecodeString(reg.Auth)
		if err != nil {
			return "", err
		}
		userAndPass := strings.SplitN(string(bytes), ":", 2)
		if len(userAndPass) != 2 {
			return "", errors.Errorf("auth field of docker authConfig must be base64ed user:pass")
		}
		reg.Username, reg.Password = userAndPass[0], userAndPass[1]
		reg.Auth = ""
	}
	bs, err := json.Marshal(reg)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bs), nil
}

func (d *Client) sendPullLogs(ctx context.Context, r io.Reader, p events.Publisher[Event]) error {
	plf := pullLogFormatter{Known: map[string]*pullInfo{}}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log := jsonmessage.JSONMessage{}
		if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
			return fmt.Errorf("error parsing log message: %#v: %w", log, err)
		}

		logMsg := plf.Update(log)
		if logMsg == nil {
			continue
		}

		if err := p.Publish(ctx, NewLogEvent(model.LogLevelInfo, *logMsg)); err != nil {
			return err
		}
	}
	// Always print the complete progress bar, regardless of the backoff time.
	finalLogMsg := plf.RenderProgress()
	if err := p.Publish(ctx, NewLogEvent(model.LogLevelInfo, finalLogMsg)); err != nil {
		return err
	}
	return scanner.Err()
}
