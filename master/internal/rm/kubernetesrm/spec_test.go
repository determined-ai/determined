//nolint:exhaustruct
package kubernetesrm

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"

	k8sV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	alphaGatewayTyped "sigs.k8s.io/gateway-api/apis/v1alpha2"
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

func TestConfigureProxyResources(t *testing.T) {
	j := &job{
		namespace: "podnamespace",
		jobName:   "sharedName",
	}
	taskSpec := &tasks.TaskSpec{
		AllocationID: "allocID",
	}
	require.Nil(t, j.configureProxyResources(taskSpec))

	j.req = &sproto.AllocateRequest{
		ProxyPorts: []*sproto.ProxyPortConfig{
			{
				ServiceID: "test_unused",
				Port:      12345,
				ProxyTCP:  false, // We still want a TCP proxy.
			},
		},
	}
	j.internalTaskGWConfig = &config.InternalTaskGatewayConfig{
		GatewayName:      "gatewayname",
		GatewayNamespace: "gatewaynamespace",
	}

	svc := &k8sV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      j.jobName + "-0",
			Namespace: "podnamespace",
			Labels:    map[string]string{determinedLabel: "allocID"},
			Annotations: map[string]string{
				jobNameAnnotation: j.jobName,
			},
		},
		Spec: k8sV1.ServiceSpec{
			Ports: []k8sV1.ServicePort{
				{
					Protocol: k8sV1.ProtocolTCP,
					Port:     12345,
				},
			},
			Selector: map[string]string{determinedLabel: "allocID"},
			Type:     k8sV1.ServiceTypeClusterIP,
		},
	}

	tcp := &alphaGatewayTyped.TCPRoute{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      j.jobName + "-0",
			Namespace: "podnamespace",
			Labels:    map[string]string{determinedLabel: "allocID"},
			Annotations: map[string]string{
				jobNameAnnotation: j.jobName,
			},
		},
		Spec: alphaGatewayTyped.TCPRouteSpec{
			CommonRouteSpec: alphaGatewayTyped.CommonRouteSpec{
				ParentRefs: []alphaGatewayTyped.ParentReference{
					{
						Namespace: ptrs.Ptr(alphaGatewayTyped.Namespace("gatewaynamespace")),
						Name:      alphaGatewayTyped.ObjectName("gatewayname"),
						Port:      ptrs.Ptr(alphaGatewayTyped.PortNumber(12345)),
						SectionName: ptrs.Ptr(alphaGatewayTyped.SectionName(
							generateListenerName(12345),
						)),
					},
				},
			},
			Rules: []alphaGatewayTyped.TCPRouteRule{
				{
					BackendRefs: []alphaGatewayTyped.BackendRef{
						{
							BackendObjectReference: alphaGatewayTyped.BackendObjectReference{
								Name: alphaGatewayTyped.ObjectName(j.jobName + "-0"),
								Kind: ptrs.Ptr(alphaGatewayTyped.Kind("Service")),
								Port: ptrs.Ptr(alphaGatewayTyped.PortNumber(12345)),
							},
						},
					},
				},
			},
		},
	}

	listener := createListenerForPod(12345)

	require.Equal(t, []gatewayProxyResource{
		{
			serviceSpec:     svc,
			tcpRouteSpec:    tcp,
			gatewayListener: listener,
		},
	}, (j.configureProxyResources(taskSpec))([]int{12345}))
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
	copyPod := p.DeepCopy()
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")
	require.Equal(t, copyPod, p)
}

func TestAddDisallowedNodesToPodSpec(t *testing.T) {
	p := &k8sV1.Pod{}
	addNodeDisabledAffinityToPodSpec(p, "cluster-id")

	copyPod := p.DeepCopy()

	// No block list adds anything.
	addDisallowedNodesToPodSpec(&sproto.AllocateRequest{
		BlockedNodes: nil,
	}, p)
	require.Equal(t, copyPod, p)

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

func TestDetProxyThroughGatewayEnv(t *testing.T) {
	env := expconf.EnvironmentConfig{
		RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
			RawCPU: []string{},
		},
	}

	t.Run("with gateway", func(t *testing.T) {
		j := job{
			internalTaskGWConfig: &config.InternalTaskGatewayConfig{},
		}

		actual, err := j.configureEnvVars(make(map[string]string), env, device.CPU)
		require.NoError(t, err)
		require.Contains(t, actual, k8sV1.EnvVar{Name: "DET_PROXY_THROUGH_GATEWAY", Value: "true"})
	})

	t.Run("without gateway", func(t *testing.T) {
		j := job{
			internalTaskGWConfig: nil,
		}

		actual, err := j.configureEnvVars(make(map[string]string), env, device.CPU)
		require.NoError(t, err)
		var keys []string
		for _, a := range actual {
			keys = append(keys, a.Name)
		}
		require.NotContains(t, keys, "DET_PROXY_THROUGH_GATEWAY")
	})
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

	p := job{}
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

	p := job{}
	actual, err := p.configureEnvVars(make(map[string]string), env, device.CPU)
	require.NoError(t, err)
	require.Contains(t, actual, k8sV1.EnvVar{Name: "test", Value: expectedValue})
	require.Contains(t, actual, k8sV1.EnvVar{Name: "test2", Value: ""})
	require.Contains(t, actual, k8sV1.EnvVar{Name: "func", Value: "f(x)=x"})
}

func TestValidatePodLabelValues(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{"valid all alpha", "simpleCharacters", "simpleCharacters"},
		{"valid all alphanumeric", "simple4Characters", "simple4Characters"},
		{"valid contains non-alphanumeric", "simple-Characters.With_Other", "simple-Characters.With_Other"},
		{"invalid chars", "letters contain *@ other chars -=%", "letters_contain____other_chars"},
		{"invalid leading chars", "-%4-simpleCharacters0", "4-simpleCharacters0"},
		{"invalid trailing chars", "simple-Characters4%-.#", "simple-Characters4"},
		{
			"invalid too many chars", "simpleCharactersGoesOnForWayTooLong36384042444648505254565860-_AndThenSome",
			"simpleCharactersGoesOnForWayTooLong36384042444648505254565860",
		},
		{"invalid email-style input", "name@domain.com", "name_domain.com"},
		{"invalid chars only", "-.*$%#$...", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			testOutput, err := validatePodLabelValue(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.output, testOutput, tt.name+" failed")
		})
	}
}

func TestDetMasterEnvVar(t *testing.T) {
	cases := []struct {
		name            string
		masterTLSConfig model.TLSClientConfig
		masterScheme    string
		expected        string
	}{
		{
			"tls off",
			model.TLSClientConfig{Enabled: false},
			"", "http://example.com:1234",
		},
		{
			"tls on",
			model.TLSClientConfig{Enabled: true},
			"", "https://example.com:1234",
		},
		{
			"tls off scheme https",
			model.TLSClientConfig{Enabled: false},
			"https", "https://example.com:1234",
		},
		{
			"tls on scheme https",
			model.TLSClientConfig{Enabled: true},
			"https", "https://example.com:1234",
		},
		{
			"tls on scheme http",
			model.TLSClientConfig{Enabled: true},
			"http", "http://example.com:1234",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := expconf.EnvironmentConfig{
				RawEnvironmentVariables: &expconf.EnvironmentVariablesMap{
					RawCPU: []string{},
				},
			}

			j := &job{
				masterHost:      "example.com",
				masterPort:      1234,
				masterScheme:    c.masterScheme,
				masterTLSConfig: c.masterTLSConfig,
			}

			envVars, err := j.configureEnvVars(make(map[string]string), env, device.CPU)
			require.NoError(t, err)
			require.Contains(t, envVars, k8sV1.EnvVar{Name: "DET_MASTER", Value: c.expected})
		})
	}
}

func TestConfigureResourcesRequirements(t *testing.T) {
	cases := []struct {
		deviceType   device.Type
		resourceName k8sV1.ResourceName
		quantity     resource.Quantity
	}{
		{
			deviceType:   device.CPU,
			resourceName: "cpu",
			quantity:     *resource.NewMilliQuantity(3000, resource.DecimalSI),
		},
		{
			deviceType:   device.CUDA,
			resourceName: resourceTypeNvidia,
			quantity:     *resource.NewQuantity(3, resource.DecimalSI),
		},
		{
			deviceType:   device.ROCM,
			resourceName: resourceTypeAMD,
			quantity:     *resource.NewQuantity(3, resource.DecimalSI),
		},
	}
	for _, c := range cases {
		t.Run(string(c.deviceType), func(t *testing.T) {
			j := &job{
				slotType:             c.deviceType,
				slotsPerPod:          3,
				slotResourceRequests: config.PodSlotResourceRequests{CPU: 1.0},
			}

			expected := k8sV1.ResourceRequirements{
				Limits: map[k8sV1.ResourceName]resource.Quantity{
					c.resourceName: c.quantity,
				},
				Requests: map[k8sV1.ResourceName]resource.Quantity{
					c.resourceName: c.quantity,
				},
			}
			require.Equal(t, expected, j.configureResourcesRequirements())
		})
	}
}

func TestDeterminedLabels(t *testing.T) {
	// Fill out task spec.
	taskSpec := tasks.TaskSpec{
		Owner: &model.User{
			ID:       1,
			Username: "determined",
			Active:   true,
			Admin:    false,
		},
		Workspace:   "test-workspace",
		TaskType:    model.TaskTypeCommand,
		TaskID:      model.NewTaskID().String(),
		ContainerID: "container-id",
		ExtraPodLabels: map[string]string{
			"k1": "v1",
			"k2": "v2",
		},
	}

	p := job{
		req: &sproto.AllocateRequest{
			ResourcePool: "test-rp",
		},
	}

	// Define expectations.
	expectedLabels := map[string]string{
		determinedLabel:   taskSpec.AllocationID,
		userLabel:         taskSpec.Owner.Username,
		workspaceLabel:    taskSpec.Workspace,
		resourcePoolLabel: p.req.ResourcePool,
		taskTypeLabel:     string(taskSpec.TaskType),
		taskIDLabel:       taskSpec.TaskID,
		containerIDLabel:  taskSpec.ContainerID,
		allocationIDLabel: taskSpec.AllocationID,
	}
	for k, v := range taskSpec.ExtraPodLabels {
		expectedLabels[labelPrefix+k] = v
	}

	spec := p.configureJobSpec(
		&taskSpec, make([]k8sV1.Volume, 1), k8sV1.Container{},
		k8sV1.Container{}, make([]k8sV1.Container, 1), &k8sV1.Pod{}, "scheduler",
	)

	// Confirm pod spec has required labels.
	require.NotNil(t, spec)
	require.Equal(t, expectedLabels, spec.ObjectMeta.Labels)
}
