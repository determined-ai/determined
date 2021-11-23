package internal

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"syscall"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
)

type dockerActor struct {
	*client.Client
	credentialStores map[string]*credentialStore
}

type (
	signalContainer struct {
		dockerID string
		signal   syscall.Signal
	}
	pullImage struct {
		cproto.PullSpec
		Name string
	}
	runContainer struct {
		cproto.RunSpec
	}
	reattachContainer struct {
		ID cproto.ID
	}

	imagePulled      struct{}
	containerStarted struct {
		dockerID      string
		containerInfo types.ContainerJSON
	}
	containerTerminated struct {
		ExitCode aproto.ExitCode
	}
	dockerErr struct{ Error error }
)

// registryToString converts the Registry struct to a base64 encoding for json strings.
func registryToString(reg types.AuthConfig) (string, error) {
	bs, err := json.Marshal(reg)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bs), nil
}

func (d *dockerActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		stores, err := getAllCredentialStores()
		if err != nil {
			ctx.Log().Infof(
				"can't find any docker credential stores, continuing without them %v", err)
		}
		d.credentialStores = stores

	case pullImage:
		go d.pullImage(ctx, msg)

	case reattachContainer:
		go d.reattachContainer(ctx, msg.ID)

	case runContainer:
		go d.runContainer(ctx, msg.RunSpec)

	case signalContainer:
		go d.signalContainer(ctx, msg)

	case actor.PostStop:
	}
	return nil
}

func (d *dockerActor) pullImage(ctx *actor.Context, msg pullImage) {
	ref, err := reference.ParseNormalizedNamed(msg.Name)
	if err != nil {
		sendErr(ctx, errors.Wrapf(err, "error parsing image name: %s", msg.Name))
		return
	}
	ref = reference.TagNameOnly(ref)

	_, _, err = d.ImageInspectWithRaw(context.Background(), ref.String())
	switch {
	case msg.ForcePull:
		if err == nil {
			d.sendAuxLog(ctx, fmt.Sprintf(
				"image present, but force_pull_image is set; checking for updates: %s",
				ref.String(),
			))
		}
	case err == nil:
		d.sendAuxLog(ctx, fmt.Sprintf("image already found, skipping pull phase: %s", ref.String()))
		ctx.Tell(ctx.Sender(), imagePulled{})
		return
	case client.IsErrNotFound(err):
		d.sendAuxLog(ctx, fmt.Sprintf("image not found, pulling image: %s", ref.String()))
	default:
		sendErr(ctx, errors.Wrapf(err, "error checking if image exists: %s", ref.String()))
		return
	}

	// TODO: replace with command.EncodeAuthToBase64
	reg := ""
	if msg.Registry != nil {
		if reg, err = registryToString(*msg.Registry); err != nil {
			sendErr(ctx, errors.Wrap(err, "error encoding registry credentials"))
			return
		}
	} else if store, ok := d.credentialStores[reference.Domain(ref)]; ok {
		var creds types.AuthConfig
		creds, err = store.get()
		if err != nil {
			sendErr(ctx, errors.Wrap(err, "unable to get credentials from helper"))
			return
		}
		reg, err = registryToString(creds)
		if err != nil {
			sendErr(ctx, errors.Wrap(err, "error encoding registry credentials from helper"))
			return
		}
	}

	opts := types.ImagePullOptions{
		All:          false,
		RegistryAuth: reg,
	}

	logs, err := d.ImagePull(context.Background(), ref.String(), opts)
	if err != nil {
		sendErr(ctx, errors.Wrapf(err, "error pulling image: %s", ref.String()))
		return
	}

	if err = d.sendPullLogs(ctx, logs); err != nil {
		sendErr(ctx, errors.Wrap(err, "error parsing log stream"))
		return
	}
	if err = logs.Close(); err != nil {
		sendErr(ctx, errors.Wrap(err, "error closing log stream"))
		return
	}
	ctx.Tell(ctx.Sender(), imagePulled{})
}

func (d *dockerActor) runContainer(ctx *actor.Context, msg cproto.RunSpec) {
	useFluentLogging := msg.UseFluentLogging
	if !useFluentLogging {
		msg.HostConfig.AutoRemove = false
	}

	response, err := d.ContainerCreate(
		context.Background(), &msg.ContainerConfig, &msg.HostConfig, &msg.NetworkingConfig, "")
	if err != nil {
		sendErr(ctx, errors.Wrap(err, "error creating container"))
		return
	}
	containerID := response.ID
	for _, w := range response.Warnings {
		d.sendAuxLog(ctx, fmt.Sprintf("warning when creating container: %s", w))
	}

	if !useFluentLogging {
		defer func() {
			if err = d.Client.ContainerRemove(
				context.Background(), containerID, types.ContainerRemoveOptions{},
			); err != nil {
				sendErr(ctx, errors.Wrap(err, "error removing container"))
			}
		}()
	}

	for _, copyArx := range msg.Archives {
		d.sendAuxLog(ctx, fmt.Sprintf("copying files to container: %s", copyArx.Path))
		files, aerr := archive.ToIOReader(copyArx.Archive)
		if aerr != nil {
			sendErr(ctx, errors.Wrap(aerr, "error converting RunSpec Archive files to io.Reader"))
			return
		}
		if cerr := d.CopyToContainer(
			context.Background(),
			containerID,
			copyArx.Path,
			files,
			copyArx.CopyOptions,
		); cerr != nil {
			sendErr(ctx, errors.Wrap(cerr, "error copying files to container"))
			return
		}
	}

	exit, eerr := d.ContainerWait(
		context.Background(), containerID, dcontainer.WaitConditionNextExit)

	if err = d.ContainerStart(context.Background(), containerID,
		types.ContainerStartOptions{}); err != nil {
		sendErr(ctx, errors.Wrap(err, "error starting container"))
		return
	}

	// If we specified a port to expose but not the host port to bind, Docker assigns an arbitrary host
	// port, which we ask for here. (If we did specify a host port, this gives the same one back.)
	containerInfo, err := d.ContainerInspect(context.Background(), containerID)
	if err != nil {
		sendErr(ctx, errors.Wrap(err, "error inspecting container"))
		return
	}

	ctx.Tell(
		ctx.Sender(),
		containerStarted{dockerID: response.ID, containerInfo: containerInfo},
	)

	if !useFluentLogging {
		if lerr := trackLogs(ctx, d.Client, containerID, ctx.Sender()); lerr != nil {
			sendErr(ctx, lerr)
		}
	}
	select {
	case err = <-eerr:
		sendErr(ctx, errors.Wrap(err, "error while waiting for container to exit"))
	case exit := <-exit:
		ctx.Tell(ctx.Sender(), containerTerminated{ExitCode: aproto.ExitCode(exit.StatusCode)})
	}
}

func (d *dockerActor) reattachContainer(ctx *actor.Context, id cproto.ID) {
	containers, err := d.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", dockerContainerIDLabel+"="+id.String()),
		),
	})
	if err != nil {
		sendErr(ctx, errors.Wrap(err, "error while reattaching container"))
		return
	}

	for _, cont := range containers {
		// Subscribe to termination notifications first.
		exit, eerr := d.ContainerWait(context.Background(), cont.ID, dcontainer.WaitConditionNextExit)

		// Restore containerInfo.
		containerInfo, err := d.ContainerInspect(context.Background(), cont.ID)
		if err != nil {
			sendErr(ctx, errors.Wrap(err, "error inspecting reattached container"))
			return
		}

		// Check if container has exited while we were trying to reattach it.
		if !containerInfo.State.Running {
			ctx.Tell(
				ctx.Sender(),
				containerTerminated{ExitCode: aproto.ExitCode(containerInfo.State.ExitCode)})
		} else {
			select {
			case err = <-eerr:
				sendErr(ctx, errors.Wrap(err, "error while waiting for reattached container to exit"))
			case exit := <-exit:
				ctx.Tell(ctx.Sender(), containerTerminated{ExitCode: aproto.ExitCode(exit.StatusCode)})
			}
		}
	}
}

func (d *dockerActor) signalContainer(ctx *actor.Context, msg signalContainer) {
	err := d.ContainerKill(context.Background(), msg.dockerID, unix.SignalName(msg.signal))
	if err != nil {
		sendErr(ctx, errors.Wrap(err, "error while killing container"))
		return
	}
}

func sendErr(ctx *actor.Context, err error) {
	ctx.Tell(ctx.Sender(), dockerErr{Error: err})
}

func (d *dockerActor) sendAuxLog(ctx *actor.Context, msg string) {
	ctx.Tell(ctx.Sender(), aproto.ContainerLog{
		Timestamp:  time.Now().UTC(),
		AuxMessage: &msg,
	})
}

func (d *dockerActor) sendPullLogs(ctx *actor.Context, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log := jsonmessage.JSONMessage{}
		if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
			return errors.Wrapf(err, "error parsing log message: %#v", log)
		}
		ctx.Tell(ctx.Sender(), aproto.ContainerLog{
			Timestamp:   time.Now().UTC(),
			PullMessage: &log,
		})
	}
	return scanner.Err()
}

type demultiplexer struct {
	ctx       *actor.Context
	stdType   stdcopy.StdType
	recipient *actor.Ref
}

func (d demultiplexer) Write(p []byte) (n int, err error) {
	d.ctx.Tell(d.recipient, aproto.ContainerLog{
		Timestamp: time.Now().UTC(),
		RunMessage: &aproto.RunMessage{
			Value:   string(p),
			StdType: d.stdType,
		},
	})
	return len(p), nil
}

func trackLogs(
	ctx *actor.Context, docker *client.Client, containerID string, recipient *actor.Ref,
) error {
	logs, lErr := docker.ContainerLogs(
		context.Background(),
		containerID,
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
	if lErr != nil {
		return errors.Wrap(lErr, "error grabbing container logs")
	}

	stdout := demultiplexer{ctx: ctx, stdType: stdcopy.Stdout, recipient: recipient}
	stderr := demultiplexer{ctx: ctx, stdType: stdcopy.Stderr, recipient: recipient}
	if _, lErr = stdcopy.StdCopy(stdout, stderr, logs); lErr != nil {
		return errors.Wrap(lErr, "error scanning logs")
	}
	if lErr = logs.Close(); lErr != nil {
		return errors.Wrap(lErr, "error closing log stream")
	}
	return nil
}
