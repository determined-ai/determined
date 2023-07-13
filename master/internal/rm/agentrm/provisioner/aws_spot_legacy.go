package provisioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// There was a bug where spot requests and instances were not being tagged with
// the resource pool, leading to multiple resource pools trying to manage the
// same instances. The code now uses a resource pool tag, but that means that
// instances with the old tag format are ignored. To make sure we aren't leaving
// orphaned instances after a version upgrade, we identify instances with the old
// format and clean them up. Because the spot API is eventually consistent, we
// repeat the cleanup after 5 minutes. This function can be removed once all users
// are on versions newer than 0.15.5.
func (c *awsCluster) cleanupLegacySpotInstances() {
	loggerSpotLegacy := c.syslog.WithField("codepath", "spotLegacy")

	// Repeat after 5 minutes to handle the fact that the AWS API is eventually consistent.
	go func() {
		loggerSpotLegacy.Infof("starting legacy spot cleanup operation")
		c.legacyCleanupActiveSpotRequestsAndInstances()
		c.legacyCleanupCanceledButInstanceRunningSpot()

		time.Sleep(5 * time.Minute)

		loggerSpotLegacy.Debugf("starting second pass of legacy spot cleanup")
		c.legacyCleanupActiveSpotRequestsAndInstances()
		c.legacyCleanupCanceledButInstanceRunningSpot()
		loggerSpotLegacy.Debugf("completed legacy spot cleanup")
	}()
}

func (c *awsCluster) legacyCleanupActiveSpotRequestsAndInstances() {
	loggerSpotLegacy := c.syslog.WithField("codepath", "spotLegacy")

	// List spot requests with the old format
	activeSpotReqs, err := c.legacyListActiveSpotInstanceRequests()
	if err != nil {
		loggerSpotLegacy.
			WithError(err).
			Errorf("cannot list active legacy spot requests")
		return
	}
	if activeSpotReqs.numReqs() == 0 {
		loggerSpotLegacy.Debugf("no active legacy spot requests to clean up")
		return
	}
	// Delete spot requests with old format
	loggerSpotLegacy.Infof(
		"terminating %d active legacy spot requests: %s",
		activeSpotReqs.numReqs(),
		strings.Join(activeSpotReqs.idsAsList(), ","))

	_, err = c.terminateSpotInstanceRequests(activeSpotReqs.idsAsListOfPointers(), false)
	if err != nil {
		loggerSpotLegacy.
			WithError(err).
			Errorf("unable to terminate active legacy spot requests")
	}

	// Delete spot instances associated with the active requests
	instancesToTerminate := activeSpotReqs.instanceIds()
	if len(instancesToTerminate) == 0 {
		loggerSpotLegacy.Debugf(
			"no instances associated with active legacy spot requests to terminate",
		)
		return
	}

	var instanceListToLog strings.Builder
	for idx, instanceID := range instancesToTerminate {
		if idx > 0 {
			instanceListToLog.WriteString(", ")
		}
		instanceListToLog.WriteString(*instanceID)
	}
	loggerSpotLegacy.Infof(
		"terminating %d legacy spot instances associated with active legacy spot requests: %s",
		len(instancesToTerminate),
		instanceListToLog.String())

	_, err = c.terminateInstances(instancesToTerminate)
	if err != nil {
		loggerSpotLegacy.
			WithError(err).
			Errorf("unable to terminate instances associated with active legacy spot requests")
	}
}

func (c *awsCluster) legacyCleanupCanceledButInstanceRunningSpot() {
	loggerSpotLegacy := c.syslog.WithField("codepath", "spotLegacy")

	loggerSpotLegacy.Debugf("listing CanceledButInstanceRunning requests")
	canceledButInstanceRunningSpotReqs, err := c.legacyListCanceledButInstanceRunningSpotRequests()
	if err != nil {
		loggerSpotLegacy.
			WithError(err).
			Debugf("unable to list CanceledButInstanceRunning legacy spot requests")
		return
	}
	// Delete any instances associated with canceledButInstanceRunning spot requests
	if canceledButInstanceRunningSpotReqs.numReqs() == 0 {
		loggerSpotLegacy.Debugf("no CanceledButInstanceRunning legacy spot requests to clean up")
		return
	}

	loggerSpotLegacy.
		Infof(
			"terminating %d legacy spot instances where requests are CanceledButInstanceRunning: %s",
			canceledButInstanceRunningSpotReqs.numReqs(),
			strings.Join(canceledButInstanceRunningSpotReqs.idsAsList(), ","),
		)

	_, err = c.terminateInstances(canceledButInstanceRunningSpotReqs.instanceIds())
	if err != nil {
		loggerSpotLegacy.
			WithError(err).
			Debugf("cannot terminate EC2 instances associated with canceled spot requests")
	}
}

func isLegacy(request *ec2.SpotInstanceRequest) bool {
	for _, tag := range request.Tags {
		if tag.Key != nil && *tag.Key == "determined-resource-pool" {
			return false
		}
	}
	return true
}

func (c *awsCluster) legacyListCanceledButInstanceRunningSpotRequests() (
	reqs *setOfSpotRequests, err error,
) {
	input := &ec2.DescribeSpotInstanceRequestsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String(fmt.Sprintf("tag:%s", c.config.TagKey)),
				Values: []*string{
					aws.String(c.config.TagValue),
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

func (c *awsCluster) legacyListActiveSpotInstanceRequests() (reqs *setOfSpotRequests, err error) {
	input := &ec2.DescribeSpotInstanceRequestsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String(fmt.Sprintf("tag:%s", c.config.TagKey)),
				Values: []*string{
					aws.String(c.config.TagValue),
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
		return nil, err
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
