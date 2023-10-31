//nolint:exhaustivestruct
package kubernetesrm

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	k8sV1 "k8s.io/api/core/v1"
)

func TestGetDetContainerSecurityContext(t *testing.T) {
	// Agent user group specified.
	aug := &model.AgentUserGroup{
		UID: 1001,
		GID: 1002,
	}
	secContext := getDetContainerSecurityContext(aug, nil)
	require.NotNil(t, secContext.RunAsUser)
	require.Equal(t, int64(aug.UID), *secContext.RunAsUser)
	require.NotNil(t, secContext.RunAsGroup)
	require.Equal(t, int64(aug.GID), *secContext.RunAsGroup)

	// Trying to specify RunAsUser in pod spec gets overwritten.
	expectedCaps := []k8sV1.Capability{"TEST"}
	spec := &expconf.PodSpec{
		Spec: k8sV1.PodSpec{
			Containers: []k8sV1.Container{
				{
					Name: model.DeterminedK8ContainerName,
					SecurityContext: &k8sV1.SecurityContext{
						Capabilities: &k8sV1.Capabilities{
							Add: expectedCaps,
						},
						RunAsUser:  ptrs.Ptr(int64(32)),
						RunAsGroup: ptrs.Ptr(int64(33)),
					},
				},
			},
		},
	}
	secContext = getDetContainerSecurityContext(aug, spec)
	require.NotNil(t, secContext.RunAsUser)
	require.Equal(t, int64(aug.UID), *secContext.RunAsUser)
	require.NotNil(t, secContext.RunAsGroup)
	require.Equal(t, int64(aug.GID), *secContext.RunAsGroup)
	require.Equal(t, expectedCaps, secContext.Capabilities.Add)

	// No agent user group still gets overwritten.
	secContext = getDetContainerSecurityContext(nil, spec)
	require.Nil(t, secContext.RunAsUser)
	require.Nil(t, secContext.RunAsGroup)
	require.Equal(t, expectedCaps, secContext.Capabilities.Add)
}

func TestAddNodeDisabledAffinityToPodSpec(t *testing.T) {
	hasDisabledLabel := func(p *k8sV1.Pod) {
		actualList := p.Spec.Affinity.
			NodeAffinity.
			RequiredDuringSchedulingIgnoredDuringExecution.
			NodeSelectorTerms[0].
			MatchExpressions
		expectedItem := k8sV1.NodeSelectorRequirement{
			Key:      "cluster-id",
			Operator: k8sV1.NodeSelectorOpDoesNotExist,
		}
		require.Contains(t, actualList, expectedItem)
	}

	p := &k8sV1.Pod{}
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")
	hasDisabledLabel(p)

	p = &k8sV1.Pod{
		Spec: k8sV1.PodSpec{
			Affinity: &k8sV1.Affinity{
				PodAffinity: &k8sV1.PodAffinity{},
			},
		},
	}
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")
	hasDisabledLabel(p)
	require.NotNil(t, p.Spec.Affinity.PodAffinity) // Didn't overwrite.

	pref := make([]k8sV1.PreferredSchedulingTerm, 7)
	p = &k8sV1.Pod{
		Spec: k8sV1.PodSpec{
			Affinity: &k8sV1.Affinity{
				NodeAffinity: &k8sV1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: pref,
				},
			},
		},
	}
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")
	hasDisabledLabel(p)
	require.Len(t, p.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution, 7)

	nodeSelectorTerm := k8sV1.NodeSelectorRequirement{
		Key:      "other-id",
		Operator: k8sV1.NodeSelectorOpDoesNotExist,
	}
	p = &k8sV1.Pod{
		Spec: k8sV1.PodSpec{
			Affinity: &k8sV1.Affinity{
				NodeAffinity: &k8sV1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &k8sV1.NodeSelector{
						NodeSelectorTerms: []k8sV1.NodeSelectorTerm{
							{
								MatchExpressions: []k8sV1.NodeSelectorRequirement{
									nodeSelectorTerm,
								},
							},
						},
					},
				},
			},
		},
	}
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")
	hasDisabledLabel(p)
	require.Contains(t, p.Spec.Affinity.
		NodeAffinity.
		RequiredDuringSchedulingIgnoredDuringExecution.
		NodeSelectorTerms[0].
		MatchExpressions, nodeSelectorTerm)

	// Test idempotency.
	copy := p.DeepCopy()
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")
	require.Equal(t, copy, p)
}

func TestLaterEnvironmentVariablesGetSet(t *testing.T) {
	dontBe := k8sV1.EnvVar{Name: "var", Value: "dontbe"}
	shouldBe := k8sV1.EnvVar{Name: "var", Value: "shouldbe"}
	env := expconf.EnvironmentConfig{
		RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
			RawCPU: []string{
				dontBe.Name + "=" + dontBe.Value,
				shouldBe.Name + "=" + shouldBe.Value,
			},
		},
	}

	p := pod{}
	actual, err := p.configureEnvVars(make(map[string]string), env, device.CPU)
	require.NoError(t, err)
	require.NotContains(t, actual, dontBe, "earlier variable set")
	require.Contains(t, actual, shouldBe, "later variable not set")
}

func TestAllPrintableCharactersInEnv(t *testing.T) {
	expectedValue := ""
	for i := 0; i <= 1024; i++ {
		if unicode.IsPrint(rune(i)) {
			expectedValue += string([]rune{rune(i)})
		}
	}

	env := expconf.EnvironmentConfig{
		RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
			RawCPU: []string{
				"test=" + expectedValue,
				"test2",
				"func=f(x)=x",
			},
		},
	}

	p := pod{}
	actual, err := p.configureEnvVars(make(map[string]string), env, device.CPU)
	require.NoError(t, err)
	require.Contains(t, actual, k8sV1.EnvVar{Name: "test", Value: expectedValue})
	require.Contains(t, actual, k8sV1.EnvVar{Name: "test2", Value: ""})
	require.Contains(t, actual, k8sV1.EnvVar{Name: "func", Value: "f(x)=x"})
}
