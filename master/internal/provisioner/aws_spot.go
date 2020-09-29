package provisioner

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// Spot instances are created asynchronously. You create a spot request, the
// request is validated and, if there is available capacity at the given price,
// an instance will be created (spot request fulfilled). We use one-time spot
// requests rather than persistent requests - this means that if an instance is
// shut down, the spot request will not try to automatically launch a new instance.
// The main reason we do this is that it makes state management slightly simpler
// since AWS will not be doing any automatic provisioning.
//
// The link between spot request and instance is a little complicated. Once the
// spot request has ben fulfilled, the request will have a pointer to the instance
// id. If the spot request is cancelled, the instance will continue to run. The
// spot request will enter the status "request-canceled-and-instance-running".
// If the instance is stopped or terminated, either manually or automatically due
// to capacity, the spot request will enter a terminal state (either cancelled,
// closed or disabled).
//
// The scaleDecider interacts with the awsCluster through list, terminate, and
// launch calls. With spot, list will return the running spot instances. Terminate
// will terminate both the instances and the associated spot requests (technically
// not necessary, but good citizenship). Launch will adjust the number of open spot
// requests to match the desired cluster size.
//
// The main complexity in this code is around spot requests that have not yet been
// fulfilled. The scaleDecider is not aware of these because list only returns running
// instances. During list, the awsCluster records how many pending requests exist that
// the scaleDecider is not aware of. Then during launch, it looks at how many additional
// instances the scaleDecider is requesting and adjusts the number of spot requests to match.
//
// This means that spot is tightly coupled to the current implementation of scaleDecider
// where there is always a list call prior to a launch call.
//
// In some cases spot requests will never be able to be fulfilled and the user will need to
// make adjustments (no instances of a given type in a certain AZ, AWS capacity limits hit).
// This is surfaced via error logs.
//
// More information about the spot instance lifecycle -
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-request-status.html#spot-instance-bid-status-understand

func (c *awsCluster) listSpot(ctx *actor.Context) ([]*Instance, error) {
	// 1. List all non-terminal spot instance requests
	// 2. For any requests that indicate a failure, write out an error message
	//    and clean up the spot request (if needed).
	// 3. For each spot instance request, find the matching instances
	// 4. Keep track of the spot requests that haven't been fulfilled
	resp, err := c.listActiveSpotInstanceRequests(ctx, false)
	if err != nil {
		return nil, errors.Wrap(err, "cannot describe EC2 spot requests")
	}

	runningSpotInstances, pendingRequests, unfulfillableRequests := parseDescribeSpotInstanceRequestsResponse(resp)
	c.handleUnfulfillableRequests(ctx, unfulfillableRequests)

	c.pendingSpotRequestIds = pendingRequests

	instancesToReturn, err := c.describeInstancesById(runningSpotInstances, false)
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
	// 1. Look at how many new instances the scaleDecider is asking for and compare it to
	//    the number of pendingSpotRequests that the scaleDecider is not aware of
	// 2. Launch or terminate the appropriate number of requests
	instType, ok := instanceType.(ec2InstanceType)
	if !ok {
		panic("cannot pass non-ec2InstanceType to ec2Cluster")
	}

	if instanceNum < 0 {
		return
	}

	// There may be pending spot requests that have been fulfilled in the time since we told the
	// scaleDecider how many instances were running.
	// Look at instanceNum and pendingSpotRequests
	// to decide what action to take. There are three cases:
	// 1. numInstancesToLaunch == len(pendingSpotRequests)
	// 	  - nothing needs to be done
	// 2. numInstancesToLaunch < len(pendingSpotRequests)
	//    - we need to cancel spotRequests, prioritizing those that do not have running instances
	// 3. numInstancesToLaunch > len(pendingSpotRequests)
	//    - we need to create more spotRequests
	// First we need to inspect the pendingSpotRequests and clean up any requests that
	// are unfulfillable and so should not be included in this calculation
	listSpotRequestResp, err := c.listSpotRequestsById(ctx, c.pendingSpotRequestIds, false)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot describe EC2 spot requests")
		return
	}

	runningInstanceIds, pendingRequests, unfulfillableRequests := parseDescribeSpotInstanceRequestsResponse(listSpotRequestResp)
	c.handleUnfulfillableRequests(ctx, unfulfillableRequests)

	numNewInstanceRunningOrPending := len(pendingRequests) + len(runningInstanceIds)
	numNewInstancesDesired := instanceNum
	numAdditionalRequestsNeeded := numNewInstancesDesired - numNewInstanceRunningOrPending

	switch {
	case numAdditionalRequestsNeeded == 0:
		ctx.Log().Debugf("The number of desired instances will be met by the current set of spot requests. " +
			"No need to launch more spot requests")
		return
	case numAdditionalRequestsNeeded > 0:
		ctx.Log().Debugf("More instances are desired than can be met by the current set of spot requests. "+
			"Creating %d additional requests", numAdditionalRequestsNeeded)
		ctx.Log().Infof("launching %d EC2 spot requests", numAdditionalRequestsNeeded)
		resp, err := c.createSpotInstanceRequest(ctx, numAdditionalRequestsNeeded, false, instType)
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
	case numAdditionalRequestsNeeded < 0:
		ctx.Log().Infof("The set of current spot requests exceeds the desired number of instances." +
			" Shutting down requests.")
		var numPendingRequestsToDelete int
		var numRunningInstancesToDelete int
		if numAdditionalRequestsNeeded <= len(pendingRequests) {
			numPendingRequestsToDelete = numAdditionalRequestsNeeded
			numRunningInstancesToDelete = 0
		} else {
			numPendingRequestsToDelete = len(pendingRequests)
			numRunningInstancesToDelete = numAdditionalRequestsNeeded - numPendingRequestsToDelete
		}
		if numPendingRequestsToDelete > 0 {
			spotRequestsToCancel := pendingRequests[0:numPendingRequestsToDelete]
			_, err := c.terminateSpotInstanceRequest(ctx, spotRequestsToCancel, false)
			if err != nil {
				ctx.Log().WithError(err).Error("cannot cancel spot requests")
				return
			}

			// Remember that the requests may have been fulfilled since we checked, so
			// make sure we don't leave behind orphaned instances!
			listSpotRequestResp, err = c.listSpotRequestsById(ctx, c.pendingSpotRequestIds, false)
			if err != nil {
				ctx.Log().WithError(err).Error("cannot describe EC2 spot requests")
				return
			}
			_, _, unfulfillableRequests := parseDescribeSpotInstanceRequestsResponse(listSpotRequestResp)
			c.handleUnfulfillableRequests(ctx, unfulfillableRequests)

		}
		if numRunningInstancesToDelete > 0 {
			instanceIdsToTerminate := runningInstanceIds[0:numRunningInstancesToDelete]
			c.terminateSpot(ctx, instanceIdsToTerminate)
		}

		return
	}
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

type unfulfillableSpotRequest struct {
	SpotRequestId string
	State         string
	StatusCode    string
	StatusMessage string
	InstanceId    *string
}

func spotRequestIsUnfulfillable(
	requestInfo ec2.SpotInstanceRequest,
) bool {
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-request-status.html#spot-instance-bid-status-understand

	// Unfulfillable:
	// state=closed
	// state=cancelled
	// state=disabled

	// Unfulfillable not requiring cleanup
	// canceled-before-fulfillment
	// instance-terminated-by-price
	// instance-terminated-by-schedule
	// instance-terminated-by-service
	// instance-terminated-by-user
	// spot-instance-terminated-by-user
	// instance-terminated-launch-group-constraint
	// instance-terminated-no-capacity
	// marked-for-stop
	// marked-for-termination
	// instance-stopped-by-price
	// instance-stopped-by-user
	// instance-stopped-no-capacity
	// schedule-expired
	// system-error

	// Unfulfillable, maybe requiring cleanup
	// request-canceled-and-instance-running

	// Unfulfillable due to reason requiring user correction
	// status-code=bad-parameters, state=closed
	// constraint-not-fulfillable
	// limit-exceeded

	// Fulfillable (status-code)
	// az-group-constraint
	// capacity-not-available
	// fulfilled
	// launch-group-constraint
	// not-scheduled-yet
	// pending-evaluation
	// pending-fulfillment
	// placement-group-constraint
	// price-too-low
	return *requestInfo.State == "closed" || *requestInfo.State == "disabled" || *requestInfo.State == "cancelled"
}

func (c *awsCluster) handleUnfulfillableRequests(ctx *actor.Context, unfulfillableRequests []*unfulfillableSpotRequest) {
	// For requests that are in a terminal state,
	// clean up any orphaned instances and create error logs if there is user action required

	for _, unfulfillableRequest := range unfulfillableRequests {
		switch unfulfillableRequest.StatusCode {
		case
			"cancelled-before-fulfillment",
			"instance-terminated-by-price",
			"instance-terminated-by-schedule",
			"instance-terminated-by-service",
			"instance-terminated-by-user",
			"spot-instance-terminated-by-user",
			"instance-terminated-launch-group-constraint",
			"instance-terminated-no-capacity",
			"marked-for-stop",
			"marked-for-termination",
			"instance-stopped-by-price",
			"instance-stopped-by-user",
			"instance-stopped-no-capacity",
			"schedule-expired",
			"system-error":
			// Unfulfillable, not requiring cleanup
			continue
		case "request-canceled-and-instance-running":
			// Unfulfillable, maybe requiring cleanup
			c.terminateSpot(ctx, []*string{unfulfillableRequest.InstanceId})
			continue
		case
			"bad-parameters",
			"constraint-not-fulfillable",
			"limit-exceeded":
			// Unfulfillable, requiring user attention
			ctx.Log().
				WithField("spot-request-status-code", unfulfillableRequest.StatusCode).
				WithField("spot-request-status-message", unfulfillableRequest.StatusMessage).
				Error("a spot request cannot be fulfilled and the error message indicates that this is a permanent error requiring the user to fix something")
			continue
		}
	}

	return
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
		LaunchSpecification: &ec2.RequestSpotLaunchSpecification{
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
			EbsOptimized: nil, // TODO: We should enable this, but we need to confirm that all allowable instance support this
			ImageId:      aws.String(c.ImageID),
			InstanceType: aws.String(instanceType.name()),
			KeyName:      aws.String(c.SSHKeyName),

			UserData: aws.String(base64.StdEncoding.EncodeToString(c.ec2UserData)),
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
		SpotPrice: aws.String(c.AWSClusterConfig.SpotMaxPrice),
		ValidFrom: aws.Time(validFrom),
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
		DryRun: aws.Bool(dryRun),
		Filters: []*ec2.Filter{
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

	if len(spotRequestIds) == 0 {
		return &ec2.DescribeSpotInstanceRequestsOutput{}, nil
	}

	input := &ec2.DescribeSpotInstanceRequestsInput{
		DryRun:                 aws.Bool(dryRun),
		SpotInstanceRequestIds: spotRequestIds,
	}

	return c.client.DescribeSpotInstanceRequests(input)
}

// This function takes the output of a DescribeSpotInstanceRequests and
// divides the results into three groups:
//   1. Spot requests that have been fulfilled.
//      Represented by the instanceId.
//   2. Spot requests that are pending and will be fulfilled in time.
//      Represented by the spotRequestId.
//   3. Spot requests that have entered a permanently failed state.
//      Represented by an unfulfillableSpotRequest struct.
func parseDescribeSpotInstanceRequestsResponse(
	response *ec2.DescribeSpotInstanceRequestsOutput,
) (runningInstanceIds []*string, healthyPendingRequests []*string, unfulfillableRequests []*unfulfillableSpotRequest) {

	unfulfillableRequests = make([]*unfulfillableSpotRequest, 0, 0)
	healthyPendingRequests = make([]*string, 0, 0)
	runningInstanceIds = make([]*string, 0, 0)

	for _, request := range response.SpotInstanceRequests {
		if spotRequestIsUnfulfillable(*request) {
			unfulfillableRequests = append(unfulfillableRequests, &unfulfillableSpotRequest{
				SpotRequestId: *request.SpotInstanceRequestId,
				State:         *request.State,
				StatusCode:    *request.Status.Code,
				StatusMessage: *request.Status.Message,
				InstanceId:    request.InstanceId,
			})
			continue
		}
		if request.InstanceId == nil {
			healthyPendingRequests = append(healthyPendingRequests, request.SpotInstanceRequestId)
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

	if len(instanceIds) == 0 {
		return make([]*string, 0, 0), nil
	}

	input := &ec2.DescribeSpotInstanceRequestsInput{
		DryRun: aws.Bool(dryRun),
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-id"),
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
