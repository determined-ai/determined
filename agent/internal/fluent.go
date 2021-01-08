package internal

import (
	"archive/tar"
	"context"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"time"

	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/fluent"
)

// The names of environment variables whose values should be included in log entries that Docker or
// the agent sends to the Fluent Bit logger.
var fluentEnvVarNames = []string{containerIDEnvVar, trialIDEnvVar}

var fluentLogLineRegexp = regexp.MustCompile(`\[[^]]*\] \[ *([^]]*)\] (.*)`)

func removeContainerByName(docker *client.Client, name string) error {
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
		log.WithFields(log.Fields{"name": name, "id": cont.ID}).Infof(
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

func pullImageByName(docker *client.Client, imageName string) error {
	_, _, err := docker.ImageInspectWithRaw(context.Background(), imageName)
	switch {
	case err == nil:
		// No error means the image is present; do nothing.
	case client.IsErrNotFound(err):
		// This error means the call to Docker went fine but the image doesn't exist; pull it now.
		log.Infof("pulling Docker image %s", imageName)
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
	docker *client.Client,
	opts Options,
	masterSetOpts aproto.MasterSetAgentOptions,
) (int, string, error) {
	const containerName = "determined-fluent"
	imageName := opts.Fluent.Image

	if err := removeContainerByName(docker, containerName); err != nil {
		return 0, "", errors.Wrap(err, "failed to kill old logging container")
	}

	if err := pullImageByName(docker, imageName); err != nil {
		return 0, "", errors.Wrap(err, "failed to pull logging image")
	}

	masterHost := opts.MasterHost
	masterPort := opts.MasterPort
	if opts.ContainerMasterHost != "" {
		masterHost = opts.ContainerMasterHost
	}
	if opts.ContainerMasterPort != 0 {
		masterPort = opts.ContainerMasterPort
	}

	var tlsConfig model.TLSClientConfig
	switch l := masterSetOpts.LoggingOptions; {
	case l.DefaultLoggingConfig != nil:
		t := opts.Security.TLS
		tlsConfig = model.TLSClientConfig{
			Enabled:         t.Enabled,
			SkipVerify:      t.SkipVerify,
			CertificatePath: t.MasterCert,
			CertificateName: t.MasterCertName,
		}
		if err := tlsConfig.Resolve(); err != nil {
			return 0, "", err
		}
	case l.ElasticLoggingConfig != nil:
		tlsConfig = l.ElasticLoggingConfig.Security.TLS
	}

	const fluentBaseDir = "/run/determined/fluent"
	//nolint:govet // Allow unkeyed struct fields -- it really looks much better like this.
	fluentArgs, fluentFiles := fluent.ContainerConfig(
		masterHost,
		masterPort,
		[]fluent.ConfigSection{
			{
				{"Name", "forward"},
			},
		},
		[]fluent.ConfigSection{
			{
				{"Name", "modify"},
				{"Match", "*"},
				// Delete Docker's container information, which we don't want.
				{"Remove", "container_id"},
				{"Remove", "container_name"},
				// Rename environment variables to normal names.
				{"Rename", containerIDEnvVar + " container_id"},
				{"Rename", trialIDEnvVar + " trial_id"},
				{"Add", "agent_id " + opts.AgentID},
				{"Rename", "source stdtype"},
			},
		},
		masterSetOpts.LoggingOptions,
		tlsConfig,
	)

	createResponse, err := docker.ContainerCreate(
		context.Background(),
		&container.Config{
			Image:      imageName,
			Cmd:        fluentArgs,
			WorkingDir: fluentBaseDir,
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

	var fluentArchive archive.Archive
	for name, content := range fluentFiles {
		fluentArchive = append(fluentArchive, archive.Item{
			Path:     filepath.Join(fluentBaseDir, name),
			Type:     tar.TypeReg,
			FileMode: 0444,
			Content:  content,
		})
	}

	filesReader, err := archive.ToIOReader(fluentArchive)
	if err != nil {
		return 0, "", err
	}

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

	log.Infof("Fluent Bit listening on host port %d", opts.Fluent.Port)

	return opts.Fluent.Port, createResponse.ID, nil
}

// fluentActor manages the lifecycle of the Fluent Bit container that is run by the agent for the
// purpose of forwarding container logs.
type fluentActor struct {
	opts            Options
	masterSetOpts   aproto.MasterSetAgentOptions
	port            int
	containerID     string
	docker          *client.Client
	fluentLogs      []*aproto.RunMessage
	fluentLogsCount int
}

func newFluentActor(
	opts Options,
	masterSetOpts aproto.MasterSetAgentOptions,
) (*fluentActor, error) {
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.40"))
	if err != nil {
		return nil, errors.Wrap(err, "error connecting to Docker daemon")
	}

	t0 := time.Now()
	hostPort, cid, err := startLoggingContainer(docker, opts, masterSetOpts)
	if err != nil {
		return nil, err
	}
	log.Infof("Fluent Bit started in %s", time.Since(t0))

	return &fluentActor{
		opts:          opts,
		masterSetOpts: masterSetOpts,
		port:          hostPort,
		containerID:   cid,
		docker:        docker,
		fluentLogs:    make([]*aproto.RunMessage, 50),
	}, nil
}

// fluentFailedDetected is a message sent when the trackLogs sees fluent has failed.
type fluentFailureDetected struct{}

func (f *fluentActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		go f.trackLogs(ctx)
	case aproto.ContainerLog:
		if msg.RunMessage == nil {
			return nil
		}
		f.fluentLogs[f.fluentLogsCount%len(f.fluentLogs)] = msg.RunMessage
		f.fluentLogsCount++
		match := fluentLogLineRegexp.FindStringSubmatch(msg.RunMessage.Value)
		if match == nil {
			return nil
		}
		switch level, message := match[1], match[2]; level {
		case "warn":
			ctx.Log().Warnf("Fluent Bit: %s", message)
		case "error":
			ctx.Log().Errorf("Fluent Bit: %s", message)
		}
	case fluentFailureDetected:
		return errors.New("detected Fluent Bit exit")
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

func (f *fluentActor) trackLogs(ctx *actor.Context) {
	exitChan, errChan := f.docker.ContainerWait(
		context.Background(), f.containerID, container.WaitConditionNotRunning)

	if err := trackLogs(ctx, f.docker, f.containerID, ctx.Self()); err != nil {
		ctx.Log().Errorf("error tracking Fluent Bit logs: %s", err)
	}
	// This message also allows us to synchronize with the buffer before dumping logs.
	ctx.Ask(ctx.Self(), aproto.ContainerLog{
		Timestamp: time.Now(),
		RunMessage: &aproto.RunMessage{
			Value: "detected Fluent Bit exit",
		},
	}).Get()

	select {
	case err := <-errChan:
		ctx.Log().WithError(err).Error("failed to wait for Fluent Bit to exit")
	case exit := <-exitChan:
		ctx.Log().Errorf("Fluent Bit exit status: %+v", exit)
	}

	ctx.Log().Error("Fluent Bit failed, dumping all recent logs")
	i0 := f.fluentLogsCount - len(f.fluentLogs)
	if i0 < 0 {
		i0 = 0
	}
	for i := i0; i < f.fluentLogsCount; i++ {
		ctx.Log().Error(f.fluentLogs[i%len(f.fluentLogs)].Value)
	}
	ctx.Tell(ctx.Self(), fluentFailureDetected{})
}
