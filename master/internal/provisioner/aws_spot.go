package provisioner

import (
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/google/uuid"
	"time"
)

func (c *awsCluster) createSpotInstanceRequest(
	ctx *actor.Context,
	numInstances int,
	dryRun bool,
	instanceType ec2InstanceType,
) (*ec2.RequestSpotInstancesOutput, error) {

	if dryRun {
		ctx.Log().Debug("dry run of createSpotInstanceRequest.")
	}
	idempotencyToken := uuid.New().String()

	validFrom := time.Now().Local().Add(time.Second * time.Duration(10))
	spotInput := &ec2.RequestSpotInstancesInput{
		ClientToken:                  aws.String(idempotencyToken),
		DryRun:                       aws.Bool(dryRun),
		InstanceCount:                aws.Int64(int64(numInstances)),
		InstanceInterruptionBehavior: aws.String("terminate"),
		LaunchSpecification:          &ec2.RequestSpotLaunchSpecification{
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
			EbsOptimized:        nil,  // TODO: We should enable this.
			ImageId:      aws.String(c.ImageID),
			InstanceType: aws.String(instanceType.name()),
			KeyName:      aws.String(c.SSHKeyName),
			UserData:            aws.String(base64.StdEncoding.EncodeToString(c.ec2UserData)),
		},
		SpotPrice:                    aws.String(c.AWSClusterConfig.SpotMaxPrice),
		ValidFrom:                    aws.Time(validFrom),
	}

	// TODO: Add tags
	//input := &ec2.RunInstancesInput{
	//	TagSpecifications: []*ec2.TagSpecification{
	//		{
	//			ResourceType: aws.String("instance"),
	//			Tags: []*ec2.Tag{
	//				{
	//					Key:   aws.String("Name"),
	//					Value: aws.String(c.InstanceName),
	//				},
	//				{
	//					Key:   aws.String(c.TagKey),
	//					Value: aws.String(c.TagValue),
	//				},
	//				{
	//					Key:   aws.String("determined-master-address"),
	//					Value: aws.String(c.masterURL.String()),
	//				},
	//			},
	//		},
	//	},
	//}

	spotInput.LaunchSpecification.NetworkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{
		{
			AssociatePublicIpAddress: aws.Bool(c.NetworkInterface.PublicIP),
			DeleteOnTermination:      aws.Bool(true),
			Description:              aws.String("network interface created by Determined"),
			DeviceIndex:              aws.Int64(0),
		},
	}
	if c.NetworkInterface.SubnetID != "" {
		spotInput.LaunchSpecification.NetworkInterfaces[0].SubnetId = aws.String(c.NetworkInterface.SubnetID)
	}
	if c.NetworkInterface.SecurityGroupID != "" {
		spotInput.LaunchSpecification.NetworkInterfaces[0].Groups = []*string{
			aws.String(c.NetworkInterface.SecurityGroupID),
		}
	}

	if c.IamInstanceProfileArn != "" {
		spotInput.LaunchSpecification.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
			Arn: aws.String(c.IamInstanceProfileArn),
		}
	}

	return c.client.RequestSpotInstances(spotInput)
}

// List all Determined-managed spot instance requests in an active state
func (c *awsCluster) listActiveSpotInstanceRequests(
	ctx *actor.Context,
	dryRun bool,
) (*ec2.DescribeSpotInstanceRequestsOutput, error) {

	if dryRun {
		ctx.Log().Debug("dry run of listActiveSpotInstanceRequests.")
	}

	input := &ec2.DescribeSpotInstanceRequestsInput{
		DryRun:                 aws.Bool(dryRun),
		Filters:                []*ec2.Filter{
			{
				Name: aws.String(fmt.Sprintf("tag:%s", c.TagKey)),
				Values: []*string{
					aws.String(c.TagValue),
				},
			},
			{
				Name: aws.String("state"),
				Values: []*string{
					aws.String("open"),
					aws.String("active"),
				},
			},
		},
	}

	return c.client.DescribeSpotInstanceRequests(input)
}


func (c *awsCluster) getSpotRequestIdsGivenInstanceIds(
	ctx *actor.Context,
	instanceId string,
	dryRun bool,
) (*ec2.DescribeSpotInstanceRequestsOutput, error) {

	if dryRun {
		ctx.Log().Debug("dry run of getSpotRequestIdsGivenInstanceIds.")
	}

	input := &ec2.DescribeSpotInstanceRequestsInput{
		DryRun:                 aws.Bool(dryRun),
		Filters:                []*ec2.Filter{
			{
				Name: aws.String("instance-id"),
				Values: []*string{
					aws.String(instanceId),
				},
			},
		},
	}

	return c.client.DescribeSpotInstanceRequests(input)
}


func (c *awsCluster) terminateSpotInstanceRequest(
	ctx *actor.Context,
	spotRequestIds []*string,
	dryRun bool,
) (*ec2.CancelSpotInstanceRequestsOutput, error) {
	if len(spotRequestIds) == 0 {
		return &ec2.CancelSpotInstanceRequestsOutput{}, nil
	}
	input := &ec2.CancelSpotInstanceRequestsInput{
		DryRun:                 aws.Bool(dryRun),
		SpotInstanceRequestIds: spotRequestIds,
	}

	return c.client.CancelSpotInstanceRequests(input)

}