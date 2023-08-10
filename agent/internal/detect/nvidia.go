package detect

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/device"
)

var (
	detectMIGEnabled = []string{
		"nvidia-smi", "--query-gpu=mig.mode.current", "--format=csv,noheader",
	}
	detectMIGRegExp    = regexp.MustCompile(`(?P<dev>MIG \S+).+\(UUID.+(?P<uuid>MIG.+)\)`)
	detectCudaDevices  = []string{"nvidia-smi", "-L"} // Lists both GPUs and MIG instances
	detectCudaGPUsArgs = []string{
		"nvidia-smi", "--query-gpu=index,name,uuid", "--format=csv,noheader",
	}
	detectCudaGPUsIDFlagTpl = "--id=%v"
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
		log.Warn("empty nvidia-smi driver version")
		return "", nil
	case err != nil:
		return "", errors.Wrap(err, "error parsing output of nvidia-smi as csv")
	case len(record) != 1:
		return "", errors.New(
			"error parsing output of nvidia-smi; GPU record should have exactly 1 field")
	}
	return record[0], nil
}

// detectCudaGPUs returns the list of available Nvidia GPUs.
func detectCudaGPUs(visibleGPUs string) ([]device.Device, error) {
	devices, err := detectMigInstances(visibleGPUs)
	if err == nil && devices != nil && len(devices) > 0 {
		return devices, nil
	}

	flags := detectCudaGPUsArgs[1:]
	if visibleGPUs != "" {
		flags = append(flags, fmt.Sprintf(detectCudaGPUsIDFlagTpl, visibleGPUs))
	}

	// #nosec G204
	cmd := exec.Command(detectCudaGPUsArgs[0], flags...)
	out, err := cmd.Output()

	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return nil, nil
	} else if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf(
			"error while executing nvidia-smi to detect GPUs")
		return nil, nil
	}

	devices = make([]device.Device, 0)

	r := csv.NewReader(strings.NewReader(string(out)))
	cudaVisibleDevices := parseVisibleDevices()
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
		if deviceNotAllocated(cudaVisibleDevices, record) {
			continue // skip device outside of our allocation
		}
		index, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			return nil, errors.Wrap(
				err, "error parsing output of nvidia-smi; index of GPU cannot be converted to int")
		}

		brand := strings.TrimSpace(record[1])
		uuid := strings.TrimSpace(record[2])

		devices = append(devices, device.Device{
			ID:    device.ID(index),
			Brand: brand,
			UUID:  uuid,
			Type:  device.CUDA,
		})
	}
}

func parseVisibleDevices() []string {
	devices, found := os.LookupEnv("CUDA_VISIBLE_DEVICES")
	if !found {
		return nil
	}
	log.Tracef("CUDA_VISIBLE_DEVICES: %s", devices)
	return strings.Split(devices, ",")
}

func deviceNotAllocated(devices []string, device []string) bool {
	if devices == nil {
		return false
	}
	for _, d := range devices {
		if d == device[0] {
			return false
		}
	}
	return true
}

// detect if MIG is enabled and if there are instances configured.
func detectMigInstances(visibleGPUs string) ([]device.Device, error) {
	// Fail fast if MIG isn't even enabled
	// #nosec G204
	cmd := exec.Command(detectMIGEnabled[0], detectMIGEnabled[1:]...)
	out, err := cmd.Output()
	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return nil, nil
	} else if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf(
			"error while executing nvidia-smi to detect MIG mode")
		return nil, nil
	}
	if !strings.HasPrefix(string(out), "Enabled") {
		return nil, nil
	}

	// #nosec G204
	cmd = exec.Command(detectCudaDevices[0], detectCudaDevices[1:]...)
	out, err = cmd.Output()
	if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf(
			"error while executing nvidia-smi to detect MIG instances")
		return nil, nil
	}

	devices := make([]device.Device, 0)
	deviceIndex := 0

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()

		if detectMIGRegExp.MatchString(line) {
			matches := detectMIGRegExp.FindStringSubmatch(line)
			if len(matches) != 3 {
				continue
			}
			brand := matches[1]
			uuid := matches[2]
			devices = append(
				devices,
				device.Device{
					ID:    device.ID(deviceIndex),
					Brand: brand,
					UUID:  uuid,
					Type:  device.CUDA,
				},
			)
			deviceIndex++
		}
	}
	return devices, nil
}
