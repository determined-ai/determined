package main

import (
	"testing"

	"github.com/spf13/viper"
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
	err := expected.Resolve()
	assert.NilError(t, err)
	err = mergeConfigBytesIntoViper([]byte(raw))
	assert.NilError(t, err)
	config, err := getConfig(viper.AllSettings())
	assert.NilError(t, err)
	assert.DeepEqual(t, config, expected)
}

func TestUnmarshalMasterConfiguration(t *testing.T) {
	config, err := getConfig(viper.AllSettings())
	assert.NilError(t, err)

	c, err := config.CheckpointStorage.ToModel()
	if err != nil {
		t.Fatal(err)
	}

	if f := c.SaveTrialBest; f <= 0 {
		t.Errorf("SaveTrialBest %d <= 0", f)
	}
}
