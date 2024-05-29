//go:build integration

package kubernetesrm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/determined-ai/determined/master/internal/mocks"
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

	ns1 := &mocks.PodInterface{}
	ns1.On("List", mock.Anything, mock.Anything).Once().
		Return(&k8sV1.PodList{Items: append(hiddenPods, expectedPods[0])}, nil)

	ns2 := &mocks.PodInterface{}
	ns2.On("List", mock.Anything, mock.Anything).Once().
		Return(&k8sV1.PodList{Items: append(hiddenPods, expectedPods[1])}, nil)

	p := jobsService{
		podInterfaces: map[string]typedV1.PodInterface{
			"ns1": ns1,
			"ns2": ns2,
		},
	}

	actualPods, err := p.getNonDetPods()
	require.NoError(t, err)
	require.ElementsMatch(t, expectedPods, actualPods)
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
