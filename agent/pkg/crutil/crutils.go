package crutil

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

var logLevel = regexp.MustCompile("(?P<level>INFO|WARN|ERROR|FATAL):    (?P<log>.*)")

// ShipPodmanCmdLogs does what you might expect from its name.
func ShipPodmanCmdLogs(
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
		if matches := logLevel.FindStringSubmatch(line); len(matches) == 3 {
			level, log = matches[1], matches[2]
		} else {
			level, log = model.LogLevelInfo, line
		}

		if err := p.Publish(ctx, docker.NewTypedLogEvent(level, log, stdtype)); err != nil {
			logrus.WithError(err).Trace("log stream terminated")
			return
		}
	}
}

// PprintCommand provides pretty printing of the given command to trace & log outputs.
func PprintCommand(
	ctx context.Context,
	command string,
	args []string,
	p events.Publisher[docker.Event],
	log *logrus.Entry,
) error {
	toPrint := command
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

	log.Trace(toPrint)
	if err := p.Publish(ctx, docker.NewLogEvent(
		model.LogLevelDebug,
		toPrint,
	)); err != nil {
		return err
	}
	return nil
}

// CanonicalizeImage returns the canonicalized image name.
func CanonicalizeImage(image string) string {
	url, err := url.Parse(image)
	isURIForm := err == nil
	isFSForm := path.IsAbs(image)
	if isFSForm || (isURIForm && url.Scheme != "") {
		return image
	}
	return fmt.Sprintf("docker://%s", image)
}

// PullImage implements code sharing for singularity & podman.
func PullImage(
	ctx context.Context,
	req docker.PullImage,
	p events.Publisher[docker.Event],
	wg waitgroupx.Group,
	log logrus.Entry,
	getPullCommand func(docker.PullImage, string) (string, []string),
) (err error) {
	if err = p.Publish(ctx, docker.NewBeginStatsEvent(docker.ImagePullStatsKind)); err != nil {
		return err
	}
	defer func() {
		if err = p.Publish(ctx, docker.NewEndStatsEvent(docker.ImagePullStatsKind)); err != nil {
			log.WithError(err).Warn("did not send image pull done stats")
		}
	}()

	image := CanonicalizeImage(req.Name)

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

	// TODO(DET-9078): Support registry auth. Investigate other auth mechanisms with podman.
	// args := []string{"pull"}
	// if req.ForcePull {
	// 	args = append(args, "--force") // Use 'podman image rm'?
	// }
	// args = append(args, image)
	command, args := getPullCommand(req, image)

	if err = PprintCommand(ctx, command, args, p, &log); err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, command, args...) // #nosec G204 'command' is under our control
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	// The return codes from `podman pull` aren't super helpful in determining the error, so we
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
	wg.Go(func(ctx context.Context) { ShipPodmanCmdLogs(ctx, stdout, stdcopy.Stdout, p) })
	wg.Go(func(ctx context.Context) {
		defer close(ignoreErrorsSig)
		ShipPodmanCmdLogs(ctx, stderr, stdcopy.Stderr, checkIgnoreErrors)
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
