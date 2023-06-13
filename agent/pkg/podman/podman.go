package podman

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/agent/pkg/cruntimes"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

var podmanWrapperEntrypoint = path.Join(tasks.RunDir, tasks.SingularityEntrypointWrapperScript)

const (
	hostNetworking   = "host"
	bridgeNetworking = "bridge"
	envFileName      = "envfile"
	archivesName     = "archives"
)

// PodmanContainer captures the state of a container.
type PodmanContainer struct {
	PID     int            `json:"pid"`
	Req     cproto.RunSpec `json:"req"`
	TmpDir  string         `json:"tmp_dir"`
	Proc    *os.Process    `json:"-"`
	Started atomic.Bool    `json:"started"`
}

// PodmanClient implements ContainerRuntime.
type PodmanClient struct {
	log        *logrus.Entry
	opts       options.PodmanOptions
	mu         sync.Mutex
	wg         waitgroupx.Group
	containers map[cproto.ID]*PodmanContainer
	agentTmp   string
	debug      bool
}

// New returns a new podman client, which launches and tracks containers.
func New(opts options.Options) (*PodmanClient, error) {
	agentTmp, err := cruntimes.BaseTempDirName(opts.AgentID)
	if err != nil {
		return nil, fmt.Errorf("unable to compose agentTmp directory path: %w", err)
	}

	if err := os.RemoveAll(agentTmp); err != nil {
		return nil, fmt.Errorf("removing agent tmp from previous runs: %w", err)
	}

	if err := os.MkdirAll(agentTmp, 0o700); err != nil {
		return nil, fmt.Errorf("preparing agent tmp: %w", err)
	}

	return &PodmanClient{
		log:        logrus.WithField("component", "podman"),
		opts:       opts.PodmanOptions,
		wg:         waitgroupx.WithContext(context.Background()),
		containers: make(map[cproto.ID]*PodmanContainer),
		agentTmp:   agentTmp,
		debug:      opts.Debug,
	}, nil
}

// Close the client, killing all running containers and removing our scratch space.
func (s *PodmanClient) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Since we launch procs with exec.CommandContext under s.wg's context, this cleans them up.
	s.wg.Close()

	if err := os.RemoveAll(s.agentTmp); err != nil {
		return fmt.Errorf("cleaning up agent tmp: %w", err)
	}
	return nil
}

func getPullCommand(req docker.PullImage, image string) (string, []string) {
	// C.f. singularity where if req.ForcePull is set then a 'pull --force' is done.
	// podman does not have this option, though it does have '--pull always' on the
	// run command.
	return "podman", []string{"pull", image}
}

// PullImage implements container.ContainerRuntime.
func (s *PodmanClient) PullImage(
	ctx context.Context,
	req docker.PullImage,
	p events.Publisher[docker.Event],
) (err error) {
	return cruntimes.PullImage(ctx, req, p, &s.wg, s.log, getPullCommand)
}

// CreateContainer implements container.ContainerRuntime.
func (s *PodmanClient) CreateContainer(
	ctx context.Context,
	id cproto.ID,
	req cproto.RunSpec,
	p events.Publisher[docker.Event],
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.containers[id] = &PodmanContainer{Req: req}
	return id.String(), nil
}

// RunContainer implements container.ContainerRuntime.
// nolint: golint,maintidx // Both contexts can't both be first / TODO refactor.
func (s *PodmanClient) RunContainer(
	ctx context.Context,
	waitCtx context.Context,
	id string,
	p events.Publisher[docker.Event],
) (*docker.Container, error) {
	s.mu.Lock()
	var cont *PodmanContainer
	for cID, rcont := range s.containers {
		if cproto.ID(id) != cID {
			continue
		}
		cont = rcont
		break
	}
	s.mu.Unlock()

	if cont == nil {
		return nil, container.ErrMissing
	}
	req := cont.Req

	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("checking user: %w", err)
	}

	uidgid := fmt.Sprintf("%s:%s", u.Uid, u.Gid)
	if req.ContainerConfig.User != uidgid {
		return nil, fmt.Errorf(
			"agent running as %s cannot launch as user %s",
			uidgid, req.ContainerConfig.User,
		)
	}

	tmpdir, err := os.MkdirTemp(s.agentTmp, fmt.Sprintf("*-%s", id))
	if err != nil {
		return nil, fmt.Errorf("making tmp dir for archives: %w", err)
	}

	var args []string
	args = append(args, "run")
	args = append(args, "--rm")
	args = append(args, "--workdir", req.ContainerConfig.WorkingDir)

	// Env. variables. c.f. launcher PodmanOverSlurm.java
	args = append(args, "--env", "SLURM_*")
	args = append(args, "--env", "CUDA_VISIBLE_DEVICES")
	args = append(args, "--env", "NVIDIA_VISIBLE_DEVICES")
	args = append(args, "--env", "ROCR_VISIBLE_DEVICES")
	args = append(args, "--env", "HIP_VISIBLE_DEVICES")
	if s.debug {
		args = append(args, "--env", "DET_DEBUG=1")
	}
	envFilePath := path.Join(tmpdir, envFileName)
	envFile, err := os.OpenFile(
		envFilePath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o600,
	) // #nosec G304 // We made this filepath, and it is randomized.
	if err != nil {
		return nil, fmt.Errorf("creating envfile %s: %w", envFilePath, err)
	}
	args = append(args, "--env-file", envFilePath)

	req.ContainerConfig.Env = append(req.ContainerConfig.Env, "DET_NO_FLUENT=true")
	req.ContainerConfig.Env = append(req.ContainerConfig.Env, "DET_SHIPPER_EMIT_STDOUT_LOGS=False")

	for _, env := range req.ContainerConfig.Env {
		_, err = envFile.WriteString(env + "\n")
		if err != nil {
			return nil, fmt.Errorf("writing to envfile: %w", err)
		}
	}
	if err = envFile.Close(); err != nil {
		return nil, fmt.Errorf("closing envfile: %w", err)
	}

	switch {
	case req.HostConfig.NetworkMode == bridgeNetworking && s.opts.AllowNetworkCreation:
		// ?? publish ports only for bridgeNetworking
		for port := range req.ContainerConfig.ExposedPorts {
			p := port.Int()
			args = append(args, "-p", fmt.Sprintf("%d:%d/tcp", p, p))
		}
		args = append(args, "--network=bridge")
	case req.HostConfig.NetworkMode == bridgeNetworking:
		if err = p.Publish(ctx, docker.NewLogEvent(
			model.LogLevelDebug,
			"container requested network virtualization, but network creation isn't allowed; "+
				"overriding to host networking",
		)); err != nil {
			return nil, err
		}
		req.HostConfig.NetworkMode = hostNetworking
		fallthrough
	case req.HostConfig.NetworkMode == hostNetworking:
		args = append(args, "--network=host")
	default:
		return nil, fmt.Errorf("unsupported network mode %s", req.HostConfig.NetworkMode)
	}

	archivesPath := filepath.Join(tmpdir, archivesName)
	mountPoints, wErr := cruntimes.ArchiveMountPoints(ctx, req, p, archivesPath, s.log)
	if wErr != nil {
		return nil, fmt.Errorf("determining mount points: %w", err)
	}
	for _, m := range mountPoints {
		args = append(args, "--volume", fmt.Sprintf("%s:%s", path.Join(archivesPath, m), m))
	}

	for _, m := range req.HostConfig.Mounts {
		args = hostMountsToPodmanArgs(m, args)
	}

	// from master task_container_defaults.shm_size_bytes
	if shmsize := req.HostConfig.ShmSize; shmsize != 4294967296 { // 4294967296 is the default.
		args = append(args, "--shm-size", fmt.Sprintf("%d", shmsize))
	}

	args = capabilitiesToPodmanArgs(req, args)

	image := cruntimes.CanonicalizeImage(req.ContainerConfig.Image)
	args = append(args, image)
	args = append(args, podmanWrapperEntrypoint)
	args = append(args, req.ContainerConfig.Cmd...)

	if err = s.pprintPodmanCommand(ctx, args, p); err != nil {
		return nil, err
	}

	// #nosec G204 // We launch arbitrary user code as a service.
	cmd := exec.CommandContext(waitCtx, "podman", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}
	s.wg.Go(func(ctx context.Context) { s.shipPodmanCmdLogs(ctx, stdout, stdcopy.Stdout, p) })
	s.wg.Go(func(ctx context.Context) { s.shipPodmanCmdLogs(ctx, stderr, stdcopy.Stderr, p) })

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting podman container: %w", err)
	}

	cont.PID = cmd.Process.Pid
	cont.Proc = cmd.Process
	cont.TmpDir = tmpdir
	cont.Started.Store(true)
	at := time.Now().String()
	s.log.Infof("started container %s with pid %d", id, cont.PID)

	return &docker.Container{
		ContainerInfo: types.ContainerJSON{
			ContainerJSONBase: &types.ContainerJSONBase{
				ID:      strconv.Itoa(cont.Proc.Pid),
				Created: at,
				Path:    podmanWrapperEntrypoint,
				Args:    req.ContainerConfig.Cmd,
				State: &types.ContainerState{
					Status:    "running",
					Running:   true,
					Pid:       cont.Proc.Pid,
					StartedAt: at,
				},
				Image: image,
				HostConfig: &dcontainer.HostConfig{
					NetworkMode: req.HostConfig.NetworkMode,
				},
			},
			Config: &dcontainer.Config{
				ExposedPorts: req.ContainerConfig.ExposedPorts,
			},
		},
		ContainerWaiter: s.waitOnContainer(cproto.ID(id), cont, p),
	}, nil
}

func capabilitiesToPodmanArgs(req cproto.RunSpec, args []string) []string {
	for _, cap := range req.HostConfig.CapAdd {
		args = append(args, "--cap-add", cap)
	}
	for _, cap := range req.HostConfig.CapDrop {
		args = append(args, "--cap-drop", cap)
	}
	return args
}

func hostMountsToPodmanArgs(m mount.Mount, args []string) []string {
	var mountOptions []string
	if m.ReadOnly {
		mountOptions = append(mountOptions, "ro")
	}
	if m.BindOptions != nil && string(m.BindOptions.Propagation) != "" {
		mountOptions = append(mountOptions, string(m.BindOptions.Propagation))
	}
	var options string
	if len(mountOptions) > 0 {
		options = fmt.Sprintf(":%s", strings.Join(mountOptions, ","))
	}
	return append(args, "--volume", fmt.Sprintf("%s:%s%s", m.Source, m.Target, options))
}

// ReattachContainer implements container.ContainerRuntime.
// TODO(DET-9082): Ensure orphaned processes are cleaned up on reattach.
func (s *PodmanClient) ReattachContainer(
	ctx context.Context,
	reattachID cproto.ID,
) (*docker.Container, *aproto.ExitCode, error) {
	return nil, nil, container.ErrMissing
}

// RemoveContainer implements container.ContainerRuntime.
func (s *PodmanClient) RemoveContainer(ctx context.Context, id string, force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cont, ok := s.containers[cproto.ID(id)]
	if !ok {
		return container.ErrMissing
	}

	if cont.Started.Load() {
		return cont.Proc.Kill()
	}
	return fmt.Errorf("cannot kill container %s that is not started", id)
}

// SignalContainer implements container.ContainerRuntime.
func (s *PodmanClient) SignalContainer(
	ctx context.Context,
	id string,
	sig syscall.Signal,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cont, ok := s.containers[cproto.ID(id)]
	if !ok {
		return container.ErrMissing
	}

	if cont.Started.Load() {
		return cont.Proc.Signal(sig)
	}
	return fmt.Errorf("cannot signal container %s with %s that is not started", id, sig)
}

// ListRunningContainers implements container.ContainerRuntime.
func (s *PodmanClient) ListRunningContainers(
	ctx context.Context,
	fs filters.Args,
) (map[cproto.ID]types.Container, error) {
	resp := make(map[cproto.ID]types.Container)

	s.mu.Lock()
	defer s.mu.Unlock()
	for id, cont := range s.containers {
		resp[id] = types.Container{
			ID:     string(id),
			Labels: cont.Req.ContainerConfig.Labels,
		}
	}
	return resp, nil
}

func (s *PodmanClient) waitOnContainer(
	id cproto.ID,
	cont *PodmanContainer,
	p events.Publisher[docker.Event],
) docker.ContainerWaiter {
	wchan := make(chan dcontainer.ContainerWaitOKBody, 1)
	errchan := make(chan error)
	s.wg.Go(func(ctx context.Context) {
		defer close(wchan)
		defer close(errchan)

		var body dcontainer.ContainerWaitOKBody
		switch state, err := cont.Proc.Wait(); {
		case ctx.Err() != nil && err == nil && state.ExitCode() == -1:
			s.log.Trace("detached from container process")
			return
		case err != nil:
			s.log.Tracef("proc %d for container %s exited: %s", cont.PID, id, err)
			body.Error = &dcontainer.ContainerWaitOKBodyError{Message: err.Error()}
		default:
			s.log.Tracef("proc %d for container %s exited with %d", cont.PID, id, state.ExitCode())
			body.StatusCode = int64(state.ExitCode())
		}

		select {
		case wchan <- body:
		case <-ctx.Done():
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()
		s.log.Tracef("forgetting completed container: %s", id)
		delete(s.containers, id)

		// Defer file cleanup until restart if debug logging is enabled.
		if s.log.Logger.Level <= logrus.DebugLevel {
			if err := p.Publish(ctx, docker.NewLogEvent(
				model.LogLevelDebug,
				fmt.Sprintf("leaving tmpdir %s for inspection", cont.TmpDir),
			)); err != nil {
				return
			}
		} else {
			if err := os.RemoveAll(cont.TmpDir); err != nil {
				if err = p.Publish(ctx, docker.NewLogEvent(
					model.LogLevelWarning,
					fmt.Sprintf("failed to cleanup tmpdir (ephemeral mounts, etc): %s", err),
				)); err != nil {
					logrus.WithError(err).Error("publishing cleanup failure warning")
					return
				}
			}
		}
	})
	return docker.ContainerWaiter{Waiter: wchan, Errs: errchan}
}

func (s *PodmanClient) shipPodmanCmdLogs(
	ctx context.Context,
	r io.ReadCloser,
	stdtype stdcopy.StdType,
	p events.Publisher[docker.Event],
) {
	cruntimes.ShipContainerCommandLogs(ctx, r, stdtype, p)
}

func (s *PodmanClient) pprintPodmanCommand(
	ctx context.Context,
	args []string,
	p events.Publisher[docker.Event],
) error {
	return cruntimes.PprintCommand(ctx, "podman", args, p, s.log)
}
