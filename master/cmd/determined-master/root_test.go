package main

import (
	"testing"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"google.golang.org/api/compute/v1"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/internal/provisioner"
)

func TestUnmarshalMasterConfigurationViaViper(t *testing.T) {
	raw := `
provisioner:
  provider: gcp
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
	expected.Provisioner = provisioner.DefaultConfig()
	expected.Provisioner.GCP = provisioner.DefaultGCPClusterConfig()
	expected.Provisioner.GCP.BaseConfig = &compute.Instance{
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

	if config.CheckpointStorage != nil {
		if f := *config.CheckpointStorage.SaveTrialBest; f <= 0 {
			t.Errorf("SaveTrialBest %d <= 0", f)
		}
	}
}
