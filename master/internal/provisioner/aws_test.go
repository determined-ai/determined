package provisioner

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
)

func newAWSTestConfig() *AWSClusterConfig {
	return &AWSClusterConfig{
		Region:         "us-west-2",
		ImageID:        "ami-07fbb063a8beac623",
		InstanceName:   "ci-determined-dynamic-agents",
		SSHKeyName:     "integrations-test",
		RootVolumeSize: 100,
		InstanceType:   "p2.xlarge",
		MaxInstances:   5,
	}
}

// These tests ends with `Cloud` require credentials in the form of a shared credential
// file IAM role with permissions. It also requires a `integrations-test` key and a security
// group `default`.
func TestAWSRequestWorkflowCloud(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	config := DefaultConfig()
	config.MasterURL = "http://test.master:8080"
	config.AWS = newAWSTestConfig()
	err := check.Validate(config)
	assert.NilError(t, err)

	cluster, err := newAWSCluster(config, fmt.Sprintf("ci-test-%s", uuid.New()))
	assert.NilError(t, err)
	err = cluster.dryRunRequests()
	assert.NilError(t, err)
	defer cleanupAWSInstances(t, cluster)

	launched := testAWSLaunchNewInstance(t, cluster)
	ids := make([]*string, 0, len(launched))
	for _, inst := range launched {
		ids = append(ids, &inst.ID)
	}
	testAWSTerminateInstances(t, cluster, ids)
}

func cleanupAWSInstances(t *testing.T, c *awsCluster) {
	instances, err := c.describeInstances(false)
	if err != nil {
		t.Error(err)
		return
	}

	if insts := c.newInstances(instances); len(insts) > 0 {
		t.Logf("during cleanup, found instances %s\n", fmtInstances(insts))
	}

	var ids []*string
	for _, inst := range instances {
		ids = append(ids, inst.InstanceId)
	}

	terminated, err := c.terminateInstances(ids, false)
	if err != nil {
		t.Error(err)
		return
	}

	if insts := c.newInstancesFromTerminateInstancesOutput(terminated); len(insts) > 0 {
		t.Logf("during cleanup, terminated %s\n", fmtInstances(insts))
	}
}

func testAWSLaunchNewInstance(t *testing.T, c *awsCluster) []*Instance {
	assert.Assert(t, c.client != nil)

	before, err := c.describeInstances(false)
	assert.NilError(t, err)
	t.Logf(
		"before launching, described EC2 instances: %v\n",
		fmtInstances(c.newInstances(before)),
	)

	launched, err := c.launchInstances(c.InstanceType, 1, false)
	assert.NilError(t, err)
	t.Logf(
		"launched EC2 instances: %v\n",
		fmtInstances(c.newInstances(launched.Instances)),
	)

	for i := 0; i < 5; i++ {
		after, err := c.describeInstances(false)
		assert.NilError(t, err)
		t.Logf("after launching, described EC2 instances: %v\n",
			fmtInstances(c.newInstances(after)),
		)
		if len(before)+len(launched.Instances) == len(after) {
			return c.newInstances(launched.Instances)
		}
		time.Sleep(3 * time.Second)
	}

	t.Fatal("could not find launched instances")
	return nil
}

func testAWSTerminateInstances(t *testing.T, c *awsCluster, ids []*string) {
	assert.Assert(t, c.client != nil)

	before, err := c.describeInstances(false)
	assert.NilError(t, err)
	t.Logf(
		"before terminating, described EC2 instances: %v\n",
		fmtInstances(c.newInstances(before)),
	)

	terminated, err := c.terminateInstances(ids, false)
	assert.NilError(t, err)

	t.Logf(
		"terminated EC2 instances: %v\n",
		fmtInstances(c.newInstancesFromTerminateInstancesOutput(terminated)),
	)

	for i := 0; i < 5; i++ {
		after, err := c.describeInstances(false)
		assert.NilError(t, err)
		t.Logf(
			"after terminating, described EC2 instances: %v\n",
			fmtInstances(c.newInstances(after)),
		)
		if len(before)-len(terminated.TerminatingInstances) == len(after) {
			return
		}
		time.Sleep(3 * time.Second)
	}

	t.Fatal("could not find terminated instances")
}
