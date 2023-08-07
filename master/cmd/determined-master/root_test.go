package main

import (
	"testing"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"google.golang.org/api/compute/v1"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
)

const testWebhookSigningKey = "testWebhookSigningKey"

func TestUnmarshalMasterConfigurationViaViper(t *testing.T) {
	raw := `
resource_pools:
  - pool_name: default
    provider:
      type: gcp
      base_config:
        disks:
          - mode: READ_ONLY
            boot: false
            initializeParams:
              sourceImage: projects/determined-ai/global/images/determined-agent
              diskSizeGb: "200"
              diskType: projects/determined-ai/zones/us-central1-a/diskTypes/pd-ssd
            autoDelete: true
task_container_defaults:
  cpu_pod_spec:
    apiVersion: v1
    kind: Pod
    metadata:
      labels:
        "app.kubernetes.io/name": "cpu-label"
    spec:
      containers:
        - name: determined-container
webhooks:
    signing_key: testWebhookSigningKey
`

	expected := config.DefaultConfig()
	expected.Webhooks.SigningKey = testWebhookSigningKey
	providerConf := provconfig.DefaultConfig()
	providerConf.GCP = provconfig.DefaultGCPClusterConfig()
	providerConf.GCP.BaseConfig = &compute.InstanceProperties{
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Mode:       "READ_ONLY",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskSizeGb:  200,
					DiskType:    "projects/determined-ai/zones/us-central1-a/diskTypes/pd-ssd",
					SourceImage: "projects/determined-ai/global/images/determined-agent",
				},
			},
		},
	}
	expected.TaskContainerDefaults = model.TaskContainerDefaultsConfig{
		ShmSizeBytes: 4294967296,
		NetworkMode:  "bridge",
	}
	expected.ResourcePools = []config.ResourcePoolConfig{
		{
			PoolName:                 "default",
			Provider:                 providerConf,
			MaxAuxContainersPerAgent: 100,
			AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
		},
	}
	expected.TaskContainerDefaults.CPUPodSpec = &k8sV1.Pod{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Spec: k8sV1.PodSpec{
			Containers: []k8sV1.Container{
				{
					Name: "determined-container",
				},
			},
		},
		ObjectMeta: metaV1.ObjectMeta{
			Labels: map[string]string{
				"app.kubernetes.io/name": "cpu-label",
			},
		},
	}
	err := expected.Resolve()
	assert.NilError(t, err)
	err = mergeConfigBytesIntoViper([]byte(raw))
	assert.NilError(t, err)
	config, err := getConfig(v.AllSettings())
	assert.NilError(t, err)
	assert.DeepEqual(t, config, expected)
}

func TestMergeUnmarshalMasterConfigurationViaViper(t *testing.T) {
	raw1 := `
resource_pools:
  - pool_name: default
    provider:
      type: gcp
      base_config:
        disks:
          - mode: READ_ONLY
            boot: false
            initializeParams:
              sourceImage: projects/determined-ai/global/images/determined-agent
              diskSizeGb: "200"
              diskType: projects/determined-ai/zones/us-central1-a/diskTypes/pd-ssd
            autoDelete: true
task_container_defaults:
  cpu_pod_spec:
    apiVersion: v1
    kind: Pod
    metadata:
      labels:
        "app.kubernetes.io/nameToLowerCase": "cpu-label"
    spec:
      containers:
        - name: determined-container
webhooks:
    signing_key: testWebhookSigningKey
`
	raw2 := `
task_container_defaults:
  cpu_pod_spec:
    metadata:
      labels:
        "app.kubernetes.io/nameToLowerCase": "cpu-label"
`

	expected := config.DefaultConfig()
	expected.Webhooks.SigningKey = testWebhookSigningKey
	providerConf := provconfig.DefaultConfig()
	providerConf.GCP = provconfig.DefaultGCPClusterConfig()
	providerConf.GCP.BaseConfig = &compute.InstanceProperties{
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Mode:       "READ_ONLY",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskSizeGb:  200,
					DiskType:    "projects/determined-ai/zones/us-central1-a/diskTypes/pd-ssd",
					SourceImage: "projects/determined-ai/global/images/determined-agent",
				},
			},
		},
	}
	expected.TaskContainerDefaults = model.TaskContainerDefaultsConfig{
		ShmSizeBytes: 4294967296,
		NetworkMode:  "bridge",
	}
	expected.ResourcePools = []config.ResourcePoolConfig{
		{
			PoolName:                 "default",
			Provider:                 providerConf,
			MaxAuxContainersPerAgent: 100,
			AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
		},
	}
	expected.TaskContainerDefaults.CPUPodSpec = &k8sV1.Pod{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Spec: k8sV1.PodSpec{
			Containers: []k8sV1.Container{
				{
					Name: "determined-container",
				},
			},
		},
		ObjectMeta: metaV1.ObjectMeta{
			Labels: map[string]string{
				"app.kubernetes.io/nametolowercase": "cpu-label",
			},
		},
	}
	err := expected.Resolve()
	assert.NilError(t, err)

	// Merge the two configs
	err = mergeConfigBytesIntoViper([]byte(raw1))
	assert.NilError(t, err)
	err = mergeConfigBytesIntoViper([]byte(raw2))
	assert.NilError(t, err)

	config, err := getConfig(v.AllSettings())
	assert.NilError(t, err)
	assert.DeepEqual(t, config, expected)
}

func TestUnmarshalMasterConfiguration(t *testing.T) {
	config, err := getConfig(v.AllSettings())
	assert.NilError(t, err)

	c := schemas.WithDefaults(config.CheckpointStorage)

	if f := c.SaveTrialBest(); f <= 0 {
		t.Errorf("SaveTrialBest %d <= 0", f)
	}
}

func TestApplyBackwardsCompatibility(t *testing.T) {
	type testcase struct {
		name     string
		before   map[string]interface{}
		expected map[string]interface{}
		err      error
	}
	tcs := []testcase{
		{
			before: map[string]interface{}{
				"scheduler": map[string]interface{}{
					"fit": "best",
				},
				"provisioner": map[string]interface{}{
					"max_idle_agent_period":     "30s",
					"max_agent_starting_period": "30s",
				},
			},
			expected: map[string]interface{}{
				"resource_manager": map[string]interface{}{
					"type": "agent",
					"scheduler": map[string]interface{}{
						"fitting_policy": "best",
						"type":           "fair_share",
					},
					"default_cpu_resource_pool": "default",
					"default_gpu_resource_pool": "default",
				},
				"resource_pools": []map[string]interface{}{
					{
						"pool_name": "default",
						"provider": map[string]interface{}{
							"max_idle_agent_period":     "30s",
							"max_agent_starting_period": "30s",
						},
					},
				},
			},
		},
		{
			before: map[string]interface{}{
				"scheduler": map[string]interface{}{
					"fit": "best",
					"resource_provider": map[string]interface{}{
						"type":                "kubernetes",
						"master_service_name": "k8s-det",
					},
				},
			},
			expected: map[string]interface{}{
				"resource_manager": map[string]interface{}{
					"type": "kubernetes",
					"scheduler": map[string]interface{}{
						"fitting_policy": "best",
						"type":           "fair_share",
					},
					"master_service_name": "k8s-det",
				},
			},
		},
	}
	for ix := range tcs {
		tc := tcs[ix]
		t.Run(tc.name, func(t *testing.T) {
			after, err := applyBackwardsCompatibility(tc.before)
			assert.Equal(t, err, tc.err)
			assert.DeepEqual(t, after, tc.expected)
		})
	}
}
