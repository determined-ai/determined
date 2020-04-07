package internal

import (
	"context"
	"encoding/csv"
	"io"
	"os/exec"
	"strings"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

const (
	nvidiaRuntime        = "nvidia"
	unknownNvidiaVersion = "Unknown"
)

func getNvidiaVersion() (string, error) {
	// #nosec G204
	cmd := exec.Command("nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader")
	out, err := cmd.Output()

	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return "", nil
	} else if err != nil {
		return "", errors.Wrapf(err, "error while executing nvidia-smi (output: %s)", string(out))
	}

	r := csv.NewReader(strings.NewReader(string(out)))
	record, err := r.Read()
	switch {
	case err == io.EOF:
		return unknownNvidiaVersion, nil
	case err != nil:
		return "", errors.Wrap(err, "error parsing output of nvidia-smi as csv")
	case len(record) != 1:
		return "", errors.New(
			"error parsing output of nvidia-smi; GPU record should have exactly 1 field")
	}
	return record[0], nil
}

func nvidiaRuntimeInstalled() (bool, error) {
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false, errors.Wrap(err, "error connecting to docker daemon")
	}

	info, err := c.Info(context.Background())
	if err != nil {
		return false, errors.Wrap(err, "error retrieving docker system info")
	}

	_, ok := info.Runtimes[nvidiaRuntime]
	return ok, nil
}
