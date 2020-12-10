package main

import (
	"testing"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"google.golang.org/api/compute/v1"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
)

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
`
	expected := internal.DefaultConfig()
	providerConf := provisioner.DefaultConfig()
	providerConf.GCP = provisioner.DefaultGCPClusterConfig()
	providerConf.GCP.BaseConfig = &compute.Instance{
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
	expected.ResourcePools = []resourcemanagers.ResourcePoolConfig{
		{
			PoolName:                 "default",
			Provider:                 providerConf,
			MaxCPUContainersPerAgent: 100,
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

func TestUnmarshalMasterConfiguration(t *testing.T) {
	config, err := getConfig(v.AllSettings())
	assert.NilError(t, err)

	c, err := config.CheckpointStorage.ToModel()
	if err != nil {
		t.Fatal(err)
	}

	if f := c.SaveTrialBest; f <= 0 {
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
