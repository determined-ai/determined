package singularity

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

var singularityWrapperEntrypoint = path.Join(tasks.RunDir, tasks.SingularityEntrypointWrapperScript)

const (
	// TODO(DET-9111): Parameterize this by agent ID.
	agentTmp         = "/tmp/determined/agent"
	hostNetworking   = "host"
	bridgeNetworking = "bridge"
	envFileName      = "envfile"
	archivesName     = "archives"
)

// SingularityContainer captures the state of a container.
type SingularityContainer struct {
	PID     int            `json:"pid"`
	Req     cproto.RunSpec `json:"req"`
	TmpDir  string         `json:"tmp_dir"`
	Proc    *os.Process    `json:"-"`
	Started atomic.Bool    `json:"started"`
}

// SingularityClient implements ContainerRuntime.
type SingularityClient struct {
	log        *logrus.Entry
	opts       options.SingularityOptions
	mu         sync.Mutex
	wg         waitgroupx.Group
	containers map[cproto.ID]*SingularityContainer
}

// New returns a new singularity client, which launches and tracks containers.
func New(opts options.SingularityOptions) (*SingularityClient, error) {
	if err := os.RemoveAll(agentTmp); err != nil {
		return nil, fmt.Errorf("removing agent tmp from previous runs: %w", err)
	}

	if err := os.MkdirAll(agentTmp, 0o700); err != nil {
		return nil, fmt.Errorf("preparing agent tmp: %w", err)
	}

	return &SingularityClient{
		log:        logrus.WithField("compotent", "singularity"),
		opts:       opts,
		wg:         waitgroupx.WithContext(context.Background()),
		containers: make(map[cproto.ID]*SingularityContainer),
	}, nil
}

// Close the client, killing all running containers and removing our scratch space.
func (s *SingularityClient) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Since we launch procs with exec.CommandContext under s.wg's context, this cleans them up.
	s.wg.Close()

	if err := os.RemoveAll(agentTmp); err != nil {
		return fmt.Errorf("cleaning up agent tmp: %w", err)
	}
	return nil
}

// PullImage implements container.ContainerRuntime.
func (s *SingularityClient) PullImage(
	ctx context.Context,
	req docker.PullImage,
	p events.Publisher[docker.Event],
) (err error) {
	if err = p.Publish(ctx, docker.NewBeginStatsEvent(docker.ImagePullStatsKind)); err != nil {
		return err
	}
	defer func() {
		if err = p.Publish(ctx, docker.NewEndStatsEvent(docker.ImagePullStatsKind)); err != nil {
			s.log.WithError(err).Warn("did not send image pull done stats")
		}
	}()

	image := s.canonicalizeImage(req.Name)

	uri, err := url.Parse(image)
	if err != nil || uri.Scheme == "" {
		if err = p.Publish(ctx, docker.NewLogEvent(
			model.LogLevelInfo,
			fmt.Sprintf("image %s isn't a pullable URI; skipping pull", image),
		)); err != nil {
			return err
		}
		return nil
	}

	// TODO(DET-9078): Support registry auth. Investigate other auth mechanisms with singularity.
	args := []string{"pull"}
	if req.ForcePull {
		args = append(args, "--force")
	}
	args = append(args, image)

	if err = s.pprintSingularityCommand(ctx, args, p); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "singularity", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	// The return codes from `singularity pull` aren't super helpful in determining the error, so we
	// wrap the publisher and skim logs to see what happened as we ship them.
	ignoreErrorsSig := make(chan bool)
	checkIgnoreErrors := events.FuncPublisher[docker.Event](
		func(ctx context.Context, t docker.Event) error {
			if t.Log != nil && strings.Contains(t.Log.Message, "Image file already exists") {
				ignoreErrorsSig <- true
			}
			return p.Publish(ctx, t)
		},
	)
	s.wg.Go(func(ctx context.Context) { s.shipSingularityCmdLogs(ctx, stdout, stdcopy.Stdout, p) })
	s.wg.Go(func(ctx context.Context) {
		defer close(ignoreErrorsSig)
		s.shipSingularityCmdLogs(ctx, stderr, stdcopy.Stderr, checkIgnoreErrors)
	})

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("starting pull command: %w", err)
	}

	var ignoreErrors bool
	select {
	case ignoreErrors = <-ignoreErrorsSig:
	case <-ctx.Done():
		return ctx.Err()
	}

	if err = cmd.Wait(); err != nil && !ignoreErrors {
		return fmt.Errorf("pulling %s: %w", image, err)
	}
	return nil
}

// CreateContainer implements container.ContainerRuntime.
func (s *SingularityClient) CreateContainer(
	ctx context.Context,
	id cproto.ID,
	req cproto.RunSpec,
	p events.Publisher[docker.Event],
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.containers[id] = &SingularityContainer{Req: req}
	return id.String(), nil
}

// RunContainer implements container.ContainerRuntime.
// nolint: golint,maintidx // Both contexts can't both be first / TODO refactor.
func (s *SingularityClient) RunContainer(
	ctx context.Context,
	waitCtx context.Context,
	id string,
	p events.Publisher[docker.Event],
) (*docker.Container, error) {
	s.mu.Lock()
	var cont *SingularityContainer
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

	tmpdir, err := os.MkdirTemp(agentTmp, fmt.Sprintf("*-%s", id))
	if err != nil {
		return nil, fmt.Errorf("making tmp dir for archives: %w", err)
	}

	var args []string
	args = append(args, "run")
	args = append(args, "--writable-tmpfs")
	args = append(args, "--pwd", req.ContainerConfig.WorkingDir)

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

	var b64EnvVars []string
	for _, env := range req.ContainerConfig.Env {
		parts := strings.SplitN(env, "=", 2)

		var formattedEnv string
		switch len(parts) {
		case 0:
			continue // Must be empty envvar.
		case 1:
			formattedEnv = env
		case 2:
			// Don't even attempt to escape quotes, strconv.Quote doesn't work - singularity seems
			// to unescape it multiple times.
			if strings.Contains(parts[1], "\"") {
				b64EnvVars = append(b64EnvVars, parts[0])
				formattedEnv = fmt.Sprintf(
					"%s=\"%s\"",
					parts[0], base64.StdEncoding.EncodeToString([]byte(parts[1])),
				)
			} else {
				formattedEnv = fmt.Sprintf("%s=%s", parts[0], strconv.Quote(parts[1]))
			}
		}

		_, err = envFile.WriteString(formattedEnv + "\n")
		if err != nil {
			return nil, fmt.Errorf("writing to envfile: %w", err)
		}
	}

	_, err = envFile.WriteString(fmt.Sprintf(
		"DET_B64_ENCODED_ENVVARS=%s",
		strings.Join(b64EnvVars, ","),
	))
	if err != nil {
		return nil, fmt.Errorf("writing to envfile: %w", err)
	}

	switch {
	case req.HostConfig.NetworkMode == bridgeNetworking && s.opts.AllowNetworkCreation:
		// --net sets up a bridge network by default
		// (see https://apptainer.org/user-docs/3.0/networking.html#net)
		args = append(args, "--net")
		// Do the equivalent of Docker's PublishAllPorts = true
		for port := range req.ContainerConfig.ExposedPorts {
			p := port.Int()
			args = append(args, "--network-args", fmt.Sprintf("portmap=%d:%d/tcp", p, p))
		}
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
	default:
		return nil, fmt.Errorf("unsupported network mode %s", req.HostConfig.NetworkMode)
	}

	archivesPath := filepath.Join(tmpdir, archivesName)
	for _, a := range req.Archives {
		src := filepath.Join(archivesPath, a.Path)
		if wErr := archive.Write(src, a.Archive, func(level, log string) error {
			return p.Publish(ctx, docker.NewLogEvent(level, log))
		}); wErr != nil {
			return nil, fmt.Errorf("writing archive for %s: %w", a.Path, err)
		}
	}

	// Do not mount top level dirs that are likely to conflict inside of the container, since
	// these mounts do not overlay. Instead, mount their children.
	ignoredPathPrefixes := []string{"/", "/etc", "/opt", "/run", "/etc/ssh"}
	var mountPoints []string
	// This depends on walkdir walking in lexical order, which is documented.
	if wErr := filepath.WalkDir(archivesPath, func(src string, d fs.DirEntry, err error) error {
		p := strings.TrimPrefix(src, archivesPath)

		// If an existing mount point covers this path, nothing to add
		for _, m := range mountPoints {
			if strings.HasPrefix(p, m) {
				return nil
			}
		}

		dirPaths := filepath.SplitList(p)
		prefix := ""
		// Search to find the top-most unmounted path
		for i := 0; i < len(dirPaths); i++ {
			prefix = filepath.Join(prefix, dirPaths[i])

			s.log.Trace("Checking mountPoint prefix {}", prefix)
			if !slices.Contains(ignoredPathPrefixes, prefix) {
				s.log.Trace("Add mountPoint {}", prefix)
				mountPoints = append(mountPoints, prefix)
				return nil
			}
		}
		s.log.Warnf("could not determine where to mount %s", src)
		return nil
	}); wErr != nil {
		return nil, fmt.Errorf("determining mount points: %w", err)
	}
	for _, m := range mountPoints {
		args = append(args, "--bind", fmt.Sprintf("%s:%s", path.Join(archivesPath, m), m))
	}

	for _, m := range req.HostConfig.Mounts {
		// TODO(DET-9079): Investigate handling these options.
		if m.ReadOnly {
			if err = p.Publish(ctx, docker.NewLogEvent(model.LogLevelWarning, fmt.Sprintf(
				"mount %s:%s was requested as readonly but singularity does not support this; "+
					"will bind mount anyway, without it being readonly",
				m.Source, m.Target,
			))); err != nil {
				return nil, err
			}
		}
		if m.BindOptions != nil && m.BindOptions.Propagation != "rprivate" { // rprivate is default.
			if err = p.Publish(ctx, docker.NewLogEvent(model.LogLevelWarning, fmt.Sprintf(
				"mount %s:%s had propagation settings but singularity does not support this; "+
					"will bind mount anyway, without them",
				m.Source, m.Target,
			))); err != nil {
				return nil, err
			}
		}
		args = append(args, "--bind", fmt.Sprintf("%s:%s", m.Source, m.Target))
	}

	if shmsize := req.HostConfig.ShmSize; shmsize != 4294967296 { // 4294967296 is the default.
		if err = p.Publish(ctx, docker.NewLogEvent(model.LogLevelWarning, fmt.Sprintf(
			"shmsize was requested as %d but singularity does not support this; "+
				"we do not launch with `--contain`, so we inherit the configuration of the host",
			shmsize,
		))); err != nil {
			return nil, err
		}
	}

	// TODO(DET-9075): Un-dockerize the RunContainer API so we can know to pass `--rocm` without
	// regexing on devices.
	// TODO(DET-9080): Test this on ROCM devices.
	rocmDevice := regexp.MustCompile("/dev/dri/by-path/pci-.*-card")
	for _, d := range req.HostConfig.Devices {
		if rocmDevice.MatchString(d.PathOnHost) {
			args = append(args, "--rocm")
			break
		}
	}

	// Visible devices are set later by modifying the exec.Command's env.
	var cudaVisibleDevices []string
	for _, d := range cont.Req.HostConfig.DeviceRequests {
		if d.Driver == "nvidia" {
			cudaVisibleDevices = append(cudaVisibleDevices, d.DeviceIDs...)
		}
	}
	if len(cudaVisibleDevices) > 0 {
		// TODO(DET-9081): We need to move to --nvccli --nv, because --nv does not provide
		// sufficient isolation (e.g., nvidia-smi see all GPUs on the machine, not just ours).
		args = append(args, "--nv")
	}

	// TODO(DET-9079): It is unlikely we can handle this, but we should do better at documenting.
	if len(req.HostConfig.CapAdd) != 0 || len(req.HostConfig.CapDrop) != 0 {
		if err = p.Publish(ctx, docker.NewLogEvent(model.LogLevelWarning, fmt.Sprintf(
			"cap add or drop was requested but singularity does not support this; "+
				"will be ignored (cap_add: %+v, cap_drop: %+v)", req.HostConfig.CapAdd,
			req.HostConfig.CapDrop,
		))); err != nil {
			return nil, err
		}
	}

	image := s.canonicalizeImage(req.ContainerConfig.Image)
	args = append(args, image)
	args = append(args, singularityWrapperEntrypoint)
	args = append(args, req.ContainerConfig.Cmd...)

	if err = s.pprintSingularityCommand(ctx, args, p); err != nil {
		return nil, err
	}

	// #nosec G204 // We launch arbitrary user code as a service.
	cmd := exec.CommandContext(waitCtx, "singularity", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}
	s.wg.Go(func(ctx context.Context) { s.shipSingularityCmdLogs(ctx, stdout, stdcopy.Stdout, p) })
	s.wg.Go(func(ctx context.Context) { s.shipSingularityCmdLogs(ctx, stderr, stdcopy.Stderr, p) })

	cudaVisibleDevicesVar := strings.Join(cudaVisibleDevices, ",")
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("SINGULARITYENV_CUDA_VISIBLE_DEVICES=%s", cudaVisibleDevicesVar),
		fmt.Sprintf("APPTAINERENV_CUDA_VISIBLE_DEVICES=%s", cudaVisibleDevicesVar),
	)

	// HACK(singularity): without this, --nv doesn't work right. If the singularity run command
	// cannot find nvidia-smi, the --nv fails to make it available inside the container, e.g.,
	// env -i /usr/bin/singularity run --nv \\
	//   docker://determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-24586f0 nvidia-smi
	cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting singularity container: %w", err)
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
				Path:    singularityWrapperEntrypoint,
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

// ReattachContainer implements container.ContainerRuntime.
// TODO(DET-9082): Ensure orphaned processes are cleaned up on reattach.
func (s *SingularityClient) ReattachContainer(
	ctx context.Context,
	reattachID cproto.ID,
) (*docker.Container, *aproto.ExitCode, error) {
	return nil, nil, container.ErrMissing
}

// RemoveContainer implements container.ContainerRuntime.
func (s *SingularityClient) RemoveContainer(ctx context.Context, id string, force bool) error {
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
func (s *SingularityClient) SignalContainer(
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
func (s *SingularityClient) ListRunningContainers(
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

func (s *SingularityClient) waitOnContainer(
	id cproto.ID,
	cont *SingularityContainer,
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

var singularityLogLevel = regexp.MustCompile("(?P<level>INFO|WARN|ERROR|FATAL):    (?P<log>.*)")

func (s *SingularityClient) shipSingularityCmdLogs(
	ctx context.Context,
	r io.ReadCloser,
	stdtype stdcopy.StdType,
	p events.Publisher[docker.Event],
) {
	for scan := bufio.NewScanner(r); scan.Scan(); {
		line := scan.Text()
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		var level, log string
		if matches := singularityLogLevel.FindStringSubmatch(line); len(matches) == 3 {
			level, log = matches[1], matches[2]
		} else {
			level, log = model.LogLevelInfo, line
		}

		if err := p.Publish(ctx, docker.NewTypedLogEvent(level, log, stdtype)); err != nil {
			logrus.WithError(err).Trace("log stream terminated")
			return
		}
	}
	return
}

func (s *SingularityClient) pprintSingularityCommand(
	ctx context.Context,
	args []string,
	p events.Publisher[docker.Event],
) error {
	toPrint := "singularity"
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") { // print each arg on a new line
			toPrint += " \\\n"
			toPrint += "\t"
			toPrint += arg
		} else {
			toPrint += " "
			toPrint += arg
		}
	}

	s.log.Trace(toPrint)
	if err := p.Publish(ctx, docker.NewLogEvent(
		model.LogLevelDebug,
		toPrint,
	)); err != nil {
		return err
	}
	return nil
}

func (s *SingularityClient) canonicalizeImage(image string) string {
	url, err := url.Parse(image)
	isURIForm := err == nil
	isFSForm := path.IsAbs(image)
	if isFSForm || (isURIForm && url.Scheme != "") {
		return image
	}
	return fmt.Sprintf("docker://%s", image)
}
