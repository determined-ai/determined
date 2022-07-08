//nolint:exhaustivestruct
package model

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestEnvironmentVarsDefaultMerging(t *testing.T) {
	defaults := &TaskContainerDefaultsConfig{
		EnvironmentVariables: &RuntimeItems{
			CPU:  []string{"cpu=default"},
			CUDA: []string{"cuda=default"},
			ROCM: []string{"rocm=default"},
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
}
