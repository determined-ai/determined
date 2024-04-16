package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	singularity = "singularity"
	podman      = "podman"
	enroot      = "enroot"
)

// job labeling modes.
const (
	Project     = "project"
	Workspace   = "workspace"
	Label       = "label"
	LabelPrefix = "label:"
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
	UserName             string `json:"user_name"`
	GroupName            string `json:"group_name"`
	SingularityImageRoot string `json:"singularity_image_root"`
	ApptainerImageRoot   string `json:"apptainer_image_root"`
	JobStorageRoot       string `json:"job_storage_root"`
	Path                 string `json:"path"`
	LdLibraryPath        string `json:"ld_library_path"`
	LauncherJvmArgs      string `json:"launcher_jvm_args"`
	SudoAuthorized       string `json:"sudo_authorized"`
	// Configuration parameters handled by DispatchRM within master
	TresSupported              bool    `json:"tres_supported"`
	GresSupported              bool    `json:"gres_supported"`
	DefaultAuxResourcePool     *string `json:"default_aux_resource_pool"`
	DefaultComputeResourcePool *string `json:"default_compute_resource_pool"`
	JobProjectSource           *string `json:"job_project_source"`

	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`

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
	if c.ApptainerImageRoot != "" && c.SingularityImageRoot != "" {
		return []error{fmt.Errorf("apptainer_image_root and singularity_image_root cannot be both set")}
	}
	if c.SlotType != nil {
		switch *c.SlotType {
		case device.CPU, device.CUDA, device.ROCM:
			break
		default:
			return []error{fmt.Errorf(
				"invalid slot_type '%s'.  Specify one of cuda, rocm, or cpu", *c.SlotType)}
		}
	}

	return c.validateJobProjectSource()
}

func (c DispatcherResourceManagerConfig) validateJobProjectSource() []error {
	switch {
	case c.JobProjectSource == nil:
	case *c.JobProjectSource == Project:
	case *c.JobProjectSource == Workspace:
	case *c.JobProjectSource == Label:
	case strings.HasPrefix(*c.JobProjectSource, LabelPrefix):
	default:
		return []error{fmt.Errorf(
			"invalid job_project_source value: '%s'. "+
				"Specify one of project, workspace or label[:value]",
			*c.JobProjectSource)}
	}
	return nil
}

var defaultDispatcherResourceManagerConfig = DispatcherResourceManagerConfig{
	LauncherPort:             8181,
	LauncherProtocol:         "http",
	TresSupported:            true,
	GresSupported:            true,
	LauncherContainerRunType: singularity,
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *DispatcherResourceManagerConfig) UnmarshalJSON(data []byte) error {
	*c = defaultDispatcherResourceManagerConfig
	type DefaultParser *DispatcherResourceManagerConfig
	if err := json.Unmarshal(data, DefaultParser(c)); err != nil {
		return err
	}
	if c.ApptainerImageRoot != "" {
		c.SingularityImageRoot = c.ApptainerImageRoot
	}
	if c.LauncherContainerRunType == "apptainer" {
		c.LauncherContainerRunType = "singularity"
	}
	return nil
}

// ResolveSlotType resolves the slot type by first looking for a partition-specific setting,
// then falling back to the master config, and finally falling back to what we can infer.
func (c DispatcherResourceManagerConfig) ResolveSlotType(partition string) *device.Type {
	return c.resolveSlotTypeWithDefault(partition, c.SlotType)
}

// ResolveSlotTypeFromOverrides scans the available partition overrides for a slot type
// definition for the specified partition.
func (c DispatcherResourceManagerConfig) ResolveSlotTypeFromOverrides(
	partition string,
) *device.Type {
	return c.resolveSlotTypeWithDefault(partition, nil)
}

func (c DispatcherResourceManagerConfig) resolveSlotTypeWithDefault(
	partition string, defaultResult *device.Type,
) *device.Type {
	for name, overrides := range c.PartitionOverrides {
		if name != partition {
			continue
		}
		if overrides.SlotType == nil {
			break
		}
		return overrides.SlotType
	}
	return defaultResult
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
		if !strings.EqualFold(name, partition) {
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
	Description                 string                             `json:"description"`
}
