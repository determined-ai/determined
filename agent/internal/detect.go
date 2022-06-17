package internal

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/device"
)

const (
	osDarwin = "darwin"
)

func (a *agent) detect() error {
	switch {
	case a.ArtificialSlots > 0:
		h := sha256.New()
		h.Write([]byte(a.AgentID))
		rndSource, bytesRead := binary.Varint(h.Sum(nil)[:8])
		if bytesRead <= 0 {
			return fmt.Errorf(
				"failed to init random source for artificial slots ids. bytes read: %d", bytesRead)
		}
		rnd := rand.New(rand.NewSource(rndSource)) // nolint:gosec
		for i := 0; i < a.ArtificialSlots; i++ {
			u, err := uuid.NewRandomFromReader(rnd)
			if err != nil {
				return err
			}
			id := u.String()
			a.Devices = append(a.Devices, device.Device{
				ID: device.ID(i), Brand: "Artificial", UUID: id, Type: device.CPU,
			})
		}
	case a.SlotType == "none":
		a.Devices = []device.Device{}
	case a.SlotType == "cuda" || a.SlotType == "gpu":
		// Support "gpu" for backwards compatibility.
		devices, err := detectCudaGPUs(a.Options.VisibleGPUs)
		if err != nil {
			return errors.Wrap(err, "error while gathering GPU info through nvidia-smi command")
		}

		a.Devices = devices
	case a.SlotType == "rocm":
		devices, err := detectRocmGPUs(a.Options.VisibleGPUs)
		if err != nil {
			return errors.Wrap(err, "error while gathering GPU info through rocm-smi command")
		}

		a.Devices = devices
	case a.SlotType == "cpu":
		devices, err := detectCPUs()
		if err != nil {
			return err
		}
		a.Devices = devices
	case a.SlotType == "auto":
		devices, err := detectCudaGPUs(a.Options.VisibleGPUs)
		if err != nil {
			return errors.Wrap(err, "error while gathering GPU info through nvidia-smi command")
		}
		if len(devices) == 0 {
			devices, err = detectRocmGPUs(a.Options.VisibleGPUs)
			if err != nil {
				return errors.Wrap(err, "error while gathering GPU info through rocm-smi command")
			}
		}
		if len(devices) == 0 {
			devices, err = detectCPUs()
			if err != nil {
				return err
			}
		}
		a.Devices = devices
	default:
		panic("unrecognized slot type")
	}
	return nil
}

// detectCPUs returns the list of available CPUs; all the cores are returned as a single device.
func detectCPUs() ([]device.Device, error) {
	switch cpuInfo, err := cpu.Info(); {
	case err != nil:
		// Apple M1 does not report CPU frequency,
		// and that can result in an error here
		if errno, ok := err.(syscall.Errno); ok {
			switch errno {
			case syscall.ENOENT:
				if runtime.GOARCH == "arm64" && runtime.GOOS == osDarwin {
					return []device.Device{{ID: 0, Brand: "Apple", UUID: "AppleSilicon", Type: device.CPU}}, nil
				}
				return nil, errors.Wrap(err, "error while gathering CPU info on a non-AppleM1 system")
			default:
				return nil, errors.Wrap(err, "error while gathering CPU info")
			}
		}
		return nil, errors.Wrap(err, "error while gathering CPU info")
	case len(cpuInfo) == 0:
		return nil, errors.New("no CPUs detected")
	default:
		// Use uuid from the first `cpuinfo` entry.
		// All cores are exposed as a single slot; we aggregate the core counts by model name
		// to produce a display string for device description.
		uuid := cpuInfo[0].VendorID

		coreCounts := map[string]int32{}
		for _, entry := range cpuInfo {
			coreCounts[entry.ModelName] += entry.Cores
		}

		brands := []string{}
		for modelName := range coreCounts {
			brands = append(brands, fmt.Sprintf("%s x %d cores", modelName, coreCounts[modelName]))
		}

		brand := strings.Join(brands, ", ")
		return []device.Device{{ID: 0, Brand: brand, UUID: uuid, Type: device.CPU}}, nil
	}
}

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
			devices = append(devices,
				device.Device{ID: device.ID(deviceIndex), Brand: brand, UUID: uuid, Type: device.CUDA})
			deviceIndex++
		}
	}
	return devices, nil
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

		devices = append(devices, device.Device{
			ID:    device.ID(index),
			Brand: brand,
			UUID:  uuid,
			Type:  device.CUDA,
		})
	}
}
