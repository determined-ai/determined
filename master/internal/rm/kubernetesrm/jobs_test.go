//go:build integration

package kubernetesrm

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"
	k8error "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestGetNonDetPods(t *testing.T) {
	hiddenPods := []k8sV1.Pod{
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "no node name",
			},
		},
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name:   "has det label",
				Labels: map[string]string{determinedLabel: "t"},
			},
		},
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name:   "has det system label",
				Labels: map[string]string{determinedSystemLabel: "f"},
			},
		},
	}
	expectedPods := []k8sV1.Pod{
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "ns1",
			},
			Spec: k8sV1.PodSpec{
				NodeName: "a",
			},
		},
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "ns2",
			},
			Spec: k8sV1.PodSpec{
				NodeName: "a",
			},
		},
	}

	emptyNS := &mocks.PodInterface{}
	emptyNS.On("List", mock.Anything, mock.Anything).Once().
		Return(&k8sV1.PodList{Items: append(hiddenPods, expectedPods[0], expectedPods[1])}, nil)

	js := jobsService{
		podInterfaces: map[string]typedV1.PodInterface{
			"ns1": &mocks.PodInterface{},
			"ns2": &mocks.PodInterface{},
			"":    emptyNS,
		},
	}

	actualPods, err := js.getNonDetPods()
	require.NoError(t, err)
	require.ElementsMatch(t, expectedPods, actualPods)
}

func TestListPodsInAllNamespaces(t *testing.T) {
	detPods := []k8sV1.Pod{
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "ns1",
			},
			Spec: k8sV1.PodSpec{
				NodeName: "a",
			},
		},
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "ns2",
			},
			Spec: k8sV1.PodSpec{
				NodeName: "a",
			},
		},
	}

	ns1 := &mocks.PodInterface{}
	ns2 := &mocks.PodInterface{}

	emptyNS := &mocks.PodInterface{}

	js := jobsService{
		podInterfaces: map[string]typedV1.PodInterface{
			"ns1": ns1,
			"ns2": ns2,
			"":    emptyNS,
		},
	}

	// This pod is not part of js.podInterfaces.
	outsidePod := k8sV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "ns3",
		},
		Spec: k8sV1.PodSpec{
			NodeName: "b",
		},
	}

	var expectedPods []k8sV1.Pod
	copy(expectedPods, append(detPods, outsidePod))
	expectedPodList := k8sV1.PodList{Items: expectedPods}
	emptyNS.On("List", mock.Anything, mock.Anything).Once().
		Return(&k8sV1.PodList{Items: expectedPods}, nil)

	ctx := context.Background()
	opts := metaV1.ListOptions{}
	actualPodList, err := js.listPodsInAllNamespaces(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, actualPodList)
	require.ElementsMatch(t, expectedPodList.Items, actualPodList)

	forbiddenErr := k8error.NewForbidden(schema.GroupResource{}, "forbidden",
		fmt.Errorf("forbidden"))

	emptyNS.On("List", mock.Anything, mock.Anything).Twice().
		Return(nil, forbiddenErr)

	ns1.On("List", mock.Anything, mock.Anything).Twice().
		Return(&k8sV1.PodList{Items: []k8sV1.Pod{detPods[0]}}, nil)

	ns2.On("List", mock.Anything, mock.Anything).Once().
		Return(&k8sV1.PodList{Items: []k8sV1.Pod{detPods[1]}}, nil)

	actualPodList, err = js.listPodsInAllNamespaces(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, actualPodList)
	require.ElementsMatch(t, detPods, actualPodList)

	listErr := fmt.Errorf("something bad happened")
	ns2.On("List", mock.Anything, mock.Anything).Once().
		Return(nil, listErr)
	actualPodList, err = js.listPodsInAllNamespaces(ctx, opts)
	require.ErrorIs(t, err, listErr)
	require.Nil(t, actualPodList)
}

func TestHealthStatus(t *testing.T) {
	detPods := []k8sV1.Pod{
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "ns1",
			},
			Spec: k8sV1.PodSpec{
				NodeName: "a",
			},
		},
		{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "ns2",
			},
			Spec: k8sV1.PodSpec{
				NodeName: "a",
			},
		},
	}

	ns1 := &mocks.PodInterface{}
	ns2 := &mocks.PodInterface{}

	emptyNS := &mocks.PodInterface{}

	js := jobsService{
		podInterfaces: map[string]typedV1.PodInterface{
			"ns1": ns1,
			"ns2": ns2,
			"":    emptyNS,
		},
	}

	// This pod is not part of js.podInterfaces.
	outsidePod := k8sV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "ns3",
		},
		Spec: k8sV1.PodSpec{
			NodeName: "b",
		},
	}

	var expectedPods []k8sV1.Pod
	copy(expectedPods, append(detPods, outsidePod))
	emptyNS.On("List", mock.Anything, mock.Anything).Once().
		Return(&k8sV1.PodList{Items: expectedPods}, nil)

	health := js.HealthStatus(context.TODO())
	require.Equal(t, model.Healthy, health)

	emptyNS.On("List", mock.Anything, mock.Anything).Once().
		Return(nil, fmt.Errorf("couldnt list all pods"))

	health = js.HealthStatus(context.TODO())
	require.Equal(t, model.Unhealthy, health)

	forbiddenErr := k8error.NewForbidden(schema.GroupResource{}, "forbidden",
		fmt.Errorf("forbidden"))

	emptyNS.On("List", mock.Anything, mock.Anything).Twice().
		Return(nil, forbiddenErr)

	ns1.On("List", mock.Anything, mock.Anything).Twice().
		Return(&k8sV1.PodList{Items: []k8sV1.Pod{detPods[0]}}, nil)

	ns2.On("List", mock.Anything, mock.Anything).Once().
		Return(&k8sV1.PodList{Items: []k8sV1.Pod{detPods[1]}}, nil)

	health = js.HealthStatus(context.TODO())
	require.Equal(t, model.Healthy, health)

	ns2.On("List", mock.Anything, mock.Anything).Once().
		Return(nil, fmt.Errorf("couldnt list pods in namespace ns2"))
	health = js.HealthStatus(context.TODO())
	require.Equal(t, model.Unhealthy, health)
}

func TestJobScheduledStatus(t *testing.T) {
	// Pod has been created, but has zero PodConditions yet.
	pendingPod := k8sV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "test-pod",
		},
		Status: k8sV1.PodStatus{
			Conditions: make([]k8sV1.PodCondition, 0),
		},
	}
	js := jobsService{
		jobNameToPodNameToSchedulingState: make(map[string]map[string]sproto.SchedulingState),
	}
	jobName := "test-job"
	js.updatePodSchedulingState(jobName, pendingPod)
	actualState := js.jobSchedulingState(jobName)
	expectedState := sproto.SchedulingStateQueued
	require.Equal(t, expectedState, actualState)

	// Pod has been created, but the PodScheduled PodCondition is false.
	notScheduledPod := k8sV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "test-pod",
		},
		Status: k8sV1.PodStatus{
			Conditions: []k8sV1.PodCondition{
				{
					Type:   k8sV1.PodScheduled,
					Status: k8sV1.ConditionFalse,
				},
			},
		},
	}
	js.updatePodSchedulingState(jobName, notScheduledPod)
	actualState = js.jobSchedulingState(jobName)
	expectedState = sproto.SchedulingStateQueued
	require.Equal(t, expectedState, actualState)

	// Pod has been created, and the PodScheduled PodCondition is true.
	scheduledPod := k8sV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "test-pod",
		},
		Status: k8sV1.PodStatus{
			Conditions: []k8sV1.PodCondition{
				{
					Type:   k8sV1.PodScheduled,
					Status: k8sV1.ConditionTrue,
				},
			},
		},
	}
	js.updatePodSchedulingState(jobName, scheduledPod)
	actualState = js.jobSchedulingState(jobName)
	expectedState = sproto.SchedulingStateScheduled
	require.Equal(t, expectedState, actualState)
}

func TestTaintTolerated(t *testing.T) {
	cases := []struct {
		expected    bool
		taint       k8sV1.Taint
		tolerations []k8sV1.Toleration
	}{
		{
			expected: true,
			taint:    taintFooBar,
			tolerations: []k8sV1.Toleration{{
				Key:      taintFooBar.Key,
				Value:    taintFooBar.Value,
				Operator: k8sV1.TolerationOpEqual,
			}},
		}, {
			expected: true,
			taint:    taintFooBar,
			tolerations: []k8sV1.Toleration{{
				Key:      taintFooBar.Key,
				Operator: k8sV1.TolerationOpExists,
			}},
		}, {
			expected: true,
			taint:    taintFooBar,
			tolerations: []k8sV1.Toleration{
				{
					Key:      taintFooBar.Key,
					Value:    taintFooBar.Value,
					Operator: k8sV1.TolerationOpEqual,
				}, {
					Key:      "baz",
					Value:    "qux",
					Operator: k8sV1.TolerationOpEqual,
				},
			},
		}, {
			expected: false,
			taint:    taintFooBar,
			tolerations: []k8sV1.Toleration{{
				Key:      taintFooBar.Key,
				Value:    taintFooBar.Value + taintFooBar.Value,
				Operator: k8sV1.TolerationOpEqual,
			}},
		}, {
			expected:    false,
			taint:       taintFooBar,
			tolerations: []k8sV1.Toleration{},
		}, {
			expected:    false,
			taint:       taintFooBar,
			tolerations: nil,
		},
	}

	for i, c := range cases {
		actual := taintTolerated(c.taint, c.tolerations)
		require.Equal(t, c.expected, actual, "test case %d failed", i)
	}
}

func TestAllTaintsTolerated(t *testing.T) {
	cases := []struct {
		expected    bool
		taints      []k8sV1.Taint
		tolerations []k8sV1.Toleration
	}{
		{
			expected:    true,
			taints:      nil,
			tolerations: nil,
		}, {
			expected: true,
			taints:   nil,
			tolerations: []k8sV1.Toleration{
				{
					Key:      taintFooBar.Key,
					Value:    taintFooBar.Value,
					Operator: k8sV1.TolerationOpEqual,
				},
			},
		}, {
			expected:    false,
			taints:      []k8sV1.Taint{taintFooBar},
			tolerations: nil,
		},
	}

	for i, c := range cases {
		actual := allTaintsTolerated(c.taints, c.tolerations)
		require.Equal(t, c.expected, actual, "test case %d failed", i)
	}
}

func TestPodsCanBeScheduledOnNode(t *testing.T) {
	cases := []struct {
		name          string
		pod           *k8sV1.Pod
		node          *k8sV1.Node
		selectorMatch bool
		affinityMatch bool
	}{
		{"no task containers default", nil, nil, true, true},
		{
			"node labels nil", &k8sV1.Pod{Spec: k8sV1.PodSpec{NodeSelector: map[string]string{"baz": "bar"}}},
			setNodeLabels(nil), false, true,
		},
		{"no selector terms or node labels", &k8sV1.Pod{}, &k8sV1.Node{}, true, true},
		{
			" pod spec defined + no subset match", &k8sV1.Pod{Spec: k8sV1.PodSpec{
				NodeSelector: map[string]string{"baz": "bar"},
			}},
			setNodeLabels(map[string]string{"foo": "bar", "baz": "boo"}), false, true,
		},
		{
			" pod spec defined + match", &k8sV1.Pod{Spec: k8sV1.PodSpec{
				NodeSelector: map[string]string{"foo": "bar"},
			}},
			setNodeLabels(map[string]string{"foo": "bar"}), true, true,
		},
		{
			"affinity pod spec defined + no match", &k8sV1.Pod{Spec: k8sV1.PodSpec{
				Affinity: &k8sV1.Affinity{NodeAffinity: &k8sV1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &k8sV1.NodeSelector{
						NodeSelectorTerms: []k8sV1.NodeSelectorTerm{{
							MatchFields: []k8sV1.NodeSelectorRequirement{{
								Key: "abc", Operator: k8sV1.NodeSelectorOpIn, Values: []string{"aaa", "bbb"},
							}},
						}},
					},
				}},
			}}, setNodeLabels(map[string]string{"abc": "ccc"}), true, false,
		},
		{
			"affinity pod spec defined + match", &k8sV1.Pod{Spec: k8sV1.PodSpec{
				Affinity: &k8sV1.Affinity{NodeAffinity: &k8sV1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &k8sV1.NodeSelector{
						NodeSelectorTerms: []k8sV1.NodeSelectorTerm{{
							MatchExpressions: []k8sV1.NodeSelectorRequirement{{
								Key: "abc", Operator: k8sV1.NodeSelectorOpIn, Values: []string{"aaa", "bbb"},
							}},
						}},
					},
				}},
			}}, setNodeLabels(map[string]string{"abc": "bbb"}), true, true,
		},
		{
			"affinity + selector pod spec defined + match", &k8sV1.Pod{Spec: k8sV1.PodSpec{
				NodeSelector: map[string]string{"foo": "bar"},
				Affinity: &k8sV1.Affinity{NodeAffinity: &k8sV1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &k8sV1.NodeSelector{
						NodeSelectorTerms: []k8sV1.NodeSelectorTerm{{
							MatchExpressions: []k8sV1.NodeSelectorRequirement{{
								Key: "abc", Operator: k8sV1.NodeSelectorOpIn, Values: []string{"aaa", "bbb"},
							}},
						}},
					},
				}},
			}}, setNodeLabels(map[string]string{"foo": "bar", "baz": "boo", "abc": "aaa"}), true, true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			j := newTestJobsService(t)
			selectors, affinities := extractNodeSelectors(tt.pod)

			// Make sure that the node selectors match as intended
			require.Equal(t, tt.selectorMatch, j.podsCanBeScheduledOnNode(selectors, tt.node))

			// Make sure that the node affinities match as intended
			require.Equal(t, tt.affinityMatch, j.podsCanBeScheduledOnNode(affinities, tt.node))
		})
	}
}

func TestGetNodeResourcePoolMappingWithNodeSelectors(t *testing.T) {
	auxNode1, auxNode2, compNode1, compNode2 := setupNodes()

	// Add the following labels/node selectors to each node
	auxNode1.SetLabels(map[string]string{
		"labelIn": "aaa", "fieldNotIn": "ccc",
	})
	auxNode2.SetLabels(map[string]string{
		"foo": "bar", "baz": "boo",
		"labelIn": "aaa", "fieldNotIn": "bbb",
	})
	compNode1.SetLabels(map[string]string{
		"foo":     "bar",
		"labelIn": "bbb", "fieldNotIn": "bbb",
	})
	compNode2.SetLabels(map[string]string{
		"foo": "bar", "abc": "def",
		"labelIn": "aaa", "fieldNotIn": "bbb",
	})

	// Create a test job service with multiple nodes & taints/tolerations/selectors
	gpuJobService := createMockJobsService(map[string]*k8sV1.Node{
		auxNode1Name:  auxNode1,
		auxNode2Name:  auxNode2,
		compNode1Name: compNode1,
		compNode2Name: compNode2,
	}, device.CUDA, true)

	cpuJobService := createMockJobsService(map[string]*k8sV1.Node{
		auxNode1Name:  auxNode1,
		auxNode2Name:  auxNode2,
		compNode1Name: compNode1,
		compNode2Name: compNode2,
	}, device.CPU, true)

	cases := []struct {
		name            string
		jobsService     *jobsService
		rpConfigs       []config.ResourcePoolConfig
		poolsToNodesLen map[string]int
		nodesToPoolsLen map[string]int
	}{
		{
			"empty case", cpuJobService,
			[]config.ResourcePoolConfig{{
				PoolName:              "default",
				TaskContainerDefaults: &model.TaskContainerDefaultsConfig{CPUPodSpec: &k8sV1.Pod{}},
			}},
			map[string]int{"default": 5},
			map[string]int{"NonDetermined": 1, "comp": 1, "comp2": 1, "aux": 1, "aux2": 1},
		},
		{
			"no selectors, 1 resource pool",
			gpuJobService,
			[]config.ResourcePoolConfig{testRPConfig(true, "default", nil, nil)},
			map[string]int{"default": 5},
			map[string]int{"NonDetermined": 1, "comp": 1, "comp2": 1, "aux": 1, "aux2": 1},
		},
		{
			// When only 1 resource pool config is defined, it also matches the non-Det pod.
			"only selectors, 1 resource pool", gpuJobService,
			[]config.ResourcePoolConfig{
				testRPConfig(false, "pool-1", map[string]string{"abc": "def"}, nil),
			},
			map[string]int{"pool-1": 2},
			map[string]int{"NonDetermined": 1, "comp2": 1},
		},
		{
			"only selectors, 2 resource pools", cpuJobService,
			[]config.ResourcePoolConfig{
				testRPConfig(true, "pool-1", map[string]string{"abc": "def"}, nil),
				testRPConfig(true, "pool-2", map[string]string{"foo": "bar"}, nil),
			},
			map[string]int{"pool-1": 1, "pool-2": 3},
			map[string]int{"comp": 1, "comp2": 2, "aux2": 1},
		},
		{
			// The nonDet pod matches to the gpu pool, with a cpu job service.
			"selectors, gpu + cpu pod Spec, cpu job service", cpuJobService,
			[]config.ResourcePoolConfig{
				testRPConfig(true, "cpu-pool", map[string]string{"foo": "bar"}, nil),
				testRPConfig(false, "gpu-pool", map[string]string{"foo": "bar"}, nil),
			},
			map[string]int{"cpu-pool": 3, "gpu-pool": 4},
			map[string]int{"NonDetermined": 1, "comp": 2, "comp2": 2, "aux2": 2},
		},
		{
			// The nonDet pod matches to the gpu pool, with a gpu job service.
			"selectors, gpu + cpu pod Spec, gpu job service", gpuJobService,
			[]config.ResourcePoolConfig{
				testRPConfig(true, "cpu-pool", map[string]string{"foo": "bar"}, nil),
				testRPConfig(false, "gpu-pool", map[string]string{"foo": "bar"}, nil),
			},
			map[string]int{"cpu-pool": 3, "gpu-pool": 4},
			map[string]int{"NonDetermined": 1, "comp": 2, "comp2": 2, "aux2": 2},
		},
		{
			// only the nonDet pod is matched, to the gpu pod service.
			"selectors, no match", cpuJobService,
			[]config.ResourcePoolConfig{
				testRPConfig(true, "cpu-pool", map[string]string{"foo": "baz"}, nil),
				testRPConfig(false, "gpu-pool", map[string]string{"foo": "abc"}, nil),
			},
			map[string]int{"gpu-pool": 1},
			map[string]int{"NonDetermined": 1},
		},
		{
			// mismatch between job service & rp config type, empty results.
			"mismatch between job service & rp config type", cpuJobService, []config.ResourcePoolConfig{
				testRPConfig(true, "cpu-pool", map[string]string{"foo": "abc"}, nil),
			}, map[string]int{}, map[string]int{},
		},
		{
			"node selectors + affinities, 1 resource pool", gpuJobService,
			[]config.ResourcePoolConfig{
				testRPConfig(false, "gpu-pool", map[string]string{"foo": "bar"}, []string{"aaa", "ccc"}),
			},
			map[string]int{"gpu-pool": 3},
			map[string]int{"NonDetermined": 1, "comp2": 1, "aux2": 1},
		},
		{
			"node selectors + affinities, 2 resource pools", gpuJobService,
			[]config.ResourcePoolConfig{
				testRPConfig(false, "gpu-pool1", map[string]string{"foo": "bar"}, []string{"aaa", "ccc"}),
				testRPConfig(false, "gpu-pool2", map[string]string{"baz": "boo"}, []string{"aaa"}),
			},
			map[string]int{"gpu-pool1": 3, "gpu-pool2": 2},
			map[string]int{"NonDetermined": 2, "aux2": 2, "comp2": 1},
		},

		{
			"only node affinities, 1 resource pool", gpuJobService, []config.ResourcePoolConfig{
				testRPConfig(false, "gpu-pool", nil, []string{"aaa"}),
			}, map[string]int{"gpu-pool": 4}, map[string]int{"NonDetermined": 1, "aux": 1, "aux2": 1, "comp2": 1},
		},
		{
			"only node affinities, 2 resource pool", gpuJobService, []config.ResourcePoolConfig{
				testRPConfig(false, "gpu-pool1", nil, []string{"bbb"}),
				testRPConfig(false, "gpu-pool2", nil, []string{"bbb", "aaa"}),
			}, map[string]int{"gpu-pool1": 1, "gpu-pool2": 2}, map[string]int{"NonDetermined": 2, "aux": 1},
		},
		{
			// Empty selectors won't match with anything. So this test case only matches the nonDet pod.
			"empty selectors + node affinities, 1 resource pool", gpuJobService, []config.ResourcePoolConfig{
				testRPConfig(false, "gpu-pool", map[string]string{}, []string{"aaa"}),
			}, map[string]int{"gpu-pool": 1}, map[string]int{"NonDetermined": 1},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Add the test resource pool configs to the jobs service
			tt.jobsService.resourcePoolConfigs = tt.rpConfigs

			// Set the node summaries.
			poolsToNodes, nodesToPools := tt.jobsService.getNodeResourcePoolMapping(
				map[string]model.AgentSummary{
					auxNode1Name: {Slots: model.SlotsSummary{"s1": model.SlotSummary{
						Device: device.Device{Type: device.CUDA},
					}}}, auxNode2Name: {Slots: model.SlotsSummary{"s1": model.SlotSummary{
						Device: device.Device{Type: device.CUDA},
					}}}, compNode1Name: {Slots: model.SlotsSummary{"s1": model.SlotSummary{
						Device: device.Device{Type: device.CUDA},
					}}}, compNode2Name: {Slots: model.SlotsSummary{"s1": model.SlotSummary{
						Device: device.Device{Type: device.CUDA},
					}}},
				})

			require.Lenf(t, poolsToNodes, len(tt.poolsToNodesLen),
				fmt.Sprintf("total pools found: %v", len(poolsToNodes)))
			for poolName, nodes := range poolsToNodes {
				require.Lenf(t, nodes, tt.poolsToNodesLen[poolName], "pool "+poolName)
			}

			require.Lenf(t, nodesToPools, len(tt.nodesToPoolsLen),
				fmt.Sprintf("total nodes found: %v", len(nodesToPools)))
			for nodeName, pools := range nodesToPools {
				require.Lenf(t, pools, tt.nodesToPoolsLen[nodeName], "node "+nodeName)
			}
		})
	}
}

func Test_readClientConfig(t *testing.T) {
	customPath := "test_kube.config"
	err := os.WriteFile(customPath, []byte(fakeKubeconfig), 0o600)
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(customPath); err != nil {
			t.Logf("failed to cleanup %s", err)
		}
	}()

	tests := []struct {
		name           string
		kubeconfigPath string
		want           string
	}{
		{
			name:           "fallback to in cluster config",
			kubeconfigPath: "",
			want:           "unable to load in-cluster configuration",
		},
		{
			name:           "custom kubeconfig",
			kubeconfigPath: customPath,
			want:           "",
		},
		{
			name:           "custom kubeconfig with homedir expansion at least tried the correct file",
			kubeconfigPath: "~",
			want:           "is a directory", // Bit clever, but we're sure we expanded it with this error.
		},
		{
			name:           "this test can actually fail",
			kubeconfigPath: "a_file_that_doesn't_exist.config",
			want:           "no such file or",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readClientConfig(tt.kubeconfigPath)
			if tt.want != "" {
				require.ErrorContains(t, err, tt.want)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func setNodeLabels(labels map[string]string) *k8sV1.Node {
	node := &k8sV1.Node{}
	node.SetLabels(labels)
	return node
}

func testRPConfig(cpu bool, name string, selectors map[string]string, affinitySet []string) config.ResourcePoolConfig {
	// If affinity variables are set, define the nodeSelectorTerms
	var ns []k8sV1.NodeSelectorTerm
	if affinitySet != nil {
		ns = []k8sV1.NodeSelectorTerm{{
			MatchExpressions: []k8sV1.NodeSelectorRequirement{
				{Key: "fieldNotIn", Operator: k8sV1.NodeSelectorOpNotIn, Values: affinitySet},
				{Key: "labelIn", Operator: k8sV1.NodeSelectorOpIn, Values: affinitySet},
			},
		}}
	}
	pod := &k8sV1.Pod{Spec: k8sV1.PodSpec{Affinity: &k8sV1.Affinity{NodeAffinity: &k8sV1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &k8sV1.NodeSelector{NodeSelectorTerms: ns},
	}}}}

	if selectors != nil {
		pod.Spec.NodeSelector = selectors
	}

	if cpu {
		return config.ResourcePoolConfig{
			PoolName:              name,
			TaskContainerDefaults: &model.TaskContainerDefaultsConfig{CPUPodSpec: pod},
		}
	}
	return config.ResourcePoolConfig{
		PoolName:              name,
		TaskContainerDefaults: &model.TaskContainerDefaultsConfig{GPUPodSpec: pod},
	}
}

var taintFooBar = k8sV1.Taint{
	Key:    "foo",
	Value:  "bar",
	Effect: k8sV1.TaintEffectNoSchedule,
}

const fakeKubeconfig = `
apiVersion: v1
clusters:
- cluster:
    extensions:
    - extension:
        last-update: Mon, 04 Mar 2024 18:53:00 EST
        provider: minikube.sigs.k8s.io
        version: v1.29.0
      name: cluster_info
    server: https://127.0.0.1:49216
  name: minikube
contexts:
- context:
    cluster: minikube
    extensions:
    - extension:
        last-update: Mon, 04 Mar 2024 18:53:00 EST
        provider: minikube.sigs.k8s.io
        version: v1.29.0
      name: context_info
    namespace: default
    user: minikube
  name: minikube
current-context: minikube
kind: Config
preferences: {}
`
