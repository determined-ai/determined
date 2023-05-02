package kubernetesrm

import (
	"testing"

	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"
)

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
