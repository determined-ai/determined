package fluent

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/determined-ai/determined/agent/internal/container"
	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"

	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/fluent"
)

// EnvVarNames are the of environment variables whose values should be included in log entries that
// Docker or the agent sends to the Fluent Bit logger.
var EnvVarNames = []string{
	container.ContainerIDEnvVar,
	container.AllocationIDEnvVar,
	container.TaskIDEnvVar,
}

var fluentLogLineRegexp = regexp.MustCompile(`\[[^]]*\] \[ *([^]]*)\] (.*)`)

// Fluent manages the lifecycle of the Fluent Bit container that is run by the agent for
// the purpose of forwarding container logs.
type Fluent struct {
	// Configuration details.
	opts  options.Options
	mopts aproto.MasterSetAgentOptions

	// System dependencies.
	log *log.Entry
	dcl *docker.Client

	// Internal state.
	fluentLogs      []*aproto.RunMessage
	fluentLogsCount int
	wg              errgroupx.Group // Fluentbit-scoped group.
	err             error
	Done            chan struct{} // Closed when FluentBit exits.
}

// Start constructs and runs a Fluent Bit daemon, and returns a handle to interact with it.
func Start(
	ctx context.Context,
	opts options.Options,
	mopts aproto.MasterSetAgentOptions,
	dcl *docker.Client,
) (*Fluent, error) {
	f := &Fluent{
		opts:  opts,
		mopts: mopts,

		log: log.WithField("component", "fluentbit"),
		dcl: dcl,

		fluentLogs: make([]*aproto.RunMessage, 50),
		wg:         errgroupx.WithContext(context.Background()),
		Done:       make(chan struct{}),
	}

	cID, err := f.startContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting logging container: %w", err)
	}

	f.wg.Go(func(ctx context.Context) error {
		defer f.wg.Cancel()
		switch err := f.monitor(ctx, cID); {
		case errors.Is(err, context.Canceled):
			return nil
		case err != nil:
			return err
		default:
			return nil
		}
	})

	go func() {
		f.err = f.wg.Wait()
		close(f.Done)
	}()

	return f, nil
}

// Error returns the error associated with an exit, if there is any. It returns nil when running.
func (f *Fluent) Error() error {
	return f.err
}

// Wait for the daemon to exit.
func (f *Fluent) Wait() error {
	return f.wg.Wait()
}

// Close the Fluent Bit daemon.
func (f *Fluent) Close() error {
	return f.wg.Close()
}

func (f *Fluent) monitor(ctx context.Context, cID string) error {
	exitC, errC := f.dcl.Inner().ContainerWait(
		ctx,
		cID,
		dcontainer.WaitConditionNotRunning,
	)

	logs := make(chan *aproto.ContainerLog, 64)
	loggroup := errgroupx.WithContext(ctx)
	loggroup.Go(func(ctx context.Context) error {
		defer close(logs)
		return f.trackLogs(ctx, cID, logs)
	})
	loggroup.Go(func(ctx context.Context) error {
		for {
			select {
			case log, ok := <-logs:
				if !ok {
					return nil
				}
				if log.RunMessage == nil {
					continue
				}
				f.recordLog(log.RunMessage)
				match := fluentLogLineRegexp.FindStringSubmatch(log.RunMessage.Value)
				if match == nil {
					continue
				}
				switch level, message := match[1], match[2]; level {
				case "warn":
					f.log.Warnf("Fluent Bit: %s", message)
				case "error":
					f.log.Errorf("Fluent Bit: %s", message)
				}
			case <-ctx.Done():
				return fmt.Errorf("fluent bit canceled: %w", ctx.Err())
			}
		}
	})
	switch err := loggroup.Wait(); {
	case errors.Is(err, context.Canceled):
		f.log.WithError(err).Warn("orphaning fluent due to cancellation")
		return err
	case err != nil:
		f.log.WithError(err).Warnf("Fluent Bit logs failed unexpectedly")
	default:
		f.log.Warnf("Fluent Bit logs ended unexpectedly")
	}

	f.printRecentLogs()
	if err := f.dcl.RemoveContainer(ctx, cID, true); err != nil {
		f.log.WithError(err).Debug("ensuring Fluent Bit is cleaned up")
	}
	select {
	case err := <-errC:
		return fmt.Errorf("failure waiting for Fluent Bit to exit: %w", err)
	case exit := <-exitC:
		if exit.Error != nil {
			return fmt.Errorf("failure in Fluent Bit (%s)", exit.Error.Message)
		}
		return fmt.Errorf("unexpected Fluent Bit exit (%d)", exit.StatusCode)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// startContainer starts a Fluent Bit container running in host mode. It returns the port
// that Fluent Bit is listening on and the ID of the container.
// TODO(Brad): make fluent just use "containers" package, or do the move fluent tech debt.
func (f *Fluent) startContainer(ctx context.Context) (string, error) {
	f.log.Trace("starting fluent container")
	t0 := time.Now()

	imageName := f.opts.Fluent.Image

	if err := removeContainerByName(ctx, f.dcl, f.opts.Fluent.ContainerName); err != nil {
		return "", errors.Wrap(err, "failed to kill old logging container")
	}

	if err := pullImageByName(ctx, f.dcl, imageName); err != nil {
		return "", errors.Wrap(err, "failed to pull logging image")
	}

	masterHost := f.opts.MasterHost
	masterPort := f.opts.MasterPort
	if f.opts.ContainerMasterHost != "" {
		masterHost = f.opts.ContainerMasterHost
	}
	if f.opts.ContainerMasterPort != 0 {
		masterPort = f.opts.ContainerMasterPort
	}

	var tlsConfig model.TLSClientConfig
	switch l := f.mopts.LoggingOptions; {
	case l.DefaultLoggingConfig != nil:
		t := f.opts.Security.TLS
		tlsConfig = model.TLSClientConfig{
			Enabled:         t.Enabled,
			SkipVerify:      t.SkipVerify,
			CertificatePath: t.MasterCert,
			CertificateName: t.MasterCertName,
		}
		if err := tlsConfig.Resolve(); err != nil {
			return "", err
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
				{"Port", strconv.Itoa(f.opts.Fluent.Port)},
				// Setting mem_buf_limit allows Fluent Bit to buffer log data to disk if the rest of the
				// pipeline is backed up. In combination with setting the Docker log driver to run in
				// non-blocking mode, that lets us avoid impacting application performance when there are bursts
				// in log output.
				//
				// This scheme is described in more detail at:
				// https://docs.fluentbit.io/manual/administration/buffering-and-storage
				{"mem_buf_limit", "100M"},
			},
		},
		[]fluent.ConfigSection{
			{
				{"Name", "modify"},
				{"Match", "*"},
				// Delete Docker's container information, which we don't want.
				{"Remove", "container_id"},
				{"Remove", "container_name"},
				// Remove metadata about the container parent that we set on the container.
				{"Remove", "ai.determined.container.parent"},
				// Rename environment variables to normal names.
				{"Rename", container.ContainerIDEnvVar + " container_id"},
				{"Rename", container.AllocationIDEnvVar + " allocation_id"},
				{"Rename", container.TaskIDEnvVar + " task_id"},
				{"Add", "agent_id " + f.opts.AgentID},
				{"Rename", "source stdtype"},
			},
		},
		f.mopts.LoggingOptions,
		tlsConfig,
	)

	createResponse, err := f.dcl.Inner().ContainerCreate(
		ctx,
		&dcontainer.Config{
			Image:      imageName,
			Cmd:        fluentArgs,
			WorkingDir: fluentBaseDir,
		},
		&dcontainer.HostConfig{
			// Set autoremove to reduce the number of states that the container is likely to be in and what
			// we have to do to manage it cleanly. Restart on failure could be useful, but it conflcts with
			// autoremove; we may want to consider switching to that instead at some point.
			AutoRemove: true,
			// Always use host mode to simplify the space of networking scenarios we have to consider.
			NetworkMode: "host",
			// Provide some reasonable resource limits on the container just to be safe.
			Resources: dcontainer.Resources{
				Memory:   1 << 30,
				NanoCPUs: 1000000000,
			},
		},
		nil,
		nil,
		f.opts.Fluent.ContainerName,
	)
	if err != nil {
		return "", err
	}

	var fluentArchive archive.Archive
	for name, content := range fluentFiles {
		fluentArchive = append(fluentArchive, archive.Item{
			Path:     filepath.Join(fluentBaseDir, name),
			Type:     tar.TypeReg,
			FileMode: 0o444,
			Content:  content,
		})
	}

	filesReader, err := archive.ToIOReader(fluentArchive)
	if err != nil {
		return "", err
	}

	err = f.dcl.Inner().CopyToContainer(
		ctx,
		createResponse.ID,
		"/",
		filesReader,
		types.CopyToContainerOptions{},
	)
	if err != nil {
		return "", err
	}

	err = f.dcl.Inner().ContainerStart(ctx, createResponse.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}

	log.Infof("Fluent Bit listening on host port %d", f.opts.Fluent.Port)
	switch {
	case f.mopts.LoggingOptions.DefaultLoggingConfig != nil:
		log.Infof("Fluent Bit shipping to Determined at %s:%d", f.opts.MasterHost, f.opts.MasterPort)
	case f.mopts.LoggingOptions.ElasticLoggingConfig != nil:
		eopts := f.mopts.LoggingOptions.ElasticLoggingConfig
		log.Infof("Fluent Bit shipping to Elastic at %s:%d", eopts.Host, eopts.Port)
	}

	f.log.Infof("Fluent Bit started in %s", time.Since(t0))
	return createResponse.ID, nil
}

func pullImageByName(ctx context.Context, dcl *docker.Client, imageName string) error {
	_, _, err := dcl.Inner().ImageInspectWithRaw(ctx, imageName)
	switch {
	case err == nil:
		// No error means the image is present; do nothing.
	case client.IsErrNotFound(err):
		// This error means the call to Docker went fine but the image doesn't exist; pull it now.
		log.Infof("pulling Docker image %s", imageName)
		pullResponse, pErr := dcl.Inner().ImagePull(ctx, imageName, types.ImagePullOptions{})
		if pErr != nil {
			return pErr
		}
		if _, pErr = io.ReadAll(pullResponse); pErr != nil {
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

func (f *Fluent) recordLog(l *aproto.RunMessage) {
	f.fluentLogs[f.fluentLogsCount%len(f.fluentLogs)] = l
	f.fluentLogsCount++
}

func (f *Fluent) trackLogs(
	ctx context.Context,
	id string,
	logs chan<- *aproto.ContainerLog,
) error {
	reader, err := f.dcl.Inner().ContainerLogs(
		ctx,
		id,
		types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Since:      "",
			Timestamps: false,
			Follow:     true,
			Tail:       "all",
			Details:    true,
		},
	)
	if err != nil {
		return errors.Wrap(err, "error grabbing container logs")
	}
	defer func() {
		if err = reader.Close(); err != nil {
			f.log.WithError(err).Warn("error closing log stream")
		}
	}()

	if _, err = stdcopy.StdCopy(
		newStdStream(ctx, stdcopy.Stdout, logs),
		newStdStream(ctx, stdcopy.Stderr, logs),
		reader,
	); err != nil {
		return fmt.Errorf("error scanning logs: %w", err)
	}
	return nil
}

func removeContainerByName(
	ctx context.Context,
	dcl *docker.Client,
	name string,
) error {
	cont, err := containerByName(ctx, dcl, name)
	switch {
	case err != nil:
		return err
	case cont == nil:
		return nil
	}

	log.WithField("id", cont.ID).Infof("removing Docker container %s", name)
	err = dcl.Inner().ContainerRemove(ctx, cont.ID, docker.ForceRemoveOpts)
	switch {
	case strings.Contains(err.Error(), docker.NoSuchContainer):
		log.WithField("id", cont.ID).Tracef(err.Error())
		return nil
	case docker.RemovalInProgress.MatchString(err.Error()):
		log.WithField("id", cont.ID).Tracef(err.Error())
		exitC, errC := dcl.Inner().ContainerWait(ctx, cont.ID, dcontainer.WaitConditionRemoved)
		select {
		case resp := <-exitC:
			err := resp.Error
			if err == nil {
				return fmt.Errorf("failed to wait for container %s removal: %s", cont.ID, err)
			}
			return nil
		case err := <-errC:
			return fmt.Errorf("failed to wait for container %s removal: %w", cont.ID, err)
		case <-ctx.Done():
			return ctx.Err()
		}
	case err != nil:
		return fmt.Errorf("failed to remove container %s: %w", cont.ID, err)
	default:
		return nil
	}
}

func containerByName(
	ctx context.Context,
	dcl *docker.Client,
	name string,
) (*types.Container, error) {
	containers, err := dcl.Inner().ContainerList(ctx, types.ContainerListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("name", name),
		),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list containers by name")
	}

	for _, c := range containers {
		// ContainerList by name filters by prefix.
		// Check for an exact match while accounting for / prefix.
		for _, containerName := range c.Names {
			if strings.TrimLeft(containerName, "/") != name {
				continue
			}

			return &c, nil
		}
	}
	log.Tracef("no container found with name %s", name)
	return nil, nil
}

func (f *Fluent) printRecentLogs() {
	i0 := f.fluentLogsCount - len(f.fluentLogs)
	if i0 < 0 {
		i0 = 0
	}
	for i := i0; i < f.fluentLogsCount; i++ {
		f.log.Error(f.fluentLogs[i%len(f.fluentLogs)].Value)
	}
}

type stdStream struct {
	//nolint: containedctx // Disagree.
	ctx     context.Context
	stdType stdcopy.StdType
	out     chan<- *aproto.ContainerLog
}

func newStdStream(
	ctx context.Context,
	stdType stdcopy.StdType,
	out chan<- *aproto.ContainerLog,
) *stdStream {
	return &stdStream{
		ctx:     ctx,
		stdType: stdType,
		out:     out,
	}
}

func (s stdStream) Write(p []byte) (n int, err error) {
	log := &aproto.ContainerLog{
		Timestamp: time.Now().UTC(),
		RunMessage: &aproto.RunMessage{
			Value:   string(p),
			StdType: s.stdType,
		},
	}
	select {
	case s.out <- log:
		return len(p), nil
	case <-s.ctx.Done():
		return 0, fmt.Errorf("canceled while writing log: %w", s.ctx.Err())
	}
}
