package config

import (
	"encoding/json"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	singularity = "singularity"
	podman      = "podman"
	enroot      = "enroot"
)

// DispatcherResourceManagerConfig is the object that stores the values of
// the "resource_manager" section of "tools/devcluster.yaml".
type DispatcherResourceManagerConfig struct {
	MasterHost                 string       `json:"master_host"`
	MasterPort                 int          `json:"master_port"`
	LauncherHost               string       `json:"host"`
	LauncherPort               int          `json:"port"`
	LauncherProtocol           string       `json:"protocol"`
	SlotType                   *device.Type `json:"slot_type"`
	LauncherAuthFile           string       `json:"auth_file"`
	LauncherContainerRunType   string       `json:"container_run_type"`
	RendezvousNetworkInterface string       `json:"rendezvous_network_interface"`
	ProxyNetworkInterface      string       `json:"proxy_network_interface"`
	// Configuration parameters that are proxies for launcher.conf
	// and will be applied there by the init script.
	UserName                   string  `json:"user_name"`
	GroupName                  string  `json:"group_name"`
	SingularityImageRoot       string  `json:"singularity_image_root"`
	JobStorageRoot             string  `json:"job_storage_root"`
	Path                       string  `json:"path"`
	LdLibraryPath              string  `json:"ld_library_path"`
	TresSupported              bool    `json:"tres_supported"`
	GresSupported              bool    `json:"gres_supported"`
	DefaultAuxResourcePool     *string `json:"default_aux_resource_pool"`
	DefaultComputeResourcePool *string `json:"default_compute_resource_pool"`

	Security           *DispatcherSecurityConfig                     `json:"security"`
	PartitionOverrides map[string]DispatcherPartitionOverrideConfigs `json:"partition_overrides"`
}

// DispatcherSecurityConfig configures security-related options for the elastic logging backend.
type DispatcherSecurityConfig struct {
	TLS model.TLSClientConfig `json:"tls"`
}

// Validate performs validation.
func (c DispatcherResourceManagerConfig) Validate() []error {
	// Allowed values for the container run type are either 'singularity', 'podman' or 'enroot'
	if !(c.LauncherContainerRunType == singularity ||
		c.LauncherContainerRunType == podman ||
		c.LauncherContainerRunType == enroot) {
		return []error{fmt.Errorf("invalid launch container run type: '%s'", c.LauncherContainerRunType)}
	}
	return nil
}

var defaultDispatcherResourceManagerConfig = DispatcherResourceManagerConfig{
	TresSupported:            true,
	GresSupported:            true,
	LauncherContainerRunType: singularity,
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *DispatcherResourceManagerConfig) UnmarshalJSON(data []byte) error {
	*c = defaultDispatcherResourceManagerConfig
	type DefaultParser *DispatcherResourceManagerConfig
	return json.Unmarshal(data, DefaultParser(c))
}

// ResolveSlotType resolves the slot type by first looking for a partition-specific setting,
// then falling back to the master config, and finally falling back to what we can infer.
func (c DispatcherResourceManagerConfig) ResolveSlotType(partition string) *device.Type {
	for name, overrides := range c.PartitionOverrides {
		if name != partition {
			continue
		}
		if overrides.SlotType == nil {
			break
		}
		return overrides.SlotType
	}
	return c.SlotType
}

// ResolveRendezvousNetworkInterface resolves the rendezvous network interface by first looking for
// a partition-specific setting and then falling back to the master config.
func (c DispatcherResourceManagerConfig) ResolveRendezvousNetworkInterface(
	partition string,
) string {
	for name, overrides := range c.PartitionOverrides {
		if name != partition {
			continue
		}
		if overrides.RendezvousNetworkInterface == nil {
			break
		}
		return *overrides.RendezvousNetworkInterface
	}
	return c.RendezvousNetworkInterface
}

// ResolveProxyNetworkInterface resolves the proxy network interface by first looking for a
// partition-specific setting and then falling back to the master config.
func (c DispatcherResourceManagerConfig) ResolveProxyNetworkInterface(partition string) string {
	for name, overrides := range c.PartitionOverrides {
		if name != partition {
			continue
		}
		if overrides.ProxyNetworkInterface == nil {
			break
		}
		return *overrides.ProxyNetworkInterface
	}
	return c.ProxyNetworkInterface
}

// ResolveTaskContainerDefaults resolves the task container defaults by first looking for
// a partition-specific setting and then falling back to the master config.
func (c DispatcherResourceManagerConfig) ResolveTaskContainerDefaults(
	partition string,
) *model.TaskContainerDefaultsConfig {
	for name, overrides := range c.PartitionOverrides {
		if name != partition {
			continue
		}
		if overrides.TaskContainerDefaultsConfig == nil {
			break
		}
		return overrides.TaskContainerDefaultsConfig
	}
	return nil
}

// DispatcherPartitionOverrideConfigs describes per-partition overrides.
type DispatcherPartitionOverrideConfigs struct {
	//nolint:lll // I honestly don't know how to break this line within Go's grammar.
	RendezvousNetworkInterface  *string                            `json:"rendezvous_network_interface"`
	ProxyNetworkInterface       *string                            `json:"proxy_network_interface"`
	SlotType                    *device.Type                       `json:"slot_type"`
	TaskContainerDefaultsConfig *model.TaskContainerDefaultsConfig `json:"task_container_defaults"`
}
