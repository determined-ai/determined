package internal

import (
	"archive/tar"
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
)

// The names of environment variables whose values should be included in log entries that Docker or
// the agent sends to the Fluent Bit logger.
var fluentEnvVarNames = []string{"DET_TRIAL_ID", "DET_CONTAINER_ID"}

const fluentListenPort = 24224

// fluentConfig computes the command-line arguments and extra files needed to start Fluent Bit with
// an appropriate configuration.
func fluentConfig(opts Options) ([]string, archive.Archive, error) {
	const baseDir = "/run/determined/fluent/"
	const luaPath = baseDir + "tonumber.lua"
	const configPath = baseDir + "fluent.conf"
	const parserConfigPath = baseDir + "parsers.conf"
	const masterCertPath = baseDir + "master.crt"

	var files archive.Archive

	luaCode := `
-- Do some tweaking of values that can't be expressed with the normal filters.
function run(tag, timestamp, record)
    record.trial_id = tonumber(record.trial_id)

    -- TODO: Only do this if it's not a partial record.
    record.log = record.log .. '\n'

    return 2, timestamp, record
end
`
	files = append(files,
		archive.Item{
			Path:     luaPath,
			Type:     tar.TypeReg,
			FileMode: 0444,
			Content:  []byte(luaCode),
		},
	)

	parserConfig := `
[PARSER]
  Name log_level
  Format regex
  # Parse out something that looks like a log level at the start of the line (e.g., "INFO: xxx"); if
  # nothing is found, return an empty log level and the full message back.
  Regex ((?<level>^(DEBUG|INFO|WARNING|ERROR|CRITICAL)): |(?<level>))(?<log>.*)
`

	files = append(files,
		archive.Item{
			Path:     parserConfigPath,
			Type:     tar.TypeReg,
			FileMode: 0444,
			Content:  []byte(parserConfig),
		},
	)

	baseConfig := fmt.Sprintf(`
[SERVICE]
  # Flush every .05 seconds to reduce latency for users.
  Flush .05
  Parsers_File %s

[INPUT]
  Name forward
`, parserConfigPath)

	filterConfig := fmt.Sprintf(`
# Attempt to parse the log level out of output lines.
[FILTER]
  Name parser
  Match *
  Key_Name log
  Parser log_level
  Reserve_Data true

# Move around fields to create the desired shape of object.
[FILTER]
  Name modify
  Match *
  # Delete Docker's container information, which we don't want.
  Remove container_id
  Remove container_name
  # Rename environment variables to normal names.
  Rename DET_TRIAL_ID trial_id
  Rename DET_CONTAINER_ID container_id

  Add agent_id %s
  Rename source stdtype

# Apply the Lua code for miscellaneous field tweaking.
[FILTER]
  Name lua
  Match *
  Script %s
  Call run
`, opts.AgentID, luaPath)

	outputConfig := fmt.Sprintf(`
# TMP: Output to stdout as well.
[OUTPUT]
  Name stdout
  Match *

[OUTPUT]
  Name http
  Match *
  Host %s
  Port %d
  URI /trial_logs
  Header_tag X-Fluent-Tag
  Format json
  Json_date_key timestamp
  Json_date_format iso8601
`, opts.MasterHost, opts.MasterPort)

	if opts.Security.TLS.Enabled {
		outputConfig += "  tls On\n"

		if opts.Security.TLS.SkipVerify {
			outputConfig += "  tls.verify Off\n"
		}
		if opts.Security.TLS.MasterCert != "" {
			outputConfig += "  tls.ca_file " + masterCertPath + "\n"

			certBytes, cErr := ioutil.ReadFile(opts.Security.TLS.MasterCert)
			if cErr != nil {
				return nil, nil, cErr
			}

			files = append(files,
				archive.Item{
					Path:     masterCertPath,
					Type:     tar.TypeReg,
					FileMode: 0444,
					Content:  certBytes,
				},
			)
		}
	}

	files = append(files,
		archive.Item{
			Path:     configPath,
			Type:     tar.TypeReg,
			FileMode: 0444,
			Content:  []byte(baseConfig + filterConfig + outputConfig),
		})

	args := []string{"/fluent-bit/bin/fluent-bit", "-c", configPath}

	return args, files, nil
}

func killContainer(ctx *actor.Context, c *client.Client, containerID string) error {
	if err := c.ContainerKill(context.Background(), containerID, "KILL"); err != nil {
		return errors.Wrap(err, "failed to kill container")
	}
	return nil
}

func killContainerByName(ctx *actor.Context, docker *client.Client, name string) error {
	containers, err := docker.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.Arg("name", name),
		),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list containers by name")
	}
	for _, cont := range containers {
		ctx.Log().WithFields(logrus.Fields{"name": name, "id": cont.ID}).Infof("killing Docker container")
		if err = killContainer(ctx, docker, cont.ID); err != nil {
			return err
		}
	}
	return nil
}

func pullImageByName(ctx *actor.Context, docker *client.Client, imageName string) error {
	_, _, err := docker.ImageInspectWithRaw(context.Background(), imageName)
	switch {
	case err == nil:
		// No error means the image is present; do nothing.
	case client.IsErrNotFound(err):
		// This error means the call to Docker went fine but the image doesn't exist; pull it now.
		ctx.Log().Infof("pulling Docker image %s", imageName)
		pullResponse, pErr := docker.ImagePull(context.Background(), imageName, types.ImagePullOptions{})
		if pErr != nil {
			return pErr
		}
		if _, pErr = ioutil.ReadAll(pullResponse); pErr != nil {
			return pErr
		}
		if pErr = pullResponse.Close(); pErr != nil {
			return pErr
		}
	default:
		// Something unexpected happened; propagate the error.
		return err
	}
	return nil
}

// startLoggingContainer starts a Fluent Bit container. It returns the host and port that the agent
// should use to connect to the daemon, the exposed host port of the container, and the ID of the
// container.
func startLoggingContainer(
	ctx *actor.Context,
	docker *client.Client,
	opts Options,
) (string, int, int, string, error) {
	const containerName = "determined-fluent"
	imageName := opts.FluentLoggingImage
	exposedPort := nat.Port(fmt.Sprintf("%d/tcp", fluentListenPort))

	if err := killContainerByName(ctx, docker, containerName); err != nil {
		return "", 0, 0, "", errors.Wrap(err, "failed to kill old logging container")
	}

	if err := pullImageByName(ctx, docker, imageName); err != nil {
		return "", 0, 0, "", errors.Wrap(err, "failed to pull logging image")
	}

	fluentArgs, fluentFiles, err := fluentConfig(opts)
	if err != nil {
		return "", 0, 0, "", errors.Wrap(err, "failed to configure Fluent Bit")
	}

	// Decide what Docker network to use for Fluent Bit. If this agent is itself running inside Docker,
	// use one of the networks that this container is using. If not, use the default network
	// ("bridge").
	dockerNet, err := getDockerNetwork(docker)
	if err != nil {
		return "", 0, 0, "", errors.Wrap(err, "failed to get Docker network")
	}
	fluentDockerNet := dockerNet
	if fluentDockerNet == "" {
		fluentDockerNet = "bridge"
	}
	ctx.Log().Infof("running Fluent Bit on Docker network %q", fluentDockerNet)

	createResponse, err := docker.ContainerCreate(
		context.Background(),
		&container.Config{
			Image:        imageName,
			Cmd:          fluentArgs,
			ExposedPorts: nat.PortSet{exposedPort: struct{}{}},
		},
		&container.HostConfig{
			AutoRemove:  true,
			NetworkMode: container.NetworkMode(fluentDockerNet),
			PortBindings: nat.PortMap{
				// A port of 0 makes Docker automatically assign a free port, which we read back below.
				exposedPort: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "0"}},
			},
			Resources: container.Resources{
				Memory:   1 << 30,
				NanoCPUs: 1000000000,
			},
		},
		nil,
		containerName,
	)
	if err != nil {
		return "", 0, 0, "", err
	}

	filesReader, _ := archive.ToIOReader(fluentFiles)

	err = docker.CopyToContainer(context.Background(),
		createResponse.ID,
		"/",
		filesReader,
		types.CopyToContainerOptions{},
	)
	if err != nil {
		return "", 0, 0, "", err
	}

	err = docker.ContainerStart(context.Background(), createResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", 0, 0, "", err
	}

	container, _ := docker.ContainerInspect(context.Background(), createResponse.ID)
	portStr := container.NetworkSettings.Ports[exposedPort][0].HostPort
	hostPort, _ := strconv.Atoi(portStr)
	ctx.Log().Infof("Fluent Bit listening on host port %d", hostPort)

	// We'll need to use either the container-internal or the host-visible address and port to connect
	// to Fluent Bit, depending on whether this agent is running inside Docker or not.
	addr := container.NetworkSettings.Networks[fluentDockerNet].IPAddress
	port := fluentListenPort
	if dockerNet == "" {
		addr = "localhost"
		port = hostPort
	}

	return addr, port, hostPort, createResponse.ID, nil
}

// getDockerNetwork returns the name of one of the Docker networks attached to the container that
// this process is in. It returns an empty string if it does not seem to be running inside Docker or
// is in Docker but on the host network, which should be treated equivalently.
func getDockerNetwork(docker *client.Client) (string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", errors.Wrap(err, "error opening cgroup file")
	}

	// This regex matches lines like "12:pids:/docker/<container ID>".
	re := regexp.MustCompile(`^[0-9]+:[a-z_=]+:/docker/([0-9a-f]+)$`)

	var cid string
	lines := bufio.NewScanner(f)
	for lines.Scan() {
		if err = lines.Err(); err != nil {
			return "", errors.Wrap(err, "error reading cgroup file")
		}
		line := lines.Text()

		match := re.FindStringSubmatch(line)
		if match != nil {
			cid = match[1]
			break
		}
	}

	if cid == "" {
		return "", nil
	}

	info, err := docker.ContainerInspect(context.Background(), cid)
	if err != nil {
		return "", errors.Wrap(err, "error inspecting this container")
	}

	for name := range info.NetworkSettings.Networks {
		if name == "host" {
			name = ""
		}
		return name, nil
	}

	return "", errors.New("running in a container but no networks found")
}

// fluentActor manages the lifecycle of the Fluent Bit container that is run by the agent for the
// purpose of forwarding container logs.
type fluentActor struct {
	opts        Options
	client      *fluent.Fluent
	port        int
	containerID string
	docker      *client.Client
}

func newFluentActor(ctx *actor.Context, opts Options) (*fluentActor, error) {
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.40"))
	if err != nil {
		return nil, errors.Wrap(err, "error connecting to Docker daemon")
	}

	t0 := time.Now()
	addr, port, hostPort, cid, err := startLoggingContainer(ctx, docker, opts)
	if err != nil {
		return nil, err
	}
	ctx.Log().Infof("Fluent Bit started in %s", time.Since(t0))

	config := fluent.Config{
		FluentHost:         addr,
		FluentPort:         port,
		SubSecondPrecision: true,
		RequestAck:         true,
	}

	var client *fluent.Fluent
	const retries = 5
	for i := 0; i < retries; i++ {
		client, err = fluent.New(config)
		if err != nil {
			if i == retries-1 {
				return nil, errors.Wrap(err, "failed to connect to Fluent Bit")
			}
			time.Sleep(time.Second)
		}
	}

	return &fluentActor{
		opts:        opts,
		client:      client,
		port:        hostPort,
		containerID: cid,
		docker:      docker,
	}, nil
}

func (f *fluentActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case fluentLog:
		return f.client.Post("determined.agent", msg)
	case actor.PostStop:
		t0 := time.Now()
		err := killContainer(ctx, f.docker, f.containerID)
		ctx.Log().Infof("Fluent Bit killed in %s", time.Since(t0))
		return err
	}
	return nil
}
