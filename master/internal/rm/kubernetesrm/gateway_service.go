package kubernetesrm

import (
	"context"
	"fmt"
	"slices"
	"sync"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"
	gateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1"

	"github.com/determined-ai/determined/master/pkg/port"
)

// I wanted to do this all in patches, but Gateways don't yet support strategic merge patch.
// Its pretty easy to use json patch to add the port, but removing the port is a lot harder
// as I don't think you can remove by value. Instead just serialize reading then submitting
// updates on the gateway.
type gatewayService struct {
	mu               sync.Mutex
	gatewayInterface gateway.GatewayInterface
	gatewayName      string
}

type gatewayResourceComm struct {
	requestedPorts     int
	resourceDescriptor proxyResourceGenerator
	reportResources    func([]gatewayProxyResource)
}

func genSectionName(port int) string {
	return fmt.Sprintf("section-%d", port)
}

func newGatewayService(gatewayInterface gateway.GatewayInterface, gatewayName string) (*gatewayService, error) {
	// TODO: make port range configurable by user. We currently assume we own the controller and
	// the service.
	// DOCS: note this limit on number of active proxied tasks.
	// TODO: validate existing port on the gateway on startup
	g := &gatewayService{
		gatewayInterface: gatewayInterface,
		gatewayName:      gatewayName,
	}
	return g, nil
}

// PortMap is a map of pod ports to gateway ports.
type PortMap map[int]int

func (g *gatewayService) generateAndAddListeners(count int) ([]int, error) {
	var ports []int
	if err := g.updateGateway(func(gateway *gatewayTyped.Gateway) error {
		var err error
		listeners := make([]gatewayTyped.Listener, count)
		ports, err = pickNFreePorts(gateway, len(listeners))
		if err != nil {
			return err
		}
		for i := 0; i < count; i++ {
			listeners[i] = createListenerForPod(ports[i], genSectionName(ports[i]))
		}
		gateway.Spec.Listeners = append(gateway.Spec.Listeners, listeners...)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("adding %d listeners to gateway: %w", count, err)
	}
	return ports, nil
}

func pickNFreePorts(gateway *gatewayTyped.Gateway, count int) ([]int, error) {
	usedPorts := make([]int, 0, len(gateway.Spec.Listeners))
	for _, listener := range gateway.Spec.Listeners {
		usedPorts = append(usedPorts, int(listener.Port))
	}
	portRange, err := port.NewRange(49152, 65535, usedPorts)
	if err != nil {
		return nil, fmt.Errorf("creating port range: %w", err)
	}
	return portRange.GetAndMarkUsed(count)
}

func (g *gatewayService) freePorts(ports []int) error {
	if err := g.updateGateway(func(gateway *gatewayTyped.Gateway) error {
		var newListeners []gatewayTyped.Listener
		for _, l := range gateway.Spec.Listeners {
			if !slices.Contains(ports, int(l.Port)) {
				newListeners = append(newListeners, l)
			}
		}

		gateway.Spec.Listeners = newListeners
		return nil
	}); err != nil {
		return fmt.Errorf("freeing ports %v from gateway: %w", ports, err)
	}
	return nil
}

func (g *gatewayService) updateGateway(update func(*gatewayTyped.Gateway) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	gateway, err := g.gatewayInterface.Get(context.TODO(), g.gatewayName, metaV1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting gateway with name '%s': %w", g.gatewayName, err)
	}

	err = update(gateway)
	if err != nil {
		return err
	}

	if _, err := g.gatewayInterface.
		Update(context.TODO(), gateway, metaV1.UpdateOptions{}); err != nil {
		return fmt.Errorf("updating gateway with name '%s': %w", g.gatewayName, err)
	}

	return nil
}
