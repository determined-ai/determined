//go:build integration
// +build integration

package tasks

import (
	"path/filepath"
	"strings"
	"testing"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

//nolint:exhaustruct
func Test_GCCkptSpec_ToTaskSpec(t *testing.T) {
	tests := map[string]struct {
		expectedType         model.TaskType
		expectedEntrypoint   string
		expectedDescription  string
		expectedExtraEnvVars map[string]string
		expectedPodSpec      k8sV1.Pod
		gctaskSpec           GCCkptSpec
	}{
		"CheckpointGCPodSpecOverwriteTestCase": {
			expectedDescription:  "gc",
			expectedEntrypoint:   filepath.Join("/run/determined/checkpoint_gc", etc.GCCheckpointsEntrypointResource),
			expectedType:         model.TaskTypeCheckpointGC,
			expectedExtraEnvVars: map[string]string{"DET_TASK_TYPE": string(model.TaskTypeCheckpointGC)},
			expectedPodSpec: k8sV1.Pod{
				Spec: k8sV1.PodSpec{
					Volumes: []k8sV1.Volume{
						{
							Name: "CheckpointGC Pod Spec",
						},
					},
				},
			},
			gctaskSpec: GCCkptSpec{
				Base: TaskSpec{
					TaskContainerDefaults: model.TaskContainerDefaultsConfig{
						CheckpointGCPodSpec: &k8sV1.Pod{
							Spec: k8sV1.PodSpec{
								Volumes: []k8sV1.Volume{
									{
										Name: "CheckpointGC Pod Spec",
									},
								},
							},
						},
					},
				},
				LegacyConfig: expconf.LegacyConfig{
					Environment: expconf.EnvironmentConfig{
						RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
							RawCPU:  []string{"HOME=/where/the/heart/is"},
							RawCUDA: []string{"HOME=/where/the/heart/is"},
							RawROCM: []string{"HOME=/where/the/heart/is"},
						},
						RawPodSpec: &expconf.PodSpec{
							Spec: k8sV1.PodSpec{
								Volumes: []k8sV1.Volume{
									{
										Name: "Legacy Pod Spec",
									},
								},
							},
						},
					},
				},
			},
		},
		"LegacyPodSpecTestCase": {
			expectedDescription:  "gc",
			expectedEntrypoint:   filepath.Join("/run/determined/checkpoint_gc", etc.GCCheckpointsEntrypointResource),
			expectedType:         model.TaskTypeCheckpointGC,
			expectedExtraEnvVars: map[string]string{"DET_TASK_TYPE": string(model.TaskTypeCheckpointGC)},
			expectedPodSpec: k8sV1.Pod{
				Spec: k8sV1.PodSpec{
					Volumes: []k8sV1.Volume{
						{
							Name: "Legacy Pod Spec",
						},
					},
				},
			},
			gctaskSpec: GCCkptSpec{
				Base: TaskSpec{
					TaskContainerDefaults: model.TaskContainerDefaultsConfig{
						CPUPodSpec: &k8sV1.Pod{
							Spec: k8sV1.PodSpec{
								Volumes: []k8sV1.Volume{
									{
										Name: "CPU Pod Spec",
									},
								},
							},
						},
					},
				},
				LegacyConfig: expconf.LegacyConfig{
					Environment: expconf.EnvironmentConfig{
						RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
							RawCPU:  []string{"HOME=/where/the/heart/is"},
							RawCUDA: []string{"HOME=/where/the/heart/is"},
							RawROCM: []string{"HOME=/where/the/heart/is"},
						},
						RawPodSpec: &expconf.PodSpec{
							Spec: k8sV1.PodSpec{
								Volumes: []k8sV1.Volume{
									{
										Name: "Legacy Pod Spec",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for testCase, testVars := range tests {
		t.Run(testCase, func(t *testing.T) {
			err := etc.SetRootPath("../../static/srv/")
			require.NoError(t, err)
			res := testVars.gctaskSpec.ToTaskSpec()
			require.Equal(t, testVars.expectedDescription, strings.Split(res.Description, "-")[0])
			require.Equal(t, testVars.expectedPodSpec.Spec, res.Environment.RawPodSpec.Spec)
			require.Equal(t, testVars.expectedEntrypoint, res.Entrypoint[0])
			require.Equal(t, testVars.expectedExtraEnvVars, res.ExtraEnvVars)
			require.Equal(t, testVars.expectedType, res.TaskType)
		})
	}
}
