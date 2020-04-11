package provisioner

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
)

const awsAgentID = `$(ec2metadata --instance-id)`

func getEC2MetadataSess() (*ec2metadata.EC2Metadata, error) {
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AWS session")
	}
	return ec2metadata.New(sess), nil
}

func getEC2Metadata(field string) (string, error) {
	ec2Metadata, err := getEC2MetadataSess()
	if err != nil {
		return "", err
	}
	return ec2Metadata.GetMetadata(field)
}

func onEC2() bool {
	ec2Metadata, err := getEC2MetadataSess()
	if err != nil {
		return false
	}
	return ec2Metadata.Available()
}

// awsCluster wraps a ec2 client.
// Determined recognizes agent EC2 instances by:
// 1. a specific key/value pair tag
// 2. names of agents that are equal to the instance IDs
type awsCluster struct {
	*AWSClusterConfig
	masterURL   url.URL
	ec2UserData []byte
	client      *ec2.EC2
}

func newAWSCluster(config *Config) (*awsCluster, error) {
	if err := config.AWS.initDefaultValues(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize auto configuration")
	}
	// This following AWS session is created using AWS Credentials without explicitly configuration
	// in the code. However you need to do the following settings.
	// See https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html
	// 1. Use IAM roles for Amazon EC2
	//    The following roles on any resources:
	//    "ec2:DescribeInstances",
	//    "ec2:TerminateInstances",
	//    "ec2:CreateTags",
	//    "ec2:RunInstances".
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

	cluster := &awsCluster{
		AWSClusterConfig: config.AWS,
		masterURL:        *masterURL,
		client:           ec2.New(sess),
		ec2UserData: mustMakeAgentSetupScript(agentSetupScriptConfig{
			MasterHost:          masterURL.Hostname(),
			MasterPort:          masterURL.Port(),
			StartupScriptBase64: base64.StdEncoding.EncodeToString([]byte(config.StartupScript)),
			AgentDockerRuntime:  config.AgentDockerRuntime,
			AgentNetwork:        config.AgentDockerNetwork,
			AgentDockerImage:    config.AgentDockerImage,
			AgentID:             awsAgentID,
			LogOptions:          config.AWS.buildDockerLogString(),
		}),
	}
	return cluster, nil
}

func (c *awsCluster) instanceType() instanceType {
	return c.InstanceType
}

func (c *awsCluster) maxInstanceNum() int {
	return c.MaxInstances
}

func (c *awsCluster) agentNameFromInstance(inst *ec2.Instance) string {
	return *inst.InstanceId
}

// See https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-lifecycle.html.
var ec2InstanceStates = map[string]InstanceState{
	"pending": Starting,
	"running": Running,
	"stopped": Stopped,
}

func (c *awsCluster) stateFromEC2State(state *ec2.InstanceState) InstanceState {
	if res, ok := ec2InstanceStates[*state.Name]; ok {
		return res
	}
	return Unknown
}

func (c *awsCluster) dryRunRequests() error {
	const (
		DryRunOperationErrorCode  = "DryRunOperation"
		InstanceLimitExceededCode = "InstanceLimitExceeded"
	)
	_, err := c.describeInstances(true)
	if awsErr, ok := errors.Cause(err).(awserr.Error); !ok ||
		awsErr.Code() != DryRunOperationErrorCode {
		return err
	}
	_, err = c.launchInstances(c.InstanceType, 1, true)
	if awsErr, ok := errors.Cause(err).(awserr.Error); !ok ||
		awsErr.Code() != DryRunOperationErrorCode && awsErr.Code() != InstanceLimitExceededCode {
		return err
	}
	return nil
}

func (c *awsCluster) list(ctx *actor.Context) ([]*Instance, error) {
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

func (c *awsCluster) launch(
	ctx *actor.Context,
	instanceType instanceType,
	instanceNum int,
) {
	instType, ok := instanceType.(ec2InstanceType)
	if !ok {
		panic("cannot pass non-ec2InstanceType to ec2Cluster")
	} else if instanceNum <= 0 {
		return
	}

	ctx.Log().Infof("launching %d EC2 instances", instanceNum)
	instances, err := c.launchInstances(instType, instanceNum, false)
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

func (c *awsCluster) terminate(ctx *actor.Context, instanceIDs []string) {
	if len(instanceIDs) == 0 {
		return
	}

	ctx.Log().Infof(
		"terminating %d EC2 instances: %s",
		len(instanceIDs),
		instanceIDs,
	)
	ids := make([]*string, 0, len(instanceIDs))
	for _, id := range instanceIDs {
		idCopy := id
		ids = append(ids, &idCopy)
	}
	res, err := c.terminateInstances(ids, false)
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
				Name: aws.String(fmt.Sprintf("tag:%s", c.TagKey)),
				Values: []*string{
					aws.String(c.TagValue),
				},
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

func (c *awsCluster) launchInstances(
	instanceType ec2InstanceType,
	instanceNum int,
	dryRun bool,
) (*ec2.Reservation, error) {
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
		InstanceType: aws.String(instanceType.name()),
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
						Key:   aws.String("determined-master-address"),
						Value: aws.String(c.masterURL.String()),
					},
				},
			},
		},
		UserData: aws.String(base64.StdEncoding.EncodeToString(c.ec2UserData)),
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
	dryRun bool,
) (*ec2.TerminateInstancesOutput, error) {
	if len(ids) == 0 {
		return &ec2.TerminateInstancesOutput{}, nil
	}
	input := &ec2.TerminateInstancesInput{
		DryRun:      aws.Bool(dryRun),
		InstanceIds: ids,
	}
	return c.client.TerminateInstances(input)
}
