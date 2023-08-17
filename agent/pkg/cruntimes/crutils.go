package cruntimes

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/agent/pkg/docker"
	"github.com/determined-ai/determined/agent/pkg/events"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
)

var logLevel = regexp.MustCompile("(?P<level>INFO|WARN|ERROR|FATAL):    (?P<log>.*)")

// BaseTempDirName returns a per-user directory name that is unique
// for the use and specified id (agentID), but consistently named
// between agent runs to enable cleanup of earlier tmp files.
func BaseTempDirName(id string) (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to get username: %w", err)
	}

	return fmt.Sprintf("/tmp/determined-%s-%s", user.Username, id), nil
}

// ShipContainerCommandLogs forwards the given output stream to the specified publisher.
// It is used to reveal the result of container command lines, e.g. podman pull...
func ShipContainerCommandLogs(
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
	wg *waitgroupx.Group,
	log *logrus.Entry,
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

	// TODO(DET-9078): Support registry auth. Investigate other auth mechanisms
	// with singularity & podman.
	command, args := getPullCommand(req, image)

	if err = PprintCommand(ctx, command, args, p, log); err != nil {
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
	wg.Go(func(ctx context.Context) { ShipContainerCommandLogs(ctx, stdout, stdcopy.Stdout, p) })
	wg.Go(func(ctx context.Context) {
		defer close(ignoreErrorsSig)
		ShipContainerCommandLogs(ctx, stderr, stdcopy.Stderr, checkIgnoreErrors)
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

// ArchiveMountPoints places the experiment archives and returns a list of mount
// points to be made available inside the container.
func ArchiveMountPoints(ctx context.Context,
	req cproto.RunSpec,
	p events.Publisher[docker.Event],
	archivesPath string,
	log *logrus.Entry,
) ([]string, error) {
	for _, a := range req.Archives {
		src := filepath.Join(archivesPath, a.Path)
		if wErr := archive.Write(src, a.Archive, func(level, log string) error {
			return p.Publish(ctx, docker.NewLogEvent(level, log))
		}); wErr != nil {
			return nil, fmt.Errorf("writing archive for %s: %w", a.Path, wErr)
		}
	}
	// Do not mount top level dirs that are likely to conflict inside of the container, since
	// these mounts do not overlay. Instead, mount their children.
	ignoredPathPrefixes := []string{"/", "/etc", "/opt", "/run", "/etc/ssh"}
	var mountPoints []string
	// This depends on walkdir walking in lexical order, which is documented.
	if err := filepath.WalkDir(archivesPath, func(src string, d fs.DirEntry, err error) error {
		p := strings.TrimPrefix(src, archivesPath)

		for _, m := range mountPoints {
			if strings.HasPrefix(p, m) {
				return nil
			}
		}

		dirPaths := filepath.SplitList(p)
		prefix := ""

		for i := 0; i < len(dirPaths); i++ {
			prefix = filepath.Join(prefix, dirPaths[i])

			log.Trace("Checking mountPoint prefix {}", prefix)
			if !slices.Contains(ignoredPathPrefixes, prefix) {
				log.Trace("Add mountPoint {}", prefix)
				mountPoints = append(mountPoints, prefix)
				return nil
			}
		}
		log.Warnf("could not determine where to mount %s", src)
		return nil
	}); err != nil {
		return nil, err
	}
	return mountPoints, nil
}
