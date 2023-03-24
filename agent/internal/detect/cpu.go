package detect

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"

	"github.com/determined-ai/determined/master/pkg/device"
)

const (
	osDarwin = "darwin"
)

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
					return []device.Device{
						{ID: 0, Brand: "Apple", UUID: "AppleSilicon", Type: device.CPU},
					}, nil
				}
				return nil, errors.Wrap(
					err,
					"error while gathering CPU info on a non-AppleM1 system",
				)
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
