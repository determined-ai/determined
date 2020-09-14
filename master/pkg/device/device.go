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
	// GPU represents a GPU device.
	GPU Type = "gpu"
	// ZeroSlot represents a unspecified device.
	ZeroSlot Type = ""
)

// Proto returns the proto representation of the device type.
func (t Type) Proto() devicev1.Type {
	switch t {
	case CPU:
		return devicev1.Type_TYPE_CPU
	case GPU:
		return devicev1.Type_TYPE_GPU
	case ZeroSlot:
		return devicev1.Type_TYPE_UNSPECIFIED
	default:
		return devicev1.Type_TYPE_UNSPECIFIED
	}
}

// Device represents a single computational device on an agent.
type Device struct {
	ID    int    `json:"id"`
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
