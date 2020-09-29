package provisioner

import (
	"encoding/base64"
	"fmt"
	"strings"
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

const spotRequestAsInstancePrefix = "spot-request-instance"

type spotRequest struct {
	SpotRequestId string
	State         string
	StatusCode    *string
	StatusMessage *string
	InstanceId    *string
}

func (c *awsCluster) listSpot(ctx *actor.Context) ([]*Instance, error) {
	activeSpotRequests, inactiveSpotRequests, err := c.listSpotInstanceRequests(ctx, false)
	if err != nil {
		return nil, errors.Wrap(err, "cannot describe EC2 spot requests")
	}

	ctx.Log().
		WithField("log-type", "listSpot.querySpotRequest").
		Infof("Retrieved spot requests: %d active requests and %d inactive requests",
			len(activeSpotRequests), len(inactiveSpotRequests))

	// Clean up time!
	// If there are spot requests that are cancelled but still have an active instance, delete the instance
	// If there are requests that failed and the error indicates that the user need to do something, log it
	c.cleanupInactiveSpotRequests(ctx, inactiveSpotRequests)

	// Next, update the requestSnapshot. It is the API response + the previous state
	// for any requests not included in the response. They might not be included because
	// the the requests were just submitted and the API hasn't caught up yet. Or they
	// could have transitioned from active to inactive (in which case we don't want to
	// include them in the snapshot).

	// In the case of a master restart, we may have active spot requests, but the
	// internal state will have been lost, so we always build a fresh snapshot
	// instead of updating in-place

	inactiveSpotRequestIds := make(map[string]bool)
	for _, inactiveRequest := range inactiveSpotRequests {
		inactiveSpotRequestIds[*inactiveRequest.SpotInstanceRequestId] = true
	}

	newActiveSpotRequestSnapshot := make(map[string]*spotRequest)
	for _, request := range activeSpotRequests {
		newActiveSpotRequestSnapshot[*request.SpotInstanceRequestId] = &spotRequest{
			SpotRequestId: *request.SpotInstanceRequestId,
			State:         *request.State,
			StatusCode:    request.Status.Code,
			StatusMessage: request.Status.Message,
			InstanceId:    request.InstanceId,
		}
	}

	// Go through the previous snapshot and add any spotRequests that are too new to be returned by the EC2 API
	for _, previousSpotRequestInfo := range c.activeSpotRequests {
		if _, ok := newActiveSpotRequestSnapshot[previousSpotRequestInfo.SpotRequestId]; !ok {
			// This requests was not one of the active spotRequests returned by the API
			if _, ok2 := inactiveSpotRequestIds[previousSpotRequestInfo.SpotRequestId]; !ok2 {
				// This requests was also not one of the inactive spotRequests returned by the API.
				// This means that it is a fresh spot request and the API just hasn't updated yet.
				// Include it in the snapshot as it
				newActiveSpotRequestSnapshot[previousSpotRequestInfo.SpotRequestId] = previousSpotRequestInfo
			}
		}
	}

	ctx.Log().
		WithField("log-type", "listSpot.updateActiveSpotRequestSnapshot").
		Infof("built a new snapshot of the active spot requests. there are %d active requests.",
			len(newActiveSpotRequestSnapshot))

	c.activeSpotRequests = newActiveSpotRequestSnapshot

	// Take the active spot requests and generate the Instances that will be returned. For spot requests
	// that have been fulfilled, just read the InstanceId and then query EC2 for details. For unfulfilled
	// spot instance requests, use placeholder instances so the scaleDecider is aware that more instances
	// are pending.
	runningSpotInstanceIds := make([]*string, 0, 0)
	pendingSpotRequestsAsInstances := make([]*Instance, 0, 0)
	for _, activeRequest := range c.activeSpotRequests {
		if activeRequest.InstanceId != nil {
			runningSpotInstanceIds = append(runningSpotInstanceIds, activeRequest.InstanceId)
		} else {
			dummyInstanceId := fmt.Sprintf("%s-%s", spotRequestAsInstancePrefix, activeRequest.SpotRequestId)
			pendingSpotRequestsAsInstances = append(pendingSpotRequestsAsInstances, &Instance{
				ID:         dummyInstanceId,
				LaunchTime: time.Now(),
				AgentName:  dummyInstanceId,
				State:      SpotRequestPendingAWS,
			})
		}
	}



	instancesToReturn, err := c.describeInstancesById(runningSpotInstanceIds, false)
	if err != nil {
		return []*Instance{}, errors.Wrap(err, "cannot describe EC2 instances")
	}
	realInstances := c.newInstances(instancesToReturn)
	for _, inst := range realInstances {
		if inst.State == Unknown {
			ctx.Log().Errorf("unknown instance state for instance %v", inst.ID)
		}
	}

	combined := append(realInstances, pendingSpotRequestsAsInstances...)
	return combined, nil
}


func (c *awsCluster) terminateSpot(ctx *actor.Context, instanceIDs []*string) {
	if len(instanceIDs) == 0 {
		return
	}

	instancesToTerminate := make([]*string, 0, 0)
	pendingSpotRequestsToTerminate := make([]*string, 0, 0)

	for _, instanceId := range instanceIDs {
		if strings.HasPrefix(*instanceId, spotRequestAsInstancePrefix) {
			spotRequestId := strings.TrimPrefix(*instanceId, spotRequestAsInstancePrefix)
			pendingSpotRequestsToTerminate = append(pendingSpotRequestsToTerminate, &spotRequestId)
		} else {
			instancesToTerminate = append(instancesToTerminate, instanceId)
		}
	}

	ctx.Log().Infof(
		"terminating %d EC2 instances and %d spot requests: %s,  %s",
		len(instancesToTerminate),
		len(pendingSpotRequestsToTerminate),
		instancesToTerminate,
		pendingSpotRequestsToTerminate,
	)

	terminateInstancesResponse, err := c.terminateInstances(instanceIDs, false)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot terminate EC2 instances")
	} else {
		terminated := c.newInstancesFromTerminateInstancesOutput(terminateInstancesResponse)
		ctx.Log().Infof(
			"terminated %d/%d EC2 instances: %s",
			len(terminated),
			len(instanceIDs),
			fmtInstances(terminated),
		)
	}

	_, err = c.terminateSpotInstanceRequests(ctx, pendingSpotRequestsToTerminate, false)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot terminate spot requests")
	} else {
		ctx.Log().Infof(
			"terminated %d spot requests: %s",
			len(pendingSpotRequestsToTerminate),
			pendingSpotRequestsToTerminate,
		)
	}

	// TODO: We could clean up the spot instances here. But it will be cleaned up on the next call to list() anyway
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

	ctx.Log().Infof("launching %d EC2 spot requests", instanceNum)
	resp, err := c.createSpotInstanceRequest(ctx, instanceNum, false, instType)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot launch EC2 spot requests")
		return
	}

	// Update the internal spotRequest tracker because there can be a large delay
	// before the API start returning information about this spot request and if
	// we don't track it internally, we will end up overprovisioning.
	for _, request := range resp.SpotInstanceRequests {
		c.activeSpotRequests[*request.SpotInstanceRequestId] = &spotRequest{
			SpotRequestId: *request.SpotInstanceRequestId,
			State:         *request.State,
			StatusCode:    request.Status.Code,
			StatusMessage: request.Status.Message,
			InstanceId:    nil,
		}

		ctx.Log().Infof(
			"Launching spot request, %s, %s",
			*request.SpotInstanceRequestId,
			*request.State,
		)
	}
	return

}





func (c *awsCluster) cleanupInactiveSpotRequests(ctx *actor.Context, inactiveSpotRequests []*ec2.SpotInstanceRequest) {
	// For requests that are in a terminal state, clean up any orphaned
	// instances and create error logs if there is user action required

	instancesToTerminate := make([]*string, 0, 0)

	for _, request := range inactiveSpotRequests {
		switch *request.Status.Code {
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
			instancesToTerminate = append(instancesToTerminate, request.InstanceId)
			continue
		case
			"bad-parameters",
			"constraint-not-fulfillable",
			"limit-exceeded":
			// Unfulfillable, requiring user attention
			// TODO: This could get extremely noisy in the logs. We should track whether we have already notified the user about the error related to this spot request
			ctx.Log().
				WithField("spot-request-status-code", request.Status.Code).
				WithField("spot-request-status-message", request.Status.Message).
				Error("a spot request cannot be fulfilled and the error message indicates that this is a permanent error requiring the user to fix something")
			continue
		}
	}

	_, err := c.terminateInstances(instancesToTerminate, false)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot terminate EC2 instances associated with inactive spot requests")
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

	validFrom := time.Now().Local().Add(time.Second * time.Duration(10))  // Potential bug with clock skew?
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


func (c *awsCluster) listSpotInstanceRequests(
	ctx *actor.Context,
	dryRun bool,
) (activeRequests []*ec2.SpotInstanceRequest, inactiveRequests []*ec2.SpotInstanceRequest, err error) {

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
		},
	}

	response, err := c.client.DescribeSpotInstanceRequests(input)
	if err != nil {
		return
	}

	activeRequests = make([]*ec2.SpotInstanceRequest, 0, 0)
	inactiveRequests = make([]*ec2.SpotInstanceRequest, 0, 0)

	for _, request := range response.SpotInstanceRequests {
		if *request.State == "open" || *request.State == "active" {
			activeRequests = append(activeRequests, request)
		} else {
			inactiveRequests = append(inactiveRequests, request)
		}
	}
	return
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

func (c *awsCluster) terminateSpotInstanceRequests(
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
