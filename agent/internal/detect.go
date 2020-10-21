package internal

import (
	"encoding/csv"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/device"
)

// detectCPUs returns the list of available CPUs; each core is returned as a single device.
func detectCPUs() ([]device.Device, error) {
	switch cpuInfo, err := cpu.Info(); {
	case err != nil:
		return nil, errors.Wrap(err, "error while gathering CPU info")
	case len(cpuInfo) == 0:
		return nil, errors.New("no CPUs detected")
	default:
		brand := fmt.Sprintf("%s x %d physical cores", cpuInfo[0].ModelName, cpuInfo[0].Cores)
		uuid := cpuInfo[0].VendorID
		return []device.Device{{ID: 0, Brand: brand, UUID: uuid, Type: device.CPU}}, nil
	}
}

var detectGPUsArgs = []string{"nvidia-smi", "--query-gpu=index,name,uuid", "--format=csv,noheader"}
var detectGPUsIDFlagTpl = "--id=%v"

// detectGPUs returns the list of available Nvidia GPUs.
func detectGPUs(visibleGPUs string) ([]device.Device, error) {
	flags := detectGPUsArgs[1:]
	if visibleGPUs != "" {
		flags = append(flags, fmt.Sprintf(detectGPUsIDFlagTpl, visibleGPUs))
	}

	// #nosec G204
	cmd := exec.Command(detectGPUsArgs[0], flags...)
	out, err := cmd.Output()

	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return nil, nil
	} else if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf("error while executing nvidia-smi")
		return nil, nil
	}

	devices := make([]device.Device, 0)

	r := csv.NewReader(strings.NewReader(string(out)))
	for {
		record, err := r.Read()
		switch {
		case err == io.EOF:
			return devices, nil
		case err != nil:
			return nil, errors.Wrap(err, "error parsing output of nvidia-smi as CSV")
		case len(record) != 3:
			return nil, errors.New(
				"error parsing output of nvidia-smi; GPU record should have exactly 3 fields")
		}

		index, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			return nil, errors.Wrap(
				err, "error parsing output of nvidia-smi; index of GPU cannot be converted to int")
		}

		brand := strings.TrimSpace(record[1])
		uuid := strings.TrimSpace(record[2])

		devices = append(devices, device.Device{ID: index, Brand: brand, UUID: uuid, Type: device.GPU})
	}
}
