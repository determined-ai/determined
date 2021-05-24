package provisioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// There was a bug where spot requests and instances were not being tagged with
// the resource pool, leading to multiple resource pools trying to manage the
// same instances. The code now uses a resource pool tag, but that means that
// instances with the old tag format are ignored. To make sure we aren't leaving
// orphaned instances after a version upgrade, we identify instances with the old
// format and clean them up. Because the spot API is eventually consistent, we
// repeat the cleanup after 5 minutes. This function can be removed once all users
// are on versions newer than 0.15.5.
func (c *awsCluster) cleanupLegacySpotInstances(ctx *actor.Context) {
	cleanup := func(ctx *actor.Context) {
		ctx.Log().
			WithField("codepath", "spotLegacy").
			Infof("Starting Legacy Spot cleanup operation")
		// List spot requests with the old format
		activeSpotReqs, err := c.legacyListActiveSpotInstanceRequests(ctx)
		if err != nil {
			ctx.Log().
				WithField("codepath", "spotLegacy").
				WithError(err).
				Debugf("cannot list active spot requests")
		} else {
			ctx.Log().
				WithField("codepath", "spotLegacy").
				Infof("Terminating %d active spot instance requests", activeSpotReqs.numReqs())
			// Delete spot requests with old format
			_, err2 := c.terminateSpotInstanceRequests(ctx, activeSpotReqs.idsAsListOfPointers(), false)
			if err2 != nil {
				ctx.Log().
					WithField("codepath", "spotLegacy").
					WithError(err).
					Debugf("unable to terminate legacy spot instance requests")
			}

			// Delete any instance associated with the active requests
			instancesToTerminate := newSetOfStrings()
			for _, req := range activeSpotReqs.iter() {
				if req.InstanceID != nil {
					instancesToTerminate.add(*req.InstanceID)
				}
			}
			ctx.Log().
				WithField("codepath", "spotLegacy").
				Infof("Terminating %d active spot instances", instancesToTerminate.length())
			_, err3 := c.terminateInstances(instancesToTerminate.asListOfPointers())
			if err3 != nil {
				ctx.Log().
					WithField("codepath", "spotLegacy").
					WithError(err).
					Debugf("unable to terminate instances associated with legacy spot instance requests")
			}
		}

		ctx.Log().
			WithField("codepath", "spotLegacy").
			Infof("Listing CanceledButInstanceRunning requests")
		canceledButInstanceRunningSpotReqs, err := c.legacyListCanceledButInstanceRunningSpotRequests(ctx)
		if err != nil {
			ctx.Log().
				WithField("codepath", "spotLegacy").
				WithError(err).
				Debugf("unable to list canceled but instance running legacy spot instance requests")
			return
		}
		// Delete any instances associated with canceledButInstanceRunning spot requests
		if canceledButInstanceRunningSpotReqs.numReqs() > 0 {
			ctx.Log().
				WithField("codepath", "spotLegacy").
				Infof(
					"Terminating %d spot instances where requests are "+
						"canceled but instance is running",
					canceledButInstanceRunningSpotReqs.numReqs(),
				)

			ctx.Log().
				WithField("codepath", "spotLegacy").
				Debugf(
					"terminating EC2 instances associated with canceled spot requests: %s",
					strings.Join(canceledButInstanceRunningSpotReqs.idsAsList(), ","),
				)
			_, err = c.terminateInstances(canceledButInstanceRunningSpotReqs.instanceIds())
			if err != nil {
				ctx.Log().
					WithField("codepath", "spotLegacy").
					WithError(err).
					Debugf("cannot terminate EC2 instances associated with canceled spot requests")
			}
		}
		ctx.Log().
			WithField("codepath", "spotLegacy").
			Infof("Completed Legacy Spot cleanup")
	}

	// Repeat after 5 minutes
	go func() {
		cleanup(ctx)
		time.Sleep(5 * time.Minute)
		cleanup(ctx)
	}()
}

func isLegacy(request *ec2.SpotInstanceRequest) bool {
	for _, tag := range request.Tags {
		if *tag.Key == "determined-resource-pool" {
			return false
		}
	}
	return true
}

func (c *awsCluster) legacyListCanceledButInstanceRunningSpotRequests(
	ctx *actor.Context,
) (reqs *setOfSpotRequests, err error) {
	input := &ec2.DescribeSpotInstanceRequestsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String(fmt.Sprintf("tag:%s", c.TagKey)),
				Values: []*string{
					aws.String(c.TagValue),
				},
			},
			{
				Name: aws.String("status-code"),
				Values: []*string{
					aws.String("request-canceled-and-instance-running"),
				},
			},
		},
	}

	response, err := c.client.DescribeSpotInstanceRequests(input)
	if err != nil {
		return
	}

	ret := newSetOfSpotRequests()
	for _, req := range response.SpotInstanceRequests {
		if isLegacy(req) {
			ret.add(&spotRequest{
				SpotRequestID: *req.SpotInstanceRequestId,
				State:         *req.State,
				StatusCode:    req.Status.Code,
				StatusMessage: req.Status.Message,
				InstanceID:    req.InstanceId,
				CreationTime:  *req.CreateTime,
			})
		}
	}

	return &ret, nil
}

func (c *awsCluster) legacyListActiveSpotInstanceRequests(
	ctx *actor.Context,
) (reqs *setOfSpotRequests, err error) {
	input := &ec2.DescribeSpotInstanceRequestsInput{
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

	response, err := c.client.DescribeSpotInstanceRequests(input)
	if err != nil {
		return
	}

	ret := newSetOfSpotRequests()
	for _, req := range response.SpotInstanceRequests {
		if isLegacy(req) {
			ret.add(&spotRequest{
				SpotRequestID: *req.SpotInstanceRequestId,
				State:         *req.State,
				StatusCode:    req.Status.Code,
				StatusMessage: req.Status.Message,
				InstanceID:    req.InstanceId,
				CreationTime:  *req.CreateTime,
			})
		}
	}
	return &ret, nil
}
