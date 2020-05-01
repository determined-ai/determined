package internal

import (
	"encoding/csv"
	"io"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	unknownNvidiaVersion = "Unknown"
)

func getNvidiaVersion() (string, error) {
	// #nosec G204
	cmd := exec.Command("nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader")
	out, err := cmd.Output()

	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return "", nil
	} else if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf("error while executing nvidia-smi")
		return "", nil
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
