package tasks

import (
	"testing"

	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestTaskSpecClone(t *testing.T) {
	//nolint:exhaustruct
	orig := &TaskSpec{
		Environment: expconf.EnvironmentConfig{
			RawPodSpec: &expconf.PodSpec{
				Spec: k8sV1.PodSpec{
					ServiceAccountName: "test",
				},
			},
		},
		ExtraEnvVars: map[string]string{"a": "true"},
	}

	cloned, err := orig.Clone()
	require.NoError(t, err)
	require.Equal(t, orig, cloned)

	// Actually deep cloned.
	orig.ExtraEnvVars["a"] = "diff"
	require.Equal(t, map[string]string{"a": "true"}, cloned.ExtraEnvVars)
}
