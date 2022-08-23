package provisioner

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/api/compute/v1"

	"github.com/determined-ai/determined/master/pkg/actor"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
)

// gcpCluster wraps a GCE client. Determined recognizes agent GCE instances by:
// 1. A specific key/value pair label.
// 2. Names of agents that are equal to the instance names.
type gcpCluster struct {
	*provconfig.GCPClusterConfig
	resourcePool string
	masterURL    url.URL
	metadata     []*compute.MetadataItems

	client *compute.Service
}

func newGCPCluster(
	resourcePool string, config *provconfig.Config, cert *tls.Certificate,
) (*gcpCluster, error) {
	if err := config.GCP.InitDefaultValues(); err != nil {
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

	var certBytes []byte
	if masterURL.Scheme == secureScheme && cert != nil {
		for _, c := range cert.Certificate {
			b := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: c,
			})
			certBytes = append(certBytes, b...)
		}
	}
	masterCertBase64 := base64.StdEncoding.EncodeToString(certBytes)

	startupScript := string(mustMakeAgentSetupScript(agentSetupScriptConfig{
		MasterHost:                   masterURL.Hostname(),
		MasterPort:                   masterURL.Port(),
		MasterCertName:               config.MasterCertName,
		SlotType:                     config.GCP.SlotType(),
		AgentNetwork:                 config.AgentDockerNetwork,
		AgentDockerRuntime:           config.AgentDockerRuntime,
		AgentDockerImage:             config.AgentDockerImage,
		AgentFluentImage:             config.AgentFluentImage,
		AgentReconnectAttempts:       config.AgentReconnectAttempts,
		AgentReconnectBackoff:        config.AgentReconnectBackoff,
		StartupScriptBase64:          startupScriptBase64,
		ContainerStartupScriptBase64: containerScriptBase64,
		MasterCertBase64:             masterCertBase64,
		AgentID: `$(curl "http://metadata.google.internal/computeMetadata/v1/instance/` +
			`name" -H "Metadata-Flavor: Google")`,
		ResourcePool: resourcePool,
		LogOptions:   config.GCP.BuildDockerLogString(),
	}))

	cluster := &gcpCluster{
		resourcePool:     resourcePool,
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

func (c *gcpCluster) slotsPerInstance() int {
	return c.GCPClusterConfig.SlotsPerInstance()
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

func (c *gcpCluster) prestart(ctx *actor.Context) {
	petname.NonDeterministicMode()
}

func (c *gcpCluster) list(ctx *actor.Context) ([]*Instance, error) {
	clientCtx := context.Background()
	var instances []*compute.Instance
	filter := fmt.Sprintf(
		"(labels.%s=%s) AND (labels.determined-resource-pool=%s)",
		c.LabelKey, c.LabelValue, c.resourcePool,
	)
	req := c.client.Instances.List(c.Project, c.Zone).Filter(filter)
	if err := req.Pages(
		clientCtx,
		func(page *compute.InstanceList) error {
			instances = append(instances, page.Items...)
			return nil
		},
	); err != nil {
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

func (c *gcpCluster) launch(ctx *actor.Context, instanceNum int) {
	if instanceNum <= 0 {
		return
	}

	var ops []*compute.Operation
	for i := 0; i < instanceNum; i++ {
		clientCtx := context.Background()

		rb := c.Merge()
		rb.Name = c.generateInstanceName()
		if rb.Labels == nil {
			rb.Labels = make(map[string]string)
		}
		rb.Labels["determined-master-host"] = strings.ReplaceAll(c.masterURL.Hostname(), ".", "-")
		rb.Labels["determined-master-port"] = c.masterURL.Port()
		rb.Labels["determined-resource-pool"] = c.resourcePool
		if rb.Metadata == nil {
			rb.Metadata = &compute.Metadata{}
		}
		rb.Metadata.Items = append(c.metadata, rb.Metadata.Items...)

		rb.MinCpuPlatform = provconfig.GetCPUPlatform(rb.MachineType)

		resp, err := c.client.Instances.Insert(c.Project, c.Zone, rb).Context(clientCtx).Do()
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

	var ops []*compute.Operation
	for _, inst := range instances {
		ClientCtx := context.Background()
		resp, err := c.client.Instances.Delete(c.Project, c.Zone, inst).Context(ClientCtx).Do()
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
