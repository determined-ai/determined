package detect

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/device"
)

func getRocmVersion() (string, error) {
	cmd := exec.Command("rocm-smi", "--showdriverversion", "--csv")
	out, err := cmd.Output()

	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return "", nil
	} else if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf("error while executing rocm-smi")
		return "", nil
	}

	r := csv.NewReader(strings.NewReader(string(out)))

	var record []string

	// First line is the header, second line is data.
	// Example input to be parsed:
	//
	// device,Driver version
	// cardsystem,5.11.32.21.40
	//
	for i := 0; i < 2; i++ {
		record, err = r.Read()
		switch {
		case err == io.EOF:
			return "", errors.New("empty rocm-smi output")
		case err != nil:
			return "", errors.Wrap(err, "error parsing output of rocm-smi as csv")
		case len(record) != 2:
			return "", errors.New(
				"error parsing output of rocm-smi; GPU record should have exactly 1 field")
		case i == 0:
			continue
		}
	}

	return record[1], nil
}

// RocmDevice metadata.
type RocmDevice struct {
	UUID       string `json:"Unique ID"`
	CardSKU    string `json:"Card SKU"`
	CardVendor string `json:"Card vendor"`
	CardModel  string `json:"Card model"`
	PCIBus     string `json:"PCI Bus"`
	Index      int
}

// Cache discovered devices for runtime lookups.
var discoveredRocmDevices []RocmDevice

func parseRocmSmi(jsonData []byte) ([]RocmDevice, error) {
	parsed := map[string]RocmDevice{}
	err := json.Unmarshal(jsonData, &parsed)
	if err != nil {
		return nil, err
	}
	allocatedDevices := parseVisibleDevices()
	result := []RocmDevice{}

	for k, d := range parsed {
		d.Index, err = strconv.Atoi(k[len("card"):])
		if err != nil {
			return nil, errors.Wrap(
				err, "failed to parse card index")
		}
		if !deviceAllocated(d.Index, allocatedDevices) {
			log.Tracef("Device not allocated: %d", d.Index)
			continue
		}
		result = append(result, d)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Index < result[j].Index
	})
	return result, nil
}

func deviceAllocated(deviceIndex int, allocatedDevices []string) bool {
	if allocatedDevices == nil {
		return true
	}
	for _, d := range allocatedDevices {
		if d == fmt.Sprintf("%d", deviceIndex) {
			return true
		}
	}
	return false
}

func parseVisibleDevices() []string {
	devices, found := os.LookupEnv("ROCR_VISIBLE_DEVICES")
	if !found {
		return nil
	}
	log.Tracef("ROCR_VISIBLE_DEVICES: '%s'", devices)
	return strings.Split(devices, ",")
}

func detectRocmGPUs(visibleGPUs string) ([]device.Device, error) {
	args := []string{"--showuniqueid", "--showproductname", "--showbus", "--json"}

	if visibleGPUs != "" {
		gpuIds := strings.Split(visibleGPUs, ",")
		args = append(args, "-d")
		args = append(args, gpuIds...)
	}

	cmd := exec.Command("rocm-smi", args...)

	out, err := cmd.Output()
	if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
		return nil, nil
	} else if err != nil {
		// An rocm-smi bug causes --showproductname to throw up if the info does not exist
		// As a workaround, try again without --showproductname
		for i, arg := range args {
			if arg == "--showproductname" {
				args = append(args[:i], args[i+1:]...)
				break
			}
		}

		cmd := exec.Command("rocm-smi", args...)

		out, err = cmd.Output()
		if execError, ok := err.(*exec.Error); ok && execError.Err == exec.ErrNotFound {
			return nil, nil
		} else if err != nil {
			log.WithError(err).WithField("output", string(out)).Warnf(
				"error while executing rocm-smi to detect GPUs")
			return nil, nil
		}
		log.Warn("rocm-smi detected a card without a product name, firmware issue possible")
	}

	discoveredRocmDevices, err = parseRocmSmi(out)
	if err != nil {
		log.WithError(err).WithField("output", string(out)).Warnf(
			"error while parsing rocm-smi output")
		return nil, nil
	}

	result := []device.Device{}

	for _, rocmDevice := range discoveredRocmDevices {
		result = append(result, device.Device{
			ID:    device.ID(rocmDevice.Index),
			Brand: rocmDevice.CardVendor,
			UUID:  rocmDevice.UUID,
			Type:  device.ROCM,
		})
	}

	return result, nil
}

// GetRocmDeviceByUUID gets a RocmDevice by UUID from the singleton discovered Rocm devices.
func GetRocmDeviceByUUID(uuid string) *RocmDevice {
	for _, d := range discoveredRocmDevices {
		if d.UUID == uuid {
			return &d
		}
	}

	return nil
}
