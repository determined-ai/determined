package provisioner

import (
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"sort"
	"strings"
	"time"
)

// Spot instances are created asynchronously. You create a spot request, the
// request is validated and, if there is available capacity at the given price,
// an instance will be created (spot request fulfilled). We use one-time spot
// requests rather than persistent requests - if an instance is
// shut down, the spot request will not try to automatically launch a new instance.
// We do this so state management is slightly simpler
// because AWS will not be doing any provisioning outside of our code that we need to account for.
//
// Once the
// spot request has been fulfilled, the request will have a pointer to the instance
// id. If the spot request is cancelled, the instance will continue to run. The
// spot request will have the status "request-canceled-and-instance-running".
// If the instance is stopped or terminated, either manually or automatically by AWS,
// the spot request will enter a terminal state (either cancelled,
// closed or disabled).
//
// The Spot Request API is eventually
// consistent and there may be a 30 second delay between creating a spot request
// and having it appear in listSpotRequests. We maintain an internal
// list of the spot requests we've created to prevent overprovisioning.
//
// In some cases spot requests will not be able to be fulfilled. Some errors may be permanently fatal (e.g. AWS does not have
// the instance type in this AZ) and requires user interaction to fix. In other cases, the error is transient (e.g. AWS account
// limits hit, internal system error) and may disappear without user interaction, but the user should be made aware of it. In these cases, we should not continue to spam requests because that
// will create many failed requests, hurting performance when querying the API. It is not clear how to differentiate these cases, so we handle them identically.
//
// When they occur, the error is surfaced via error logs. If all spotRequests created within the lookbackWindow failed,
// we will stop creating new requests for either the backoffDuration or until one of the existing spotRequest is fulfilled.
//
// More information about the spot instance lifecycle -
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-request-status.html#spot-instance-bid-status-understand


const spotRequestIdPrefix = "sir-"
const lookbackWindow = time.Minute * 2
const backoffDuration = time.Minute * 2
const clockSkewCorrectionFactor = 2

type spotRequest struct {
	SpotRequestId string
	State         string
	StatusCode    *string
	StatusMessage *string
	InstanceId    *string
	CreationTime *time.Time

}

type spotLoopState struct {
	activeSpotRequests map[string]*spotRequest
	onlyLogErrorOnceTracker map[string]bool
	inBackoffState bool
	backoffStart time.Time
	approximateClockSkew *time.Duration
	launchTimeOffset time.Duration
}


func (c *awsCluster) listSpot(ctx *actor.Context) ([]*Instance, error) {
	// This list operation will become slower as the number of spot requests grows. Spot Instance
	// requests are deleted from the AWS API four hours after they are canceled and their instances
	// are terminated. In normal usage this shouldn't be a performance issue, but it's important we
	// avoid endlessly resubmitting requests that will instantly fail.
	spotRequests, err := c.listSpotInstanceRequests(ctx, false)
	if err != nil {
		return nil, errors.Wrap(err, "cannot describe EC2 spot requests")
	}

	activeRequests, inactiveRequests := c.splitRequestsIntoActiveAndInactive(ctx, spotRequests)

	ctx.Log().
		WithField("log-type", "listSpot.querySpotRequest").
		Debugf("Retrieved spot requests: %d active requests and %d inactive requests",
			len(activeRequests), len(inactiveRequests))

	c.updateBackoffState(ctx, spotRequests)
	c.cleanup(ctx, spotRequests)
	c.updateActiveRequestSnapshot(ctx, activeRequests, inactiveRequests)
	instances, err := c.buildInstanceListFromActiveRequestSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return instances, nil
}



func (c *awsCluster) terminateSpot(ctx *actor.Context, instanceIDs []*string) {
	if len(instanceIDs) == 0 {
		return
	}

	instancesToTerminate := make([]*string, 0, 0)
	pendingSpotRequestsToTerminate := make([]*string, 0, 0)

	for _, instanceId := range instanceIDs {
		if strings.HasPrefix(*instanceId, spotRequestIdPrefix) {
			spotRequestId := instanceId
			pendingSpotRequestsToTerminate = append(pendingSpotRequestsToTerminate, spotRequestId)
		} else {
			instancesToTerminate = append(instancesToTerminate, instanceId)
		}
	}

	ctx.Log().Debugf(
		"terminating %d EC2 instances and %d spot requests: %s,  %s",
		len(instancesToTerminate),
		len(pendingSpotRequestsToTerminate),
		instancesToTerminate,
		pendingSpotRequestsToTerminate,
	)

	terminateInstancesResponse, err := c.terminateInstances(instancesToTerminate, false)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot terminate EC2 instances")
	} else {
		terminated := c.newInstancesFromTerminateInstancesOutput(terminateInstancesResponse)
		ctx.Log().Debugf(
			"terminated %d EC2 instances: %s",
			len(terminated),
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

	// TODO: Race condition - an instance could have been created between listing
	//       and terminating spot request. We could clean up the spot instances here.
	//       But it will be cleaned up on the next call to list() anyway.
}




func (c *awsCluster) launchSpot(
	ctx *actor.Context,
	instanceType instanceType,
	instanceNum int,
) {
	if c.spotLoopState.inBackoffState {
		backoffEnd := c.spotLoopState.backoffStart.Add(backoffDuration)
		if  time.Now().Before(backoffEnd) {
			ctx.Log().Infof("AWS spot provider refusing to launch spot. spot provider is in ErrorBackoff state because all recent spot requests have failed")
			return
		}
	}

	instType, ok := instanceType.(ec2InstanceType)
	if !ok {
		panic("cannot pass non-ec2InstanceType to ec2Cluster")
	}

	if instanceNum < 0 {
		return
	}

	ctx.Log().Infof("launching %d EC2 spot requests", instanceNum)
	resp, err := c.createSpotInstanceRequestCorrectingForClockSkew(ctx, instanceNum, false, instType)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot launch EC2 spot requests")
		return
	}

	// Update the internal spotRequest tracker because there can be a large delay
	// before the API starts including these requests in listSpotRequest API calls,
	// and if we don't track it internally, we will end up overprovisioning.
	for _, request := range resp.SpotInstanceRequests {
		c.spotLoopState.activeSpotRequests[*request.SpotInstanceRequestId] = &spotRequest{
			SpotRequestId: *request.SpotInstanceRequestId,
			State:         *request.State,
			StatusCode:    request.Status.Code,
			StatusMessage: request.Status.Message,
			CreationTime:  request.CreateTime,
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

func (c *awsCluster) attemptToApproximateClockSkew(ctx *actor.Context) {
	// Create a spot request to try to approximate how different the local clock is from the AWS API clock
	// If it fails, we assume
	if c.spotLoopState.approximateClockSkew == nil {
		ctx.Log().Infof("new AWS spot provisioner. launching spot request to determined approximate clock skew between local machine and AWS API.")
		localCreateTime := time.Now()
		resp, err := c.createSpotInstanceRequest(ctx, 1, false, c.AWSClusterConfig.InstanceType, time.Hour * 100)
		if err != nil {
			ctx.Log().
				WithError(err).
				Infof("error while launching spot request during clock skew approximation. Non-fatal error, " +
					"defaulting to assumption that AWS clock and local clock have minimal clock skew")
			zeroDur := time.Second * 0
			c.spotLoopState.approximateClockSkew = &zeroDur
			return
		}
		awsCreateTime := resp.SpotInstanceRequests[0].CreateTime
		approxClockSkew := awsCreateTime.Sub(localCreateTime)
		ctx.Log().Infof("AWS API clock is approximately %s ahead of local machine clock", approxClockSkew.String())
		// This spot request is one that is easy to
		for {
			ctx.Log().Infof("attempting to clean up spot request used to approximate clock skew")
			_, err = c.terminateSpotInstanceRequests(ctx, []*string{resp.SpotInstanceRequests[0].SpotInstanceRequestId}, false)
			if err == nil {
				ctx.Log().Infof("Successfully cleaned up spot request used to approximate clock skew")
				break
			}
			if awsErr, ok := err.(awserr.Error); ok {
				ctx.Log().Infof("AWS error while terminating spot request during clock skew approximation, %s, %s", awsErr.Code(), awsErr.Message())
				if awsErr.Code() != "InvalidSpotInstanceRequestID.NotFound" {
					return
				}
			} else {
				ctx.Log().Errorf("unknown error while launch spot instances, %s", err.Error())
				return
			}
			time.Sleep(time.Second*2)
		}

		clockSkewRoundedUp := roundDurationUp(approxClockSkew)
		c.spotLoopState.approximateClockSkew = &clockSkewRoundedUp
	}

}



func (c *awsCluster) splitRequestsIntoActiveAndInactive(ctx *actor.Context, spotRequests []*ec2.SpotInstanceRequest) (activeRequests []*ec2.SpotInstanceRequest, inactiveRequests []*ec2.SpotInstanceRequest){
	activeRequests = make([]*ec2.SpotInstanceRequest, 0, 0)
	inactiveRequests = make([]*ec2.SpotInstanceRequest, 0, 0)

	for _, request := range spotRequests {
		if *request.State == "open" || *request.State == "active" {
			activeRequests = append(activeRequests, request)
		} else {
			inactiveRequests = append(inactiveRequests, request)
		}
	}
	return activeRequests, inactiveRequests
}


func (c *awsCluster) updateActiveRequestSnapshot(ctx *actor.Context, activeRequests []*ec2.SpotInstanceRequest, inactiveRequests []*ec2.SpotInstanceRequest) {
	// Next, update the activeRequestSnapshot. It is the list API response + the previous state
	// for any requests not included in the response. They might not be included because
	// the the requests were just submitted and the API hasn't caught up yet. Or they
	// could have transitioned from active to inactive (in which case we don't want to
	// include them in the snapshot).

	// In the case of a master restart, we may have active spot requests, but the
	// internal state will have been lost, so we always build a fresh snapshot
	// instead of updating in-place
	inactiveSpotRequestIds := make(map[string]bool)
	for _, inactiveRequest := range inactiveRequests {
		inactiveSpotRequestIds[*inactiveRequest.SpotInstanceRequestId] = true
	}

	newActiveSpotRequestSnapshot := make(map[string]*spotRequest)
	for _, request := range activeRequests {
		newActiveSpotRequestSnapshot[*request.SpotInstanceRequestId] = &spotRequest{
			SpotRequestId: *request.SpotInstanceRequestId,
			State:         *request.State,
			StatusCode:    request.Status.Code,
			StatusMessage: request.Status.Message,
			InstanceId:    request.InstanceId,
			CreationTime: request.CreateTime,
		}
	}

	// Go through the previous snapshot and add any spotRequests that are too new to be returned by the EC2 API
	for _, previousSpotRequestInfo := range c.spotLoopState.activeSpotRequests {
		if _, ok := newActiveSpotRequestSnapshot[previousSpotRequestInfo.SpotRequestId]; !ok {
			// This requests was not one of the active spotRequests returned by the API
			if _, ok2 := inactiveSpotRequestIds[previousSpotRequestInfo.SpotRequestId]; !ok2 {
				// This requests was also not one of the inactive spotRequests returned by the API.
				// This means that it is a fresh spot request and the API just hasn't updated yet.
				// Include it in the snapshot as is.
				newActiveSpotRequestSnapshot[previousSpotRequestInfo.SpotRequestId] = previousSpotRequestInfo
			}
		}
	}

	ctx.Log().
		WithField("log-type", "listSpot.updateActiveSpotRequestSnapshot").
		Debugf("built a new snapshot of the active spot requests. there are %d active requests.",
			len(newActiveSpotRequestSnapshot))

	c.spotLoopState.activeSpotRequests = newActiveSpotRequestSnapshot
}


func (c *awsCluster) buildInstanceListFromActiveRequestSnapshot(ctx *actor.Context) ([]*Instance, error) {
	// Take the active spot requests and generate the Instances that will be returned. For spot requests
	// that have been fulfilled, just read the InstanceId and then query EC2 for details. For unfulfilled
	// spot instance requests, use placeholder instances so the scaleDecider is aware that more instances
	// are pending.
	runningSpotInstanceIds := make([]*string, 0, 0)
	pendingSpotRequestsAsInstances := make([]*Instance, 0, 0)
	for _, activeRequest := range c.spotLoopState.activeSpotRequests {
		if activeRequest.InstanceId != nil {
			runningSpotInstanceIds = append(runningSpotInstanceIds, activeRequest.InstanceId)
		} else {
			pendingSpotRequestsAsInstances = append(pendingSpotRequestsAsInstances, &Instance{
				ID:         activeRequest.SpotRequestId,
				LaunchTime: *activeRequest.CreationTime,
				AgentName:  activeRequest.SpotRequestId,
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
	ctx.Log().
		WithField("log-type", "listSpot.returnCombinedList").
		Debugf("Returning list of instances: %d EC2 instances and %d dummy spot instances for %d total.",
			len(realInstances), len(pendingSpotRequestsAsInstances), len(combined))
	return combined, nil
}


func roundDurationUp(d time.Duration) time.Duration {
	roundInterval := time.Second * 10
	rounded := d.Round(roundInterval)
	if rounded < d {
		rounded = rounded + roundInterval
	}
	return rounded
}

func (c *awsCluster) updateBackoffState(ctx *actor.Context, allSpotRequests []*ec2.SpotInstanceRequest) {
	// Look back over the lookbackWindow to see if the requests we have been creating have succeeded.
	// If there are no useful requests. don't update the backoff state. If all requests have failed,
	// enter backoffState if we aren't in it already. If there are any successful requests in the
	// window, exit backoffState.
	if c.spotLoopState.inBackoffState {
		backoffEnd := c.spotLoopState.backoffStart.Add(backoffDuration)
		if time.Now().After(backoffEnd) {
			c.spotLoopState.inBackoffState = false
		}
	}

	sort.SliceStable(allSpotRequests, func(i, j int) bool {
		return allSpotRequests[i].CreateTime.Before(*allSpotRequests[j].CreateTime)
	})

	requestsInLookbackWindow := make([]*ec2.SpotInstanceRequest, 0, 0)
	for _, request := range allSpotRequests {

		adjustedLocalTime := time.Now().Add(*c.spotLoopState.approximateClockSkew)
		if adjustedLocalTime.Before(request.CreateTime.Add(lookbackWindow)) {
			// pending-evaluation means the request has neither succeeded not failed, so don't include it.
			if *request.Status.Code != "pending-evaluation" {
				requestsInLookbackWindow = append(requestsInLookbackWindow, request)
			}
		}
	}
	if len(requestsInLookbackWindow) != 0 {
		successfulRequestFoundInLookbackWindow := false
		for _, request := range requestsInLookbackWindow {
			if *request.State == "open" || *request.State == "active" {
				successfulRequestFoundInLookbackWindow = true
				break
			}
		}
		if successfulRequestFoundInLookbackWindow {
			c.spotLoopState.inBackoffState = false
		} else {
			if !c.spotLoopState.inBackoffState {
				c.spotLoopState.inBackoffState = true
				c.spotLoopState.backoffStart = time.Now()
			}
		}
	}
}


func (c *awsCluster) cleanup(ctx *actor.Context, allSpotRequests []*ec2.SpotInstanceRequest) {
	// For requests that are in a terminal state, clean up any orphaned instances. For requests that
	// might require user attention to fix, log the error, but make sure we only log the error once

	instancesToTerminate := make([]*string, 0, 0)
	allRequestsInApi := make(map[string]bool)
	spotRequestsToNotifyUserAbout := make([]*ec2.SpotInstanceRequest, 0, 0)

	for _, request := range allSpotRequests {
		allRequestsInApi[*request.SpotInstanceRequestId] = true
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
			if _, ok := c.spotLoopState.onlyLogErrorOnceTracker[*request.SpotInstanceRequestId]; !ok {
				spotRequestsToNotifyUserAbout = append(spotRequestsToNotifyUserAbout, request)
				c.spotLoopState.onlyLogErrorOnceTracker[*request.SpotInstanceRequestId] = true
			}
			continue
		}
	}

	// Log errors in chronological order
	sort.SliceStable(spotRequestsToNotifyUserAbout, func(i, j int) bool {
		return spotRequestsToNotifyUserAbout[i].CreateTime.Before(*spotRequestsToNotifyUserAbout[j].CreateTime)
	})
	for _, request := range spotRequestsToNotifyUserAbout {
		ctx.Log().
			WithField("spot-request-status-code", request.Status.Code).
			WithField("spot-request-status-message", request.Status.Message).
			WithField("spot-request-creation-time", request.CreateTime).
			Error("a spot request cannot be fulfilled and may require user intervention")
	}

	_, err := c.terminateInstances(instancesToTerminate, false)
	if err != nil {
		ctx.Log().WithError(err).Error("cannot terminate EC2 instances associated with inactive spot requests")
	}

	// Keep the size of the onlyLogErrorOnceTracker small by removing items after
	// the AWS API stops keeping track of the request
	for spotRequestId, _ := range c.spotLoopState.onlyLogErrorOnceTracker {
		if _, ok := allRequestsInApi[spotRequestId]; !ok {
			delete(c.spotLoopState.onlyLogErrorOnceTracker, spotRequestId)
		}
	}

	return
}

// EC2 calls
func (c *awsCluster) createSpotInstanceRequestCorrectingForClockSkew(
	ctx *actor.Context,
	numInstances int,
	dryRun bool,
	instanceType ec2InstanceType,
) (resp *ec2.RequestSpotInstancesOutput, err error) {
	// Spot requests need to have a "launchTime" that is in the future. We build the
	// launch time locally while AWS evaluates it on their server, meaning that clock
	// skew can easily make requests invalid. This function retries launching spot
	// instances if the error is an invalid parameter (invalid time), increasing how
	// far in the future we set launchTime
	maxRetries := 5
	for numRetries := 0; numRetries <= maxRetries; numRetries += 1 {
		offset := *c.spotLoopState.approximateClockSkew + c.spotLoopState.launchTimeOffset
		resp, err := c.createSpotInstanceRequest(ctx, numInstances, dryRun, instanceType, offset)
		if err == nil {
			return resp, nil
		}

		if awsErr, ok := err.(awserr.Error); ok {
			ctx.Log().Infof("AWS error while launch spot instances, %s, %s", awsErr.Code(), awsErr.Message())
			if awsErr.Code() == "InvalidTime" {
				c.spotLoopState.launchTimeOffset = c.spotLoopState.launchTimeOffset * clockSkewCorrectionFactor
				ctx.Log().Infof("AWS error while launch spot instances - InvalidTime. Increasing launchOffset to %s to correct for clock skew", c.spotLoopState.launchTimeOffset.String())
			}
		} else {
			ctx.Log().Errorf("unknown error while launch spot instances, %s", err.Error())
			return nil, err
		}
	}
	return nil, err
}

func (c *awsCluster) createSpotInstanceRequest(
	ctx *actor.Context,
	numInstances int,
	dryRun bool,
	instanceType ec2InstanceType,
	launchTimeOffset time.Duration,
) (*ec2.RequestSpotInstancesOutput, error) {

	if dryRun {
		ctx.Log().Debug("dry run of createSpotInstanceRequest.")
	}
	idempotencyToken := uuid.New().String()

	validFrom := time.Now().Local().Add(launchTimeOffset)  // Potential bug with clock skew?
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
		ValidFrom: aws.Time(validFrom),
	}

	// Excluding the SpotPrice param automatically uses the on-demand price
	if c.SpotMaxPrice != SpotPriceNotSetPlaceholder {
		spotInput.SpotPrice = aws.String(c.AWSClusterConfig.SpotMaxPrice)
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
) (requests []*ec2.SpotInstanceRequest, err error) {

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

	return response.SpotInstanceRequests, nil
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


