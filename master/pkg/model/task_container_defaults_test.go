//nolint:exhaustivestruct
package model

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestEnvironmentVarsDefaultMerging(t *testing.T) {
	gpuType := "tesla"
	pbsSlotsPerNode := 99
	defaults := &TaskContainerDefaultsConfig{
		EnvironmentVariables: &RuntimeItems{
			CPU:  []string{"cpu=default"},
			CUDA: []string{"cuda=default"},
			ROCM: []string{"rocm=default"},
		},
		Slurm:                  expconf.SlurmConfigV0{
			RawGpuType:      &gpuType,
		},
		Pbs: expconf.PbsConfigV0{
			RawSlotsPerNode: &pbsSlotsPerNode,
		},
	}
	conf := expconf.ExperimentConfig{
		RawEnvironment: &expconf.EnvironmentConfig{
			RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
				RawCPU:  []string{"cpu=expconf"},
				RawCUDA: []string{"extra=expconf"},
			},
		},
	}
	defaults.MergeIntoExpConfig(&conf)

	require.Equal(t, conf.RawEnvironment.RawEnvironmentVariables,
		&expconf.EnvironmentVariablesMap{
			RawCPU:  []string{"cpu=default", "cpu=expconf"},
			RawCUDA: []string{"cuda=default", "extra=expconf"},
			RawROCM: []string{"rocm=default"},
		})

	require.Equal(t, *conf.RawSlurmConfig.RawGpuType, gpuType)
	require.Equal(t, *conf.RawPbsConfig.RawSlotsPerNode, pbsSlotsPerNode)
}
