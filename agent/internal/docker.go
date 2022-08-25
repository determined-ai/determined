package internal

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"syscall"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

type dockerActor struct {
	*client.Client
	credentialStores    map[string]*credentialStore
	authConfigs         map[string]types.AuthConfig
	reattachContainerID *cproto.ID
}

type (
	signalContainer struct {
		dockerID string
		signal   syscall.Signal
	}
	pullImage struct {
		cproto.PullSpec
		Name         string
		TaskType     string
		AllocationID model.AllocationID
	}
	runContainer struct {
		cproto.RunSpec
	}

	imagePulled      struct{}
	containerStarted struct {
		dockerID      string
		containerInfo types.ContainerJSON
	}
	containerReattached struct {
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

func (d *dockerActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		stores, auths, err := processDockerConfig()
		if err != nil {
			ctx.Log().Infof("couldn't process ~/.docker/config.json %v", err)
		}
		if len(stores) == 0 {
			ctx.Log().Info("can't find any docker credential stores, continuing without them")
		}
		if len(auths) == 0 {
			ctx.Log().Info("can't find any auths in ~/.docker/config.json, continuing without them")
		}
		d.credentialStores, d.authConfigs = stores, auths

		if d.reattachContainerID != nil {
			d.reattachContainer(ctx, *d.reattachContainerID)
		}
	case pullImage:
		go d.pullImage(ctx, msg)

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
			d.sendAuxLog(ctx, ptrs.Ptr(model.LogLevelInfo), fmt.Sprintf(
				"image present, but force_pull_image is set; checking for updates: %s",
				ref.String(),
			))
		}
	case err == nil:
		d.sendAuxLog(ctx, ptrs.Ptr(model.LogLevelInfo),
			fmt.Sprintf("image already found, skipping pull phase: %s", ref.String()))
		ctx.Tell(ctx.Sender(), imagePulled{})
		return
	case client.IsErrNotFound(err):
		d.sendAuxLog(ctx, ptrs.Ptr(model.LogLevelInfo),
			fmt.Sprintf("image not found, pulling image: %s", ref.String()))
	default:
		sendErr(ctx, errors.Wrapf(err, "error checking if image exists: %s", ref.String()))
		return
	}

	now := time.Now().UTC()
	ctx.Tell(ctx.Self().Parent(), aproto.ContainerStatsRecord{
		EndStats: false,
		TaskType: model.TaskType(msg.TaskType),
		Stats: &model.TaskStats{
			AllocationID: msg.AllocationID,
			EventType:    "IMAGEPULL",
			StartTime:    &now,
		},
	})

	defer func() {
		now := time.Now().UTC()
		ctx.Tell(ctx.Self().Parent(), aproto.ContainerStatsRecord{
			EndStats: true,
			TaskType: model.TaskType(msg.TaskType),
			Stats: &model.TaskStats{
				AllocationID: msg.AllocationID,
				EventType:    "IMAGEPULL",
				EndTime:      &now,
			},
		})
	}()

	auth, err := d.getDockerAuths(ctx, msg.Registry, ref)
	if err != nil {
		sendErr(ctx, errors.Wrap(err, "could not get docker authentication"))
		return
	}
	authString, err := registryToString(auth)
	if err != nil {
		sendErr(ctx, errors.Wrap(err, "error encoding docker credentials"))
		return
	}
	opts := types.ImagePullOptions{
		All:          false,
		RegistryAuth: authString,
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

func (d *dockerActor) getDockerAuths(
	ctx *actor.Context,
	expconfReg *types.AuthConfig,
	image reference.Named,
) (types.AuthConfig, error) {
	imageDomain := reference.Domain(image)
	// Try expconf registry auth config.
	if expconfReg != nil {
		didNotPassServerAddress := expconfReg.ServerAddress == ""
		if didNotPassServerAddress {
			d.sendAuxLog(ctx, ptrs.Ptr(model.LogLevelWarning), "setting registry_auth "+
				"without registry_auth.serveraddress is deprecated and will soon be required")
		}

		expconfDomain := registry.ConvertToHostname(expconfReg.ServerAddress)
		if expconfDomain == imageDomain ||
			didNotPassServerAddress { // TODO remove didNotPassServerAddress when it becomes required.
			return *expconfReg, nil
		}
		d.sendAuxLog(ctx, ptrs.Ptr(model.LogLevelWarning),
			fmt.Sprintf("not using expconfig registry_auth since expconf "+
				"registry_auth.serverAddress %s did not match the image serverAddress %s",
				expconfDomain, imageDomain))
	}

	// Try using credential stores specified in ~/.docker/config.json.
	if store, ok := d.credentialStores[imageDomain]; ok {
		creds, err := store.get()
		if err == nil {
			d.sendAuxLog(ctx, ptrs.Ptr(model.LogLevelInfo), fmt.Sprintf(
				"domain '%s' found in 'credHelpers' config. Using credentials helper.", imageDomain))
		}
		return creds, errors.Wrap(err, "unable to get credentials from helper")
	}

	// Finally try using auths section of users ~/.docker/config.json.
	index, err := registry.ParseSearchIndexInfo(image.String())
	if err != nil {
		return types.AuthConfig{}, errors.Wrap(err, "error invalid docker repo name")
	}
	reg := registry.ResolveAuthConfig(d.authConfigs, index)
	if reg != (types.AuthConfig{}) {
		d.sendAuxLog(ctx, ptrs.Ptr(model.LogLevelInfo), fmt.Sprintf(
			"domain '%s' found in 'auths' ~/.docker/config.json", imageDomain))
	}
	return reg, nil
}

func (d *dockerActor) runContainer(ctx *actor.Context, msg cproto.RunSpec) {
	response, err := d.ContainerCreate(
		context.Background(), &msg.ContainerConfig, &msg.HostConfig, &msg.NetworkingConfig, nil, "")
	if err != nil {
		sendErr(ctx, errors.Wrap(err, "error creating container"))
		return
	}
	containerID := response.ID
	for _, w := range response.Warnings {
		d.sendAuxLog(ctx, ptrs.Ptr(model.LogLevelWarning),
			fmt.Sprintf("warning when creating container: %s", w))
	}

	for _, copyArx := range msg.Archives {
		d.sendAuxLog(ctx, ptrs.Ptr(model.LogLevelInfo),
			fmt.Sprintf("copying files to container: %s", copyArx.Path))
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

	select {
	case err = <-eerr:
		sendErr(ctx, errors.Wrap(err, "error while waiting for container to exit"))
	case exit := <-exit:
		if exit.Error != nil {
			sendErr(ctx, fmt.Errorf("error receiving container exit: %s", exit.Error.Message))
			return
		}
		ctx.Tell(ctx.Sender(), containerTerminated{ExitCode: aproto.ExitCode(exit.StatusCode)})
	}
}

func (d *dockerActor) reattachContainer(ctx *actor.Context, id cproto.ID) {
	// Since this method is called in PreStart,
	// use actor parent instead of message sender to communicate.
	senderRef := ctx.Self().Parent()

	containers, err := d.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", dockerContainerIDLabel+"="+id.String()),
		),
	})
	if err != nil {
		sendErrParent(ctx, errors.Wrap(err, "error while reattaching container"))
		return
	}

	for _, cont := range containers {
		// Subscribe to termination notifications first.
		exit, eerr := d.ContainerWait(context.Background(), cont.ID, dcontainer.WaitConditionNextExit)

		// Restore containerInfo.
		containerInfo, err := d.ContainerInspect(context.Background(), cont.ID)
		if err != nil {
			sendErrParent(ctx, errors.Wrap(err, "error inspecting reattached container"))
			return
		}

		// Check if container has exited while we were trying to reattach it.
		if !containerInfo.State.Running {
			ctx.Tell(
				senderRef,
				containerTerminated{ExitCode: aproto.ExitCode(containerInfo.State.ExitCode)})
		} else {
			ctx.Tell(
				senderRef,
				containerReattached{dockerID: cont.ID, containerInfo: containerInfo},
			)
			select {
			case err = <-eerr:
				sendErrParent(ctx, errors.Wrap(err, "error while waiting for reattached container to exit"))
			case exit := <-exit:
				ctx.Tell(
					senderRef,
					containerTerminated{ExitCode: aproto.ExitCode(exit.StatusCode)})
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

func sendErrParent(ctx *actor.Context, err error) {
	ctx.Tell(ctx.Self().Parent(), dockerErr{Error: err})
}

func (d *dockerActor) sendAuxLog(ctx *actor.Context, level *string, msg string) {
	ctx.Tell(ctx.Sender(), aproto.ContainerLog{
		Timestamp:  time.Now().UTC(),
		Level:      level,
		AuxMessage: &msg,
	})
}

type pullInfo struct {
	DownloadStarted bool
	ExtractStarted  bool
	Total           int64
	Downloaded      int64
	Extracted       int64
}

type pullLogFormatter struct {
	Order   []string
	Known   map[string]*pullInfo
	Backoff time.Time
}

// renderProgress generates human-readable and log-file-friendly progress messages.
//
// Every layer goes through the following stages:
// - 1 Pulling fs layer (ID but no size)
// - 1 Waiting (ID but no size)
// - 1+ Downloading
// - 1 Verifying Checksum
// - 1 Download Complete
// - 1+ Extracting
// - 1 Pull Complete
//
// You can't really estimate global progress because the log stream doesn't tell you how big the
// full download size is at any point, it only tells you how big each layer is, and only when that
// layer starts downloading.  The downloads are staggered, so when many layers are present you
// wouldn't know the full download size until you're basically done.
//
// Showing a per-layer status bar is practically impossible without an interactive terminal (as
// docker run would have).
//
// So instead we create a weighted-average status bar, where every layer's download and extraction
// count as equal parts.  The status bar ends up pretty jerky but it still gives a "sensation" of
// progress; things don't look frozen, the user has a rough idea of how far along you are, and the
// logs are still sane afterwards.
func (f *pullLogFormatter) RenderProgress() string {
	var downloaded int64
	var extracted int64
	progress := 0.0
	for _, id := range f.Order {
		info := f.Known[id]
		downloaded += info.Downloaded
		extracted += info.Extracted
		switch {
		case !info.DownloadStarted:
			// no progress on this layer
		case info.Extracted == info.Total:
			// this layer is complete
			progress += 1.0
		case info.Downloaded == info.Total:
			// download complete, extraction in progress
			progress += 0.5 + 0.5*float64(info.Extracted)/float64(info.Total)
		default:
			progress += 0.5 * float64(info.Downloaded) / float64(info.Total)
		}
	}

	// Normalize by layer count.
	progress /= float64(len(f.Known))

	// 40-character progress bar
	prog := int(40.0 * progress)

	bar := ""
	for i := 0; i < 40; i++ {
		if i <= prog {
			if prog == 40 || i+1 <= prog {
				// Download is full, or middle of bar.
				bar += "="
			} else {
				// Boundary between bar and spaces.
				bar += ">"
			}
		} else {
			bar += " "
		}
	}

	return fmt.Sprintf(
		"[%v] Downloaded: %.1fMB, Extracted %.1fMB",
		bar,
		float64(downloaded)/1e6,
		float64(extracted)/1e6,
	)
}

func (f *pullLogFormatter) backoffOrRenderProgress() *string {
	// log at most one line every 1 second
	now := time.Now().UTC()
	if now.Before(f.Backoff) {
		return nil
	}
	f.Backoff = now.Add(1 * time.Second)

	progress := f.RenderProgress()
	return &progress
}

// Update returns nil or a rendered progress update for the end user.
func (f *pullLogFormatter) Update(msg jsonmessage.JSONMessage) *string {
	if msg.Error != nil {
		log.Errorf("%d: %v", msg.Error.Code, msg.Error.Message)
		return nil
	}

	var info *pullInfo
	var ok bool

	switch msg.Status {
	case "Pulling fs layer":
		fallthrough
	case "Waiting":
		if _, ok = f.Known[msg.ID]; !ok {
			// New layer!
			f.Known[msg.ID] = &pullInfo{}
			f.Order = append(f.Order, msg.ID)
		}
		return nil

	case "Downloading":
		if info, ok = f.Known[msg.ID]; !ok {
			log.Error("message ID not found for downloading message!")
			return nil
		}
		if info.ExtractStarted {
			log.Error("got downloading message after extraction started!")
			return nil
		}
		info.Downloaded = msg.Progress.Current
		// The first "Downloading" msg is important, as it gives us the layer size.
		if !info.DownloadStarted {
			info.DownloadStarted = true
			info.Total = msg.Progress.Total
		}
		return f.backoffOrRenderProgress()

	case "Extracting":
		if info, ok = f.Known[msg.ID]; !ok {
			log.Error("message ID not found for extracting message!")
			return nil
		}
		info.Extracted = msg.Progress.Current
		if !info.ExtractStarted {
			info.ExtractStarted = true
			// Forcibly mark Downloaded as completed.
			info.Downloaded = info.Total
		}
		return f.backoffOrRenderProgress()

	case "Pull complete":
		if info, ok = f.Known[msg.ID]; !ok {
			log.Error("message ID not found for completed message!")
			return nil
		}
		// Forcibly mark Extracted as completed.
		info.Extracted = info.Total
		return f.backoffOrRenderProgress()
	}

	return nil
}

func (d *dockerActor) sendPullLogs(ctx *actor.Context, r io.Reader) error {
	plf := pullLogFormatter{Known: map[string]*pullInfo{}}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log := jsonmessage.JSONMessage{}
		if err := json.Unmarshal(scanner.Bytes(), &log); err != nil {
			return errors.Wrapf(err, "error parsing log message: %#v", log)
		}

		logMsg := plf.Update(log)
		if logMsg != nil {
			ctx.Tell(ctx.Sender(), aproto.ContainerLog{
				Timestamp:   time.Now().UTC(),
				PullMessage: logMsg,
			})
		}
	}
	// Always print the complete progress bar, regardless of the backoff time.
	finalLogMsg := plf.RenderProgress()
	ctx.Tell(ctx.Sender(), aproto.ContainerLog{
		Timestamp:   time.Now().UTC(),
		PullMessage: &finalLogMsg,
	})
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
