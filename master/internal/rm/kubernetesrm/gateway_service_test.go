package kubernetesrm

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"
	alphaGateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1alpha2"

	"github.com/determined-ai/determined/master/internal/mocks"
)

func TestGatewayServiceAddListeners(t *testing.T) {
	gatewayMock := &mocks.GatewayInterface{}
	TCPRouteMock := &mocks.TCPRouteInterface{}
	tcpInterfaces := map[string]alphaGateway.TCPRouteInterface{
		"default": TCPRouteMock,
	}
	g, err := newGatewayService(
		gatewayMock, tcpInterfaces, "gatewayname",
	)
	g.portRangeStart = 1
	require.NoError(t, err)

	toReturn := &gatewayTyped.Gateway{
		Spec: gatewayTyped.GatewaySpec{
			Listeners: []gatewayTyped.Listener{
				createListenerForPod(1),
			},
		},
	}

	expected := &gatewayTyped.Gateway{
		Spec: gatewayTyped.GatewaySpec{
			Listeners: []gatewayTyped.Listener{
				createListenerForPod(1),
				createListenerForPod(2),
				createListenerForPod(3),
			},
		},
	}

	gatewayMock.On("Get", mock.Anything, "gatewayname", metaV1.GetOptions{}).Return(toReturn, nil)
	gatewayMock.On("Update", mock.Anything, expected, metaV1.UpdateOptions{}).Return(nil, nil)
	ports, err := g.generateAndAddListeners(2)
	require.Len(t, ports, 2)
	require.Equal(t, []int{2, 3}, ports)
	require.NoError(t, err)
	gatewayMock.AssertExpectations(t)
}

func TestGatewayServiceFreePorts(t *testing.T) {
	gatewayMock := &mocks.GatewayInterface{}
	TCPRouteMock := &mocks.TCPRouteInterface{}
	tcpInterfaces := map[string]alphaGateway.TCPRouteInterface{
		"default": TCPRouteMock,
	}
	g, err := newGatewayService(
		gatewayMock, tcpInterfaces, "gatewayname",
	)
	require.NoError(t, err)

	toReturn := &gatewayTyped.Gateway{
		Spec: gatewayTyped.GatewaySpec{
			Listeners: []gatewayTyped.Listener{
				{
					Port: 1,
				},
				{
					Port: 2,
				},
				{
					Port: 3,
				},
			},
		},
	}

	expected := &gatewayTyped.Gateway{
		Spec: gatewayTyped.GatewaySpec{
			Listeners: []gatewayTyped.Listener{
				{
					Port: 2,
				},
			},
		},
	}

	gatewayMock.On("Get", mock.Anything, "gatewayname", metaV1.GetOptions{}).Return(toReturn, nil)
	gatewayMock.On("Update", mock.Anything, expected, metaV1.UpdateOptions{}).Return(nil, nil)
	require.NoError(t, g.freePorts([]int{1, 3, 4}))

	gatewayMock.AssertExpectations(t)
}

func TestGatewayServicePickPorts(t *testing.T) {
	testCases := []struct {
		name           string
		rangeStart     int
		rangeEnd       int
		usedPorts      []int
		requestedCount int
		expectedPorts  []int
	}{
		{
			name:           "no ports available",
			rangeStart:     1,
			rangeEnd:       9,
			usedPorts:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
			requestedCount: 1,
			expectedPorts:  nil,
		},
		{
			name:           "some ports available",
			rangeStart:     1,
			rangeEnd:       9,
			usedPorts:      []int{1, 2, 3, 4, 5, 6},
			requestedCount: 2,
			expectedPorts:  []int{7, 8},
		},
		{
			name:           "exact number of ports available",
			rangeStart:     1,
			rangeEnd:       9,
			usedPorts:      []int{1, 2, 3, 4, 5, 6, 7},
			requestedCount: 2,
			expectedPorts:  []int{8, 9},
		},
		{
			name:           "more requested than available",
			rangeStart:     1,
			rangeEnd:       5,
			usedPorts:      []int{1, 2},
			requestedCount: 4,
			expectedPorts:  nil,
		},
		{
			name:           "all ports available",
			rangeStart:     10,
			rangeEnd:       20,
			usedPorts:      []int{},
			requestedCount: 3,
			expectedPorts:  []int{10, 11, 12},
		},
		{
			name:           "some ports used in the middle of the range",
			rangeStart:     1,
			rangeEnd:       10,
			usedPorts:      []int{5, 6, 7},
			requestedCount: 3,
			expectedPorts:  []int{1, 2, 3},
		},
		{
			name:           "used ports at the end of range",
			rangeStart:     1,
			rangeEnd:       5,
			usedPorts:      []int{4, 5},
			requestedCount: 2,
			expectedPorts:  []int{1, 2},
		},
		{
			name:           "non-sequential used ports",
			rangeStart:     1,
			rangeEnd:       10,
			usedPorts:      []int{2, 4, 6, 8, 10},
			requestedCount: 3,
			expectedPorts:  []int{1, 3, 5},
		},
		{
			name:           "used ports outside range, enough ports available",
			rangeStart:     10,
			rangeEnd:       20,
			usedPorts:      []int{1, 2, 3, 11, 12},
			requestedCount: 3,
			expectedPorts:  []int{10, 13, 14},
		},
		{
			name:           "used ports outside range, not enough ports available",
			rangeStart:     10,
			rangeEnd:       15,
			usedPorts:      []int{1, 2, 3, 11, 12},
			requestedCount: 5,
			expectedPorts:  nil,
		},
		{
			name:           "used ports within and outside range",
			rangeStart:     20,
			rangeEnd:       30,
			usedPorts:      []int{15, 18, 21, 25, 28, 35},
			requestedCount: 4,
			expectedPorts:  []int{20, 22, 23, 24},
		},
		{
			name:           "all used ports outside range",
			rangeStart:     5,
			rangeEnd:       10,
			usedPorts:      []int{1, 2, 3, 11, 12, 13},
			requestedCount: 3,
			expectedPorts:  []int{5, 6, 7},
		},
		{
			name:           "mixed ports inside and outside range, exact count available",
			rangeStart:     100,
			rangeEnd:       105,
			usedPorts:      []int{95, 96, 100, 101, 102},
			requestedCount: 3,
			expectedPorts:  []int{103, 104, 105},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			listeners := []gatewayTyped.Listener{}
			for _, port := range tc.usedPorts {
				listeners = append(listeners, createListenerForPod(port))
			}
			gatewayGetReturn := &gatewayTyped.Gateway{
				Spec: gatewayTyped.GatewaySpec{
					Listeners: listeners,
				},
			}
			g := &gatewayService{
				portRangeStart: tc.rangeStart,
				portRangeEnd:   tc.rangeEnd,
			}
			ports, err := g.pickNFreePorts(gatewayGetReturn, tc.requestedCount)
			if tc.expectedPorts == nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expectedPorts, ports)
		})
	}
}
