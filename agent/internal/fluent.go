package internal

import (
	"archive/tar"
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
)

// The names of environment variables whose values should be included in log entries that Docker or
// the agent sends to the Fluent Bit logger.
var fluentEnvVarNames = []string{containerIDEnvVar, trialIDEnvVar}

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
    record.rank_id = tonumber(record.rank_id)
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
  Name rank_id
  Format regex
  # Look for a rank ID from the beginning of the line (e.g., "[rank=0] xxx").
  Regex ^\[rank=(?<rank_id>([0-9]+))\] (?<log>.*)

[PARSER]
  Name log_level
  Format regex
  # Look for a log level at the start of the line (e.g., "INFO: xxx").
  Regex ^(?<level>(DEBUG|INFO|WARNING|ERROR|CRITICAL)): (?<log>.*)
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
# Attempt to parse the rank ID and log level out of output lines.
[FILTER]
  Name parser
  Match *
  Key_Name log
  Parser rank_id
  Reserve_Data true

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
  Rename %s container_id
  Rename %s trial_id

  Add agent_id %s
  Rename source stdtype

# Apply the Lua code for miscellaneous field tweaking.
[FILTER]
  Name lua
  Match *
  Script %s
  Call run
`, containerIDEnvVar, trialIDEnvVar, opts.AgentID, luaPath)

	// HACK: If a host resolves to both IPv4 and IPv6 addresses, Fluent Bit seems to only try IPv6 and
	// fail if that connection doesn't work. IPv6 doesn't play well with Docker and many Linux
	// distributions ship with an `/etc/hosts` that maps "localhost" to both 127.0.0.1 (IPv4) and [::1]
	// (IPv6), so Fluent Bit will break when run in host mode. To avoid that, translate "localhost"
	// diretcly into an IP address before passing it to Fluent Bit.
	fluentMasterHost := opts.MasterHost
	if fluentMasterHost == "localhost" {
		fluentMasterHost = "127.0.0.1"
	}

	outputConfig := fmt.Sprintf(`
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
`, fluentMasterHost, opts.MasterPort)

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

func removeContainerByName(ctx *actor.Context, docker *client.Client, name string) error {
	containers, err := docker.ContainerList(context.Background(), types.ContainerListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("name", name),
		),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list containers by name")
	}
	for _, cont := range containers {
		ctx.Log().WithFields(logrus.Fields{"name": name, "id": cont.ID}).Infof(
			"killing and removing Docker container",
		)
		err := docker.ContainerRemove(
			context.Background(), cont.ID, types.ContainerRemoveOptions{Force: true},
		)
		if err != nil {
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

// startLoggingContainer starts a Fluent Bit container running in host mode. It returns the port
// that Fluent Bit is listening on and the ID of the container.
func startLoggingContainer(
	ctx *actor.Context,
	docker *client.Client,
	opts Options,
) (int, string, error) {
	const containerName = "determined-fluent"
	imageName := opts.Fluent.Image

	if err := removeContainerByName(ctx, docker, containerName); err != nil {
		return 0, "", errors.Wrap(err, "failed to kill old logging container")
	}

	if err := pullImageByName(ctx, docker, imageName); err != nil {
		return 0, "", errors.Wrap(err, "failed to pull logging image")
	}

	fluentArgs, fluentFiles, err := fluentConfig(opts)
	if err != nil {
		return 0, "", errors.Wrap(err, "failed to configure Fluent Bit")
	}

	createResponse, err := docker.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: imageName,
			Cmd:   fluentArgs,
		},
		&container.HostConfig{
			// Set autoremove to reduce the number of states that the container is likely to be in and what
			// we have to do to manage it cleanly. Restart on failure could be useful, but it conflcts with
			// autoremove; we may want to consider switching to that instead at some point.
			AutoRemove: true,
			// Always use host mode to simplify the space of networking scenarios we have to consider.
			NetworkMode: "host",
			// Provide some reasonable resource limits on the container just to be safe.
			Resources: container.Resources{
				Memory:   1 << 30,
				NanoCPUs: 1000000000,
			},
		},
		nil,
		containerName,
	)
	if err != nil {
		return 0, "", err
	}

	filesReader, _ := archive.ToIOReader(fluentFiles)

	err = docker.CopyToContainer(context.Background(),
		createResponse.ID,
		"/",
		filesReader,
		types.CopyToContainerOptions{},
	)
	if err != nil {
		return 0, "", err
	}

	err = docker.ContainerStart(context.Background(), createResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		return 0, "", err
	}

	ctx.Log().Infof("Fluent Bit listening on host port %d", opts.Fluent.Port)

	return opts.Fluent.Port, createResponse.ID, nil
}

// fluentActor manages the lifecycle of the Fluent Bit container that is run by the agent for the
// purpose of forwarding container logs.
type fluentActor struct {
	opts        Options
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
	hostPort, cid, err := startLoggingContainer(ctx, docker, opts)
	if err != nil {
		return nil, err
	}
	ctx.Log().Infof("Fluent Bit started in %s", time.Since(t0))

	return &fluentActor{
		opts:        opts,
		port:        hostPort,
		containerID: cid,
		docker:      docker,
	}, nil
}

func (f *fluentActor) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PostStop:
		t0 := time.Now()
		err := f.docker.ContainerRemove(
			context.Background(), f.containerID, types.ContainerRemoveOptions{Force: true},
		)
		ctx.Log().Infof("Fluent Bit killed in %s", time.Since(t0))
		return err
	}
	return nil
}
