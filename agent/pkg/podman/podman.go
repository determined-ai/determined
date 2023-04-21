package podman

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/stdcopy"
	"golang.org/x/exp/slices"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/agent/pkg/singularity"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

var podmanWrapperEntrypoint = path.Join(tasks.RunDir, tasks.SingularityEntrypointWrapperScript)

const (
	// TODO(DET-9111): Parameterize this by agent ID.
	agentTmp         = "/tmp/determined/agent"
	hostNetworking   = "host"
	bridgeNetworking = "bridge"
	envFileName      = "envfile"
	archivesName     = "archives"
)

// PodmanContainer captures the state of a container.
type PodmanContainer struct {
	*singularity.SingularityContainer
}

// PodmanClient implements ContainerRuntime.
type PodmanClient struct {
	*singularity.SingularityClient
}

// New returns a new podman client, which launches and tracks containers.
func New(opts options.PodmanOptions) (*PodmanClient, error) {
	client, err := singularity.New(options.SingularityOptions{})
	if err != nil {
		return nil, err
	}
	return &PodmanClient{
		SingularityClient: client,
	}, nil
}

// getPullCommand returns the command and arguments to perform the container image pull.
func (s *PodmanClient) getPullCommand(req docker.PullImage, image string) (string, []string) {
	args := []string{"pull"}
	// if req.ForcePull {
	// 	args = append(args, "--force") // Use 'podman image rm'?
	// }
	args = append(args, image)
	return "podman", args
}

// PullImage implements container.ContainerRuntime.
func (s *PodmanClient) PullImage(
	ctx context.Context,
	req docker.PullImage,
	p events.Publisher[docker.Event],
) error {
	return s.PullImageCommon(ctx, req, p, s.getPullCommand)
}

// RunContainer implements container.ContainerRuntime.
// nolint: golint,maintidx // Both contexts can't both be first / TODO refactor.
func (s *PodmanClient) RunContainer(
	ctx context.Context,
	waitCtx context.Context,
	id string,
	p events.Publisher[docker.Event],
) (*docker.Container, error) {
	cont := s.FindContainer(id)
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
	args = append(args, "--rm")
	// args = append(args, "--writable-tmpfs")
	args = append(args, "--workdir", req.ContainerConfig.WorkingDir)

	// Env. variables. c.f. launcher PodmanOverSlurm.java
	args = append(args, "--env", "SLURM_*")
	args = append(args, "--env", "CUDA_VISIBLE_DEVICES")
	args = append(args, "--env", "NVIDIA_VISIBLE_DEVICES")
	args = append(args, "--env", "ROCR_VISIBLE_DEVICES")
	args = append(args, "--env", "HIP_VISIBLE_DEVICES")

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

	switch {
	case req.HostConfig.NetworkMode == bridgeNetworking && s.Opts.AllowNetworkCreation:
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

			s.Log.Trace("Checking mountPoint prefix {}", prefix)
			if !slices.Contains(ignoredPathPrefixes, prefix) {
				s.Log.Trace("Add mountPoint {}", prefix)
				mountPoints = append(mountPoints, prefix)
				return nil
			}
		}
		s.Log.Warnf("could not determine where to mount %s", src)
		return nil
	}); wErr != nil {
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

	// TODO(DET-9075): Un-dockerize the RunContainer API so we can know to pass `--rocm` without
	// regexing on devices.
	// TODO(DET-9080): Test this on ROCM devices.
	// rocmDevice := regexp.MustCompile("/dev/dri/by-path/pci-.*-card")
	// for _, d := range req.HostConfig.Devices {
	// 	if rocmDevice.MatchString(d.PathOnHost) {
	// 		args = append(args, "--rocm")
	// 		break
	// 	}
	// }

	// Visible devices are set later by modifying the exec.Command's env.
	// var cudaVisibleDevices []string
	// for _, d := range cont.Req.HostConfig.DeviceRequests {
	// 	if d.Driver == "nvidia" {
	// 		cudaVisibleDevices = append(cudaVisibleDevices, d.DeviceIDs...)
	// 	}
	// }
	// if len(cudaVisibleDevices) > 0 {
	// 	// TODO(DET-9081): We need to move to --nvccli --nv, because --nv does not provide
	// 	// sufficient isolation (e.g., nvidia-smi see all GPUs on the machine, not just ours).
	// 	args = append(args, "--nv")
	// }

	args = capabilitiesToPodmanArgs(req, args)

	image := s.CanonicalizeImage(req.ContainerConfig.Image)
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
	s.Wg.Go(func(ctx context.Context) { s.ShipContainerCmdLogs(ctx, stdout, stdcopy.Stdout, p) })
	s.Wg.Go(func(ctx context.Context) { s.ShipContainerCmdLogs(ctx, stderr, stdcopy.Stderr, p) })

	// cudaVisibleDevicesVar := strings.Join(cudaVisibleDevices, ",")
	// cmd.Env = append(cmd.Env,
	// 	fmt.Sprintf("SINGULARITYENV_CUDA_VISIBLE_DEVICES=%s", cudaVisibleDevicesVar),
	// 	fmt.Sprintf("APPTAINERENV_CUDA_VISIBLE_DEVICES=%s", cudaVisibleDevicesVar),
	// )

	// HACK(singularity): without this, --nv doesn't work right. If the singularity run command
	// cannot find nvidia-smi, the --nv fails to make it available inside the container, e.g.,
	// env -i /usr/bin/singularity run --nv \\
	//   docker://determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpu-24586f0 nvidia-smi
	// cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting podman container: %w", err)
	}

	cont.PID = cmd.Process.Pid
	cont.Proc = cmd.Process
	cont.TmpDir = tmpdir
	cont.Started.Store(true)
	at := time.Now().String()
	s.Log.Infof("started container %s with pid %d", id, cont.PID)

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
		ContainerWaiter: s.WaitOnContainer(cproto.ID(id), cont, p),
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

func (s *PodmanClient) pprintPodmanCommand(
	ctx context.Context,
	args []string,
	p events.Publisher[docker.Event],
) error {
	return s.PprintCommand(ctx, "podman", args, p)
}
