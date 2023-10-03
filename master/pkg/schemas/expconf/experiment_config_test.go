//nolint:exhaustivestruct
package expconf

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
)

func TestBindMountsMerge(t *testing.T) {
	e1 := ExperimentConfig{
		RawBindMounts: BindMountsConfig{
			BindMount{
				RawHostPath:      "/host/e1",
				RawContainerPath: "/container/e1",
			},
		},
	}
	e2 := ExperimentConfig{
		RawBindMounts: BindMountsConfig{
			BindMount{
				RawHostPath:      "/host/e2",
				RawContainerPath: "/container/e2",
			},
		},
	}
	out := schemas.Merge(e1, e2)
	assert.Assert(t, len(out.RawBindMounts) == 2)
	assert.Assert(t, out.RawBindMounts[0].RawHostPath == "/host/e1")
	assert.Assert(t, out.RawBindMounts[1].RawHostPath == "/host/e2")
}

func TestPodSpecMerge(t *testing.T) {
	e0 := EnvironmentConfig{
		RawPodSpec: &PodSpec{
			Spec: k8sV1.PodSpec{
				Hostname:         "e0HostName",
				Subdomain:        "e0SubDomain",
				ImagePullSecrets: []k8sV1.LocalObjectReference{{Name: "e0Secret"}},
			},
		},
	}
	require.Equal(t, e0, schemas.Merge(e0, EnvironmentConfig{}))
	require.Equal(t, e0, schemas.Merge(EnvironmentConfig{}, e0))

	e1 := EnvironmentConfig{
		RawPodSpec: &PodSpec{
			Spec: k8sV1.PodSpec{
				Hostname:           "e1HostName",
				ServiceAccountName: "e1ServiceAccount",
				ImagePullSecrets:   []k8sV1.LocalObjectReference{{Name: "e1Secret"}},
			},
		},
	}

	e0PriorityExpected := EnvironmentConfig{
		RawPodSpec: &PodSpec{
			Spec: k8sV1.PodSpec{
				Hostname:           "e0HostName",
				Subdomain:          "e0SubDomain",
				ServiceAccountName: "e1ServiceAccount",
				ImagePullSecrets: []k8sV1.LocalObjectReference{
					{Name: "e0Secret"},
					{Name: "e1Secret"},
				},
			},
		},
	}
	require.Equal(t, e0PriorityExpected, schemas.Merge(e0, e1))

	e1PriorityExpected := EnvironmentConfig{
		RawPodSpec: &PodSpec{
			Spec: k8sV1.PodSpec{
				Hostname:           "e1HostName",
				Subdomain:          "e0SubDomain",
				ServiceAccountName: "e1ServiceAccount",
				ImagePullSecrets: []k8sV1.LocalObjectReference{
					{Name: "e1Secret"},
					{Name: "e0Secret"},
				},
			},
		},
	}
	require.Equal(t, e1PriorityExpected, schemas.Merge(e1, e0))
}

func TestLogPatterns(t *testing.T) {
	inp := `[
{"pattern": "a", "policy": {"type": "on_failure_dont_retry"}},
{"pattern": "b", "policy": {"type": "on_failure_exclude_node"}},
{"pattern": "c", "policy": {"type": "send_webhook"}}
]`
	expected := LogPatternPoliciesConfig{
		LogPatternPolicy{RawPattern: "a", RawPolicy: &LogPolicy{
			RawOnFailureDontRetry: &DontRetryPolicy{},
		}},
		LogPatternPolicy{RawPattern: "b", RawPolicy: &LogPolicy{
			RawOnFailureExcludeNode: &OnFailureExcludeNodePolicy{},
		}},
		LogPatternPolicy{RawPattern: "c", RawPolicy: &LogPolicy{
			RawSendWebhook: &SendWebhookPolicy{},
		}},
	}

	var actual LogPatternPoliciesConfig
	require.NoError(t, json.Unmarshal([]byte(inp), &actual))
	require.Equal(t, expected, actual)
}

func TestName(t *testing.T) {
	config := ExperimentConfig{
		RawName: Name{
			RawString: ptrs.Ptr("my_name"),
		},
	}

	// Test marshaling.
	bytes, err := json.Marshal(config)
	assert.NilError(t, err)

	rawObj := map[string]interface{}{}
	err = json.Unmarshal(bytes, &rawObj)
	assert.NilError(t, err)

	var expect interface{} = "my_name"
	assert.DeepEqual(t, rawObj["name"], expect)

	// Test unmarshaling.
	newConfig := ExperimentConfig{}
	err = json.Unmarshal(bytes, &newConfig)
	assert.NilError(t, err)

	assert.DeepEqual(t, newConfig.Name().String(), "my_name")
}
