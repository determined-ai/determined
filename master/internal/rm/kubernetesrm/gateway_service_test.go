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
