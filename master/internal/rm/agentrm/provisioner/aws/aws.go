package aws

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/internal/rm/agentrm/provisioner/agentsetup"
	"github.com/determined-ai/determined/master/pkg/model"
)

// awsCluster wraps an EC2 client. Determined recognizes agent EC2 instances by:
// 1. A specific key/value pair tag.
// 2. Names of agents that are equal to the instance IDs.
type awsCluster struct {
	config       *provconfig.AWSClusterConfig
	resourcePool string
	masterURL    url.URL
	ec2UserData  []byte
	client       *ec2.EC2

	// State that is only used if spot instances are enabled
	spot *spotState

	syslog *logrus.Entry
}

//nolint:lll  // See https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html
const ec2InstanceID = `$(curl -q -H "X-aws-ec2-metadata-token: $(curl -q -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")"  http://169.254.169.254/latest/meta-data/instance-id)`

// New creates a new AWS cluster.
func New(
	resourcePool string, config *provconfig.Config, cert *tls.Certificate,
) (agentsetup.Provider, error) {
	if err := config.AWS.InitDefaultValues(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize auto configuration")
	}

	// This following AWS session is created using AWS Credentials without explicitly configuration
	// in the code. However you need to do the following settings.
	// See https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html
	// 1. Use IAM roles for Amazon EC2
	//    The following permissions on any resources:
	//    "ec2:DescribeInstances",
	//    "ec2:TerminateInstances",
	//    "ec2:CreateTags",
	//    "ec2:RunInstances".
	//    If using spot instances, the following permissions will be required
	//    "ec2:CancelSpotInstanceRequests",
	//    "ec2:RequestSpotInstances",
	//    "ec2:DescribeSpotInstanceRequests",
	// 2. Use a shared credentials file
	//    In order to be able to connect to AWS, the credentials should be put in the
	//    file `~/.aws/credential` in the format:
	//    ```
	//    [default]
	//    aws_access_key_id = YOUR_ACCESS_KEY_ID
	//    aws_secret_access_key = YOUR_SECRET_ACCESS_KEY
	//    ```
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AWS.Region),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AWS session")
	}

	masterURL, err := url.Parse(config.MasterURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse master url")
	}

	startupScriptBase64 := base64.StdEncoding.EncodeToString([]byte(config.StartupScript))
	containerScriptBase64 := base64.StdEncoding.EncodeToString(
		[]byte(config.ContainerStartupScript),
	)

	var certBytes []byte
	if masterURL.Scheme == agentsetup.SecureScheme && cert != nil {
		for _, c := range cert.Certificate {
			b := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: c,
			})
			certBytes = append(certBytes, b...)
		}
	}
	masterCertBase64 := base64.StdEncoding.EncodeToString(certBytes)
	configFileBase64 := base64.StdEncoding.EncodeToString(config.AgentConfigFileContents)

	cluster := &awsCluster{
		resourcePool: resourcePool,
		config:       config.AWS,
		masterURL:    *masterURL,
		client:       ec2.New(sess),
		ec2UserData: agentsetup.MustMakeAgentSetupScript(agentsetup.AgentSetupScriptConfig{
			MasterHost:                   masterURL.Hostname(),
			MasterPort:                   masterURL.Port(),
			MasterCertName:               config.MasterCertName,
			StartupScriptBase64:          startupScriptBase64,
			ContainerStartupScriptBase64: containerScriptBase64,
			MasterCertBase64:             masterCertBase64,
			ConfigFileBase64:             configFileBase64,
			SlotType:                     config.AWS.SlotType(),
			AgentDockerRuntime:           config.AgentDockerRuntime,
			AgentNetwork:                 config.AgentDockerNetwork,
			AgentDockerImage:             config.AgentDockerImage,
			// deprecated, no longer in use.
			AgentFluentImage:       config.AgentFluentImage,
			AgentReconnectAttempts: config.AgentReconnectAttempts,
			AgentReconnectBackoff:  config.AgentReconnectBackoff,
			AgentID:                ec2InstanceID,
			ResourcePool:           resourcePool,
			LogOptions:             config.AWS.BuildDockerLogString(),
		}),
		syslog: logrus.WithField("aws-cluster", resourcePool),
	}

	if cluster.config.SpotEnabled {
		cluster.spot = &spotState{
			trackedReqs:          newSetOfSpotRequests(),
			approximateClockSkew: time.Second * 0,
			launchTimeOffset:     time.Second * 10,
		}
		cluster.attemptToApproximateClockSkew()
	}

	return cluster, nil
}

func (c *awsCluster) InstanceType() model.InstanceType {
	return c.config.InstanceType
}

func (c *awsCluster) SlotsPerInstance() int {
	return c.config.SlotsPerInstance()
}

func (c *awsCluster) agentNameFromInstance(inst *ec2.Instance) string {
	return *inst.InstanceId
}

// See https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-lifecycle.html.
var ec2InstanceStates = map[string]model.InstanceState{
	"pending":       model.Starting,
	"running":       model.Running,
	"stopped":       model.Stopped,
	"stopping":      model.Stopping,
	"shutting-down": model.Terminating,
}

func (c *awsCluster) stateFromEC2State(state *ec2.InstanceState) model.InstanceState {
	if res, ok := ec2InstanceStates[*state.Name]; ok {
		return res
	}
	return model.Unknown
}

func (c *awsCluster) List() ([]*model.Instance, error) {
	if c.config.SpotEnabled {
		return c.listSpot()
	}
	return c.listOnDemand()
}

func (c *awsCluster) Launch(instanceNum int) error {
	if c.config.SpotEnabled {
		return c.launchSpot(instanceNum)
	}
	return c.launchOnDemand(instanceNum)
}

func (c *awsCluster) Terminate(instanceIDs []string) {
	ids := make([]*string, 0, len(instanceIDs))
	for _, id := range instanceIDs {
		idCopy := id
		ids = append(ids, &idCopy)
	}

	if c.config.SpotEnabled {
		c.terminateSpot(ids)
	} else {
		c.terminateOnDemand(ids)
	}
}

func (c *awsCluster) listOnDemand() ([]*model.Instance, error) {
	instances, err := c.describeInstances(false)
	if err != nil {
		return nil, errors.Wrap(err, "cannot describe EC2 instances")
	}
	res := c.newInstances(instances)
	for _, inst := range res {
		if inst.State == model.Unknown {
			c.syslog.Errorf("unknown instance state for instance %v", inst.ID)
		}
	}
	return res, nil
}

func (c *awsCluster) launchOnDemand(instanceNum int) error {
	if instanceNum <= 0 {
		return nil
	}
	instances, err := c.launchInstances(instanceNum, false)
	if err != nil {
		c.syslog.WithError(err).Error("cannot launch EC2 instances")
		return err
	}
	launched := c.newInstances(instances.Instances)
	c.syslog.Infof(
		"launched %d/%d EC2 instances: %s",
		len(launched),
		instanceNum,
		model.FmtInstances(launched),
	)
	return nil
}

func (c *awsCluster) terminateOnDemand(instanceIDs []*string) {
	if len(instanceIDs) == 0 {
		return
	}

	res, err := c.terminateInstances(instanceIDs)
	if err != nil {
		c.syslog.WithError(err).Error("cannot terminate EC2 instances")
		return
	}
	terminated := c.newInstancesFromTerminateInstancesOutput(res)
	c.syslog.Infof(
		"terminated %d/%d EC2 instances: %s",
		len(terminated),
		len(instanceIDs),
		model.FmtInstances(terminated),
	)
}

func (c *awsCluster) newInstances(input []*ec2.Instance) []*model.Instance {
	output := make([]*model.Instance, 0, len(input))
	for _, inst := range input {
		output = append(output, &model.Instance{
			ID:         *inst.InstanceId,
			LaunchTime: *inst.LaunchTime,
			AgentName:  c.agentNameFromInstance(inst),
			State:      c.stateFromEC2State(inst.State),
		})
	}
	return output
}

func (c *awsCluster) newInstancesFromTerminateInstancesOutput(
	output *ec2.TerminateInstancesOutput,
) []*model.Instance {
	instances := make([]*model.Instance, 0, len(output.TerminatingInstances))
	for _, instanceChange := range output.TerminatingInstances {
		instances = append(instances, &model.Instance{
			ID:    *instanceChange.InstanceId,
			State: c.stateFromEC2State(instanceChange.CurrentState),
		})
	}
	return instances
}

func (c *awsCluster) describeInstances(dryRun bool) ([]*ec2.Instance, error) {
	input := &ec2.DescribeInstancesInput{
		DryRun: aws.Bool(dryRun),
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", c.config.TagKey)),
				Values: []*string{aws.String(c.config.TagValue)},
			},
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", "determined-resource-pool")),
				Values: []*string{aws.String(c.resourcePool)},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
					aws.String("pending"),
					aws.String("stopped"),
				},
			},
		},
	}
	result, err := c.client.DescribeInstances(input)
	if err != nil {
		return nil, err
	}
	var instances []*ec2.Instance
	for _, rsv := range result.Reservations {
		if rsv.Instances != nil {
			instances = append(instances, rsv.Instances...)
		}
	}
	return instances, nil
}

func (c *awsCluster) describeInstancesByID(
	instanceIds []*string,
	dryRun bool,
) ([]*ec2.Instance, error) {
	if len(instanceIds) == 0 {
		return make([]*ec2.Instance, 0), nil
	}
	input := &ec2.DescribeInstancesInput{
		DryRun:      aws.Bool(dryRun),
		InstanceIds: instanceIds,
	}
	result, err := c.client.DescribeInstances(input)
	if err != nil {
		return nil, err
	}
	var instances []*ec2.Instance
	for _, rsv := range result.Reservations {
		if rsv.Instances != nil {
			instances = append(instances, rsv.Instances...)
		}
	}
	return instances, nil
}

func (c *awsCluster) launchInstances(instanceNum int, dryRun bool) (*ec2.Reservation, error) {
	input := &ec2.RunInstancesInput{
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					VolumeSize:          aws.Int64(int64(c.config.RootVolumeSize)),
					VolumeType:          aws.String("gp2"),
				},
			},
		},
		DryRun:                            aws.Bool(dryRun),
		ImageId:                           aws.String(c.config.ImageID),
		InstanceInitiatedShutdownBehavior: aws.String(ec2.ShutdownBehaviorTerminate),
		InstanceType:                      aws.String(c.config.InstanceType.Name()),
		KeyName:                           aws.String(c.config.SSHKeyName),
		MaxCount:                          aws.Int64(int64(instanceNum)),
		MinCount:                          aws.Int64(1),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(c.config.InstanceName),
					},
					{
						Key:   aws.String(c.config.TagKey),
						Value: aws.String(c.config.TagValue),
					},
					{
						Key:   aws.String("determined-resource-pool"),
						Value: aws.String(c.resourcePool),
					},
					{
						Key:   aws.String("determined-master-address"),
						Value: aws.String(c.masterURL.String()),
					},
				},
			},
		},
		MetadataOptions: &ec2.InstanceMetadataOptionsRequest{
			HttpTokens: aws.String(ec2.HttpTokensStateRequired),
			// We need the additional hop due to running in a Docker container
			// with a bridge network. This adds an extra hop causing the put requests to fail
			// with the default limit of 1.
			HttpPutResponseHopLimit: aws.Int64(2),
		},
		UserData: aws.String(base64.StdEncoding.EncodeToString(c.ec2UserData)),
	}

	if c.config.CustomTags != nil {
		for _, tag := range c.config.CustomTags {
			customTag := &ec2.Tag{
				Key:   aws.String(tag.Key),
				Value: aws.String(tag.Value),
			}
			input.TagSpecifications[0].Tags = append(input.TagSpecifications[0].Tags, customTag)
		}
	}

	input.NetworkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{
		{
			AssociatePublicIpAddress: aws.Bool(c.config.NetworkInterface.PublicIP),
			DeleteOnTermination:      aws.Bool(true),
			Description:              aws.String("network interface created by Determined"),
			DeviceIndex:              aws.Int64(0),
		},
	}
	if c.config.NetworkInterface.SubnetID != "" {
		input.NetworkInterfaces[0].SubnetId = aws.String(c.config.NetworkInterface.SubnetID)
	}
	if c.config.NetworkInterface.SecurityGroupID != "" {
		input.NetworkInterfaces[0].Groups = []*string{
			aws.String(c.config.NetworkInterface.SecurityGroupID),
		}
	}

	if c.config.IamInstanceProfileArn != "" {
		input.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
			Arn: aws.String(c.config.IamInstanceProfileArn),
		}
	}

	return c.client.RunInstances(input)
}

func (c *awsCluster) terminateInstances(
	ids []*string,
) (*ec2.TerminateInstancesOutput, error) {
	if len(ids) == 0 {
		return &ec2.TerminateInstancesOutput{}, nil
	}
	input := &ec2.TerminateInstancesInput{
		InstanceIds: ids,
	}
	return c.client.TerminateInstances(input)
}
