package device

import (
	"fmt"

	"github.com/determined-ai/determined/proto/pkg/devicev1"
)

// Type is a string holding the type of the Device.
type Type string

const (
	// CPU represents a CPU device.
	CPU Type = "cpu"
	// CUDA represents a CUDA device.
	CUDA Type = "cuda"
	// ROCM represents an AMD GPU device.
	ROCM Type = "rocm"
	// ZeroSlot represents cpu devices on agents where only GPUs are modeled.
	ZeroSlot Type = ""
)

// Proto returns the proto representation of the device type.
func (t Type) Proto() devicev1.Type {
	switch t {
	case CPU:
		return devicev1.Type_TYPE_CPU
	case CUDA:
		return devicev1.Type_TYPE_CUDA
	case ROCM:
		return devicev1.Type_TYPE_ROCM
	case ZeroSlot:
		return devicev1.Type_TYPE_UNSPECIFIED
	default:
		return devicev1.Type_TYPE_UNSPECIFIED
	}
}

// ID the type of Device.ID.
type ID int

// Device represents a single computational device on an agent.
type Device struct {
	ID    ID     `json:"id"`
	Brand string `json:"brand"`
	UUID  string `json:"uuid"`
	Type  Type   `json:"type"`
}

func (d *Device) String() string {
	return fmt.Sprintf("%s%d (%s)", d.Type, d.ID, d.Brand)
}

// Proto returns the proto representation of the device.
func (d *Device) Proto() *devicev1.Device {
	if d == nil {
		return nil
	}
	return &devicev1.Device{
		Id:    int32(d.ID),
		Brand: d.Brand,
		Uuid:  d.UUID,
		Type:  d.Type.Proto(),
	}
}

// Devices is a slice of Device objects. Primarily useful for its methods.
type Devices []Device

// Proto converts Devices into its protobuf representation.
func (ds Devices) Proto() []*devicev1.Device {
	dp := make([]*devicev1.Device, len(ds))
	for i, d := range ds {
		dp[i] = d.Proto()
	}
	return dp
}
