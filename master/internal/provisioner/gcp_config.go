package provisioner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/pkg/errors"
	"google.golang.org/api/compute/v1"

	"github.com/determined-ai/determined/master/pkg"
	"github.com/determined-ai/determined/master/pkg/check"
)

// MaxNamePrefixLen is the max length of the instance name prefix. The full name of an instance
// should be 1-63 characters long and match the regular expression [a-z]([-a-z0-9]*[a-z0-9])? as
// suggested here: https://cloud.google.com/compute/docs/reference/rest/v1/instances/insert. We
// concatenate the prefix with a pet name to make an instance name. Here we made a rough estimation
// of the max length of name prefix to be 30.
const MaxNamePrefixLen = 30

// GCPClusterConfig describes the configuration for a GCP cluster managed by Determined.
type GCPClusterConfig struct {
	BaseConfig *compute.Instance `json:"base_config"`

	Project string `json:"project"`
	Zone    string `json:"zone"`

	BootDiskSize        int    `json:"boot_disk_size"`
	BootDiskSourceImage string `json:"boot_disk_source_image"`

	LabelKey   string `json:"label_key"`
	LabelValue string `json:"label_value"`
	NamePrefix string `json:"name_prefix"`

	NetworkInterface gceNetworkInterface `json:"network_interface"`
	NetworkTags      []string            `json:"network_tags"`
	ServiceAccount   gceServiceAccount   `json:"service_account"`

	InstanceType gceInstanceType `json:"instance_type"`

	OperationTimeoutPeriod Duration `json:"operation_timeout_period"`
}

// DefaultGCPClusterConfig returns the default configuration of the gcp cluster.
func DefaultGCPClusterConfig() *GCPClusterConfig {
	return &GCPClusterConfig{
		BootDiskSize:        200,
		BootDiskSourceImage: "projects/determined-ai/global/images/det-environments-1def2ee",
		LabelKey:            "managed-by",
		InstanceType: gceInstanceType{
			MachineType: "n1-standard-32",
			GPUType:     "nvidia-tesla-v100",
			GPUNum:      4,
		},
		OperationTimeoutPeriod: Duration(5 * time.Minute),
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *GCPClusterConfig) UnmarshalJSON(data []byte) error {
	*c = *DefaultGCPClusterConfig()
	type DefaultParser *GCPClusterConfig
	return json.Unmarshal(data, DefaultParser(c))
}

// Validate implements the check.Validatable interface.
func (c GCPClusterConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(c.BootDiskSize, 100, "gce VM boot disk size must be >= 100"),
	}
}

func (c *GCPClusterConfig) initDefaultValues() error {
	var err error

	if len(c.Project) == 0 {
		if c.Project, err = metadata.ProjectID(); err != nil {
			return err
		}
	}
	if len(c.Zone) == 0 {
		if c.Zone, err = metadata.Zone(); err != nil {
			return err
		}
	}

	// One common reason that metadata.InstanceName() fails is that the master is not
	// running in GCP. Use a default name here rather than holding up initializing the provider.
	identifier := pkg.DeterminedIdentifier
	if masterName, err := metadata.InstanceName(); err == nil {
		identifier = masterName
	}
	if len(identifier) >= MaxNamePrefixLen {
		identifier = identifier[:MaxNamePrefixLen]
	}
	if len(c.NamePrefix) == 0 {
		if identifier[len(identifier)-1] != '-' {
			c.NamePrefix = identifier + "-"
		} else {
			c.NamePrefix = identifier
		}
	}
	if len(c.LabelValue) == 0 {
		if identifier[len(identifier)-1] == '-' {
			c.LabelValue = identifier[:len(identifier)-1]
		} else {
			c.LabelValue = identifier
		}
	}

	if len(c.ServiceAccount.Email) > 0 && len(c.ServiceAccount.Scopes) == 0 {
		c.ServiceAccount.Scopes = []string{"https://www.googleapis.com/auth/cloud-platform"}
	}
	return nil
}

func (c *GCPClusterConfig) merge() *compute.Instance {
	rb := &compute.Instance{}
	if c.BaseConfig != nil {
		*rb = *c.BaseConfig
	}

	if len(c.InstanceType.MachineType) > 0 {
		rb.MachineType = fmt.Sprintf(
			"zones/%s/machineTypes/%s", c.Zone, c.InstanceType.MachineType,
		)
	}

	if len(c.InstanceType.GPUType) > 0 {
		rb.GuestAccelerators = []*compute.AcceleratorConfig{
			{
				AcceleratorType: fmt.Sprintf(
					"zones/%s/acceleratorTypes/%s", c.Zone, c.InstanceType.GPUType,
				),
				AcceleratorCount: int64(c.InstanceType.GPUNum),
			},
		}
	}

	if len(c.BootDiskSourceImage) > 0 {
		rb.Disks = append([]*compute.AttachedDisk{
			{
				Boot: true,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: c.BootDiskSourceImage,
					DiskSizeGb:  int64(c.BootDiskSize),
				},
				AutoDelete: true,
			},
		}, rb.Disks...)
	}

	if len(c.LabelKey) > 0 && len(c.LabelValue) > 0 {
		if rb.Labels == nil {
			rb.Labels = make(map[string]string)
		}
		rb.Labels[c.LabelKey] = c.LabelValue
	}

	if len(c.NetworkInterface.Network) > 0 && len(c.NetworkInterface.Subnetwork) > 0 {
		networkInterface := &compute.NetworkInterface{
			Network:    c.NetworkInterface.Network,
			Subnetwork: c.NetworkInterface.Subnetwork,
		}
		if c.NetworkInterface.ExternalIP {
			networkInterface.AccessConfigs = []*compute.AccessConfig{
				{
					NetworkTier: "PREMIUM",
				},
			}
		}
		rb.NetworkInterfaces = append(rb.NetworkInterfaces, networkInterface)
	}
	if len(c.NetworkTags) > 0 {
		if rb.Tags == nil {
			rb.Tags = &compute.Tags{}
		}
		rb.Tags.Items = append(rb.Tags.Items, c.NetworkTags...)
	}

	if len(c.ServiceAccount.Email) > 0 {
		rb.ServiceAccounts = append(rb.ServiceAccounts, &compute.ServiceAccount{
			Email:  c.ServiceAccount.Email,
			Scopes: c.ServiceAccount.Scopes,
		})
	}

	rb.Scheduling = &compute.Scheduling{
		OnHostMaintenance: "TERMINATE",
		Preemptible:       c.InstanceType.Preemptible,
	}
	return rb
}

type gceNetworkInterface struct {
	Network    string `json:"network"`
	Subnetwork string `json:"subnetwork"`
	ExternalIP bool   `json:"external_ip"`
}

type gceServiceAccount struct {
	Email  string   `json:"email"`
	Scopes []string `json:"scopes"`
}

var gceMachineTypes = []string{
	"n1-standard",
	"n1-highmem",
	"n1-highcpu",
	"n1-ultramem",
	"m2-ultramem",
	"n1-megamem",
	"c2-standard",
	"custom",
}

var gceGPUTypes = map[string][]int{
	"":                  {0},
	"nvidia-tesla-t4":   {0, 1, 2, 4},
	"nvidia-tesla-p100": {0, 1, 2, 4},
	"nvidia-tesla-p4":   {0, 1, 2, 4},
	"nvidia-tesla-v100": {0, 1, 2, 4, 8},
	"nvidia-tesla-k80":  {0, 1, 2, 4, 8},
}

type gceInstanceType struct {
	MachineType string `json:"machine_type"`
	GPUType     string `json:"gpu_type"`
	GPUNum      int    `json:"gpu_num"`
	Preemptible bool   `json:"preemptible"`
}

func (t gceInstanceType) name() string {
	return fmt.Sprintf("%s-%s-%d", t.MachineType, t.GPUType, t.GPUNum)
}

func (t gceInstanceType) slots() int {
	return t.GPUNum
}

func (t gceInstanceType) Validate() []error {
	var checkMachineType = errors.Errorf("gce VM machine type must be within: %v",
		strings.Join(gceMachineTypes, ", "))
	if items := strings.Split(t.MachineType, "-"); len(items) == 3 {
		for _, mType := range gceMachineTypes {
			if strings.HasPrefix(t.MachineType, mType) {
				checkMachineType = nil
				break
			}
		}
	}

	var checkGPU error
	if numsAllowed, ok := gceGPUTypes[t.GPUType]; !ok {
		strs := make([]string, 0, len(gceGPUTypes))
		for item := range gceGPUTypes {
			strs = append(strs, item)
		}
		checkGPU = errors.Errorf("gce VM gpu type must be within: %s", strings.Join(strs, ", "))
	} else {
		checkGPU = errors.Errorf("gce VM gpu type %s num must be within: %v", t.GPUType, numsAllowed)
		for _, n := range numsAllowed {
			if t.GPUNum == n {
				checkGPU = nil
				break
			}
		}
	}

	return []error{
		checkMachineType,
		checkGPU,
	}
}
