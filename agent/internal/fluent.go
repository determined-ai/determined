package internal

import (
	"archive/tar"
	"context"
	"fmt"
	"io/ioutil"
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
    record.rank = tonumber(record.rank)
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

func startLoggingContainer(ctx *actor.Context, opts Options) (int, string, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.40"))
	if err != nil {
		return 0, "", errors.Wrap(err, "error connecting to Docker daemon")
	}

	defer func() {
		if cErr := c.Close(); cErr != nil {
			logrus.WithError(cErr).Warn("failed to close Docker connection")
		}
	}()

	const imageName = "fluent/fluent-bit:1.5"
	const exposedPort = "24224/tcp"
	const containerName = "determined-fluent"

	// Check for an old container and kill it if present.
	containers, err := c.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.Arg("name", containerName),
		),
	})
	if err != nil {
		return 0, "", err
	}
	for _, cont := range containers {
		ctx.Log().Infof("killing found Fluent Bit container %s", cont.ID)
		if err = c.ContainerKill(context.Background(), cont.ID, "KILL"); err != nil {
			return 0, "", errors.Wrap(err, "failed to kill existing Fluent Bit container")
		}
	}

	// Pull the image.
	_, _, err = c.ImageInspectWithRaw(context.Background(), imageName)
	switch {
	case err == nil:
		// No error means the image is present; do nothing.
	case client.IsErrNotFound(err):
		ctx.Log().Infof("pulling Fluent Bit Docker image %s", imageName)

		// This error means the call to Docker went fine but the image doesn't exist; pull it.
		pullResponse, pErr := c.ImagePull(context.Background(), imageName, types.ImagePullOptions{})
		if pErr != nil {
			return 0, "", pErr
		}
		if _, pErr = ioutil.ReadAll(pullResponse); pErr != nil {
			return 0, "", pErr
		}
		if pErr = pullResponse.Close(); pErr != nil {
			return 0, "", pErr
		}
	default:
		// Something unexpected happened; propagate the error.
		return 0, "", errors.Wrap(err, "failed to pull logging image")
	}

	fluentArgs, fluentFiles, err := fluentConfig(opts)
	if err != nil {
		return 0, "", err
	}

	createResponse, err := c.ContainerCreate(
		context.Background(),
		&container.Config{
			Image:        imageName,
			Cmd:          fluentArgs,
			ExposedPorts: nat.PortSet{exposedPort: struct{}{}},
		},
		&container.HostConfig{
			AutoRemove: true,
			PortBindings: nat.PortMap{
				// A port of 0 makes Docker automatically assign a free port, which we read back below.
				exposedPort: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "0"}},
			},
		},
		nil,
		containerName,
	)
	if err != nil {
		return 0, "", err
	}

	filesReader, _ := archive.ToIOReader(fluentFiles)

	err = c.CopyToContainer(context.Background(),
		createResponse.ID,
		"/",
		filesReader,
		types.CopyToContainerOptions{},
	)
	if err != nil {
		return 0, "", err
	}

	err = c.ContainerStart(context.Background(), createResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		return 0, "", err
	}

	container, _ := c.ContainerInspect(context.Background(), createResponse.ID)
	portStr := container.NetworkSettings.Ports[exposedPort][0].HostPort
	port, _ := strconv.Atoi(portStr)
	ctx.Log().Infof("Fluent Bit listening on host port %d", port)

	return port, createResponse.ID, nil
}

func killContainer(containerID string) error {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.40"))
	if err != nil {
		return errors.Wrap(err, "error connecting to Docker daemon")
	}

	defer func() {
		if cErr := c.Close(); cErr != nil {
			logrus.WithError(cErr).Warn("failed to close Docker connection")
		}
	}()

	return c.ContainerKill(context.Background(), containerID, "KILL")
}

// fluentActor manages the lifecycle of the Fluent Bit container that is run by the agent for the
// purpose of forwarding container logs.
type fluentActor struct {
	opts        Options
	client      *fluent.Fluent
	port        int
	containerID string
}

func newFluentActor(ctx *actor.Context, opts Options) (*fluentActor, error) {
	t0 := time.Now()
	port, cid, err := startLoggingContainer(ctx, opts)
	if err != nil {
		return nil, err
	}
	ctx.Log().Infof("Fluent Bit started in %s", time.Since(t0))

	client, err := fluent.New(fluent.Config{
		FluentHost:         "localhost",
		FluentPort:         port,
		SubSecondPrecision: true,
		RequestAck:         true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Fluent Bit")
	}

	return &fluentActor{
		opts:        opts,
		client:      client,
		port:        port,
		containerID: cid,
	}, nil
}

func (f *fluentActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case fluentLog:
		return f.client.Post("determined.agent", msg)
	case actor.PostStop:
		t0 := time.Now()
		err := killContainer(f.containerID)
		ctx.Log().Infof("Fluent Bit killed in %s", time.Since(t0))
		return err
	}
	return nil
}
