package kubernetesrm

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/determined-ai/determined/master/pkg/port"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"
	gateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1"
)

// I wanted to do this all in patches, but Gateways don't yet support strategic merge patch.
// Its pretty easy to use json patch to add the port, but removing the port is a lot harder
// as I don't think you can remove by value. Instead just serialize reading then submitting
// updates on the gateway.
type gatewayService struct {
	mu               sync.Mutex
	gatewayInterface gateway.GatewayInterface
	gatewayName      string
	portRange        *port.Range
}

func newGatewayService(gatewayInterface gateway.GatewayInterface, gatewayName string, portRangeStart, portRangeEnd int) (*gatewayService, error) {
	portRange, err := port.NewRange(portRangeStart, portRangeEnd, make([]int, 0))
	if err != nil {
		return nil, fmt.Errorf("creating port range: %w", err)
	}
	return &gatewayService{
		gatewayInterface: gatewayInterface,
		gatewayName:      gatewayName,
		portRange:        portRange,
	}, nil
}

func (g *gatewayService) addListeners(listeners []gatewayTyped.Listener) error {
	if err := g.updateGateway(func(gateway *gatewayTyped.Gateway) {
		gateway.Spec.Listeners = append(gateway.Spec.Listeners, listeners...)
	}); err != nil {
		return fmt.Errorf("adding listeners %+v to gateway: %w", listeners, err)
	}

	return nil
}

func (g *gatewayService) freePorts(ports []int) error {
	if err := g.updateGateway(func(gateway *gatewayTyped.Gateway) {
		var newListeners []gatewayTyped.Listener
		for _, l := range gateway.Spec.Listeners {
			if !slices.Contains(ports, int(l.Port)) {
				newListeners = append(newListeners, l)
			}
		}

		gateway.Spec.Listeners = newListeners
	}); err != nil {
		return fmt.Errorf("freeing ports %v from gateway: %w", ports, err)
	}

	return nil
}

func (g *gatewayService) updateGateway(update func(*gatewayTyped.Gateway)) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	gateway, err := g.gatewayInterface.Get(context.TODO(), g.gatewayName, metaV1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting gateway with name '%s': %w", g.gatewayName, err)
	}

	update(gateway)

	if _, err := g.gatewayInterface.
		Update(context.TODO(), gateway, metaV1.UpdateOptions{}); err != nil {
		return fmt.Errorf("updating gateway with name '%s': %w", g.gatewayName, err)
	}

	return nil
}
