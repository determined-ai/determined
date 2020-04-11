package provisioner

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg"
	"github.com/determined-ai/determined/master/pkg/check"
)

// AWSClusterConfig describes the configuration for an EC2 cluster managed by Determined.
type AWSClusterConfig struct {
	Region string `json:"region"`

	RootVolumeSize int    `json:"root_volume_size"`
	ImageID        string `json:"image_id"`

	TagKey       string `json:"tag_key"`
	TagValue     string `json:"tag_value"`
	InstanceName string `json:"instance_name"`

	SSHKeyName            string              `json:"ssh_key_name"`
	NetworkInterface      ec2NetworkInterface `json:"network_interface"`
	IamInstanceProfileArn string              `json:"iam_instance_profile_arn"`

	InstanceType ec2InstanceType `json:"instance_type"`
	MaxInstances int             `json:"max_instances"`

	LogGroup  string `json:"log_group"`
	LogStream string `json:"log_stream"`
}

var defaultAWSClusterConfig = AWSClusterConfig{
	InstanceName:   "determined-ai-agent",
	RootVolumeSize: 200,
	TagKey:         "managed_by",
	NetworkInterface: ec2NetworkInterface{
		PublicIP: true,
	},
	InstanceType: "p3.8xlarge",
	MaxInstances: 5,
}

func (c *AWSClusterConfig) buildDockerLogString() string {
	logString := ""
	if c.LogGroup != "" {
		logString += "--log-driver=awslogs --log-opt awslogs-group=" + c.LogGroup
	}
	if c.LogStream != "" {
		logString += " --log-opt awslogs-stream=" + c.LogStream
	}
	return logString
}

func (c *AWSClusterConfig) initDefaultValues() error {
	metadata, err := getEC2MetadataSess()
	if err != nil {
		return err
	}

	if len(c.Region) == 0 {
		if c.Region, err = metadata.Region(); err != nil {
			return err
		}
	}

	// One common reason that metadata.GetInstanceIdentityDocument() fails is that the master is not
	// running in EC2. Use a default name here rather than holding up initializing the provider.
	identifier := pkg.DeterminedIdentifier
	idDoc, err := metadata.GetInstanceIdentityDocument()
	if err == nil {
		identifier = idDoc.InstanceID
	}

	if len(c.TagValue) == 0 {
		c.TagValue = identifier
	}
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *AWSClusterConfig) UnmarshalJSON(data []byte) error {
	*c = defaultAWSClusterConfig
	type DefaultParser *AWSClusterConfig
	return json.Unmarshal(data, DefaultParser(c))
}

// Validate implements the check.Validatable interface.
func (c AWSClusterConfig) Validate() []error {
	return []error{
		check.GreaterThan(len(c.ImageID), 0, "ec2 image ID must be non-empty"),
		check.GreaterThan(len(c.SSHKeyName), 0, "ec2 key name must be non-empty"),
		check.GreaterThanOrEqualTo(c.RootVolumeSize, 100, "ec2 root volume size must be >= 100"),
		check.GreaterThanOrEqualTo(c.MaxInstances, 0, "ec2 cluster max instance number must be >= 0"),
	}
}

type ec2NetworkInterface struct {
	PublicIP        bool   `json:"public_ip"`
	SubnetID        string `json:"subnet_id"`
	SecurityGroupID string `json:"security_group_id"`
}

type ec2InstanceType string

var ec2InstanceSlots = map[ec2InstanceType]int{
	"p2.xlarge":     1,
	"p2.8xlarge":    8,
	"p2.16xlarge":   16,
	"p3.2xlarge":    1,
	"p3.8xlarge":    4,
	"p3.16xlarge":   8,
	"p3dn.24xlarge": 8,
}

func (t ec2InstanceType) name() string {
	return string(t)
}

func (t ec2InstanceType) slots() int {
	if s, ok := ec2InstanceSlots[t]; ok {
		return s
	}
	return 0
}

func (t ec2InstanceType) Validate() []error {
	if _, ok := ec2InstanceSlots[t]; ok {
		return nil
	}
	strs := make([]string, 0, len(ec2InstanceSlots))
	for t := range ec2InstanceSlots {
		strs = append(strs, t.name())
	}
	return []error{
		errors.Errorf("ec2 instance type must be valid type: %s", strings.Join(strs, ", ")),
	}
}
