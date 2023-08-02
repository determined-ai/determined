package expconf

import (
	"encoding/json"
	"fmt"

	k8sV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/device"
)

// PodSpec is just a k8sV1.Pod with custom methods, since k8sV1.Pod is not reflect-friendly.
type PodSpec k8sV1.Pod

// Copy implements the schemas.Copyable psuedointerface.
func (p PodSpec) Copy() PodSpec {
	k8sP := k8sV1.Pod(p)
	return PodSpec(*k8sP.DeepCopy())
}

// Merge implements the schemas.Mergable psuedointerface using Kubernetes strategic merging.
// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md
//
//nolint:lll
func (p PodSpec) Merge(other PodSpec) PodSpec {
	pBytes, err := json.Marshal(p)
	if err != nil {
		panic(err) // These errors shouldn't happen.
	}
	otherBytes, err := json.Marshal(other)
	if err != nil {
		panic(err)
	}

	// Receiver should overwrite the value.
	mergedBytes, err := strategicpatch.StrategicMergePatch(otherBytes, pBytes, p)
	if err != nil {
		panic(err)
	}

	var mergedSpec PodSpec
	if err := json.Unmarshal(mergedBytes, &mergedSpec); err != nil {
		panic(err)
	}
	return mergedSpec
}

// WithDefaults implements the schemas.Defaultable psuedointerface.
func (p PodSpec) WithDefaults() PodSpec {
	pod := k8sV1.Pod(p)
	return PodSpec(*pod.DeepCopy())
}

// EnvironmentConfigV0 configures the environment of a Determined command or experiment.
//
//go:generate ../gen.sh --import github.com/docker/docker/api/types
type EnvironmentConfigV0 struct {
	RawImage                *EnvironmentImageMapV0     `json:"image"`
	RawEnvironmentVariables *EnvironmentVariablesMapV0 `json:"environment_variables"`
	RawProxyPorts           *ProxyPortsConfigV0        `json:"proxy_ports"`

	RawPorts          map[string]int    `json:"ports"`
	RawRegistryAuth   *types.AuthConfig `json:"registry_auth"`
	RawForcePullImage *bool             `json:"force_pull_image"`
	RawPodSpec        *PodSpec          `json:"pod_spec"`

	RawAddCapabilities  []string `json:"add_capabilities"`
	RawDropCapabilities []string `json:"drop_capabilities"`
}

// EnvironmentImageMapV0 configures the runtime image.
//
//go:generate ../gen.sh
type EnvironmentImageMapV0 struct {
	RawCPU  *string `json:"cpu"`
	RawCUDA *string `json:"cuda"`
	RawROCM *string `json:"rocm"`
}

// WithDefaults implements the Defaultable psuedointerface.
func (e EnvironmentImageMapV0) WithDefaults() EnvironmentImageMapV0 {
	cpu := CPUImage
	cuda := CUDAImage
	rocm := ROCMImage
	if e.RawCPU != nil {
		cpu = *e.RawCPU
	}
	if e.RawROCM != nil {
		rocm = *e.RawROCM
	}
	if e.RawCUDA != nil {
		cuda = *e.RawCUDA
	}
	return EnvironmentImageMapV0{RawCPU: &cpu, RawCUDA: &cuda, RawROCM: &rocm}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (e *EnvironmentImageMapV0) UnmarshalJSON(data []byte) error {
	var plain string
	if err := json.Unmarshal(data, &plain); err == nil {
		e.RawCPU = &plain
		e.RawCUDA = &plain
		e.RawROCM = &plain
		return nil
	}

	type DefaultParser EnvironmentImageMapV0
	var jsonItem DefaultParser
	if err := json.Unmarshal(data, &jsonItem); err != nil {
		return errors.Wrapf(err, "failed to parse runtime item")
	}

	e.RawCPU = jsonItem.RawCPU
	e.RawROCM = jsonItem.RawROCM
	e.RawCUDA = jsonItem.RawCUDA

	if e.RawCUDA == nil {
		type EnvironmentImageMapV0Compat struct {
			// Parse legacy field for compatibility.
			RawGPU *string `json:"gpu"`
		}
		var compatItem EnvironmentImageMapV0Compat
		if err := json.Unmarshal(data, &compatItem); err != nil {
			return errors.Wrapf(err, "failed to parse runtime item")
		}
		e.RawCUDA = compatItem.RawGPU
	}

	return nil
}

// For returns the value for the provided device type.
func (e EnvironmentImageMapV0) For(deviceType device.Type) string {
	switch deviceType {
	case device.CPU:
		return *e.RawCPU
	case device.CUDA:
		return *e.RawCUDA
	case device.ROCM:
		return *e.RawROCM
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}

// EnvironmentVariablesMapV0 configures the runtime environment variables.
//
//go:generate ../gen.sh
type EnvironmentVariablesMapV0 struct {
	RawCPU  []string `json:"cpu"`
	RawCUDA []string `json:"cuda"`
	RawROCM []string `json:"rocm"`
}

// Merge implemenets the mergable interface.
func (e EnvironmentVariablesMapV0) Merge(
	other EnvironmentVariablesMapV0,
) EnvironmentVariablesMapV0 {
	// Order is relevant here. We want to append items to allow the following
	// override order, expConf -> templates -> taskContainerDefaults.
	// Items placed later in the array override items placed earlier.
	var out EnvironmentVariablesMapV0
	out.RawCPU = append(out.RawCPU, other.RawCPU...)
	out.RawCUDA = append(out.RawCUDA, other.RawCUDA...)
	out.RawROCM = append(out.RawROCM, other.RawROCM...)

	out.RawCPU = append(out.RawCPU, e.RawCPU...)
	out.RawCUDA = append(out.RawCUDA, e.RawCUDA...)
	out.RawROCM = append(out.RawROCM, e.RawROCM...)

	return out
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (e *EnvironmentVariablesMapV0) UnmarshalJSON(data []byte) error {
	var plain []string
	if err := json.Unmarshal(data, &plain); err == nil {
		e.RawCPU = []string{}
		e.RawCUDA = []string{}
		e.RawROCM = []string{}

		e.RawCPU = append(e.RawCPU, plain...)
		e.RawROCM = append(e.RawROCM, plain...)
		e.RawCUDA = append(e.RawCUDA, plain...)
		return nil
	}

	type DefaultParser EnvironmentVariablesMapV0
	var jsonItems DefaultParser
	if err := json.Unmarshal(data, &jsonItems); err != nil {
		return errors.Wrapf(err, "failed to parse runtime items")
	}
	e.RawCPU = []string{}
	e.RawCUDA = []string{}
	e.RawROCM = []string{}

	if jsonItems.RawCPU != nil {
		e.RawCPU = append(e.RawCPU, jsonItems.RawCPU...)
	}
	if jsonItems.RawROCM != nil {
		e.RawROCM = append(e.RawROCM, jsonItems.RawROCM...)
	}

	if jsonItems.RawCUDA != nil {
		e.RawCUDA = append(e.RawCUDA, jsonItems.RawCUDA...)
	} else {
		type EnvironmentVariablesMapV0Compat struct {
			RawGPU []string `json:"gpu"`
		}

		var compatItems EnvironmentVariablesMapV0Compat
		if err := json.Unmarshal(data, &compatItems); err != nil {
			return errors.Wrapf(err, "failed to parse runtime items")
		}

		e.RawCUDA = append(e.RawCUDA, compatItems.RawGPU...)
	}
	return nil
}

// For returns the value for the provided device type.
func (e EnvironmentVariablesMapV0) For(deviceType device.Type) []string {
	switch deviceType {
	case device.CPU:
		return e.RawCPU
	case device.CUDA:
		return e.RawCUDA
	case device.ROCM:
		return e.RawROCM
	default:
		panic(fmt.Sprintf("unexpected device type: %s", deviceType))
	}
}

// ProxyPortV0 configures the master-proxied task ports.
//
//go:generate ../gen.sh
type ProxyPortV0 struct {
	RawProxyPort        int   `json:"proxy_port"`
	RawProxyTCP         *bool `json:"proxy_tcp"`
	RawUnauthenticated  *bool `json:"unauthenticated"`
	RawDefaultServiceID *bool `json:"default_service_id"`
}

// ProxyPortsConfigV0 is the configuration for proxy ports.
//
//go:generate ../gen.sh
type ProxyPortsConfigV0 []ProxyPortV0

// Merge implemenets the mergable interface.
func (b ProxyPortsConfigV0) Merge(other ProxyPortsConfigV0) ProxyPortsConfigV0 {
	out := ProxyPortsConfigV0{}
	out = append(out, b...)

	// Prevent duplicate container ports as a result of the merge.
	ports := map[int]bool{}
	for _, pp := range b {
		ports[pp.ProxyPort()] = true
	}
	for _, pp := range other {
		if _, ok := ports[pp.ProxyPort()]; !ok {
			out = append(out, pp)
		}
	}
	return out
}
