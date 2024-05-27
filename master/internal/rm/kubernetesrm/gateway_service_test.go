package kubernetesrm

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/determined-ai/determined/master/internal/mocks"
)

func TestGatewayServiceAddListeners(t *testing.T) {
	gatewayMock := &mocks.GatewayInterface{}
	g, err := newGatewayService(
		gatewayMock, "gatewayname",
	)
	require.NoError(t, err)

	toReturn := &gatewayTyped.Gateway{
		Spec: gatewayTyped.GatewaySpec{
			Listeners: []gatewayTyped.Listener{
				{
					Port: 1,
				},
			},
		},
	}

	expected := &gatewayTyped.Gateway{
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

	gatewayMock.On("Get", mock.Anything, "gatewayname", metaV1.GetOptions{}).Return(toReturn, nil)
	gatewayMock.On("Update", mock.Anything, expected, metaV1.UpdateOptions{}).Return(nil, nil)
	listeners, err := g.generateAndAddListeners(2)
	require.Len(t, listeners, 2)
	require.NoError(t, err)

	gatewayMock.AssertExpectations(t)
}

func TestGatewayServiceFreePorts(t *testing.T) {
	gatewayMock := &mocks.GatewayInterface{}
	g, err := newGatewayService(
		gatewayMock, "gatewayname",
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
