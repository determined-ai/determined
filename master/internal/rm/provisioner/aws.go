package provisioner

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

	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/pkg/actor"
)

// awsCluster wraps an EC2 client. Determined recognizes agent EC2 instances by:
// 1. A specific key/value pair tag.
// 2. Names of agents that are equal to the instance IDs.
type awsCluster struct {
	*provconfig.AWSClusterConfig
	resourcePool string
	masterURL    url.URL
	ec2UserData  []byte
	client       *ec2.EC2

	// State that is only used if spot instances are enabled
	spot *spotState
}

func newAWSCluster(
	resourcePool string, config *provconfig.Config, cert *tls.Certificate,
) (*awsCluster, error) {
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
	containerScriptBase64 := base64.StdEncoding.EncodeToString([]byte(config.ContainerStartupScript))

	var certBytes []byte
	if masterURL.Scheme == secureScheme && cert != nil {
		for _, c := range cert.Certificate {
			b := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: c,
			})
			certBytes = append(certBytes, b...)
		}
	}
	masterCertBase64 := base64.StdEncoding.EncodeToString(certBytes)

	cluster := &awsCluster{
		resourcePool:     resourcePool,
		AWSClusterConfig: config.AWS,
		masterURL:        *masterURL,
		client:           ec2.New(sess),
		ec2UserData: mustMakeAgentSetupScript(agentSetupScriptConfig{
			MasterHost:                   masterURL.Hostname(),
			MasterPort:                   masterURL.Port(),
			MasterCertName:               config.MasterCertName,
			StartupScriptBase64:          startupScriptBase64,
			ContainerStartupScriptBase64: containerScriptBase64,
			MasterCertBase64:             masterCertBase64,
			SlotType:                     config.AWS.SlotType(),
			AgentDockerRuntime:           config.AgentDockerRuntime,
			AgentNetwork:                 config.AgentDockerNetwork,
			AgentDockerImage:             config.AgentDockerImage,
			AgentFluentImage:             config.AgentFluentImage,
			AgentReconnectAttempts:       config.AgentReconnectAttempts,
			AgentReconnectBackoff:        config.AgentReconnectBackoff,
			AgentID:                      `$(ec2metadata --instance-id)`,
			ResourcePool:                 resourcePool,
			LogOptions:                   config.AWS.BuildDockerLogString(),
		}),
	}

	if cluster.SpotEnabled {
		cluster.spot = &spotState{
			trackedReqs:          newSetOfSpotRequests(),
			approximateClockSkew: time.Second * 0,
			launchTimeOffset:     time.Second * 10,
		}
	}

	return cluster, nil
}

func (c *awsCluster) instanceType() instanceType {
	return c.InstanceType
}

func (c *awsCluster) slotsPerInstance() int {
	return c.AWSClusterConfig.SlotsPerInstance()
}

func (c *awsCluster) agentNameFromInstance(inst *ec2.Instance) string {
	return *inst.InstanceId
}

// See https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-lifecycle.html.
var ec2InstanceStates = map[string]InstanceState{
	"pending":       Starting,
	"running":       Running,
	"stopped":       Stopped,
	"stopping":      Stopping,
	"shutting-down": Terminating,
}

func (c *awsCluster) stateFromEC2State(state *ec2.InstanceState) InstanceState {
	if res, ok := ec2InstanceStates[*state.Name]; ok {
		return res
	}
	return Unknown
}

func (c *awsCluster) prestart(ctx *actor.Context) {
	if c.SpotEnabled {
		c.attemptToApproximateClockSkew(ctx)
		c.cleanupLegacySpotInstances(ctx)
	}
}

func (c *awsCluster) list(ctx *actor.Context) ([]*Instance, error) {
	if c.SpotEnabled {
		return c.listSpot(ctx)
	}
	return c.listOnDemand(ctx)
}

func (c *awsCluster) launch(
	ctx *actor.Context,
	instanceNum int,
) {
	if c.SpotEnabled {
		c.launchSpot(ctx, instanceNum)
	} else {
		c.launchOnDemand(ctx, instanceNum)
	}
}

func (c *awsCluster) terminate(ctx *actor.Context, instanceIDs []string) {
	ids := make([]*string, 0, len(instanceIDs))
	for _, id := range instanceIDs {
		idCopy := id
		ids = append(ids, &idCopy)
	}

	if c.SpotEnabled {
		c.terminateSpot(ctx, ids)
	} else {
		c.terminateOnDemand(ctx, ids)
	}
}

func (c *awsCluster) listOnDemand(ctx *actor.Context) ([]*Instance, error) {
	instances, err := c.describeInstances(false)
	if err != nil {
		return nil, errors.Wrap(err, "cannot describe EC2 instances")
	}
	res := c.newInstances(instances)
	for _, inst := range res {
		if inst.State == Unknown {
			ctx.Log().Errorf("unknown instance state for instance %v", inst.ID)
		}
	}
	return res, nil
}

func (c *awsCluster) launchOnDemand(ctx *actor.Context, instanceNum int) {
	if instanceNum <= 0 {
		return
	}
	instances, err := c.launchInstances(instanceNum, false)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot launch EC2 instances")
		return
	}
	launched := c.newInstances(instances.Instances)
	ctx.Log().Infof(
		"launched %d/%d EC2 instances: %s",
		len(launched),
		instanceNum,
		fmtInstances(launched),
	)
}

func (c *awsCluster) terminateOnDemand(ctx *actor.Context, instanceIDs []*string) {
	if len(instanceIDs) == 0 {
		return
	}

	res, err := c.terminateInstances(instanceIDs)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot terminate EC2 instances")
		return
	}
	terminated := c.newInstancesFromTerminateInstancesOutput(res)
	ctx.Log().Infof(
		"terminated %d/%d EC2 instances: %s",
		len(terminated),
		len(instanceIDs),
		fmtInstances(terminated),
	)
}

func (c *awsCluster) newInstances(input []*ec2.Instance) []*Instance {
	output := make([]*Instance, 0, len(input))
	for _, inst := range input {
		output = append(output, &Instance{
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
) []*Instance {
	instances := make([]*Instance, 0, len(output.TerminatingInstances))
	for _, instanceChange := range output.TerminatingInstances {
		instances = append(instances, &Instance{
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
				Name:   aws.String(fmt.Sprintf("tag:%s", c.TagKey)),
				Values: []*string{aws.String(c.TagValue)},
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
					VolumeSize:          aws.Int64(int64(c.RootVolumeSize)),
					VolumeType:          aws.String("gp2"),
				},
			},
		},
		DryRun:       aws.Bool(dryRun),
		ImageId:      aws.String(c.ImageID),
		InstanceType: aws.String(c.AWSClusterConfig.InstanceType.Name()),
		KeyName:      aws.String(c.SSHKeyName),
		MaxCount:     aws.Int64(int64(instanceNum)),
		MinCount:     aws.Int64(1),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(c.InstanceName),
					},
					{
						Key:   aws.String(c.TagKey),
						Value: aws.String(c.TagValue),
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
		UserData: aws.String(base64.StdEncoding.EncodeToString(c.ec2UserData)),
	}

	if c.CustomTags != nil {
		for _, tag := range c.CustomTags {
			customTag := &ec2.Tag{
				Key:   aws.String(tag.Key),
				Value: aws.String(tag.Value),
			}
			input.TagSpecifications[0].Tags = append(input.TagSpecifications[0].Tags, customTag)
		}
	}

	input.NetworkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{
		{
			AssociatePublicIpAddress: aws.Bool(c.NetworkInterface.PublicIP),
			DeleteOnTermination:      aws.Bool(true),
			Description:              aws.String("network interface created by Determined"),
			DeviceIndex:              aws.Int64(0),
		},
	}
	if c.NetworkInterface.SubnetID != "" {
		input.NetworkInterfaces[0].SubnetId = aws.String(c.NetworkInterface.SubnetID)
	}
	if c.NetworkInterface.SecurityGroupID != "" {
		input.NetworkInterfaces[0].Groups = []*string{
			aws.String(c.NetworkInterface.SecurityGroupID),
		}
	}

	if c.IamInstanceProfileArn != "" {
		input.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
			Arn: aws.String(c.IamInstanceProfileArn),
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
