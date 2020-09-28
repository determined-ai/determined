package provisioner

import (
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"time"
)


// Spot Instances are created asynchronously - you create a request with a
// time in the future and the request will be fulfilled or not. However, the scaleDecider operates in terms of instances.
//Most of this code is about handling this. When the scaleDecider asks for a list of instances, we return the spot
//instances that have been created and we silently keep track of the spot requests
//that are pending but have not been fulfilled. We obey deletion as normal.
//During instance creation, the scaleDecider is telling us that it wants X
//more instances than it thinks we have based on the output of list. It is
//unaware of the unfulfilled pending spot requests. We look at the number of
//additional instances that the scaleDecider wants and compare that to the
//number of unfulfilled spot requests to decide whether we need to create
//more spot requests, delete spot requests or do nothing.



func (c *awsCluster) listSpot(ctx *actor.Context) ([]*Instance, error) {
	// 1. List all spot instance requests
	// 2. For any requests that indicate a failure, write out an error message and clean up the spot request (if needed)
	// 3. For each spot instance request, find the matching instances
	// 4. Keep track of the spot requests that haven't been fulfilled
	resp, err := c.listActiveSpotInstanceRequests(ctx, false)
	if err != nil {
		return nil, errors.Wrap(err, "cannot describe EC2 spot requests")
	}

	runningInstanceIds, requestsWithoutInstances, unfulfillableRequests := parseDescribeSpotInstanceRequestResponse(resp)
	c.handleUnfulfillableRequests(ctx, unfulfillableRequests)

	c.pendingSpotRequestIds = requestsWithoutInstances

	instancesToReturn, err := c.describeInstancesById(runningInstanceIds, false)
	if err != nil {
		return nil, errors.Wrap(err, "cannot describe EC2 instances")
	}
	res := c.newInstances(instancesToReturn)
	for _, inst := range res {
		if inst.State == Unknown {
			ctx.Log().Errorf("unknown instance state for instance %v", inst.ID)
		}
	}
	return res, nil
}



func (c *awsCluster) launchSpot(
	ctx *actor.Context,
	instanceType instanceType,
	instanceNum int,
) {
	// 1. Take the number of instances to launch + the number of pending spot requests to calculate how many instances to launch or shut down
	// 2. Launch or terminate the appropriate number of requests
	instType, ok := instanceType.(ec2InstanceType)
	if !ok {
		panic("cannot pass non-ec2InstanceType to ec2Cluster")
	}

	if instanceNum < 0 {
		return
	}

	// There may be pending spot requests that have been fulfilled or since we told the
	// scaleDecider how many instances were running. This means that we need to look
	// at instanceNum and pendingSpotRequest to decide what action we should actually
	// take. There are three cases:
	// 1. numInstancesToLaunch == len(pendingSpotRequests)
	// 	  - nothing needs to be done
	// 2. numInstancesToLaunch < len(pendingSpotRequests)
	//    - we need to cancel spotRequests, prioritizing those that do not have instances attached
	// 3. numInstancesToLaunch > len(pendingSpotRequests)
	//    - we need to create more spotRequests
	// First we need to inspect the pendingSpotRequests and clean up. If there are requests that
	// are unfulfillable, we should not include them in the calculation
	listSpotRequestResp, err := c.listSpotRequestsById(ctx, c.pendingSpotRequestIds, false)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot describe EC2 spot requests")
		return
	}

	runningInstanceIds, requestsWithoutInstances, unfulfillableRequests := parseDescribeSpotInstanceRequestResponse(listSpotRequestResp)
	c.handleUnfulfillableRequests(ctx, unfulfillableRequests)

	numNewInstanceRunningOrPending := len(requestsWithoutInstances) + len(runningInstanceIds)
	numNewInstancesDesired := instanceNum
	diff := numNewInstancesDesired - numNewInstanceRunningOrPending

	switch  {
	case diff == 0:
		ctx.Log().Info("The number of desired instances will be met by the current set of spot requests. " +
			"No need to launch more spot requests")
		return
	case diff > 0:
		ctx.Log().Infof("More instances are desired than can be met by the current set of spot requests. " +
			"Creating %d additional requests", diff)
		ctx.Log().Infof("launching %d EC2 spot requests", diff)
		resp, err := c.createSpotInstanceRequest(ctx, diff, false, instType)
		if err != nil {
			ctx.Log().WithError(err).Error("cannot launch EC2 spot requests")
			return
		}

		for _, request := range resp.SpotInstanceRequests {
			ctx.Log().Infof(
				"Launching spot request, %s, %s",
				*request.SpotInstanceRequestId,
				*request.State,
			)
		}
		return
	case diff < 0:
		ctx.Log().Infof("The set of current spot requests exceeds the desired number of instances." +
			" Shutting down requests.")
		var numPendingRequestsToDelete int
		var numRunningInstancesToDelete int
		if diff <= len(requestsWithoutInstances) {

			numPendingRequestsToDelete = diff
			numRunningInstancesToDelete = 0
		} else {
			numPendingRequestsToDelete = len(requestsWithoutInstances)
			numRunningInstancesToDelete = diff - numPendingRequestsToDelete
		}
		if numPendingRequestsToDelete > 0 {
			spotRequestsToCancel := requestsWithoutInstances[0: numPendingRequestsToDelete]
			_, err := c.terminateSpotInstanceRequest(ctx, spotRequestsToCancel, false)
			if err != nil {
				ctx.Log().WithError(err).Error("cannot cancel spot requests")
			}

			// Remember that the request may have been fulfilled since we checked, so
			// make sure we don't leave behind orphaned instances!

			// TODO: Do another scan of the spot requests to see if there are any instances to shut down
		}
		if numRunningInstancesToDelete > 0 {
			instanceIdsToTerminate := runningInstanceIds[0:numRunningInstancesToDelete]
			c.terminateSpot(ctx, instanceIdsToTerminate)
		}

		return
	}
}


func (c *awsCluster) handleUnfulfillableRequests(ctx *actor.Context, unfulfillableRequests []*unfulfillableSpotRequest) {
	// TODO: Add error logs
	// TODO: Clean up spot requests to the extent possible
	return
}




func (c *awsCluster) terminateSpot(ctx *actor.Context, instanceIDs []*string) {
	if len(instanceIDs) == 0 {
		return
	}

	ctx.Log().Infof(
		"terminating %d EC2 spot instances: %s",
		len(instanceIDs),
		instanceIDs,
	)

	// First we need to terminate the spot requests. This will leave behind instances
	spotRequestIds, err := c.getSpotRequestIdsGivenInstanceIds(ctx, instanceIDs, false)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot list spot request ids given EC2 instance ids")
		return
	}

	_, err = c.terminateSpotInstanceRequest(ctx, spotRequestIds, false)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot terminate EC2 spot instance requests")
		return
	}

	res, err := c.terminateInstances(instanceIDs, false)
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


// EC2 calls
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
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("spot-instances-request"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String(c.TagKey),
						Value: aws.String(c.TagValue),
					},
				},
			},
		},
		SpotPrice:                    aws.String(c.AWSClusterConfig.SpotMaxPrice),
		ValidFrom:                    aws.Time(validFrom),
	}

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


func (c *awsCluster) listSpotRequestsById(
	ctx *actor.Context,
	spotRequestIds []*string,
	dryRun bool,
) (*ec2.DescribeSpotInstanceRequestsOutput, error) {

	if dryRun {
		ctx.Log().Debug("dry run of listSpotRequestsById.")
	}

	input := &ec2.DescribeSpotInstanceRequestsInput{
		DryRun:                 aws.Bool(dryRun),
		SpotInstanceRequestIds: spotRequestIds,
	}

	return c.client.DescribeSpotInstanceRequests(input)
}


type unfulfillableSpotRequest struct {
	SpotRequestId string
	State string
	StatusCode string
	StatusMessage string
}

func parseDescribeSpotInstanceRequestResponse(
	response *ec2.DescribeSpotInstanceRequestsOutput,
) (runningInstanceIds []*string, healthyPendingRequests []*string, unfulfillableRequests []*unfulfillableSpotRequest) {

	unfulfillableRequests = make([]*unfulfillableSpotRequest, 0, 0)
	healthyPendingRequests = make([]*string, 0, 0)
	runningInstanceIds = make([]*string, 0, 0)

	for _, request := range response.SpotInstanceRequests {
		if request.InstanceId == nil {
			if spotRequestIsUnfulfillable(*request) {
				unfulfillableRequests = append(unfulfillableRequests, &unfulfillableSpotRequest{
					SpotRequestId: *request.SpotInstanceRequestId,
					State: *request.State,
					StatusCode: *request.Status.Code,
					StatusMessage: *request.Status.Message,
				})
			} else {
				healthyPendingRequests = append(healthyPendingRequests, request.SpotInstanceRequestId)
			}

		} else {
			runningInstanceIds = append(runningInstanceIds, request.InstanceId)
		}
	}
	return
}



func (c *awsCluster) getSpotRequestIdsGivenInstanceIds(
	ctx *actor.Context,
	instanceIds []*string,
	dryRun bool,
) ([]*string, error) {

	if dryRun {
		ctx.Log().Debug("dry run of getSpotRequestIdsGivenInstanceIds.")
	}

	input := &ec2.DescribeSpotInstanceRequestsInput{
		DryRun:                 aws.Bool(dryRun),
		Filters:                []*ec2.Filter{
			{
				Name: aws.String("instance-id"),
				Values: instanceIds,
			},
		},
	}

	resp, err := c.client.DescribeSpotInstanceRequests(input)
	if err != nil {
		return nil, err
	}

	spotRequestIds := make([]*string, 0, len(resp.SpotInstanceRequests))
	for _, spotRequest := range resp.SpotInstanceRequests {
		spotRequestIds = append(spotRequestIds, spotRequest.SpotInstanceRequestId)
	}

	return spotRequestIds, nil
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


func spotRequestIsUnfulfillable(
	requestInfo ec2.SpotInstanceRequest,
) bool {
	// TODO: Implement
	return false
}