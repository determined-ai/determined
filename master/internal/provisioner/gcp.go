package provisioner

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/api/compute/v1"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func init() {
	// set the seed of the random package for pet name generator
	rand.Seed(time.Now().UTC().UnixNano())
}

// gcpCluster wraps a GCE client. Determined recognizes agent GCE instances by:
// 1. A specific key/value pair label.
// 2. Names of agents that are equal to the instance names.
type gcpCluster struct {
	*GCPClusterConfig
	masterURL url.URL
	metadata  []*compute.MetadataItems

	client *compute.Service
}

func newGCPCluster(config *Config) (*gcpCluster, error) {
	if err := config.GCP.initDefaultValues(); err != nil {
		return nil, errors.Wrap(err, "failed to initialize auto configuration")
	}
	// This following GCP service is created using GCP Credentials without explicitly configuration
	// in the code. However you need to do the following settings.
	// See https://cloud.google.com/docs/authentication/production
	// 1. Use service account for GCP
	//    The following scope on cloud API access:
	//    "Compute Engine": "Read Write".
	// 2. Use a environment variable
	//    ```
	//    export GOOGLE_APPLICATION_CREDENTIALS="[PATH]"
	//    ```
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create GCP compute engine client")
	}

	masterURL, err := url.Parse(config.MasterURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse master url")
	}

	startupScriptBase64 := base64.StdEncoding.EncodeToString([]byte(config.StartupScript))
	containerScriptBase64 := base64.StdEncoding.EncodeToString([]byte(config.ContainerStartupScript))
	startupScript := string(mustMakeAgentSetupScript(agentSetupScriptConfig{
		MasterHost:                   masterURL.Hostname(),
		MasterPort:                   masterURL.Port(),
		AgentNetwork:                 config.AgentDockerNetwork,
		AgentDockerRuntime:           config.AgentDockerRuntime,
		AgentDockerImage:             config.AgentDockerImage,
		StartupScriptBase64:          startupScriptBase64,
		ContainerStartupScriptBase64: containerScriptBase64,
		AgentID: `$(curl "http://metadata.google.internal/computeMetadata/v1/instance/` +
			`name" -H "Metadata-Flavor: Google")`,
	}))

	cluster := &gcpCluster{
		GCPClusterConfig: config.GCP,
		masterURL:        *masterURL,
		metadata: []*compute.MetadataItems{
			{
				Key:   "startup-script",
				Value: &startupScript,
			},
			{
				Key:   "determined-master-address",
				Value: &masterURL.Host,
			},
		},
		client: computeService,
	}

	return cluster, nil
}

func (c *gcpCluster) instanceType() instanceType {
	return c.InstanceType
}

func (c *gcpCluster) maxInstanceNum() int {
	return c.MaxInstances
}

func (c *gcpCluster) idFromInstance(inst *compute.Instance) string {
	return fmt.Sprintf("%v", inst.Name)
}

func (c *gcpCluster) idFromOperation(op *compute.Operation) string {
	strs := strings.Split(op.TargetLink, "/")
	return strs[len(strs)-1]
}

func (c *gcpCluster) agentNameFromInstance(inst *compute.Instance) string {
	return fmt.Sprintf("%v", inst.Name)
}

// See https://cloud.google.com/compute/docs/instances/instance-life-cycle.
var gceInstanceStates = map[string]InstanceState{
	"PROVISIONING": Starting,
	"STAGING":      Starting,
	"RUNNING":      Running,
	"STOPPING":     Stopping,
	"STOPPED":      Stopped,
	"SUSPENDING":   Stopping,
	"SUSPENDED":    Stopped,
	"TERMINATED":   Stopped,
}

func (c *gcpCluster) stateFromInstance(inst *compute.Instance) InstanceState {
	if state, ok := gceInstanceStates[inst.Status]; ok {
		return state
	}
	return Unknown
}

func (c *gcpCluster) generateInstanceName() string {
	return c.NamePrefix + petname.Generate(2, "-")
}

func (c *gcpCluster) list(ctx *actor.Context) ([]*Instance, error) {
	instances, err := c.listInstances()
	if err != nil {
		return nil, errors.Wrap(err, "cannot list GCE instances")
	}
	res := c.newInstances(instances)
	for i, inst := range res {
		if inst.State == Unknown {
			ctx.Log().Errorf("unknown instance state for instance %v: %v",
				inst.ID, instances[i])
		}
	}
	return res, nil
}

func (c *gcpCluster) launch(ctx *actor.Context, instanceType instanceType, instanceNum int) {
	instType, ok := instanceType.(gceInstanceType)
	if !ok {
		panic("cannot pass non-gce instanceType to gcpCluster")
	}

	if instanceNum <= 0 {
		return
	}

	ctx.Log().Infof("inserting %d GCE instances", instanceNum)
	var ops []*compute.Operation
	for i := 0; i < instanceNum; i++ {
		resp, err := c.insertInstance(instType)
		if err != nil {
			ctx.Log().WithError(err).Errorf("cannot insert GCE instance")
		} else {
			ops = append(ops, resp)
		}
	}

	if len(ops) == 0 {
		return
	}
	if _, ok := ctx.ActorOf(
		fmt.Sprintf("track-batch-operation-%s", uuid.New()),
		&gcpBatchOperationTracker{
			config: c.GCPClusterConfig,
			client: c.client,
			ops:    ops,
			postProcess: func(doneOps []*compute.Operation) {
				inserted := c.newInstancesFromOperations(doneOps)
				ctx.Log().Infof(
					"inserted %d/%d GCE instances: %s",
					len(inserted),
					instanceNum,
					fmtInstances(inserted),
				)
			},
		},
	); !ok {
		ctx.Log().Error("internal error tracking GCP operation batch")
		return
	}
}

func (c *gcpCluster) terminate(ctx *actor.Context, instances []string) {
	if len(instances) == 0 {
		return
	}

	ctx.Log().Infof(
		"deleting %d GCE instances: %s",
		len(instances),
		instances,
	)
	var ops []*compute.Operation
	for _, inst := range instances {
		resp, err := c.deleteInstance(inst)
		if err != nil {
			ctx.Log().WithError(err).Errorf("cannot delete GCE instance: %s", inst)
		} else {
			ops = append(ops, resp)
		}
	}

	if len(ops) == 0 {
		return
	}
	if _, ok := ctx.ActorOf(
		fmt.Sprintf("track-batch-operation-%s", uuid.New()),
		&gcpBatchOperationTracker{
			config: c.GCPClusterConfig,
			client: c.client,
			ops:    ops,
			postProcess: func(doneOps []*compute.Operation) {
				deleted := c.newInstancesFromOperations(doneOps)
				ctx.Log().Infof(
					"deleted %d/%d GCE instances: %s",
					len(deleted),
					len(instances),
					fmtInstances(deleted),
				)
			},
		},
	); !ok {
		ctx.Log().Error("internal error tracking GCP operation batch")
		return
	}
}

func (c *gcpCluster) newInstances(input []*compute.Instance) []*Instance {
	output := make([]*Instance, 0, len(input))
	for _, inst := range input {
		if inst == nil {
			continue
		}
		t, err := time.Parse(time.RFC3339, inst.CreationTimestamp)
		if err != nil {
			panic(errors.Wrap(err, "cannot parse GCE instance launching time"))
		}
		output = append(output, &Instance{
			ID:         c.idFromInstance(inst),
			LaunchTime: t,
			AgentName:  c.agentNameFromInstance(inst),
			State:      c.stateFromInstance(inst),
		})
	}
	return output
}

func (c *gcpCluster) newInstancesFromOperations(operations []*compute.Operation) []*Instance {
	instances := make([]*Instance, 0, len(operations))
	for _, op := range operations {
		instances = append(instances, &Instance{
			ID: c.idFromOperation(op),
		})
	}
	return instances
}

func (c *gcpCluster) listInstances() ([]*compute.Instance, error) {
	ctx := context.Background()
	var instances []*compute.Instance
	filter := fmt.Sprintf("labels.%s=%s", c.LabelKey, c.LabelValue)
	req := c.client.Instances.List(c.Project, c.Zone).Filter(filter)
	if err := req.Pages(ctx, func(page *compute.InstanceList) error {
		instances = append(instances, page.Items...)
		return nil
	}); err != nil {
		return nil, err
	}
	return instances, nil
}

func (c *gcpCluster) insertInstance(instanceType gceInstanceType) (*compute.Operation, error) {
	ctx := context.Background()

	rb := c.merge()
	rb.Name = c.generateInstanceName()
	if rb.Labels == nil {
		rb.Labels = make(map[string]string)
	}
	rb.Labels["determined-master-host"] = strings.ReplaceAll(c.masterURL.Hostname(), ".", "-")
	rb.Labels["determined-master-port"] = c.masterURL.Port()
	if rb.Metadata == nil {
		rb.Metadata = &compute.Metadata{}
	}
	rb.Metadata.Items = append(c.metadata, rb.Metadata.Items...)

	return c.client.Instances.Insert(c.Project, c.Zone, rb).Context(ctx).Do()
}

func (c *gcpCluster) deleteInstance(id string) (*compute.Operation, error) {
	ctx := context.Background()
	return c.client.Instances.Delete(c.Project, c.Zone, id).Context(ctx).Do()
}
