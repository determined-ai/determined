//go:build integration
// +build integration

package kubernetesrm

import (
	"testing"

	k8sV1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/set"
)

func TestAddDisallowedNodesToPodSpec(t *testing.T) {
	p := &k8sV1.Pod{}
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")

	copy := p.DeepCopy()

	// No blocklist adds nothing.
	logpattern.SetDisallowedNodesCacheTest(t, nil)
	require.Equal(t, copy, p)

	taskID := model.TaskID("21")
	logpattern.SetDisallowedNodesCacheTest(t, map[model.TaskID]*set.Set[string]{
		taskID: ptrs.Ptr(set.FromSlice([]string{"a1", "a2"})),
	})

	addDisallowedNodesToPodSpec(p, taskID)

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
