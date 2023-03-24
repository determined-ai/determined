package detect

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/device"
)

// Detect the devices available. If artificial devices are configured, prefers those, otherwise,
// we detect cuda, rocm, cpu (or no) devices based on the configured slot type.
func Detect(slotType, agentID, visibleGPUs string, artificialSlots int) ([]device.Device, error) {
	// Log detected nvidia version.
	v, err := getNvidiaVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get nvidia version: %w", err)
	} else if v != "" {
		log.Infof("Nvidia driver version: %s", v)
	}

	// Log detected rocm version.
	v, err = getRocmVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get rocm version: %w", err)
	} else if v != "" {
		log.Infof("Rocm driver version: %s", v)
	}

	// Detect devices available to the agent.
	var detected []device.Device
	switch {
	case artificialSlots > 0:
		// Generate random UUIDs consistent across agent restarts as long as
		// agentID is the same.
		rnd, sErr := randFromString(agentID)
		if sErr != nil {
			return nil, sErr
		}

		for i := 0; i < artificialSlots; i++ {
			u, rErr := uuid.NewRandomFromReader(rnd)
			if rErr != nil {
				return nil, rErr
			}
			id := u.String()
			detected = append(detected, device.Device{
				ID: device.ID(i), Brand: "Artificial", UUID: id, Type: device.CPU,
			})
		}
	case slotType == "none":
		detected = []device.Device{}
	case slotType == "cuda" || slotType == "gpu":
		// Support "gpu" for backwards compatibility.
		detected, err = detectCudaGPUs(visibleGPUs)
		if err != nil {
			return nil, errors.Wrap(
				err,
				"error while gathering GPU info through nvidia-smi command",
			)
		}
	case slotType == "rocm":
		detected, err = detectRocmGPUs(visibleGPUs)
		if err != nil {
			return nil, errors.Wrap(err, "error while gathering GPU info through rocm-smi command")
		}
	case slotType == "cpu":
		detected, err = detectCPUs()
		if err != nil {
			return nil, err
		}
	case slotType == "auto":
		detected, err = detectCudaGPUs(visibleGPUs)
		if err != nil {
			return nil, errors.Wrap(
				err,
				"error while gathering GPU info through nvidia-smi command",
			)
		}
		if len(detected) == 0 {
			detected, err = detectRocmGPUs(visibleGPUs)
			if err != nil {
				return nil, errors.Wrap(
					err,
					"error while gathering GPU info through rocm-smi command",
				)
			}
		}
		if len(detected) == 0 {
			detected, err = detectCPUs()
			if err != nil {
				return nil, err
			}
		}
	default:
		panic("unrecognized slot type")
	}

	log.Info("detected compute devices:")
	for _, d := range detected {
		log.Infof("\t%s", d.String())
	}

	return detected, nil
}

// randFromString returns a random-number generated seeded from an input string.
func randFromString(seed string) (*rand.Rand, error) {
	h := sha256.New()
	h.Write([]byte(seed))
	rndSource, bytesRead := binary.Varint(h.Sum(nil)[:8])
	if bytesRead <= 0 {
		return nil, fmt.Errorf(
			"failed to init random source for artificial slots ids. bytes read: %d", bytesRead)
	}
	rnd := rand.New(rand.NewSource(rndSource)) // nolint:gosec
	return rnd, nil
}
