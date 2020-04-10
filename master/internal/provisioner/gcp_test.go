package provisioner

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
)

func newGCPTestConfig() *GCPClusterConfig {
	return &GCPClusterConfig{
		Project:             "determined-ai",
		Zone:                "us-east4-a",
		NamePrefix:          "ci-determined-dynamic-agents-",
		BootDiskSize:        100,
		BootDiskSourceImage: "projects/debian-cloud/global/images/family/debian-9",
		InstanceType: gceInstanceType{
			MachineType: "n1-standard-8",
			GPUType:     "nvidia-tesla-p4",
			GPUNum:      1,
		},
		MaxInstances: 5,
	}
}

// These tests ends with `Cloud` require credentials in the form of credential file, the path
// of which is specified by the environment variable `GOOGLE_APPLICATION_CREDENTIALS`.
func TestGCPRequestWorkflowCloud(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	t.Skip("skipping GCP integration test because we don't have credentials in our CI AMIs")

	config := Config{GCP: newGCPTestConfig()}
	err := check.Validate(&config)
	assert.NilError(t, err)
	cluster, err := newGCPCluster(&config, fmt.Sprintf("ci-test-%s", uuid.New()))
	assert.NilError(t, err)
	defer cleanupGCPInstances(t, cluster)

	inserted := testGCPInsertNewInstance(t, cluster)
	testGCPDeleteInstance(t, cluster, inserted)
}

func cleanupGCPInstances(t *testing.T, c *gcpCluster) {
	instances, err := c.listInstances()
	assert.NilError(t, err)

	insts := c.newInstances(instances)
	if len(insts) > 0 {
		t.Logf("during cleanup, found instances: %v\n", fmtInstances(insts))
	}

	var deletedIDs []string
	for _, inst := range insts {
		deleted, err := c.deleteInstance(inst.ID)
		if err != nil {
			t.Error(err)
		}

		deletedIDs = append(deletedIDs, c.idFromOperation(deleted))
	}

	if len(deletedIDs) > 0 {
		t.Logf("during cleanup, terminated GCE instances: %v\n", strings.Join(deletedIDs, ","))
	}
}

func testGCPInsertNewInstance(t *testing.T, c *gcpCluster) string {
	assert.Assert(t, c.client != nil)

	before, err := c.listInstances()
	assert.NilError(t, err)
	t.Logf(
		"before inserting, listed GCE instances: %v\n",
		fmtInstances(c.newInstances(before)),
	)

	inserted, err := c.insertInstance(c.InstanceType)
	assert.NilError(t, err)
	t.Logf("inserted GCE instance: %v\n", c.idFromOperation(inserted))

	for i := 0; i < 5; i++ {
		after, err := c.listInstances()
		assert.NilError(t, err)
		t.Logf(
			"after inserting, listed GCE instances: %v\n",
			fmtInstances(c.newInstances(after)),
		)
		if len(before)+1 == len(after) {
			return c.idFromOperation(inserted)
		}
		time.Sleep(3 * time.Second)
	}

	t.Fatal("could not find launched instances")
	return ""
}

func testGCPDeleteInstance(t *testing.T, c *gcpCluster, id string) {
	assert.Assert(t, c.client != nil)

	before, err := c.listInstances()
	assert.NilError(t, err)
	t.Logf(
		"before terminating, described GCE instances: %v\n",
		fmtInstances(c.newInstances(before)),
	)

	deleted, err := c.deleteInstance(id)
	assert.NilError(t, err)
	t.Logf("terminated GCE instances: %v\n", c.idFromOperation(deleted))
}
