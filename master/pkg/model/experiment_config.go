package model

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/docker/go-units"

	"github.com/ghodss/yaml"

	"github.com/determined-ai/determined/master/pkg/check"
)

const (
	// MinUserSchedulingPriority is the smallest priority users may specify.
	MinUserSchedulingPriority = 1
	// MaxUserSchedulingPriority is the largest priority users may specify.
	MaxUserSchedulingPriority = 99
	defaultDeviceMode         = "mrw"
)

// DevicesConfig is the configuration for devices.  It is a named type because it needs custom
// merging behavior (via UnmarshalJSON).
type DevicesConfig []DeviceConfig

// UnmarshalJSON implements the json.Unmarshaler interface so that DeviceConfigs are additive.
func (d *DevicesConfig) UnmarshalJSON(data []byte) error {
	unmarshaled := make([]DeviceConfig, 0)
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		return errors.Wrap(err, "failed to parse devices")
	}

	// Prevent duplicate container paths as a result of the merge.  Prefer the unmarshaled devices
	// to the old ones since with this unmarshaling strategy we always unmarshal in order of
	// increasing priority.
	paths := map[string]bool{}
	for _, device := range unmarshaled {
		paths[device.ContainerPath] = true
	}
	for _, device := range *d {
		if _, ok := paths[device.ContainerPath]; !ok {
			unmarshaled = append(unmarshaled, device)
		}
	}

	*d = unmarshaled
	return nil
}

// DeviceConfig configures container device access.
type DeviceConfig struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Mode          string `json:"mode"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *DeviceConfig) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err == nil {
		fields := strings.Split(plain, ":")
		if len(fields) < 2 || len(fields) > 3 {
			return errors.Errorf("invalid device string: %q", plain)
		}
		d.HostPath = fields[0]
		d.ContainerPath = fields[1]
		if len(fields) > 2 {
			d.Mode = fields[2]
		} else {
			d.Mode = defaultDeviceMode
		}
		return nil
	}

	d.Mode = defaultDeviceMode
	type DefaultParser *DeviceConfig
	return errors.Wrap(json.Unmarshal(data, DefaultParser(d)), "failed to parse device")
}

// ResourcesConfig configures resource usage for an experiment, command, notebook, or tensorboard.
type ResourcesConfig struct {
	Slots int `json:"slots"`

	MaxSlots       *int         `json:"max_slots,omitempty"`
	Weight         float64      `json:"weight"`
	NativeParallel bool         `json:"native_parallel,omitempty"`
	ShmSize        *StorageSize `json:"shm_size,omitempty"`
	ResourcePool   string       `json:"resource_pool"`
	Priority       *int         `json:"priority,omitempty"`

	Devices DevicesConfig `json:"devices"`

	// Deprecated: Use ResourcePool instead.
	AgentLabel string `json:"agent_label,omitempty"`
}

// StorageSize is a named type for custom marshaling behavior for shm_size.
type StorageSize int64

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *StorageSize) UnmarshalJSON(data []byte) error {
	var size any
	if err := json.Unmarshal(data, &size); err != nil {
		return err
	}

	switch s := size.(type) {
	case float64:
		*d = StorageSize(s)
	case string:
		b, err := units.RAMInBytes(s)
		if err != nil {
			return errors.Wrap(err, "failed to parse shm_size")
		}
		*d = StorageSize(b)
	default:
		return errors.New("shm_size needs to be a string or numeric")
	}
	return nil
}

// ParseJustResources is a helper function for breaking the circular dependency where we need the
// TaskContainerDefaults to unmarshal an ExperimentConfig, but we need the Resources.ResourcePool
// setting to know which TaskContainerDefaults to use.  It does not throw errors; if unmarshalling
// fails that can just get caught later.
func ParseJustResources(configBytes []byte) ResourcesConfig {
	// Make this function usable on experiment or command configs.
	type DummyConfig struct {
		Resources ResourcesConfig `json:"resources"`
	}

	dummy := DummyConfig{
		Resources: ResourcesConfig{
			Slots: 1,
		},
	}

	// Don't throw errors; validation should happen elsewhere.
	_ = yaml.Unmarshal(configBytes, &dummy)

	return dummy.Resources
}

// ValidatePrioritySetting checks that priority if set is within a valid range.
func ValidatePrioritySetting(priority *int) []error {
	errs := make([]error, 0)

	if priority != nil {
		errs = append(errs, check.GreaterThanOrEqualTo(
			*priority, MinUserSchedulingPriority,
			"scheduling priority must be greater than 0 and less than 100"))
		errs = append(errs, check.LessThanOrEqualTo(
			*priority, MaxUserSchedulingPriority,
			"scheduling priority must be greater than 0 and less than 100"))
	}
	return errs
}

// Validate implements the check.Validatable interface.
func (r ResourcesConfig) Validate() []error {
	errs := []error{
		check.GreaterThanOrEqualTo(r.Slots, 0, "slots must be >= 0"),
		check.GreaterThan(r.Weight, float64(0), "weight must be > 0"),
	}
	errs = append(errs, ValidatePrioritySetting(r.Priority)...)
	return errs
}

// BindMountsConfig is the configuration for bind mounts.
type BindMountsConfig []BindMount

// UnmarshalJSON implements the json.Unmarshaler interface.
func (b *BindMountsConfig) UnmarshalJSON(data []byte) error {
	unmarshaled := make([]BindMount, 0)
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		return errors.Wrap(err, "failed to parse bind mounts")
	}

	// Prevent duplicate container paths as a result of the merge.  Prefer the unmarshaled bind
	// mounts to the old ones since with this unmarshaling strategy we always unmarshal in order of
	// increasing priority.
	paths := map[string]bool{}
	for _, mount := range unmarshaled {
		paths[mount.ContainerPath] = true
	}
	for _, mount := range *b {
		if _, ok := paths[mount.ContainerPath]; !ok {
			unmarshaled = append(unmarshaled, mount)
		}
	}

	*b = unmarshaled
	return nil
}

// BindMount configures trial runner filesystem bind mounts.
type BindMount struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	ReadOnly      bool   `json:"read_only"`
	Propagation   string `json:"propagation"`
}

// Validate implements the check.Validatable interface.
func (b BindMount) Validate() []error {
	return []error{
		check.True(b.ContainerPath != ".", "container_path must not be \".\""),
		check.True(filepath.IsAbs(b.HostPath), "host_path must be an absolute path"),
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (b *BindMount) UnmarshalJSON(data []byte) error {
	b.Propagation = "rprivate"
	type DefaultParser *BindMount
	return errors.Wrap(json.Unmarshal(data, DefaultParser(b)), "failed to parse bind mounts")
}
