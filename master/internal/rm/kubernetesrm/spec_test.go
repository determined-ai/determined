//nolint:exhaustruct
package kubernetesrm

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"

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

func TestAddDisallowedNodesToPodSpec(t *testing.T) {
	p := &k8sV1.Pod{}
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")

	copy := p.DeepCopy()

	// No block list adds anything.
	addDisallowedNodesToPodSpec(&sproto.AllocateRequest{
		BlockedNodes: nil,
	}, p)
	require.Equal(t, copy, p)

	addDisallowedNodesToPodSpec(&sproto.AllocateRequest{
		BlockedNodes: []string{"a1", "a2"},
	}, p)

	expectedA1 := k8sV1.NodeSelectorRequirement{
		Key:      "metadata.name",
		Operator: k8sV1.NodeSelectorOpNotIn,
		Values:   []string{"a1"},
	}
	expectedA2 := k8sV1.NodeSelectorRequirement{
		Key:      "metadata.name",
		Operator: k8sV1.NodeSelectorOpNotIn,
		Values:   []string{"a2"},
	}

	for _, e := range []k8sV1.NodeSelectorRequirement{expectedA1, expectedA2} {
		require.Contains(t, p.Spec.Affinity.
			NodeAffinity.
			RequiredDuringSchedulingIgnoredDuringExecution.
			NodeSelectorTerms[0].
			MatchFields, e)
	}
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

func TestDeterminedLabels(t *testing.T) {
	// fill out task spec
	taskSpec := tasks.TaskSpec{
		Owner:       createUser(),
		Workspace:   "test-workspace",
		TaskType:    model.TaskTypeCommand,
		TaskID:      model.NewTaskID().String(),
		ContainerID: "container-id",
	}

	p := pod{
		req: &sproto.AllocateRequest{
			ResourcePool: "test-rp",
		},
		submissionInfo: &podSubmissionInfo{
			taskSpec: taskSpec,
		},
	}

	// define expectations
	expectedLabels := map[string]string{
		userLabel:         taskSpec.Owner.Username,
		workspaceLabel:    taskSpec.Workspace,
		resourcePoolLabel: p.req.ResourcePool,
		taskTypeLabel:     string(taskSpec.TaskType),
		taskIDLabel:       taskSpec.TaskID,
		containerIDLabel:  taskSpec.ContainerID,
	}

	spec := p.configurePodSpec(make([]k8sV1.Volume, 1), k8sV1.Container{},
		k8sV1.Container{}, make([]k8sV1.Container, 1), &k8sV1.Pod{}, "scheduler")

	// confirm pod spec has required labels
	require.NotNil(t, spec)
	for expectedKey, expectedValue := range expectedLabels {
		require.Equal(t, spec.ObjectMeta.Labels[expectedKey], expectedValue)
	}
}
func TestTrialLabels(t *testing.T) {
	// fill out task spec
	experimentID := "5"
	trialRequestID := model.NewTaskID().String()
	taskSpec := tasks.TaskSpec{
		Owner:       createUser(),
		Workspace:   "test-workspace",
		TaskType:    model.TaskTypeTrial,
		TaskID:      experimentID + "." + trialRequestID,
		ContainerID: "container-id",
	}

	p := pod{
		req: &sproto.AllocateRequest{
			ResourcePool: "test-rp",
		},
		submissionInfo: &podSubmissionInfo{
			taskSpec: taskSpec,
		},
	}

	// define expectations
	expectedLabels := map[string]string{
		userLabel:           taskSpec.Owner.Username,
		workspaceLabel:      taskSpec.Workspace,
		resourcePoolLabel:   p.req.ResourcePool,
		taskTypeLabel:       string(taskSpec.TaskType),
		taskIDLabel:         taskSpec.TaskID,
		experimentIDLabel:   experimentID,
		trialRequestIDLabel: trialRequestID,
		containerIDLabel:    taskSpec.ContainerID,
	}

	t.Run("correctly formatted", func(t *testing.T) {

		spec := p.configurePodSpec(make([]k8sV1.Volume, 1), k8sV1.Container{},
			k8sV1.Container{}, make([]k8sV1.Container, 1), &k8sV1.Pod{}, "scheduler")

		// confirm pod spec has required labels
		require.NotNil(t, spec)
		for expectedKey, expectedValue := range expectedLabels {
			require.Equal(t, expectedValue, spec.ObjectMeta.Labels[expectedKey])
		}
	})

	t.Run("badly formatted: too many", func(t *testing.T) {

		p.submissionInfo.taskSpec.TaskID = "a.b.c"

		// define expectations
		expectedLabels[taskIDLabel] = p.submissionInfo.taskSpec.TaskID
		expectedLabels[experimentIDLabel] = ""
		expectedLabels[trialRequestIDLabel] = ""

		spec := p.configurePodSpec(make([]k8sV1.Volume, 1), k8sV1.Container{},
			k8sV1.Container{}, make([]k8sV1.Container, 1), &k8sV1.Pod{}, "scheduler")

		// confirm pod spec has required labels
		require.NotNil(t, spec)
		for expectedKey, expectedValue := range expectedLabels {
			require.Equal(t, expectedValue, spec.ObjectMeta.Labels[expectedKey])
		}
	})

	t.Run("badly formatted: not enough", func(t *testing.T) {
		p.submissionInfo.taskSpec.TaskID = "a"

		// define expectations
		expectedLabels[taskIDLabel] = p.submissionInfo.taskSpec.TaskID
		expectedLabels[experimentIDLabel] = ""
		expectedLabels[trialRequestIDLabel] = ""

		spec := p.configurePodSpec(make([]k8sV1.Volume, 1), k8sV1.Container{},
			k8sV1.Container{}, make([]k8sV1.Container, 1), &k8sV1.Pod{}, "scheduler")

		// confirm pod spec has required labels
		require.NotNil(t, spec)
		for expectedKey, expectedValue := range expectedLabels {
			require.Equal(t, expectedValue, spec.ObjectMeta.Labels[expectedKey])
		}
	})
}
