package device

import "fmt"

// Type is a string holding the type of the Device.
type Type string

const (
	// CPU represents a CPU device.
	CPU Type = "cpu"
	// GPU represents a GPU device.
	GPU Type = "gpu"
)

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
