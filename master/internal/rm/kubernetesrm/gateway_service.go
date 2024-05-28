package kubernetesrm

import (
	"context"
	"fmt"
	"slices"
	"sync"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"
	gateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1"

	alphaGateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1alpha2"

	"github.com/determined-ai/determined/master/pkg/model"
)

// I wanted to do this all in patches, but Gateways don't yet support strategic merge patch.
// Its pretty easy to use json patch to add the port, but removing the port is a lot harder
// as I don't think you can remove by value. Instead just serialize reading then submitting
// updates on the gateway.
type gatewayService struct {
	mu                 sync.Mutex
	gatewayInterface   gateway.GatewayInterface
	tcpRouteInterfaces map[string]alphaGateway.TCPRouteInterface
	gatewayName        string
	portRangeStart     int
	portRangeEnd       int
}

// chat: these probably could use a better names or packaging.
type gatewayResourceComm struct {
	requestedPorts     int
	resourceDescriptor proxyResourceGenerator
	reportResources    func([]gatewayProxyResource)
}

func genSectionName(gwPort int) string {
	return fmt.Sprintf("section-%d", gwPort)
}

func newGatewayService(
	gatewayInterface gateway.GatewayInterface,
	tcpRouteInterfaces map[string]alphaGateway.TCPRouteInterface,
	gatewayName string,
) (*gatewayService, error) {
	// TODO: make port range configurable by user. We currently assume we own the controller and
	// the service.
	// DOCS: note this limit on number of active proxied tasks.
	// TODO: validate existing port on the gateway on startup
	g := &gatewayService{
		gatewayInterface:   gatewayInterface,
		tcpRouteInterfaces: tcpRouteInterfaces,
		gatewayName:        gatewayName,
		portRangeStart:     49152,
		portRangeEnd:       65535,
	}
	return g, nil
}

// PortMap is a map of internal pod ports to external gateway ports.
type PortMap map[int]int

func (g *gatewayService) generateAndAddListeners(count int) ([]int, error) {
	var ports []int
	if err := g.updateGateway(func(gateway *gatewayTyped.Gateway) error {
		var err error
		listeners := make([]gatewayTyped.Listener, count)
		ports, err = g.pickNFreePorts(gateway, len(listeners))
		if err != nil {
			return err
		}
		for i := 0; i < count; i++ {
			listeners[i] = createListenerForPod(ports[i])
		}
		gateway.Spec.Listeners = append(gateway.Spec.Listeners, listeners...)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("adding %d listeners to gateway: %w", count, err)
	}
	return ports, nil
}

func (g *gatewayService) pickNFreePorts(gateway *gatewayTyped.Gateway, count int) ([]int, error) {
	usedPorts := make(map[int]struct{}, len(gateway.Spec.Listeners))
	for _, listener := range gateway.Spec.Listeners {
		usedPorts[int(listener.Port)] = struct{}{}
	}
	ports := make([]int, 0, count)
	for port := g.portRangeStart; port <= g.portRangeEnd; port++ {
		if _, used := usedPorts[port]; !used {
			ports = append(ports, port)
			if len(ports) == count {
				return ports, nil
			}
		}
	}
	return nil, fmt.Errorf("not enough free ports in range %d-%d", g.portRangeStart, g.portRangeEnd)
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

// getDeployedPortMap returns a mapping of ports based on gw config in the cluster.
func (g *gatewayService) getDeployedPortMap() (map[model.AllocationID]PortMap, error) {
	rv := make(map[model.AllocationID]PortMap)

	for namespace, tcpRouteInterface := range g.tcpRouteInterfaces {
		tcpRoutes, err := tcpRouteInterface.List(context.TODO(), metaV1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing TCPRoutes in namespace '%s': %w", namespace, err)
		}

		for _, route := range tcpRoutes.Items {
			allocIDStr, ok := route.Labels[determinedLabel]
			if !ok {
				continue
			}
			allocID := model.AllocationID(allocIDStr)
			for _, rule := range route.Spec.Rules {
				for _, backendRef := range rule.BackendRefs {
					if _, ok := rv[allocID]; !ok {
						rv[allocID] = make(PortMap)
					}
					if backendRef.Port == nil {
						return nil, fmt.Errorf("TCPRoute '%s' has a nil port", route.Name)
					}
					if len(route.Spec.ParentRefs) != 1 {
						return nil, fmt.Errorf("TCPRoute '%s' has %d parent refs, expected 1",
							route.Name, len(route.Spec.ParentRefs))
					}
					rv[allocID][int(*backendRef.Port)] = int(*route.Spec.ParentRefs[0].Port)
				}
			}
		}
	}
	return rv, nil
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
